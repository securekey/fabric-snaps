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
	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkapi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/sdk/provider"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/sdk/service"
)

// MembershipSDKSteps scenarios will build the membership provider and directly call the membership service
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
	u := t.BDDContext.OrgUser(args[1], ADMIN)

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

// this step shows an example of injecting a membership provider into the client sdk
func (t *MembershipSDKSteps) injectMembershipProviderIntoSDK() error {
	factory := DynamicMembershipProviderFactory{
		discoveryProvider: &t.membProvider,
	}

	// Create SDK setup for channel client with dynamic selection
	// This step is performed during the test to allow normal SDK-based initialized of the selection provider
	_, err := fabsdk.New(
		fabsdk.WithConfig(t.BDDContext.ClientConfig()),
		fabsdk.WithServicePkg(&factory))
	if err != nil {
		return errors.WithMessage(err, "Failed to create new SDK with membership service provider included")
	}
	return nil
}

// DynamicMembershipProviderFactory is configured with remote (sdk) membership provider
type DynamicMembershipProviderFactory struct {
	defsvc.ProviderFactory
	discoveryProvider sdkapi.DiscoveryProvider
}

// CreateSelectionProvider returns a new implementation of dynamic selection provider
func (f *DynamicMembershipProviderFactory) NewDiscoveryProvider(config apiconfig.Config) (sdkapi.DiscoveryProvider, error) {
	return f.discoveryProvider, nil
}

func (t *MembershipSDKSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(t.BDDContext.BeforeScenario)
	s.AfterScenario(t.BDDContext.AfterScenario)
	s.Step(`^client C1 creates a new membership service provider with args "([^"]*)"$`, t.getSdkMembershipProvider)
	s.Step(`^client C1 creates a new membership service with args "([^"]*)"$`, t.getSdkMembershipService)
	s.Step(`^client C1 calls GetPeers function on membership service`, t.getPeers)
	s.Step(`^response from membership service GetPeers function to client contains value "([^"]*)"$`, t.verifyPeerFound)
	s.Step(`^client C1 uses a factory to get a membership service provider and injects it into the sdk`, t.injectMembershipProviderIntoSDK)
}
