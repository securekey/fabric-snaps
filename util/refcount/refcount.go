/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package refcount

import (
	"sync/atomic"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
)

var logger = logging.NewLogger("txnsnap")

// Closer closes the resource
type Closer func()

// ReferenceCounter is a reference counter which allows you to safely close
// a resource when it's no longer in use.
type ReferenceCounter struct {
	refCount int32
	closed   uint32
	closer   Closer
}

// New returns a new reference counter
func New(closer Closer) *ReferenceCounter {
	return &ReferenceCounter{
		closer: closer,
	}
}

// Acquire adds a reference
func (c *ReferenceCounter) Acquire() bool {
	if c.isClosed() {
		logger.Debugf("Cannot acquire reference since resource is already closed")
		return false
	}

	for {
		refCount := c.count()
		logger.Debugf("Current refCount: %d", refCount)
		if refCount < 0 {
			logger.Debugf("Cannot acquire reference since resource is already closed")
			return false
		}
		newRefCount := refCount + 1
		if c.setCount(refCount, newRefCount) {
			logger.Debugf("Updated refCount to %d", refCount)
			return true
		}
		logger.Debugf("Another thread has modified the refcount. Expected %d", refCount)
	}
}

// Release releases a reference
func (c *ReferenceCounter) Release() bool {
	for {
		refCount := c.count()
		logger.Debugf("Current refCount: %d", refCount)
		if refCount <= 0 {
			logger.Warnf("Cannot release resource since the refcount is %d", refCount)
			return false
		}

		newRefCount := refCount - 1
		if c.setCount(refCount, newRefCount) {
			logger.Debugf("Updated refCount to %d", refCount)
			if newRefCount == 0 {
				logger.Debugf("RefCount is 0. Checking if it's safe to close...")
				c.checkAndCloseResource()
			}
			return true
		}
		logger.Debugf("Another thread has modified the refcount. Expected %d", refCount)
	}
}

// Close closes the resource when the last reference is released
func (c *ReferenceCounter) Close() bool {
	if !c.close() {
		logger.Debugf("Resource already closed.")
		return false
	}

	logger.Debugf("Current refcount: %d. Resource will be closed when last reference is removed.", c.count())
	c.checkAndCloseResource()

	return true
}

func (c *ReferenceCounter) checkAndCloseResource() {
	if !c.isClosed() {
		return
	}

	// If the ref-count reaches 0 then close the client
	if c.setCount(0, -1) {
		logger.Debugf("Last reference removed - closing resource ...")
		c.closer()
		return
	}

	logger.Debugf("Last reference was not removed. Will not close resource yet.")
}

func (c *ReferenceCounter) isClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

func (c *ReferenceCounter) close() bool {
	return atomic.CompareAndSwapUint32(&c.closed, 0, 1)
}

func (c *ReferenceCounter) count() int32 {
	return atomic.LoadInt32(&c.refCount)
}

func (c *ReferenceCounter) setCount(expectValue, value int32) bool {
	return atomic.CompareAndSwapInt32(&c.refCount, expectValue, value)
}
