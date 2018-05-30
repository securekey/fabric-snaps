/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generatecsr

import (
	"encoding/pem"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/spf13/cobra"
)

const description = `
The GenerateCSR command allows the client to generate CSR by using configuration snap.
Configuration for the CSR template should be part of configuration snap config.
CLI Required Args:

-keyType (key type -> one of:
	ECDSA,	ECDSAP256, ECDSAP384, RSA, RSA1024,	RSA2048, RSA3072, RSA4096 )

-ephemeral (true/false - use false to create persistent key)

-sigAlg (signature algorithm -> one of:
	MD2WithRSA, MD5WithRSA, SHA1WithRSA, SHA256WithRSA, SHA384WithRSA, SHA512WithRSA
	DSAWithSHA1, DSAWithSHA256, ECDSAWithSHA1, ECDSAWithSHA256, ECDSAWithSHA384
	ECDSAWithSHA512, SHA256WithRSAPSS, SHA384WithRSAPSS, SHA512WithRSAPSS)

-csrCommonName (string)

`

var keyOpts = []string{"ECDSA", "ECDSAP256", "ECDSAP384", "RSA", "RSA1024", "RSA2048", "RSA3072", "RSA4096"}
var sigAlgOpts = []string{"MD2WithRSA", "MD5WithRSA", "SHA1WithRSA",
	"SHA256WithRSA", "SHA384WithRSA", "SHA512WithRSA",
	"DSAWithSHA1", "DSAWithSHA256", "ECDSAWithSHA1", "ECDSAWithSHA256", "ECDSAWithSHA384",
	"ECDSAWithSHA512", "SHA256WithRSAPSS", "SHA384WithRSAPSS", "SHA512WithRSAPSS"}

const examples = `
- Generate CSR by running command:
    $ ./configcli generateCSR --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel  --peerurl grpcs://localhost:7051 --mspid Org1MSP --peerid peer0.org1.example.com --keyType ECDSA  --ephemeral false  --sigAlg ECDSAWithSHA512 --csrCommonName certcommonname1234567
`

// Cmd returns the Query command
func Cmd() *cobra.Command {
	return newCmd(action.New())
}

type queryAction struct {
	action.Action
}

func newCmd(baseAction action.Action) *cobra.Command {
	validArgs := []string{"keyType", "ephemeral", "sigAlg", "user", "peer", "peerurl", "configfile", "clientconfig"}
	cmd := &cobra.Command{
		Use:       "generateCSR",
		Short:     "Generate CSR: mandatory flags are: 'keyType', 'ephemeral' ,'sigAlg','csrCommonName'",
		Long:      description,
		Example:   examples,
		ValidArgs: validArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			action, err := newGenerateCSRAction(baseAction)
			if err != nil {
				return errors.Wrap(err, "error while initializing generateCSR")
			}
			return action.generateCSR()
		},
	}

	flags := cmd.Flags()
	cliconfig.InitPeerURL(flags)
	cliconfig.InitChannelID(flags)
	cliconfig.InitPeerID(flags)
	cliconfig.InitKeyType(flags)
	cliconfig.InitEphemeralFlag(flags)
	cliconfig.InitSigAlg(flags)
	cliconfig.InitCSRCommonName(flags)
	return cmd
}

func newGenerateCSRAction(baseAction action.Action) (*queryAction, error) {
	action := &queryAction{
		Action: baseAction,
	}
	err := action.Initialize()
	return action, err
}

func (a *queryAction) generateCSR() error {
	ephemeralstr := cliconfig.Config().EphemeralFlag()
	_, err := strconv.ParseBool(ephemeralstr)
	if err != nil {
		return errors.Errorf("Ephemeral Flag should have \"true\"/\"false\" value")
	}
	keyType := cliconfig.Config().KeyType()
	if keyType == "" {
		return errors.Errorf("Key type is mandatory field")
	}
	b := contains(keyOpts, keyType)
	if !b {
		return errors.Errorf("Unsuported key type %s ", keyType)
	}
	sigAlg := cliconfig.Config().SigAlg()
	if sigAlg == "" {
		return errors.Errorf("SigAlg is mandatory field")
	}
	b = contains(sigAlgOpts, sigAlg)
	if !b {
		return errors.Errorf("Unsuported signature algorithm %s ", sigAlg)
	}
	csrCommonName := cliconfig.Config().CSRCommonName()
	if csrCommonName == "" {
		return errors.Errorf("csrCommonName is mandatory field")
	}

	args := [][]byte{[]byte(keyType), []byte([]byte(ephemeralstr)), []byte(sigAlg), []byte(csrCommonName)}

	cliconfig.Config().Logger().Debugf("Using generate csr args: %v\n", args)
	//invoke configuration snap -function name: 'generateCSR'
	response, err := a.Query(cliconfig.ConfigSnapID, "generateCSR", args)
	if err != nil {
		return err
	}
	cliconfig.Config().Logger().Debugf("***Generated CSR*** [%v]", response)
	fmt.Printf("\nPEM encoded CSR:[%s]", csrToPem(response))
	return nil
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func csrToPem(csr []byte) (csrPEM string) {
	csrPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	csrPEM = string(csrPEMBytes[:])
	return
}
