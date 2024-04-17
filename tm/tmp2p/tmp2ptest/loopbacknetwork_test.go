package tmp2ptest_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/rollchains/gordian/tm/tmp2p/tmp2ptest"
)

func TestLoopbackNetwork_Compliance(t *testing.T) {
	tmp2ptest.TestNetworkCompliance(
		t,
		func(ctx context.Context, log *slog.Logger) (tmp2ptest.Network, error) {
			n := tmp2ptest.NewLoopbackNetwork(ctx, log)
			return &tmp2ptest.GenericNetwork[*tmp2ptest.LoopbackConnection]{
				Network: n,
			}, nil
		},
	)
}
