package xds

import (
	"path"
	"sort"
	"testing"

	"github.com/hashicorp/consul/acl"
	"github.com/hashicorp/consul/agent/proxycfg"
	"github.com/hashicorp/consul/agent/structs"
	testinf "github.com/mitchellh/go-testing-interface"
	"github.com/stretchr/testify/require"
)

func TestMakeRBACNetworkFilter(t *testing.T) {
	testIntention := func(srcNS, src, dstNS, dst string, action structs.IntentionAction) *structs.Intention {
		ixn := structs.TestIntention(t)
		ixn.SourceNS = srcNS
		ixn.SourceName = src
		ixn.DestinationNS = dstNS
		ixn.DestinationName = dst
		ixn.Action = action
		ixn.UpdatePrecedence()
		return ixn
	}
	sorted := func(ixns ...*structs.Intention) structs.Intentions {
		sort.Slice(ixns, func(i, j int) bool {
			return ixns[j].Precedence < ixns[i].Precedence
		})
		return structs.Intentions(ixns)
	}

	tests := map[string]struct {
		create func(t testinf.T) *proxycfg.ConfigSnapshot
	}{
		"default-deny-wildcard-allow": {
			create: func(_ testinf.T) *proxycfg.ConfigSnapshot {
				return &proxycfg.ConfigSnapshot{
					Intentions: sorted(
						testIntention("*", "*", "*", "*", structs.IntentionActionAllow),
					),
					DefaultACLPolicy: acl.Deny,
				}
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			snap := tt.create(t)

			filter, err := makeRBACNetworkFilter(snap, nil)
			require.NoError(t, err)

			gotJSON := protoToJSON(t, filter)

			require.JSONEq(t, golden(t, path.Join("rbac", name), gotJSON), gotJSON)
		})
	}
}
