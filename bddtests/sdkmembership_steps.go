/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DATA-DOG/godog"
	sdkapi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/sdk/provider"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/sdk/service"
)

// MembershipSDKSteps ...
type MembershipSDKSteps struct {
	BDDContext   *BDDContext
	membUsers    []provider.ChannelUser
	membProvider provider.Impl
	membService  *service.MembershipService
	membPeers    []sdkapi.Peer
}

// NewSDKMembershipSteps ...
func NewSDKMembershipSteps(context *BDDContext) *MembershipSDKSteps {
	return &MembershipSDKSteps{BDDContext: context}
}

func (t *MembershipSDKSteps) getSdkMembershipProvider(strArgs string) error {
	args := strings.Split(strArgs, ",")
	var cusers []provider.ChannelUser
	// for testing, the org is passed explicitly here, the real client will pass in the org through the client.organization in the configs
	// as for the user, we're using the org's ADMIN client, the real client may set a list of enrolled users (1 per channelID) that have the right invocation privileges
	u, ok := t.BDDContext.users[fmt.Sprintf("%s_%s", args[1], ADMIN)]
	if !ok {
		return fmt.Errorf("failed to retrieve user %s_%s from the context users list. Available users are: %s", args[1], ADMIN, t.BDDContext.users)
	}
	cusers = append(cusers, provider.ChannelUser{ChannelID: args[0], UserID: u})
	t.membUsers = cusers
	delay, err := strconv.Atoi(args[2])
	if err != nil {
		return errors.Wrapf(err, "Failed to convert time string into int")
	}
	impl := provider.New(t.BDDContext.ClientConfig(), t.membUsers, time.Duration(delay)*time.Second)
	if impl == nil {
		return errors.New("failed to create membership provider")
	}

	t.membProvider = *impl
	return nil
}

func (t *MembershipSDKSteps) getSdkMembershipService(channelID string) error {
	membershipService, err := t.membProvider.NewDiscoveryService(channelID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get new membership service")
	}
	s, ok := membershipService.(*service.MembershipService)
	if ok {
		t.membService = s
	} else {
		return errors.New("Failed to cast MembershipService")
	}
	return nil
}

func (t *MembershipSDKSteps) getPeers() error {
	p, err := t.membService.GetPeers()
	if err != nil {
		return errors.WithMessage(err, "Failed to get list of peers from membership service")
	}
	t.membPeers = p
	return nil
}

func (t *MembershipSDKSteps) verifyPeerFound(peerURL string) error {
	for _, p := range t.membPeers {
		if p.URL() == peerURL {
			return nil
		}
	}

	return fmt.Errorf("Did not find peer '%s' in the list returned from membership service", peerURL)
}

func (t *MembershipSDKSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(t.BDDContext.BeforeScenario)
	s.AfterScenario(t.BDDContext.AfterScenario)
	s.Step(`^client C1 creates a new membership service provider with args "([^"]*)"$`, t.getSdkMembershipProvider)
	s.Step(`^client C1 creates a new membership service with args "([^"]*)"$`, t.getSdkMembershipService)
	s.Step(`^client C1 calls GetPeers function on membership service`, t.getPeers)
	s.Step(`^response from membership service GetPeers function to client contains value "([^"]*)"$`, t.verifyPeerFound)
}
