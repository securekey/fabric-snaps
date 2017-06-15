/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package interfaces

//SnapStub Implementation of the snap stub interface
type SnapStub struct {
	Args [][]byte
}

// GetArgs ...
func (sc *SnapStub) GetArgs() [][]byte {
	return [][]byte{[]byte("hello")}
}

//GetStringArgs ...
func (sc *SnapStub) GetStringArgs() []string {
	return []string{""}
}

//GetFunctionAndParameters ...
func (sc *SnapStub) GetFunctionAndParameters() (string, []string) {
	return "", []string{""}
}

//NewSnapStub ...
func NewSnapStub(args [][]byte) *SnapStub {
	ssc := SnapStub{}
	ssc.Args = args
	return &ssc
}
