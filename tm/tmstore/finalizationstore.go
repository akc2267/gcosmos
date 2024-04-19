package tmstore

import (
	"context"

	"github.com/rollchains/gordian/tm/tmconsensus"
)

type FinalizationStore interface {
	SaveFinalization(
		ctx context.Context,
		height uint64, round uint32,
		blockHash string,
		vals []tmconsensus.Validator,
		appStateHash string,
	) error

	LoadFinalizationByHeight(ctx context.Context, height uint64) (
		round uint32,
		blockHash string,
		vals []tmconsensus.Validator,
		appStateHash string,
		err error,
	)
}