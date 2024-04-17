package tmelink

import "github.com/rollchains/gordian/tm/tmconsensus"

// NetworkViewUpdate is a set of versioned round views, representing the engine's view of the network,
// that the engine is intended to send to the gossip strategy.
//
// The individual values may be nil during a particular send
// if the engine has already sent an up-to-date value to the gossip strategy.
type NetworkViewUpdate struct {
	Committing, Voting, NextRound *tmconsensus.VersionedRoundView

	// The other views are all standard,
	// but this view may be a little less obvious.
	// In the context of the Mirror component of the Engine,
	// as soon as the mirror detects a nil commit for a round,
	// it effectively discards that view and advances the round.
	// In doing so, it risks not distributing the precommits required
	// for all other validators to cross the threshold.
	//
	// In normal operations, one would expect a regular re-broadcast of current state
	// which would eventually bring validators to the same round;
	// but we can instead eagerly distribute the details that caused a round to vote nil.
	NilVotedRound *tmconsensus.VersionedRoundView
}
