/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"sync"

	"github.com/golang/protobuf/proto"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/msp"
)

// mspIDMgr manages a map of PKI IDs to MSP IDs. The map is
// populated dynamically from Gossip.
type mspIDMgr struct {
	mspIDMap map[string]string
	mutex    sync.RWMutex
}

func newMSPIDMgr(service service.GossipService) *mspIDMgr {
	m := &mspIDMgr{}

	m.mspIDMap = make(map[string]string)

	_, msgch := service.Accept(func(o interface{}) bool {
		m := o.(gossip.ReceivedMessage).GetGossipMessage()
		if m.IsPullMsg() && m.GetPullMsgType() == gossip.PullMsgType_IDENTITY_MSG {
			return m.IsDataUpdate()
		}
		return false
	}, true)

	go m.receive(msgch)

	return m
}

// GetMSPID returns the MSP ID for the given PKI ID
func (m *mspIDMgr) GetMSPID(pkiID gcommon.PKIidType) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	mspID, ok := m.mspIDMap[string(pkiID)]
	if !ok {
		logger.Warnf("MSP ID not found for PKI ID [%v]", pkiID)
	}
	return mspID
}

func (m *mspIDMgr) receive(msgch <-chan gossip.ReceivedMessage) {
	for {
		logger.Debug("Listening to gossip messages...\n")
		select {
		case msg, ok := <-msgch:
			if !ok {
				logger.Info("Gossip listener terminated")
				return
			}

			logger.Debugf("Got gossip message: %v\n", msg)
			dataUpdate := msg.GetGossipMessage().GetDataUpdate()
			if dataUpdate != nil {
				for _, envelope := range dataUpdate.Data {
					m.handleUpdate(envelope)
				}
			}
		}
	}
}

func (m *mspIDMgr) handleUpdate(envelope *gossip.Envelope) {
	msg, err := envelope.ToGossipMessage()
	if err != nil {
		logger.Error("Error occurred while getting gossip messages : %s", err)
	}
	pIdentity := msg.GetPeerIdentity()
	sID := &msp.SerializedIdentity{}
	err = proto.Unmarshal(pIdentity.Cert, sID)
	if err != nil {
		logger.Error("Error occurred while un-marshalling : %s", err)
	}
	pkiID := string(pIdentity.PkiId)

	// Only update if not already in map
	m.mutex.RLock()
	mspID, ok := m.mspIDMap[pkiID]
	m.mutex.RUnlock()

	if !ok || mspID != sID.Mspid {
		logger.Infof("Mapping PKI ID [%v] to MSP ID [%s]\n", pIdentity.PkiId, sID.Mspid)
		m.mutex.Lock()
		m.mspIDMap[pkiID] = sID.Mspid
		m.mutex.Unlock()
	}
}
