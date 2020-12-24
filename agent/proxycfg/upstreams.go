package proxycfg

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/agent/cache"
	cachetype "github.com/hashicorp/consul/agent/cache-types"
	"github.com/hashicorp/consul/agent/structs"
)

// handlerUpstreams is used by some kindHandlers that need to be aware of upstreams.
// It is not itself a kindHandler.
type handlerUpstreams handler

func (s *handlerUpstreams) handleUpdateUpstreams(ctx context.Context, u cache.UpdateEvent, snap *ConfigSnapshotUpstreams) error {
	if u.Err != nil {
		return fmt.Errorf("error filling agent cache: %v", u.Err)
	}

	switch {
	case u.CorrelationID == leafWatchID:
		leaf, ok := u.Result.(*structs.IssuedCert)
		if !ok {
			return fmt.Errorf("invalid type for response: %T", u.Result)
		}
		snap.Leaf = leaf

	case strings.HasPrefix(u.CorrelationID, "discovery-chain:"):
		resp, ok := u.Result.(*structs.DiscoveryChainResponse)
		if !ok {
			return fmt.Errorf("invalid type for response: %T", u.Result)
		}
		svc := strings.TrimPrefix(u.CorrelationID, "discovery-chain:")
		snap.DiscoveryChain[svc] = resp.Chain

		if err := s.resetWatchesFromChain(ctx, svc, resp.Chain, snap); err != nil {
			return err
		}

	case strings.HasPrefix(u.CorrelationID, "upstream-target:"):
		resp, ok := u.Result.(*structs.IndexedCheckServiceNodes)
		if !ok {
			return fmt.Errorf("invalid type for response: %T", u.Result)
		}
		correlationID := strings.TrimPrefix(u.CorrelationID, "upstream-target:")
		targetID, svc, ok := removeColonPrefix(correlationID)
		if !ok {
			return fmt.Errorf("invalid correlation id %q", u.CorrelationID)
		}

		if _, ok := snap.WatchedUpstreamEndpoints[svc]; !ok {
			snap.WatchedUpstreamEndpoints[svc] = make(map[string]structs.CheckServiceNodes)
		}
		snap.WatchedUpstreamEndpoints[svc][targetID] = resp.Nodes

	case strings.HasPrefix(u.CorrelationID, "mesh-gateway:"):
		resp, ok := u.Result.(*structs.IndexedNodesWithGateways)
		if !ok {
			return fmt.Errorf("invalid type for response: %T", u.Result)
		}
		correlationID := strings.TrimPrefix(u.CorrelationID, "mesh-gateway:")
		dc, svc, ok := removeColonPrefix(correlationID)
		if !ok {
			return fmt.Errorf("invalid correlation id %q", u.CorrelationID)
		}
		if _, ok = snap.WatchedGatewayEndpoints[svc]; !ok {
			snap.WatchedGatewayEndpoints[svc] = make(map[string]structs.CheckServiceNodes)
		}
		snap.WatchedGatewayEndpoints[svc][dc] = resp.Nodes
	default:
		return fmt.Errorf("unknown correlation ID: %s", u.CorrelationID)
	}
	return nil
}

func removeColonPrefix(s string) (string, string, bool) {
	idx := strings.Index(s, ":")
	if idx == -1 {
		return "", "", false
	}
	return s[0:idx], s[idx+1:], true
}

func (s *handlerUpstreams) resetWatchesFromChain(
	ctx context.Context,
	id string,
	chain *structs.CompiledDiscoveryChain,
	snap *ConfigSnapshotUpstreams,
) error {
	s.logger.Trace("resetting watches for discovery chain", "id", id)
	if chain == nil {
		return fmt.Errorf("not possible to arrive here with no discovery chain")
	}

	// Initialize relevant sub maps.
	if _, ok := snap.WatchedUpstreams[id]; !ok {
		snap.WatchedUpstreams[id] = make(map[string]context.CancelFunc)
	}
	if _, ok := snap.WatchedUpstreamEndpoints[id]; !ok {
		snap.WatchedUpstreamEndpoints[id] = make(map[string]structs.CheckServiceNodes)
	}
	if _, ok := snap.WatchedGateways[id]; !ok {
		snap.WatchedGateways[id] = make(map[string]context.CancelFunc)
	}
	if _, ok := snap.WatchedGatewayEndpoints[id]; !ok {
		snap.WatchedGatewayEndpoints[id] = make(map[string]structs.CheckServiceNodes)
	}

	// We could invalidate this selectively based on a hash of the relevant
	// resolver information, but for now just reset anything about this
	// upstream when the chain changes in any way.
	//
	// TODO(rb): content hash based add/remove
	for targetID, cancelFn := range snap.WatchedUpstreams[id] {
		s.logger.Trace("stopping watch of target",
			"upstream", id,
			"chain", chain.ServiceName,
			"target", targetID,
		)
		delete(snap.WatchedUpstreams[id], targetID)
		delete(snap.WatchedUpstreamEndpoints[id], targetID)
		cancelFn()
	}

	needGateways := make(map[string]struct{})
	for _, target := range chain.Targets {
		s.logger.Trace("initializing watch of target",
			"upstream", id,
			"chain", chain.ServiceName,
			"target", target.ID,
			"mesh-gateway-mode", target.MeshGateway.Mode,
		)

		// We'll get endpoints from the gateway query, but the health still has
		// to come from the backing service query.
		switch target.MeshGateway.Mode {
		case structs.MeshGatewayModeRemote:
			needGateways[target.Datacenter] = struct{}{}
		case structs.MeshGatewayModeLocal:
			needGateways[s.source.Datacenter] = struct{}{}
		}

		ctx, cancel := context.WithCancel(ctx)
		err := s.watchConnectProxyService(ctx, "upstream-target:"+target.ID+":"+id, target)
		if err != nil {
			cancel()
			return err
		}

		snap.WatchedUpstreams[id][target.ID] = cancel
	}

	for dc := range needGateways {
		if _, ok := snap.WatchedGateways[id][dc]; ok {
			continue
		}

		s.logger.Trace("initializing watch of mesh gateway in datacenter",
			"upstream", id,
			"chain", chain.ServiceName,
			"datacenter", dc,
		)

		ctx, cancel := context.WithCancel(ctx)
		err := s.watchMeshGateway(ctx, dc, id)
		if err != nil {
			cancel()
			return err
		}

		snap.WatchedGateways[id][dc] = cancel
	}

	for dc, cancelFn := range snap.WatchedGateways[id] {
		if _, ok := needGateways[dc]; ok {
			continue
		}
		s.logger.Trace("stopping watch of mesh gateway in datacenter",
			"upstream", id,
			"chain", chain.ServiceName,
			"datacenter", dc,
		)
		delete(snap.WatchedGateways[id], dc)
		delete(snap.WatchedGatewayEndpoints[id], dc)
		cancelFn()
	}

	return nil
}

func (s *handlerUpstreams) watchMeshGateway(ctx context.Context, dc string, upstreamID string) error {
	return s.cache.Notify(ctx, cachetype.InternalServiceDumpName, &structs.ServiceDumpRequest{
		Datacenter:     dc,
		QueryOptions:   structs.QueryOptions{Token: s.token},
		ServiceKind:    structs.ServiceKindMeshGateway,
		UseServiceKind: true,
		Source:         *s.source,
		EnterpriseMeta: *structs.DefaultEnterpriseMeta(),
	}, "mesh-gateway:"+dc+":"+upstreamID, s.ch)
}

func (s *handlerUpstreams) watchConnectProxyService(ctx context.Context, correlationId string, target *structs.DiscoveryTarget) error {
	return s.stateConfig.cache.Notify(ctx, cachetype.HealthServicesName, &structs.ServiceSpecificRequest{
		Datacenter: target.Datacenter,
		QueryOptions: structs.QueryOptions{
			Token:  s.serviceInstance.token,
			Filter: target.Subset.Filter,
		},
		ServiceName: target.Service,
		Connect:     true,
		// Note that Identifier doesn't type-prefix for service any more as it's
		// the default and makes metrics and other things much cleaner. It's
		// simpler for us if we have the type to make things unambiguous.
		Source:         *s.stateConfig.source,
		EnterpriseMeta: *target.GetEnterpriseMetadata(),
	}, correlationId, s.ch)
}
