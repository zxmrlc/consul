package xds

import (
	"fmt"

	envoylistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoyhttprbac "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rbac/v2"
	envoyhttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoynetrbac "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/rbac/v2"
	envoyrbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2"
	envoymatcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/hashicorp/consul/agent/proxycfg"
	"github.com/hashicorp/consul/agent/structs"
	"github.com/hashicorp/go-hclog"
)

// Requires 1.12+
//
// L4: https://www.envoyproxy.io/docs/envoy/v1.12.0/configuration/listeners/network_filters/rbac_filter
//
// L7: https://www.envoyproxy.io/docs/envoy/v1.12.0/configuration/http/http_filters/rbac_filter
//
// https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/security/rbac_filter
func makeRBACNetworkFilter(cfgSnap *proxycfg.ConfigSnapshot, _ hclog.Logger) (*envoylistener.Filter, error) {
	rules, err := makeRBACRules(cfgSnap, false)
	if err != nil {
		return nil, err
	}

	cfg := &envoynetrbac.RBAC{
		StatPrefix: "connect_authz",
		Rules:      rules,
		// ShadowRules: rules, // TODO
	}
	return makeFilter("envoy.filters.network.rbac", cfg)
}

func makeRBACHTTPFilter(cfgSnap *proxycfg.ConfigSnapshot, _ hclog.Logger) (*envoyhttp.HttpFilter, error) {
	rules, err := makeRBACRules(cfgSnap, true)
	if err != nil {
		return nil, err
	}

	cfg := &envoyhttprbac.RBAC{
		Rules: rules,
		// ShadowRules: &rules, // TODO
	}
	return makeEnvoyHTTPFilter("envoy.filters.http.rbac", cfg)
}

func makeSpiffePattern(sourceNS, sourceName string) string {
	const anyPath = `[^/]+`
	switch {
	case sourceNS != structs.WildcardSpecifier && sourceName != structs.WildcardSpecifier:
		return fmt.Sprintf(`^spiffe://%s/ns/%s/dc/%s/svc/%s$`, anyPath, sourceNS, anyPath, sourceName)
	case sourceNS != structs.WildcardSpecifier && sourceName == structs.WildcardSpecifier:
		return fmt.Sprintf(`^spiffe://%s/ns/%s/dc/%s/svc/%s$`, anyPath, sourceNS, anyPath, anyPath)
	case sourceNS == structs.WildcardSpecifier && sourceName == structs.WildcardSpecifier:
		return fmt.Sprintf(`^spiffe://%s/ns/%s/dc/%s/svc/%s$`, anyPath, anyPath, anyPath, anyPath)
	default:
		panic("TODO(rbac-ixns): not possible")
	}
}

// Requires 1.12+
//
// L4: https://www.envoyproxy.io/docs/envoy/v1.12.0/configuration/listeners/network_filters/rbac_filter
//
// L7: https://www.envoyproxy.io/docs/envoy/v1.12.0/configuration/http/http_filters/rbac_filter
//
// https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/security/rbac_filter
func makeRBACRules(cfgSnap *proxycfg.ConfigSnapshot, isHTTP bool) (*envoyrbac.RBAC, error) {
	//TODO(rbac-ixns)

	// Note that we DON'T explicitly validate the trust-domain matches ours. See
	// the PR for this change for details.

	// TODO(banks): Implement revocation list checking here.

	// First build up just the basic principal matches.
	rbacIxns := make([]*rbacIntention, 0, len(cfgSnap.ConnectProxy.Intentions))
	for _, ixn := range cfgSnap.ConnectProxy.Intentions {
		pattern := makeSpiffePattern(ixn.SourceNS, ixn.SourceName)

		id := &envoyrbac.Principal{
			Identifier: &envoyrbac.Principal_Authenticated_{
				Authenticated: &envoyrbac.Principal_Authenticated{
					PrincipalName: &envoymatcher.StringMatcher{
						MatchPattern: &envoymatcher.StringMatcher_SafeRegex{
							SafeRegex: makeEnvoyRegexMatch(pattern),
						},
					},
				},
			},
		}

		rbacIxns = append(rbacIxns, &rbacIntention{
			IDs:        []*envoyrbac.Principal{id},
			Allow:      (ixn.Action == structs.IntentionActionAllow),
			Precedence: ixn.Precedence,
		})
	}

	// Normalize: if we are in default-deny then all intentions must be allows and vice versa

	var (
		rbacAction          envoyrbac.RBAC_Action
		policyDefaultAction = (!cfgSnap.IntentionDefaultAllow)
	)
	if cfgSnap.IntentionDefaultAllow {
		// The RBAC policies deny access to principals. The rest is allowed.
		// This is block-list style access control.
		rbacAction = envoyrbac.RBAC_DENY
		policyDefaultAction = false
	} else {
		// The RBAC policies grant access to principals. The rest is denied.
		// This is safe-list style access control. This is the default type.
		rbacAction = envoyrbac.RBAC_ALLOW
		policyDefaultAction = true
	}

	// First walk backwards and if we encounter a deny, add it to all
	// subsequent statements and mark the deny rule itself for erasure.
	if len(rbacIxns) > 0 {
		for i := len(rbacIxns) - 1; i >= 0; i-- {
			if rbacIxns[i].Allow != policyDefaultAction {
				notID := &envoyrbac.Principal{
					Identifier: &envoyrbac.Principal_NotId{
						NotId: rbacIxns[i].IDs[0],
					},
				}
				for j := i + 1; j < len(rbacIxns); j++ {
					if rbacIxns[j].Skip {
						continue
					}
					rbacIxns[j].IDs = append(rbacIxns[j].IDs, notID)
				}
				// since this is default-deny, any trailing denies will just evaporate
				rbacIxns[i].Skip = true // mark for deletion
			}
		}
	}
	// At this point precedence doesn't matter since all roads lead to allow.

	var principals []*envoyrbac.Principal
	for _, rbacIxn := range rbacIxns {
		if rbacIxn.Skip {
			continue
		}
		if rbacIxn.Allow != policyDefaultAction {
			panic("oh no")
		}

		if len(rbacIxn.IDs) == 0 {
			panic("oh no")
		} else if len(rbacIxn.IDs) == 1 {
			principals = append(principals, rbacIxn.IDs[0])
		} else {
			principals = append(principals, &envoyrbac.Principal{
				Identifier: &envoyrbac.Principal_AndIds{
					AndIds: &envoyrbac.Principal_Set{
						Ids: rbacIxn.IDs,
					},
				},
			})
		}
	}

	rbac := &envoyrbac.RBAC{
		Action: rbacAction,
	}
	if len(principals) > 0 {
		policy := &envoyrbac.Policy{
			Principals: principals,
			Permissions: []*envoyrbac.Permission{
				{Rule: &envoyrbac.Permission_Any{Any: true}},
			},
		}
		rbac.Policies = map[string]*envoyrbac.Policy{
			"consul-intentions": policy,
		}
	}

	return rbac, nil
}

type rbacIntention struct {
	IDs        []*envoyrbac.Principal // these will all be ANDed
	Allow      bool
	Precedence int
	Skip       bool
}
