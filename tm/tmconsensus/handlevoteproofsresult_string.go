// Code generated by "stringer -type HandleVoteProofsResult -trimprefix=HandleVoteProofs ."; DO NOT EDIT.

package tmconsensus

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[HandleVoteProofsAccepted-1]
	_ = x[HandleVoteProofsNoNewSignatures-2]
	_ = x[HandleVoteProofsEmpty-3]
	_ = x[HandleVoteProofsBadPubKeyHash-4]
	_ = x[HandleVoteProofsRoundTooOld-5]
	_ = x[HandleVoteProofsTooFarInFuture-6]
	_ = x[HandleVoteProofsInternalError-7]
}

const _HandleVoteProofsResult_name = "AcceptedNoNewSignaturesEmptyBadPubKeyHashRoundTooOldTooFarInFutureInternalError"

var _HandleVoteProofsResult_index = [...]uint8{0, 8, 23, 28, 41, 52, 66, 79}

func (i HandleVoteProofsResult) String() string {
	i -= 1
	if i >= HandleVoteProofsResult(len(_HandleVoteProofsResult_index)-1) {
		return "HandleVoteProofsResult(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _HandleVoteProofsResult_name[_HandleVoteProofsResult_index[i]:_HandleVoteProofsResult_index[i+1]]
}
