package tmconsensus

import (
	"maps"
	"slices"

	"github.com/rollchains/gordian/gcrypto"
)

// RoundView is the engine's observed view of the state of a particular round.
//
// The RoundView may be on a later height and round, or with different validators,
// compared to the local state machine.
type RoundView struct {
	Height uint64
	Round  uint32

	Validators []Validator

	ValidatorPubKeyHash, ValidatorVotePowerHash string

	ProposedBlocks []ProposedBlock

	PrevoteProofs, PrecommitProofs map[string]gcrypto.CommonMessageSignatureProof

	VoteSummary VoteSummary
}

// Clone returns a RoundView, with values identical to v,
// and underlying slices and maps copied from v.
func (v *RoundView) Clone() RoundView {
	var prevoteClone map[string]gcrypto.CommonMessageSignatureProof
	if len(v.PrevoteProofs) > 0 {
		prevoteClone = make(map[string]gcrypto.CommonMessageSignatureProof, len(v.PrevoteProofs))
		for k, v := range v.PrevoteProofs {
			prevoteClone[k] = v.Clone()
		}
	}

	var precommitClone map[string]gcrypto.CommonMessageSignatureProof
	if len(v.PrecommitProofs) > 0 {
		precommitClone = make(map[string]gcrypto.CommonMessageSignatureProof, len(v.PrecommitProofs))
		for k, v := range v.PrecommitProofs {
			precommitClone[k] = v.Clone()
		}
	}

	return RoundView{
		Height: v.Height,
		Round:  v.Round,

		Validators: slices.Clone(v.Validators),

		ValidatorPubKeyHash:    v.ValidatorPubKeyHash,
		ValidatorVotePowerHash: v.ValidatorVotePowerHash,

		ProposedBlocks: slices.Clone(v.ProposedBlocks),

		PrevoteProofs:   prevoteClone,
		PrecommitProofs: precommitClone,

		VoteSummary: v.VoteSummary.Clone(),
	}
}

// Reset zeros out all the fields of the RoundView,
// retaining any allocated capacity for its slices and maps.
// This is helpful for reusing RoundView values to avoid unnecessary memory allocations.
func (v *RoundView) Reset() {
	v.Height = 0

	// Clear the slice to avoid retaining a reference
	// beyond length 0 but longer than the existing capacity.
	clear(v.Validators)
	v.Validators = v.Validators[:0]

	v.ValidatorPubKeyHash = ""
	v.ValidatorVotePowerHash = ""

	v.ResetForSameHeight()
	v.VoteSummary.Reset()
}

// ResetForSameHeight clears the round, proposed blocks, and vote information on v.
// It does not modify the height, validators, or validator hashes.
//
// This is intended to be used when it is known that a view is going to be reused in the same height,
// where it should be safe to keep the validator slice and validator hashes.
func (v *RoundView) ResetForSameHeight() {
	v.Round = 0

	clear(v.ProposedBlocks)
	v.ProposedBlocks = v.ProposedBlocks[:0]

	clear(v.PrevoteProofs)
	clear(v.PrecommitProofs)

	v.VoteSummary.ResetForSameHeight()
}

// VersionedRoundView is a superset of [RoundView]
// that contains version information,
// for use cases where a RoundView may be receiving live updates
// and a consumer may care to identify what has changed from one update to another.
//
// This type is used internally to the engine and exposed to the gossip strategy.
type VersionedRoundView struct {
	// Embedded network view for ease of access.
	RoundView

	// Overall version that gets incremented with each atomic change.
	// It is possible that the overall version is incremented once
	// while sub-versions are incremented multiple times.
	// It seems very unlikely that a view of a single height/round
	// would get anywhere close to 2^32 versions.
	Version uint32

	// There is no associated version for proposed blocks;
	// the length of the ProposedBlocks slice is effectively the version.

	// The overall version of the particular vote.
	// It seems very unlikely that a view of a single height/round
	// would get anywhere close to 2^32 versions.
	PrevoteVersion, PrecommitVersion uint32

	// The version of the votes we have seen for particular blocks.
	// This is independent of the overall vote version in the previous field.
	// If we see a vote for block A, then the map will contain A=>1,
	// and if that was the first update, the overall vote version may be 2
	// (because initial state is version 1).
	//
	// Then if another update occurs where we see an additional vote for A
	// and a new vote for B, this map may contain A=>2 and B=>1,
	// whereas the overall version may have been incremented from 2 to 3.
	PrevoteBlockVersions, PrecommitBlockVersions map[string]uint32
}

// Clone returns a VersionedRoundView, with values identical to v,
// and underlying slices and maps copied from v.
func (v *VersionedRoundView) Clone() VersionedRoundView {
	return VersionedRoundView{
		RoundView: v.RoundView.Clone(),

		Version: v.Version,

		PrevoteVersion:   v.PrevoteVersion,
		PrecommitVersion: v.PrecommitVersion,

		PrevoteBlockVersions:   maps.Clone(v.PrevoteBlockVersions),
		PrecommitBlockVersions: maps.Clone(v.PrecommitBlockVersions),
	}
}

// Reset zeros out all the fields of the VersionedRoundView,
// retaining any allocated capacity for its slices and maps.
// This is helpful for reusing RoundView values to avoid unnecessary memory allocations.
func (v *VersionedRoundView) Reset() {
	v.RoundView.Reset()

	v.resetVersions()
}

// ResetForSameHeight resets the version information on the VersionedRoundView
// and calls v.RoundView.ResetForSameHeight.
//
// This is particularly useful when there is existing validator information
// on v.RoundView that should not be discarded.
func (v *VersionedRoundView) ResetForSameHeight() {
	v.RoundView.ResetForSameHeight()

	v.resetVersions()
}

// resetVersions clears only the version data on v.
// It is called from both the hard Reset method and the softer ResetForSameHeight method.
func (v *VersionedRoundView) resetVersions() {
	v.Version = 0
	v.PrevoteVersion = 0
	v.PrecommitVersion = 0

	clear(v.PrevoteBlockVersions)
	clear(v.PrecommitBlockVersions)
}