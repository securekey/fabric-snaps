/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channel

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	channelConfig "github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/msp"

	"github.com/hyperledger/fabric/protos/common"
	mb "github.com/hyperledger/fabric/protos/msp"
	ab "github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"
	protos_utils "github.com/hyperledger/fabric/protos/utils"

	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	fc "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/internal"
)

// configItems contains the configuration values retrieved from the Orderer Service
type configItems struct {
	msps        []*mb.MSPConfig
	anchorPeers []*fab.OrgAnchorPeer
	orderers    []string
	versions    *versions
}

// versions ...
type versions struct {
	ReadSet  *common.ConfigGroup
	WriteSet *common.ConfigGroup
	Channel  *common.ConfigGroup
}

// Initialize initializes the channel.
// Retrieves the configuration from the primary orderer and initializes this channel
// with those values. Optionally a configuration may be passed in to initialize this channel
// without making the call to the orderer.
// config_update: Optional - A serialized form of the protobuf configuration update.
func (c *Channel) Initialize(configUpdate []byte) error {

	if len(configUpdate) > 0 {
		var err error
		if _, err = c.loadConfigUpdate(configUpdate); err != nil {
			return fmt.Errorf("Unable to load config update envelope: %v", err)
		}
		return nil
	}

	configEnvelope, err := c.channelConfig()
	if err != nil {
		return fmt.Errorf("Unable to retrieve channel configuration from orderer service: %v", err)
	}

	_, err = c.loadConfigEnvelope(configEnvelope)
	if err != nil {
		return fmt.Errorf("Unable to load config envelope: %v", err)
	}
	c.initialized = true
	return nil
}

// LoadConfigUpdateEnvelope is a utility method to load this channel with configuration information
// from an Envelope that contains a Configuration.
// data: the envelope with the configuration update items.
// See /protos/common/configtx.proto
func (c *Channel) LoadConfigUpdateEnvelope(data []byte) error {
	logger.Debugf("loadConfigUpdateEnvelope - start")

	envelope := &common.Envelope{}
	err := proto.Unmarshal(data, envelope)
	if err != nil {
		return fmt.Errorf("Unable to unmarshal envelope: %v", err)
	}

	payload, err := protos_utils.ExtractPayload(envelope)
	if err != nil {
		return fmt.Errorf("Unable to extract payload from config update envelope: %v", err)
	}

	channelHeader, err := protos_utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return fmt.Errorf("Unable to extract channel header from config update payload: %v", err)
	}

	if common.HeaderType(channelHeader.Type) != common.HeaderType_CONFIG_UPDATE {
		return fmt.Errorf("Block must be of type 'CONFIG_UPDATE'")
	}

	configUpdateEnvelope := &common.ConfigUpdateEnvelope{}
	if err := proto.Unmarshal(payload.Data, configUpdateEnvelope); err != nil {
		return fmt.Errorf("Unable to unmarshal config update envelope: %v", err)
	}

	_, err = c.loadConfigUpdate(configUpdateEnvelope.ConfigUpdate)
	return err
}

func (c *Channel) initializeFromConfig(configItems *configItems) error {
	// TODO revisit this if
	if len(configItems.msps) > 0 {
		msps, err := c.loadMSPs(configItems.msps)
		if err != nil {
			return fmt.Errorf("unable to load MSPs from config: %v", err)
		}

		if err := c.mspManager.Setup(msps); err != nil {
			return fmt.Errorf("error calling Setup on MSPManager: %v", err)
		}
	}
	c.anchorPeers = configItems.anchorPeers

	// TODO should we create orderers and endorsing peers
	return nil
}

/**
* Queries for the current config block for this channel.
* This transaction will be made to the orderer.
* @returns {ConfigEnvelope} Object containing the configuration items.
* @see /protos/orderer/ab.proto
* @see /protos/common/configtx.proto
 */
func (c *Channel) channelConfig() (*common.ConfigEnvelope, error) {
	logger.Debugf("channelConfig - start for channel %s", c.name)

	// Get the newest block
	block, err := c.block(fc.NewNewestSeekPosition())
	if err != nil {
		return nil, err
	}
	logger.Debugf("channelConfig - Retrieved newest block number: %d\n", block.Header.Number)

	// Get the index of the last config block
	lastConfig, err := fc.GetLastConfigFromBlock(block)
	if err != nil {
		return nil, fmt.Errorf("Unable to get last config from block: %v", err)
	}
	logger.Debugf("channelConfig - Last config index: %d\n", lastConfig.Index)

	// Get the last config block
	block, err = c.block(fc.NewSpecificSeekPosition(lastConfig.Index))

	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve block at index %d: %v", lastConfig.Index, err)
	}
	logger.Debugf("channelConfig - Last config block number %d, Number of tx: %d", block.Header.Number, len(block.Data.Data))

	if len(block.Data.Data) != 1 {
		return nil, fmt.Errorf("Config block must only contain one transaction but contains %d", len(block.Data.Data))
	}

	envelope := &common.Envelope{}
	if err = proto.Unmarshal(block.Data.Data[0], envelope); err != nil {
		return nil, fmt.Errorf("Error extracting envelope from config block: %v", err)
	}
	payload := &common.Payload{}
	if err := proto.Unmarshal(envelope.Payload, payload); err != nil {
		return nil, fmt.Errorf("Error extracting payload from envelope: %s", err)
	}
	channelHeader := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, channelHeader); err != nil {
		return nil, fmt.Errorf("Error extracting payload from envelope: %s", err)
	}
	if common.HeaderType(channelHeader.Type) != common.HeaderType_CONFIG {
		return nil, fmt.Errorf("Block must be of type 'CONFIG'")
	}
	configEnvelope := &common.ConfigEnvelope{}
	if err := proto.Unmarshal(payload.Data, configEnvelope); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal config envelope: %v", err)
	}
	return configEnvelope, nil
}

func (c *Channel) loadMSPs(mspConfigs []*mb.MSPConfig) ([]msp.MSP, error) {
	logger.Debugf("loadMSPs - start number of msps=%d", len(mspConfigs))

	msps := []msp.MSP{}
	for _, config := range mspConfigs {
		mspType := msp.ProviderType(config.Type)
		if mspType != msp.FABRIC {
			return nil, fmt.Errorf("MSP Configuration object type not supported: %v", mspType)
		}
		if len(config.Config) == 0 {
			return nil, fmt.Errorf("MSP Configuration object missing the payload in the 'Config' property")
		}

		fabricConfig := &mb.FabricMSPConfig{}
		err := proto.Unmarshal(config.Config, fabricConfig)
		if err != nil {
			return nil, fmt.Errorf("Unable to unmarshal FabricMSPConfig from config value: %v", err)
		}

		if fabricConfig.Name == "" {
			return nil, fmt.Errorf("MSP Configuration does not have a name")
		}

		// with this method we are only dealing with verifying MSPs, not local MSPs. Local MSPs are instantiated
		// from user enrollment materials (see User class). For verifying MSPs the root certificates are always
		// required
		if len(fabricConfig.RootCerts) == 0 {
			return nil, fmt.Errorf("MSP Configuration does not have any root certificates required for validating signing certificates")
		}

		// get the application org names
		var orgs []string
		orgUnits := fabricConfig.OrganizationalUnitIdentifiers
		for _, orgUnit := range orgUnits {
			logger.Debugf("loadMSPs - found org of :: %s", orgUnit.OrganizationalUnitIdentifier)
			orgs = append(orgs, orgUnit.OrganizationalUnitIdentifier)
		}

		// TODO: Do something with orgs

		newMSP, err := msp.NewBccspMsp()
		if err != nil {
			return nil, fmt.Errorf("error creating new MSP: %v", err)
		}

		if err := newMSP.Setup(config); err != nil {
			return nil, fmt.Errorf("error in Setup of new MSP: %v", err)
		}

		mspID, _ := newMSP.GetIdentifier()
		logger.Debugf("loadMSPs - adding msp=%s", mspID)

		msps = append(msps, newMSP)
	}

	logger.Debugf("loadMSPs - loaded %d MSPs", len(msps))
	return msps, nil
}

func loadConfigPolicy(configItems *configItems, key string, versionsPolicy *common.ConfigPolicy, configPolicy *common.ConfigPolicy, groupName string, org string) error {
	logger.Debugf("loadConfigPolicy - %s - name: %s", groupName, key)
	logger.Debugf("loadConfigPolicy - %s - version: %d", groupName, configPolicy.Version)
	logger.Debugf("loadConfigPolicy - %s - mod_policy: %s", groupName, configPolicy.ModPolicy)

	versionsPolicy.Version = configPolicy.Version
	return loadPolicy(configItems, versionsPolicy, key, configPolicy.Policy, groupName, org)
}

func loadConfigGroup(configItems *configItems, versionsGroup *common.ConfigGroup, group *common.ConfigGroup, name string, org string, top bool) error {
	logger.Debugf("loadConfigGroup - %s - START groups Org: %s", name, org)
	if group == nil {
		return nil
	}

	logger.Debugf("loadConfigGroup - %s   - version %v", name, group.Version)
	logger.Debugf("loadConfigGroup - %s   - mod policy %s", name, group.ModPolicy)
	logger.Debugf("loadConfigGroup - %s - >> groups", name)

	groups := group.GetGroups()
	if groups != nil {
		versionsGroup.Groups = make(map[string]*common.ConfigGroup)
		for key, configGroup := range groups {
			logger.Debugf("loadConfigGroup - %s - found config group ==> %s", name, key)
			// The Application group is where config settings are that we want to find
			versionsGroup.Groups[key] = &common.ConfigGroup{}
			loadConfigGroup(configItems, versionsGroup.Groups[key], configGroup, name+"."+key, key, false)
		}
	} else {
		logger.Debugf("loadConfigGroup - %s - no groups", name)
	}
	logger.Debugf("loadConfigGroup - %s - << groups", name)

	logger.Debugf("loadConfigGroup - %s - >> values", name)
	values := group.GetValues()
	if values != nil {
		versionsGroup.Values = make(map[string]*common.ConfigValue)
		for key, configValue := range values {
			versionsGroup.Values[key] = &common.ConfigValue{}
			loadConfigValue(configItems, key, versionsGroup.Values[key], configValue, name, org)
		}
	} else {
		logger.Debugf("loadConfigGroup - %s - no values", name)
	}
	logger.Debugf("loadConfigGroup - %s - << values", name)

	logger.Debugf("loadConfigGroup - %s - >> policies", name)
	policies := group.GetPolicies()
	if policies != nil {
		versionsGroup.Policies = make(map[string]*common.ConfigPolicy)
		for key, configPolicy := range policies {
			versionsGroup.Policies[key] = &common.ConfigPolicy{}
			loadConfigPolicy(configItems, key, versionsGroup.Policies[key], configPolicy, name, org)
		}
	} else {
		logger.Debugf("loadConfigGroup - %s - no policies", name)
	}
	logger.Debugf("loadConfigGroup - %s - << policies", name)
	logger.Debugf("loadConfigGroup - %s - < group", name)
	return nil
}

func loadConfigValue(configItems *configItems, key string, versionsValue *common.ConfigValue, configValue *common.ConfigValue, groupName string, org string) error {
	logger.Infof("loadConfigValue - %s - START value name: %s", groupName, key)
	logger.Infof("loadConfigValue - %s   - version: %d", groupName, configValue.Version)
	logger.Infof("loadConfigValue - %s   - modPolicy: %s", groupName, configValue.ModPolicy)

	versionsValue.Version = configValue.Version

	switch key {
	case channelConfig.AnchorPeersKey:
		anchorPeers := &pb.AnchorPeers{}
		err := proto.Unmarshal(configValue.Value, anchorPeers)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal anchor peers from config value: %v", err)
		}

		logger.Debugf("loadConfigValue - %s   - AnchorPeers :: %s", groupName, anchorPeers)

		if len(anchorPeers.AnchorPeers) > 0 {
			for _, anchorPeer := range anchorPeers.AnchorPeers {
				oap := &fab.OrgAnchorPeer{Org: org, Host: anchorPeer.Host, Port: anchorPeer.Port}
				configItems.anchorPeers = append(configItems.anchorPeers, oap)
				logger.Debugf("loadConfigValue - %s   - AnchorPeer :: %s:%d:%s", groupName, oap.Host, oap.Port, oap.Org)
			}
		}
		break

	case channelConfig.MSPKey:
		mspConfig := &mb.MSPConfig{}
		err := proto.Unmarshal(configValue.Value, mspConfig)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal MSPConfig from config value: %v", err)
		}

		logger.Debugf("loadConfigValue - %s   - MSP found", groupName)

		mspType := msp.ProviderType(mspConfig.Type)
		if mspType != msp.FABRIC {
			return fmt.Errorf("unsupported MSP type: %v", mspType)
		}

		configItems.msps = append(configItems.msps, mspConfig)
		break

	case channelConfig.ConsensusTypeKey:
		consensusType := &ab.ConsensusType{}
		err := proto.Unmarshal(configValue.Value, consensusType)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal ConsensusType from config value: %v", err)
		}

		logger.Debugf("loadConfigValue - %s   - Consensus type value :: %s", groupName, consensusType.Type)
		// TODO: Do something with this value
		break

	case channelConfig.BatchSizeKey:
		batchSize := &ab.BatchSize{}
		err := proto.Unmarshal(configValue.Value, batchSize)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal BatchSize from config value: %v", err)
		}

		logger.Debugf("loadConfigValue - %s   - BatchSize  maxMessageCount :: %d", groupName, batchSize.MaxMessageCount)
		logger.Debugf("loadConfigValue - %s   - BatchSize  absoluteMaxBytes :: %d", groupName, batchSize.AbsoluteMaxBytes)
		logger.Debugf("loadConfigValue - %s   - BatchSize  preferredMaxBytes :: %d", groupName, batchSize.PreferredMaxBytes)
		// TODO: Do something with this value
		break

	case channelConfig.BatchTimeoutKey:
		batchTimeout := &ab.BatchTimeout{}
		err := proto.Unmarshal(configValue.Value, batchTimeout)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal BatchTimeout from config value: %v", err)
		}
		logger.Debugf("loadConfigValue - %s   - BatchTimeout timeout value :: %s", groupName, batchTimeout.Timeout)
		// TODO: Do something with this value
		break

	case channelConfig.ChannelRestrictionsKey:
		channelRestrictions := &ab.ChannelRestrictions{}
		err := proto.Unmarshal(configValue.Value, channelRestrictions)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal ChannelRestrictions from config value: %v", err)
		}
		logger.Debugf("loadConfigValue - %s   - ChannelRestrictions max_count value :: %d", groupName, channelRestrictions.MaxCount)
		// TODO: Do something with this value
		break

	case channelConfig.HashingAlgorithmKey:
		hashingAlgorithm := &common.HashingAlgorithm{}
		err := proto.Unmarshal(configValue.Value, hashingAlgorithm)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal HashingAlgorithm from config value: %v", err)
		}
		logger.Debugf("loadConfigValue - %s   - HashingAlgorithm names value :: %s", groupName, hashingAlgorithm.Name)
		// TODO: Do something with this value
		break

	case channelConfig.ConsortiumKey:
		consortium := &common.Consortium{}
		err := proto.Unmarshal(configValue.Value, consortium)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal Consortium from config value: %v", err)
		}
		logger.Debugf("loadConfigValue - %s   - Consortium names value :: %s", groupName, consortium.Name)
		// TODO: Do something with this value
		break

	case channelConfig.BlockDataHashingStructureKey:
		bdhstruct := &common.BlockDataHashingStructure{}
		err := proto.Unmarshal(configValue.Value, bdhstruct)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal BlockDataHashingStructure from config value: %v", err)
		}
		logger.Debugf("loadConfigValue - %s   - BlockDataHashingStructure width value :: %s", groupName, bdhstruct.Width)
		// TODO: Do something with this value
		break

	case channelConfig.OrdererAddressesKey:
		ordererAddresses := &common.OrdererAddresses{}
		err := proto.Unmarshal(configValue.Value, ordererAddresses)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal OrdererAddresses from config value: %v", err)
		}
		logger.Debugf("loadConfigValue - %s   - OrdererAddresses addresses value :: %s", groupName, ordererAddresses.Addresses)
		if len(ordererAddresses.Addresses) > 0 {
			for _, ordererAddress := range ordererAddresses.Addresses {
				configItems.orderers = append(configItems.orderers, ordererAddress)
			}
		}
		break

	default:
		logger.Debugf("loadConfigValue - %s   - value: %s", groupName, configValue.Value)
	}
	return nil
}

func loadPolicy(configItems *configItems, versionsPolicy *common.ConfigPolicy, key string, policy *common.Policy, groupName string, org string) error {

	policyType := common.Policy_PolicyType(policy.Type)

	switch policyType {
	case common.Policy_SIGNATURE:
		sigPolicyEnv := &common.SignaturePolicyEnvelope{}
		err := proto.Unmarshal(policy.Value, sigPolicyEnv)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal SignaturePolicyEnvelope from config policy: %v", err)
		}
		logger.Debugf("loadConfigPolicy - %s - policy SIGNATURE :: %v", groupName, sigPolicyEnv.Rule)
		// TODO: Do something with this value
		break

	case common.Policy_MSP:
		// TODO: Not implemented yet
		logger.Debugf("loadConfigPolicy - %s - policy :: MSP POLICY NOT PARSED ", groupName)
		break

	case common.Policy_IMPLICIT_META:
		implicitMetaPolicy := &common.ImplicitMetaPolicy{}
		err := proto.Unmarshal(policy.Value, implicitMetaPolicy)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal ImplicitMetaPolicy from config policy: %v", err)
		}
		logger.Debugf("loadConfigPolicy - %s - policy IMPLICIT_META :: %v", groupName, implicitMetaPolicy)
		// TODO: Do something with this value
		break

	default:
		return fmt.Errorf("Unknown Policy type: %v", policyType)
	}
	return nil
}

func (c *Channel) loadConfigUpdate(configUpdateBytes []byte) (*configItems, error) {

	configUpdate := &common.ConfigUpdate{}
	if err := proto.Unmarshal(configUpdateBytes, configUpdate); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal config update: %v", err)
	}
	logger.Debugf("loadConfigUpdate - channel ::" + configUpdate.ChannelId)

	readSet := configUpdate.ReadSet
	writeSet := configUpdate.WriteSet

	versions := &versions{
		ReadSet:  readSet,
		WriteSet: writeSet,
	}

	configItems := &configItems{
		msps:        []*mb.MSPConfig{},
		anchorPeers: []*fab.OrgAnchorPeer{},
		orderers:    []string{},
		versions:    versions,
	}

	err := loadConfigGroup(configItems, configItems.versions.ReadSet, readSet, "read_set", "", false)
	if err != nil {
		return nil, err
	}
	// do the write_set second so they update anything in the read set
	err = loadConfigGroup(configItems, configItems.versions.WriteSet, writeSet, "write_set", "", false)
	if err != nil {
		return nil, err
	}
	err = c.initializeFromConfig(configItems)
	if err != nil {
		return nil, fmt.Errorf("channel initialization error: %v", err)
	}

	//TODO should we create orderers and endorsing peers
	return configItems, nil
}

func (c *Channel) loadConfigEnvelope(configEnvelope *common.ConfigEnvelope) (*configItems, error) {

	group := configEnvelope.Config.ChannelGroup

	versions := &versions{
		Channel: &common.ConfigGroup{},
	}

	configItems := &configItems{
		msps:        []*mb.MSPConfig{},
		anchorPeers: []*fab.OrgAnchorPeer{},
		orderers:    []string{},
		versions:    versions,
	}

	err := loadConfigGroup(configItems, configItems.versions.Channel, group, "base", "", true)
	if err != nil {
		return nil, fmt.Errorf("Unable to load config items from channel group: %v", err)
	}

	err = c.initializeFromConfig(configItems)

	logger.Debugf("channel config: %v", configItems)

	return configItems, err
}
