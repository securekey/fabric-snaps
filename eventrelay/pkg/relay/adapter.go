/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package relay

import (
	"github.com/hyperledger/fabric/protos/peer"
)

//EventAdapter is the interface by which a fabric event client registers interested events and
//receives messages from the fabric event Server
type EventAdapter interface {
	GetInterestedEvents() ([]*peer.Interest, error)
	Recv(msg *peer.Event) (bool, error)
	Disconnected(err error)
}
