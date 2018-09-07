/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cliconfig

import (
	"fmt"
	"strconv"
	"time"

	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/msp"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite/bccsp/multisuite"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/pflag"
)

const (
	// ConfigSnapID is the name/ID of the configuration snap
	ConfigSnapID = "configurationsnap"

	loggerName = "configcli"
)

// Flags
const (
	loggingLevelFlag        = "logging-level"
	loggingLevelDescription = "Logging level - ERROR, WARN, INFO, DEBUG"
	defaultLoggingLevel     = "ERROR"

	clientConfigFileFlag        = "clientconfig"
	clientConfigFileDescription = "The path of the client SDK's config.yaml file"
	defaultClientConfigFile     = ""

	userFlag        = "user"
	userDescription = "The user used to connect to Fabric"
	defaultUser     = ""

	passwordFlag        = "pw"
	passwordDescription = "The password of the user"
	defaultPassword     = ""

	channelIDFlag        = "cid"
	channelIDDescription = "The channel ID"
	defaultChannelID     = ""

	orgIDFlag        = "orgid"
	orgIDDescription = "A comma-separated set of peers, e.g. org1,org2. If specified then the request is made to all peers for the given organization"
	defaultOrgID     = ""

	peerURLFlag        = "peerurl"
	peerURLDescription = "A comma-separated list of target peers to query/update, e.g. 'grpcs://localhost:7051,grpcs://localhost:8051'"
	defaultPeerURL     = ""

	configKeyFlag        = "configkey"
	configKeyDescription = "The config key in JSON format. Example: {\"MspID\":\"Org1MSP\",\"PeerID\":\"peer0.org1.example.com\",\"AppName\":\"app1\",\"Version\":\"1\"}"

	configFlag        = "config"
	configDescription = "The config update string in JSON format. Example: {\"MspID\":\"Org1MSP\",\"Peers\":[{\"PeerID\":\"peer0.org1.example.com\",\"App\":[{\"AppName\":\"myapp\",\"Version\":\"1\",\"Config\":\"some config\"}]}]}"

	configFileFlag        = "configfile"
	configFileDescription = "The path to the config file"

	timeoutFlag        = "timeout"
	timeoutDescription = "The timeout (in milliseconds) for the operation"
	defaultTimeout     = "3000"

	outputFormatFlag        = "format"
	outputFormatDescription = "The output format - display, raw"
	defaultOutputFormat     = "formatted"

	mspIDFlag        = "mspid"
	mspIDDescription = "The ID of the MSP"
	defaultMSPID     = ""

	peerIDFlag        = "peerid"
	peerIDDescription = "The ID of the peer to query for"

	appNameFlag        = "appname"
	appNameDescription = "The name of the application to query for"

	appVerFlag        = "appver"
	appVerDescription = "The app version"

	componentNameFlag        = "componentname"
	componentNameDescription = "The name of the component to query for"

	componentVerFlag        = "componentver"
	componentVerDescription = "The component version"

	noPromptFlag        = "noprompt"
	noPromptDescription = "If specified then update and delete operations will not prompt for confirmation"
	defaultNoPrompt     = false

	keyType            = "keyType"
	keyTypeDescription = "Key type to be used to generate CSR"

	ephemeral            = "ephemeral"
	ephemeralDescription = "To be used in generate CSR - default false - long lived keys"
	ephemeralDefault     = "false"

	sigAlg            = "sigAlg"
	sigAlgDescription = "Signature Algorithm used to generate CSR"

	csrCommonName     = "csrCommonName"
	csrCommonNameDesc = "CSR common name"
)

var opts *options
var instance *CLIConfig

type options struct {
	user             string
	password         string
	loggingLevel     string
	orgID            string
	channelID        string
	peerURL          string
	clientConfigFile string
	configFile       string
	configKey        string
	config           string
	timeout          int64
	outputFormat     string
	mspID            string
	peerID           string
	appName          string
	appVer           string
	componentName    string
	componentVer     string
	noPrompt         bool
	keyType          string
	ephemeralFlag    string
	sigAlg           string
	csrCommonName    string
}

func init() {
	opts = &options{
		user:         defaultUser,
		password:     defaultPassword,
		loggingLevel: defaultLoggingLevel,
		channelID:    defaultChannelID,
	}
}

// CLIConfig overrides certain configuration values with those supplied on the command-line
type CLIConfig struct {
	fabApi.EndpointConfig
	mspApi.IdentityConfig
	core.CryptoSuiteConfig
	logger *logging.Logger
}

// InitConfig initializes the configuration
func InitConfig() error {
	instance = &CLIConfig{
		logger: logging.NewLogger(loggerName),
	}

	if opts.clientConfigFile == "" {
		return errors.New(errors.GeneralError, "no client config file specified")
	}

	provider := config.FromFile(opts.clientConfigFile)
	cnfg, err := provider()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "error loading the configs")
	}

	cryptoConfig := cryptosuite.ConfigFromBackend(cnfg...)

	cryptoSuiteProvider, err := multisuite.GetSuiteByConfig(cryptoConfig)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "error getting cryptosuite")
	}
	err = cryptosuite.SetDefault(cryptoSuiteProvider)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "error setting default cryptosuite")
	}

	endpointConfig, err := fab.ConfigFromBackend(cnfg...)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "from backend returned error")
	}
	identityConfig, err := msp.ConfigFromBackend(cnfg...)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "from backend returned error")
	}
	instance.EndpointConfig = endpointConfig
	instance.IdentityConfig = identityConfig
	instance.CryptoSuiteConfig = cryptoConfig
	return nil
}

// Config returns the CLI configuration
func Config() *CLIConfig {
	return instance
}

// Logger returns the Logger for the CLI tool
func (c *CLIConfig) Logger() *logging.Logger {
	return c.logger
}

// LoggingLevel specifies the logging level (DEBUG, INFO, WARNING, ERROR, or CRITICAL)
func (c *CLIConfig) LoggingLevel() string {
	return opts.loggingLevel
}

// InitLoggingLevel initializes the logging level from the provided arguments
func InitLoggingLevel(flags *pflag.FlagSet) {
	flags.StringVar(&opts.loggingLevel, loggingLevelFlag, defaultLoggingLevel, loggingLevelDescription)
}

// ClientConfigFile returns the org config file
func (c *CLIConfig) ClientConfigFile() string {
	return opts.clientConfigFile
}

// InitClientConfigFile initializes the config file path from the provided arguments
func InitClientConfigFile(flags *pflag.FlagSet) {
	flags.StringVar(&opts.clientConfigFile, clientConfigFileFlag, defaultClientConfigFile, clientConfigFileDescription)
}

// OrgID specifies the ID of the current organization. If multiple org IDs are specified then the first one is returned.
func (c *CLIConfig) OrgID() string {
	return opts.orgID
}

// InitOrgID initializes the org ID from the provided arguments
func InitOrgID(flags *pflag.FlagSet) {
	flags.StringVar(&opts.orgID, orgIDFlag, defaultOrgID, orgIDDescription)
}

// GetMspID returns the MSP ID
func (c *CLIConfig) GetMspID() string {
	return opts.mspID
}

// InitMspID initializes the MSP ID from the provided arguments
func InitMspID(flags *pflag.FlagSet) {
	flags.StringVar(&opts.mspID, mspIDFlag, defaultMSPID, mspIDDescription)
}

// ChannelID returns the channel ID
func (c *CLIConfig) ChannelID() string {
	return opts.channelID
}

// InitChannelID initializes the channel ID from the provided arguments
func InitChannelID(flags *pflag.FlagSet) {
	flags.StringVar(&opts.channelID, channelIDFlag, defaultChannelID, channelIDDescription)
}

// UserName returns the name of the enrolled user
func (c *CLIConfig) UserName() string {
	return opts.user
}

// InitUserName initializes the user name from the provided arguments
func InitUserName(flags *pflag.FlagSet) {
	flags.StringVar(&opts.user, userFlag, defaultUser, userDescription)
}

// UserPassword is the password to use when enrolling a user
func (c *CLIConfig) UserPassword() string {
	return opts.password
}

// InitUserPassword initializes the user password from the provided arguments
func InitUserPassword(flags *pflag.FlagSet) {
	flags.StringVar(&opts.password, passwordFlag, defaultPassword, passwordDescription)
}

// PeerURL returns a comma-separated list of peers in the format host1:port1,host2:port2,...
func (c *CLIConfig) PeerURL() string {
	return opts.peerURL
}

// InitPeerURL initializes the peer URL from the provided arguments
func InitPeerURL(flags *pflag.FlagSet) {
	flags.StringVar(&opts.peerURL, peerURLFlag, defaultPeerURL, peerURLDescription)
}

// PeerID returns the ID of the peer (used in the config query command)
func (c *CLIConfig) PeerID() string {
	return opts.peerID
}

// InitPeerID initializes the peer ID from the provided arguments
func InitPeerID(flags *pflag.FlagSet) {
	flags.StringVar(&opts.peerID, peerIDFlag, "", peerIDDescription)
}

// AppName returns an application name (used in the config query command)
func (c *CLIConfig) AppName() string {
	return opts.appName
}

// InitAppName initializes the application name from the provided arguments
func InitAppName(flags *pflag.FlagSet) {
	flags.StringVar(&opts.appName, appNameFlag, "", appNameDescription)
}

// AppVer returns an app ver (used in the config query command)
func (c *CLIConfig) AppVer() string {
	return opts.appVer
}

// InitAppVer initializes the app ver from the provided arguments
func InitAppVer(flags *pflag.FlagSet) {
	flags.StringVar(&opts.appVer, appVerFlag, "", appVerDescription)
}

// ComponentName returns an component name (used in the config query command)
func (c *CLIConfig) ComponentName() string {
	return opts.componentName
}

// InitComponentName initializes the component name from the provided arguments
func InitComponentName(flags *pflag.FlagSet) {
	flags.StringVar(&opts.componentName, componentNameFlag, "", componentNameDescription)
}

// ComponentVer returns an component ver (used in the config query command)
func (c *CLIConfig) ComponentVer() string {
	return opts.componentVer
}

// InitComponentVer initializes the component ver from the provided arguments
func InitComponentVer(flags *pflag.FlagSet) {
	flags.StringVar(&opts.componentVer, componentVerFlag, "", componentVerDescription)
}

// KeyType returns an KeyType name (used in the config generteCSR command)
func (c *CLIConfig) KeyType() string {
	return opts.keyType
}

// InitKeyType initializes the KeyType from the provided arguments
func InitKeyType(flags *pflag.FlagSet) {
	flags.StringVar(&opts.keyType, keyType, "", keyTypeDescription)
}

// EphemeralFlag returns an ephemeral flag (used in the config generteCSR command)
func (c *CLIConfig) EphemeralFlag() string {
	return opts.ephemeralFlag
}

// InitEphemeralFlag initializes the ephemeral flag from the provided arguments
func InitEphemeralFlag(flags *pflag.FlagSet) {
	flags.StringVar(&opts.ephemeralFlag, ephemeral, ephemeralDefault, ephemeralDescription)
}

// SigAlg returns an signature algorithm  (used in the config generteCSR command)
func (c *CLIConfig) SigAlg() string {
	return opts.sigAlg
}

// InitSigAlg initializes the signature algorithm from the provided arguments
func InitSigAlg(flags *pflag.FlagSet) {
	flags.StringVar(&opts.sigAlg, sigAlg, "", sigAlgDescription)
}

// CSRCommonName returns CSR common  name  (used in the config generteCSR command)
func (c *CLIConfig) CSRCommonName() string {
	return opts.csrCommonName
}

// InitCSRCommonName initializes the CSR common name field
func InitCSRCommonName(flags *pflag.FlagSet) {
	flags.StringVar(&opts.csrCommonName, csrCommonName, "", csrCommonNameDesc)
}

// NoPrompt is true if the user does not want top be prompted to confirm an update or delete
func (c *CLIConfig) NoPrompt() bool {
	return opts.noPrompt
}

// InitNoPrompt initializes the "no-prompt" flag from the provided arguments
func InitNoPrompt(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.noPrompt, noPromptFlag, defaultNoPrompt, noPromptDescription)
}

// ConfigKey returns the config key in JSON format
func (c *CLIConfig) ConfigKey() string {
	return opts.configKey
}

// InitConfigKey initializes the config key from the provided arguments
func InitConfigKey(flags *pflag.FlagSet) {
	flags.StringVar(&opts.configKey, configKeyFlag, "", configKeyDescription)
}

// ConfigString returns the config string in JSON format
func (c *CLIConfig) ConfigString() string {
	return opts.config
}

// InitConfigString initializes the config string from the provided arguments
func InitConfigString(flags *pflag.FlagSet) {
	flags.StringVar(&opts.config, configFlag, "", configDescription)
}

// Timeout returns the timeout (in milliseconds) for various operations
func (c *CLIConfig) Timeout(conn fabApi.TimeoutType) time.Duration {
	return time.Duration(opts.timeout) * time.Millisecond
}

// InitTimeout initializes the timeout from the provided arguments
func InitTimeout(flags *pflag.FlagSet) {
	i, err := strconv.Atoi(defaultTimeout)
	if err != nil {
		fmt.Printf("Invalid number: %s\n", defaultTimeout)
		i = 1000
	}
	flags.Int64Var(&opts.timeout, timeoutFlag, int64(i), timeoutDescription)
}

// OutputFormat returns the print (output) format for a block
func (c *CLIConfig) OutputFormat() string {
	return opts.outputFormat
}

// InitOutputFormat initializes the print format from the provided arguments
func InitOutputFormat(flags *pflag.FlagSet) {
	flags.StringVar(&opts.outputFormat, outputFormatFlag, defaultOutputFormat, outputFormatDescription)
}

// IsLoggingEnabledFor indicates whether the logger is enabled for the given logging level
func (c *CLIConfig) IsLoggingEnabledFor(level logging.Level) bool {
	return logging.IsEnabledFor(loggerName, level)
}

// ConfigFile returns the org config file
func (c *CLIConfig) ConfigFile() string {
	return opts.configFile
}

// InitConfigFile initializes the org configuration file from the provided arguments
func InitConfigFile(flags *pflag.FlagSet) {
	flags.StringVar(&opts.configFile, configFileFlag, "", configFileDescription)
}
