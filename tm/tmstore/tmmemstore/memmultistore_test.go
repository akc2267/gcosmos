package tmmemstore_test

import (
	"testing"

	"github.com/rollchains/gordian/tm/tmconsensus/tmconsensustest"
	"github.com/rollchains/gordian/tm/tmstore/tmmemstore"
	"github.com/rollchains/gordian/tm/tmstore/tmstoretest"
)

// MemMultiStore embeds all the tmmemstore types,
// in order to run [tmstoretest.TestMultiStoreCompliance].
// Since all of the store types are intended to act independently,
// we do not anticipate the multi store tests failing for the MemMultiStore.
type MemMultiStore struct {
	*tmmemstore.ActionStore
	*tmmemstore.CommittedHeaderStore
	*tmmemstore.FinalizationStore
	*tmmemstore.MirrorStore
	*tmmemstore.RoundStore
	*tmmemstore.ValidatorStore
}

func NewMemMultiStore() *MemMultiStore {
	return &MemMultiStore{
		ActionStore:          tmmemstore.NewActionStore(),
		CommittedHeaderStore: tmmemstore.NewCommittedHeaderStore(),
		FinalizationStore:    tmmemstore.NewFinalizationStore(),
		MirrorStore:          tmmemstore.NewMirrorStore(),
		RoundStore:           tmmemstore.NewRoundStore(),
		ValidatorStore:       tmmemstore.NewValidatorStore(tmconsensustest.SimpleHashScheme{}),
	}
}

func TestMemMultiStoreCompliance(t *testing.T) {
	tmstoretest.TestMultiStoreCompliance(
		t,
		func(cleanup func(func())) (*MemMultiStore, error) {
			return NewMemMultiStore(), nil
		},
	)
}
