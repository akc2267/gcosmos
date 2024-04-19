package tmstatetest

import (
	"context"
	"log/slog"
	"testing"

	"github.com/rollchains/gordian/internal/gtest"
	"github.com/rollchains/gordian/tm/tmapp"
	"github.com/rollchains/gordian/tm/tmconsensus"
	"github.com/rollchains/gordian/tm/tmconsensus/tmconsensustest"
	"github.com/rollchains/gordian/tm/tmengine/internal/tmeil"
	"github.com/rollchains/gordian/tm/tmengine/internal/tmstate"
	"github.com/rollchains/gordian/tm/tmstore/tmmemstore"
)

// Fixture is a helper type to create a [tmstate.StateMachine] and its required inputs
// for tests involving a StateMachine.
type Fixture struct {
	Log *slog.Logger

	Fx *tmconsensustest.StandardFixture

	// Exposed on the fixture explicitly as the MockConsensusStrategy type.
	CStrat *tmconsensustest.MockConsensusStrategy

	RoundTimer *MockRoundTimer

	RoundViewInCh         chan tmconsensus.VersionedRoundView
	ToMirrorCh            chan tmeil.StateMachineRoundActionSet
	FinalizeBlockRequests chan tmapp.FinalizeBlockRequest

	Cfg tmstate.StateMachineConfig
}

func NewFixture(t *testing.T, nVals int) *Fixture {
	fx := tmconsensustest.NewStandardFixture(nVals)

	cStrat := tmconsensustest.NewMockConsensusStrategy()

	rt := new(MockRoundTimer)

	roundViewInCh := make(chan tmconsensus.VersionedRoundView)
	toMirrorCh := make(chan tmeil.StateMachineRoundActionSet)
	finReq := make(chan tmapp.FinalizeBlockRequest)

	return &Fixture{
		Log: gtest.NewLogger(t),

		Fx: fx,

		CStrat: cStrat,

		RoundTimer: rt,

		RoundViewInCh:         roundViewInCh,
		ToMirrorCh:            toMirrorCh,
		FinalizeBlockRequests: finReq,

		Cfg: tmstate.StateMachineConfig{
			// Default to the first signer.
			// Caller can set to nil or a different signer if desired.
			Signer: fx.PrivVals[0].Signer,

			SignatureScheme: fx.SignatureScheme,

			HashScheme: fx.HashScheme,

			Genesis: fx.DefaultGenesis(),

			ActionStore:       tmmemstore.NewActionStore(),
			FinalizationStore: tmmemstore.NewFinalizationStore(),

			RoundTimer: rt,

			ConsensusStrategy: cStrat,

			RoundViewInCh:          roundViewInCh,
			ToMirrorCh:             toMirrorCh,
			FinalizeBlockRequestCh: finReq,
		},
	}
}

func (f *Fixture) NewStateMachine(ctx context.Context) *tmstate.StateMachine {
	sm, err := tmstate.NewStateMachine(ctx, f.Log, f.Cfg)
	if err != nil {
		panic(err)
	}
	return sm
}

func (f *Fixture) EmptyVRV(h uint64, r uint32) tmconsensus.VersionedRoundView {
	vals := f.Fx.Vals()
	vs := tmconsensus.NewVoteSummary()
	vs.SetAvailablePower(vals)
	keyHash, powHash := f.Fx.ValidatorHashes()
	return tmconsensus.VersionedRoundView{
		RoundView: tmconsensus.RoundView{
			Height:                 h,
			Round:                  r,
			Validators:             vals,
			ValidatorPubKeyHash:    keyHash,
			ValidatorVotePowerHash: powHash,

			VoteSummary: vs,
		},
	}
}