package xds

import (
	"fmt"

	envoylistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoynetrbac "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/rbac/v2"
	envoyrbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
	celcommon "github.com/google/cel-go/common"
	celparser "github.com/google/cel-go/parser"
	"github.com/hashicorp/consul/acl"
	"github.com/hashicorp/consul/agent/proxycfg"
	"github.com/hashicorp/consul/agent/structs"
	"github.com/hashicorp/go-hclog"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// Requires 1.12+
//
// L4: https://www.envoyproxy.io/docs/envoy/v1.12.0/configuration/listeners/network_filters/rbac_filter
//
// L7: https://www.envoyproxy.io/docs/envoy/v1.12.0/configuration/http/http_filters/rbac_filter
//
// https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/security/rbac_filter
func makeRBACNetworkFilter(cfgSnap *proxycfg.ConfigSnapshot, _ hclog.Logger) (*envoylistener.Filter, error) {
	rules, err := makeRBACRules(cfgSnap)
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
func makeRBACRules(cfgSnap *proxycfg.ConfigSnapshot) (*envoyrbac.RBAC, error) {
	//TODO(rbac-ixns)

	// Note that we DON'T explicitly validate the trust-domain matches ours. See
	// the PR for this change for details.

	// TODO(banks): Implement revocation list checking here.

	var cm celMemory

	cm.SetIntentions(cfgSnap.Intentions)

	rbacAction := cm.FlattenForDefaultPolicy(cfgSnap.DefaultACLPolicy)
	policies, err := cm.GenerateRBACPolicies()
	if err != nil {
		return nil, err
	}

	return &envoyrbac.RBAC{
		Action:   rbacAction,
		Policies: policies,
	}, nil
}

func (m *celMemory) GenerateRBACPolicies() (map[string]*envoyrbac.Policy, error) {
	policies := make(map[string]*envoyrbac.Policy)

	env, err := cel.NewEnv(cel.Macros(m.MacroExpander()))
	if err != nil {
		return nil, err
	}

	for i, snip := range m.snippets {
		ast, issues := env.Parse(snip.identityClause)
		if issues.Err() != nil {
			// TODO(rbac): better error context
			return nil, issues.Err()
		}

		policy := &envoyrbac.Policy{
			Principals: []*envoyrbac.Principal{
				{Identifier: &envoyrbac.Principal_Any{Any: true}},
				// {Identifier: &envoyrbac.Principal_Authenticated_{
				// 	Authenticated: &envoyrbac.Principal_Authenticated{
				// 		PrincipalName: &envoymatcher.StringMatcher{
				// 			MatchPattern: &envoymatcher.StringMatcher_Regex{
				// 				Regex: principalPatt,
				// 			},
				// 		},
				// 	},
				// }},
			},
			Permissions: []*envoyrbac.Permission{
				{Rule: &envoyrbac.Permission_Any{Any: true}},
			},
			Condition: ast.Expr(),
		}

		policyID := fmt.Sprintf("consul-compiled-intentions-%d", i)

		policies[policyID] = policy
	}

	return policies, nil
}

type celMemory struct {
	snippets        []*celSnippet
	stringConstants []string
}

func (m *celMemory) AddStringConstant(v string) int {
	constantID := len(m.stringConstants)
	m.stringConstants = append(m.stringConstants, v)
	return constantID
}

func (m *celMemory) MacroExpander() celparser.Macro {
	return celparser.NewGlobalMacro("stringvar", 1, stringIDMacroExpander(m.stringConstants))
}

func (m *celMemory) SetIntentions(ixns structs.Intentions) error {
	m.stringConstants = make([]string, 0, len(ixns))
	for _, ixn := range ixns {
		// we only care about the source end here; the dest was taken care of by the rest

		snippet := &celSnippet{
			allow:      (ixn.Action == structs.IntentionActionAllow),
			precedence: ixn.Precedence,
		}

		snippet.identityClause = fmt.Sprintf(
			"matches(connection.uri_san_peer_certificate, stringvar(%d))",
			m.AddStringConstant(
				makeSpiffePattern(ixn.SourceNS, ixn.SourceName),
			),
		)

		m.snippets = append(m.snippets, snippet)
	}

	return nil
}

// Normalize: if we are in default-deny, all of our actual clauses must be allows
//
// IDENTITY is the only thing we care about here.
func (m *celMemory) FlattenForDefaultPolicy(defaultACLPolicy acl.EnforcementDecision) envoyrbac.RBAC_Action {
	if defaultACLPolicy == acl.Deny {
		// First we walk in descending precedence and squish denies into the
		// allows that follow. This makes sense because in the common case the
		// root will be default-deny, and our RBAC will be just listing all
		// conditions that are ALLOWED.
		if len(m.snippets) > 0 {
			mod := make([]*celSnippet, 0, len(m.snippets))
			for i, snip := range m.snippets {
				if snip.allow {
					mod = append(mod, snip)
				} else {
					for j := i + 1; j < len(m.snippets); j++ {
						snip2 := m.snippets[j]

						snip2.identityClause = "(" + snip2.identityClause + ") AND !(" + snip.identityClause + ")"
					}
					// since this is default-deny, any trailing denies will just evaporate
				}
			}
			m.snippets = mod
		}

		// At this point precedence doesn't matter since all roads lead to allow.

		// Note this feels a little bit backwards.
		//
		// The RBAC policies grant access to principals. The rest is denied.
		// This is safe-list style access control. This is the default type.
		return envoyrbac.RBAC_ALLOW

	} else {
		panic("TODO")

		// The RBAC policies deny access to principals. The rest is allowed.
		// This is block-list style access control.
		//
		// return envoyrbac.RBAC_DENY
	}
}

type celSnippet struct {
	identityClause string // CEL
	allow          bool
	precedence     int // TODO(rbac-ixns): use this to optimize how the collapse works
}

// stringIDMacroExpander returns a macro expander that accepts 1 numeric
// argument and looks that up by index in a table of strings and returns that
// instead.
func stringIDMacroExpander(stringsByID []string) celparser.MacroExpander {
	return func(eh celparser.ExprHelper, _ *exprpb.Expr, args []*exprpb.Expr) (*exprpb.Expr, *celcommon.Error) {
		constExpr, ok := args[0].ExprKind.(*exprpb.Expr_ConstExpr)
		if !ok {
			location := eh.OffsetLocation(args[0].Id)
			return nil, &common.Error{
				Message:  "argument must be a constant",
				Location: location,
			}
		}
		int64Const, ok := constExpr.ConstExpr.ConstantKind.(*exprpb.Constant_Int64Value)
		if !ok {
			location := eh.OffsetLocation(args[0].Id)
			return nil, &common.Error{
				Message:  "argument must be an int64 constant",
				Location: location,
			}
		}
		constantID := int(int64Const.Int64Value)
		if constantID < 0 || constantID >= len(stringsByID) {
			location := eh.OffsetLocation(args[0].Id)
			return nil, &common.Error{
				Message:  fmt.Sprintf("stringvar %d does not exist", constantID),
				Location: location,
			}
		}
		v := stringsByID[constantID]

		return eh.LiteralString(v), nil
	}
}
