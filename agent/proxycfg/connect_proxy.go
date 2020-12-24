package proxycfg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/consul/agent/cache"
	cachetype "github.com/hashicorp/consul/agent/cache-types"
	"github.com/hashicorp/consul/agent/structs"
)

type handlerConnectProxy handler

// initWatchesConnectProxy sets up the watches needed based on current proxy registration
// state.
func (s *handlerConnectProxy) initialize(ctx context.Context) (ConfigSnapshot, error) {
	var snap ConfigSnapshot
	// Watch for root changes
	err := s.cache.Notify(ctx, cachetype.ConnectCARootName, &structs.DCSpecificRequest{
		Datacenter:   s.source.Datacenter,
		QueryOptions: structs.QueryOptions{Token: s.token},
		Source:       *s.source,
	}, rootsWatchID, s.ch)
	if err != nil {
		return snap, err
	}

	// Watch the leaf cert
	err = s.cache.Notify(ctx, cachetype.ConnectCALeafName, &cachetype.ConnectCALeafRequest{
		Datacenter:     s.source.Datacenter,
		Token:          s.token,
		Service:        s.proxyCfg.DestinationServiceName,
		EnterpriseMeta: s.proxyID.EnterpriseMeta,
	}, leafWatchID, s.ch)
	if err != nil {
		return snap, err
	}

	// Watch for intention updates
	err = s.cache.Notify(ctx, cachetype.IntentionMatchName, &structs.IntentionQueryRequest{
		Datacenter:   s.source.Datacenter,
		QueryOptions: structs.QueryOptions{Token: s.token},
		Match: &structs.IntentionQueryMatch{
			Type: structs.IntentionMatchDestination,
			Entries: []structs.IntentionMatchEntry{
				{
					Namespace: s.proxyID.NamespaceOrDefault(),
					Name:      s.proxyCfg.DestinationServiceName,
				},
			},
		},
	}, intentionsWatchID, s.ch)
	if err != nil {
		return snap, err
	}

	// Watch for service check updates
	err = s.cache.Notify(ctx, cachetype.ServiceHTTPChecksName, &cachetype.ServiceHTTPChecksRequest{
		ServiceID:      s.proxyCfg.DestinationServiceID,
		EnterpriseMeta: s.proxyID.EnterpriseMeta,
	}, svcChecksWatchIDPrefix+structs.ServiceIDString(s.proxyCfg.DestinationServiceID, &s.proxyID.EnterpriseMeta), s.ch)
	if err != nil {
		return snap, err
	}

	// default the namespace to the namespace of this proxy service
	currentNamespace := s.proxyID.NamespaceOrDefault()

	// Watch for updates to service endpoints for all upstreams
	for _, u := range s.proxyCfg.Upstreams {
		dc := s.source.Datacenter
		if u.Datacenter != "" {
			dc = u.Datacenter
		}

		ns := currentNamespace
		if u.DestinationNamespace != "" {
			ns = u.DestinationNamespace
		}

		cfg, err := parseReducedUpstreamConfig(u.Config)
		if err != nil {
			// Don't hard fail on a config typo, just warn. We'll fall back on
			// the plain discovery chain if there is an error so it's safe to
			// continue.
			s.logger.Warn("failed to parse upstream config",
				"upstream", u.Identifier(),
				"error", err,
			)
		}

		switch u.DestinationType {
		case structs.UpstreamDestTypePreparedQuery:
			err = s.cache.Notify(ctx, cachetype.PreparedQueryName, &structs.PreparedQueryExecuteRequest{
				Datacenter:    dc,
				QueryOptions:  structs.QueryOptions{Token: s.token, MaxAge: defaultPreparedQueryPollInterval},
				QueryIDOrName: u.DestinationName,
				Connect:       true,
				Source:        *s.source,
			}, "upstream:"+u.Identifier(), s.ch)
			if err != nil {
				return snap, err
			}

		case structs.UpstreamDestTypeService:
			fallthrough

		case "": // Treat unset as the default Service type
			err = s.cache.Notify(ctx, cachetype.CompiledDiscoveryChainName, &structs.DiscoveryChainRequest{
				Datacenter:             s.source.Datacenter,
				QueryOptions:           structs.QueryOptions{Token: s.token},
				Name:                   u.DestinationName,
				EvaluateInDatacenter:   dc,
				EvaluateInNamespace:    ns,
				OverrideMeshGateway:    s.proxyCfg.MeshGateway.OverlayWith(u.MeshGateway),
				OverrideProtocol:       cfg.Protocol,
				OverrideConnectTimeout: cfg.ConnectTimeout(),
			}, "discovery-chain:"+u.Identifier(), s.ch)
			if err != nil {
				return snap, err
			}

		default:
			return snap, fmt.Errorf("unknown upstream type: %q", u.DestinationType)
		}
	}

	snap = newConfigSnapshotFromServiceInstance(s.serviceInstance, s.stateConfig)
	snap.ConnectProxy.DiscoveryChain = make(map[string]*structs.CompiledDiscoveryChain)
	snap.ConnectProxy.WatchedUpstreams = make(map[string]map[string]context.CancelFunc)
	snap.ConnectProxy.WatchedUpstreamEndpoints = make(map[string]map[string]structs.CheckServiceNodes)
	snap.ConnectProxy.WatchedGateways = make(map[string]map[string]context.CancelFunc)
	snap.ConnectProxy.WatchedGatewayEndpoints = make(map[string]map[string]structs.CheckServiceNodes)
	snap.ConnectProxy.WatchedServiceChecks = make(map[structs.ServiceID][]structs.CheckType)
	snap.ConnectProxy.PreparedQueryEndpoints = make(map[string]structs.CheckServiceNodes)
	return snap, nil
}

// reducedUpstreamConfig represents the basic opaque config values that are now
// managed with the discovery chain but for backwards compatibility reasons
// should still affect how the proxy is configured.
//
// The full-blown config is agent/xds.UpstreamConfig
type reducedUpstreamConfig struct {
	Protocol         string `mapstructure:"protocol"`
	ConnectTimeoutMs int    `mapstructure:"connect_timeout_ms"`
}

func (c *reducedUpstreamConfig) ConnectTimeout() time.Duration {
	return time.Duration(c.ConnectTimeoutMs) * time.Millisecond
}

func parseReducedUpstreamConfig(m map[string]interface{}) (reducedUpstreamConfig, error) {
	var cfg reducedUpstreamConfig
	err := mapstructure.WeakDecode(m, &cfg)
	return cfg, err
}

func (s *handlerConnectProxy) handleUpdate(ctx context.Context, u cache.UpdateEvent, snap *ConfigSnapshot) error {
	if u.Err != nil {
		return fmt.Errorf("error filling agent cache: %v", u.Err)
	}

	switch {
	case u.CorrelationID == rootsWatchID:
		roots, ok := u.Result.(*structs.IndexedCARoots)
		if !ok {
			return fmt.Errorf("invalid type for response: %T", u.Result)
		}
		snap.Roots = roots
	case u.CorrelationID == intentionsWatchID:
		resp, ok := u.Result.(*structs.IndexedIntentionMatches)
		if !ok {
			return fmt.Errorf("invalid type for response: %T", u.Result)
		}
		if len(resp.Matches) > 0 {
			// RPC supports matching multiple services at once but we only ever
			// query with the one service we represent currently so just pick
			// the one result set up.
			snap.ConnectProxy.Intentions = resp.Matches[0]
		}
		snap.ConnectProxy.IntentionsSet = true

	case strings.HasPrefix(u.CorrelationID, "upstream:"+preparedQueryIDPrefix):
		resp, ok := u.Result.(*structs.PreparedQueryExecuteResponse)
		if !ok {
			return fmt.Errorf("invalid type for response: %T", u.Result)
		}
		pq := strings.TrimPrefix(u.CorrelationID, "upstream:")
		snap.ConnectProxy.PreparedQueryEndpoints[pq] = resp.Nodes

	case strings.HasPrefix(u.CorrelationID, svcChecksWatchIDPrefix):
		resp, ok := u.Result.([]structs.CheckType)
		if !ok {
			return fmt.Errorf("invalid type for service checks response: %T, want: []structs.CheckType", u.Result)
		}
		svcID := structs.ServiceIDFromString(strings.TrimPrefix(u.CorrelationID, svcChecksWatchIDPrefix))
		snap.ConnectProxy.WatchedServiceChecks[svcID] = resp
	default:
		return (*handlerUpstreams)(s).handleUpdateUpstreams(ctx, u, &snap.ConnectProxy.ConfigSnapshotUpstreams)
	}
	return nil
}
