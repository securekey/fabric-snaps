/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package pgresolver

import (
	"math/rand"
	"time"
)

type randomLBP struct {
}

// NewRandomLBP returns a random load-balance policy
func NewRandomLBP() LoadBalancePolicy {
	return &randomLBP{}
}

func (lbp *randomLBP) Choose(peerGroups []PeerGroup) PeerGroup {
	logger.Debugf("Invoking random LBP\n")

	if len(peerGroups) == 0 {
		logger.Warningf("No available peer groups\n")
		// Return an empty PeerGroup
		return NewPeerGroup()
	}

	rand.Seed(int64(time.Now().Nanosecond()))
	index := rand.Intn(len(peerGroups))

	logger.Debugf("randomLBP - Choosing index %d\n", index)
	return peerGroups[index]
}

type roundRobinLBP struct {
	index int
}

// NewRoundRobinLBP returns a round-robin load-balance policy
func NewRoundRobinLBP() LoadBalancePolicy {
	return &roundRobinLBP{index: -1}
}

func (lbp *roundRobinLBP) Choose(peerGroups []PeerGroup) PeerGroup {
	if len(peerGroups) == 0 {
		logger.Warningf("No available peer groups\n")
		// Return an empty PeerGroup
		return NewPeerGroup()
	}

	if lbp.index == -1 {
		rand.Seed(int64(time.Now().Nanosecond()))
		lbp.index = rand.Intn(len(peerGroups))
	} else {
		lbp.index++
	}

	if lbp.index >= len(peerGroups) {
		lbp.index = 0
	}

	logger.Debugf("roundRobinLBP - Choosing index %d\n", lbp.index)

	return peerGroups[lbp.index]
}
