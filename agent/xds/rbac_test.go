package xds

import (
	"path/filepath"
	"sort"
	"testing"

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
		"default-deny-double-wildcard-allow": {
			create: func(_ testinf.T) *proxycfg.ConfigSnapshot {
				snap := &proxycfg.ConfigSnapshot{
					IntentionDefaultAllow: false,
				}
				snap.ConnectProxy.Intentions = sorted(
					testIntention("*", "*", "default", "api", structs.IntentionActionAllow),
				)
				snap.ConnectProxy.IntentionsSet = true
				return snap
			},
		},
		"default-allow-double-wildcard-deny": {
			create: func(_ testinf.T) *proxycfg.ConfigSnapshot {
				snap := &proxycfg.ConfigSnapshot{
					IntentionDefaultAllow: true,
				}
				snap.ConnectProxy.Intentions = sorted(
					testIntention("*", "*", "default", "api", structs.IntentionActionDeny),
				)
				snap.ConnectProxy.IntentionsSet = true
				return snap
			},
		},
		"default-deny-one-allow": {
			create: func(_ testinf.T) *proxycfg.ConfigSnapshot {
				snap := &proxycfg.ConfigSnapshot{
					IntentionDefaultAllow: false,
				}
				snap.ConnectProxy.Intentions = sorted(
					testIntention("default", "web", "default", "api", structs.IntentionActionAllow),
				)
				snap.ConnectProxy.IntentionsSet = true
				return snap
			},
		},
		"default-deny-allow-deny-allow": {
			create: func(_ testinf.T) *proxycfg.ConfigSnapshot {
				snap := &proxycfg.ConfigSnapshot{
					IntentionDefaultAllow: false,
				}
				snap.ConnectProxy.Intentions = sorted(
					testIntention("default", "web", "default", "api", structs.IntentionActionAllow),
					testIntention("default", "*", "default", "api", structs.IntentionActionDeny),
					testIntention("*", "*", "default", "api", structs.IntentionActionAllow),
				)
				snap.ConnectProxy.IntentionsSet = true
				return snap
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

			require.JSONEq(t, golden(t, filepath.Join("rbac", name), "", gotJSON), gotJSON)
		})
	}
}
