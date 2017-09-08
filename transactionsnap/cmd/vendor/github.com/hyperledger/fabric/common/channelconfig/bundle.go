/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channelconfig

import (
	"fmt"

	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/configtx"
	configtxapi "github.com/hyperledger/fabric/common/configtx/api"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"

	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("common/channelconfig")

// RootGroupKey is the key for namespacing the channel config, especially for
// policy evaluation.
const RootGroupKey = "Channel"

// Bundle is a collection of resources which will always have a consistent
// view of the channel configuration.  In particular, for a given bundle reference,
// the config sequence, the policy manager etc. will always return exactly the
// same value.  The Bundle structure is immutable and will always be replaced in its
// entirety, with new backing memory.
type Bundle struct {
	policyManager   policies.Manager
	mspManager      msp.MSPManager
	channelConfig   *ChannelConfig
	configtxManager configtxapi.Manager
}

// PolicyManager returns the policy manager constructed for this config
func (b *Bundle) PolicyManager() policies.Manager {
	return b.policyManager
}

// MSPManager returns the MSP manager constructed for this config
func (b *Bundle) MSPManager() msp.MSPManager {
	return b.channelConfig.MSPManager()
}

// ChannelConfig returns the config.Channel for the chain
func (b *Bundle) ChannelConfig() Channel {
	return b.channelConfig
}

// OrdererConfig returns the config.Orderer for the channel
// and whether the Orderer config exists
func (b *Bundle) OrdererConfig() (Orderer, bool) {
	result := b.channelConfig.OrdererConfig()
	return result, result != nil
}

// ConsortiumsConfig() returns the config.Consortiums for the channel
// and whether the consortiums config exists
func (b *Bundle) ConsortiumsConfig() (Consortiums, bool) {
	result := b.channelConfig.ConsortiumsConfig()
	return result, result != nil
}

// ApplicationConfig returns the configtxapplication.SharedConfig for the channel
// and whether the Application config exists
func (b *Bundle) ApplicationConfig() (Application, bool) {
	result := b.channelConfig.ApplicationConfig()
	return result, result != nil
}

// ConfigtxManager returns the configtx.Manager for the channel
func (b *Bundle) ConfigtxManager() configtxapi.Manager {
	return b.configtxManager
}

// ValidateNew checks if a new bundle's contained configuration is valid to be derived from the current bundle.
// This allows checks of the nature "Make sure that the consensus type did not change." which is otherwise
func (b *Bundle) ValidateNew(nb Resources) error {
	if oc, ok := b.OrdererConfig(); ok {
		noc, ok := nb.OrdererConfig()
		if !ok {
			return fmt.Errorf("Current config has orderer section, but new config does not")
		}

		if oc.ConsensusType() != noc.ConsensusType() {
			return fmt.Errorf("Attempted to change consensus type from %s to %s", oc.ConsensusType(), noc.ConsensusType())
		}

		for orgName, org := range oc.Organizations() {
			norg, ok := noc.Organizations()[orgName]
			if !ok {
				continue
			}
			mspID := org.MSPID()
			if mspID != norg.MSPID() {
				return fmt.Errorf("Orderer org %s attempted to change MSP ID from %s to %s", orgName, mspID, norg.MSPID())
			}
		}
	}

	if ac, ok := b.ApplicationConfig(); ok {
		nac, ok := nb.ApplicationConfig()
		if !ok {
			return fmt.Errorf("Current config has consortiums section, but new config does not")
		}

		for orgName, org := range ac.Organizations() {
			norg, ok := nac.Organizations()[orgName]
			if !ok {
				continue
			}
			mspID := org.MSPID()
			if mspID != norg.MSPID() {
				return fmt.Errorf("Application org %s attempted to change MSP ID from %s to %s", orgName, mspID, norg.MSPID())
			}
		}
	}

	if cc, ok := b.ConsortiumsConfig(); ok {
		ncc, ok := nb.ConsortiumsConfig()
		if !ok {
			return fmt.Errorf("Current config has consortiums section, but new config does not")
		}

		for consortiumName, consortium := range cc.Consortiums() {
			nconsortium, ok := ncc.Consortiums()[consortiumName]
			if !ok {
				continue
			}

			for orgName, org := range consortium.Organizations() {
				norg, ok := nconsortium.Organizations()[orgName]
				if !ok {
					continue
				}
				mspID := org.MSPID()
				if mspID != norg.MSPID() {
					return fmt.Errorf("Consortium %s org %s attempted to change MSP ID from %s to %s", consortiumName, orgName, mspID, norg.MSPID())
				}
			}
		}
	}

	return nil
}

type simpleProposer struct {
	policyManager policies.Manager
}

// RootGroupKey returns RootGroupKey constant
func (sp simpleProposer) RootGroupKey() string {
	return RootGroupKey
}

// PolicyManager() returns the policy manager for considering config changes
func (sp simpleProposer) PolicyManager() policies.Manager {
	return sp.policyManager
}

// NewBundleFromEnvelope wraps the NewBundle function, extracting the needed
// information from a full configtx
func NewBundleFromEnvelope(env *cb.Envelope) (*Bundle, error) {
	payload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal payload from envelope")
	}

	configEnvelope, err := configtx.UnmarshalConfigEnvelope(payload.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config envelope from payload")
	}

	if payload.Header == nil {
		return nil, fmt.Errorf("envelope header cannot be nil")
	}

	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal channel header")
	}

	return NewBundle(chdr.ChannelId, configEnvelope.Config)
}

// NewBundle creates a new immutable bundle of configuration
func NewBundle(channelID string, config *cb.Config) (*Bundle, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.ChannelGroup == nil {
		return nil, fmt.Errorf("config must contain a channel group")
	}

	channelConfig, err := NewChannelConfig(config.ChannelGroup)
	if err != nil {
		return nil, errors.Wrap(err, "initializing channelconfig failed")
	}

	policyProviderMap := make(map[int32]policies.Provider)
	for pType := range cb.Policy_PolicyType_name {
		rtype := cb.Policy_PolicyType(pType)
		switch rtype {
		case cb.Policy_UNKNOWN:
			// Do not register a handler
		case cb.Policy_SIGNATURE:
			policyProviderMap[pType] = cauthdsl.NewPolicyProvider(channelConfig.MSPManager())
		case cb.Policy_MSP:
			// Add hook for MSP Handler here
		}
	}

	policyManager, err := policies.NewManagerImpl(RootGroupKey, policyProviderMap, config.ChannelGroup)
	if err != nil {
		return nil, errors.Wrap(err, "initializing policymanager failed")
	}

	env, err := utils.CreateSignedEnvelope(cb.HeaderType_CONFIG, channelID, nil, &cb.ConfigEnvelope{Config: config}, 0, 0)
	if err != nil {
		return nil, errors.Wrap(err, "creating envelope for configtx manager failed")
	}

	configtxManager, err := configtx.NewManagerImpl(env, simpleProposer{
		policyManager: policyManager,
	})
	if err != nil {
		return nil, errors.Wrap(err, "initializing configtx manager failed")
	}

	return &Bundle{
		policyManager:   policyManager,
		channelConfig:   channelConfig,
		configtxManager: configtxManager,
	}, nil
}
