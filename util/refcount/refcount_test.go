/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package refcount

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNoReferences(t *testing.T) {
	resourceClosed := false

	c := New(func() {
		resourceClosed = true
	})

	require.Falsef(t, resourceClosed, "Expecting resource to still be open")
	require.Truef(t, c.Close(), "Expecting Close to have succeeded")
	require.Falsef(t, c.Close(), "Expecting Close to have failed since the resource is already closed")
	require.Truef(t, resourceClosed, "Expecting resource to have been closed")
}

func TestWithReferences(t *testing.T) {
	resourceClosed := false

	c := New(func() {
		resourceClosed = true
	})

	require.Falsef(t, resourceClosed, "Expecting resource to still be open")
	require.Falsef(t, c.Release(), "Expecting Release to have failed since there are no outstanding references")
	require.Truef(t, c.Acquire(), "Expecting Acquire to have succeeded")
	require.Truef(t, c.Close(), "Expecting Close to have succeeded")
	require.Falsef(t, c.Acquire(), "Expecting Acquire to have failed since resource is closed")
	require.Falsef(t, resourceClosed, "Expecting resource to still be open since there is an outstanding reference")
	require.Truef(t, c.Release(), "Expecting Release to have succeeded")
	require.Truef(t, resourceClosed, "Expecting resource to have been closed since the last reference was released")
	require.Falsef(t, c.Release(), "Expecting Release to have failed since resource is closed")
}

func TestConcurrent(t *testing.T) {
	resourceClosed := false

	c := New(func() {
		resourceClosed = true
	})

	var wg sync.WaitGroup

	concurrency := 5
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c.Acquire() {
				time.Sleep(10 * time.Millisecond)
				c.Release()
			}
		}()
	}

	time.Sleep(10 * time.Millisecond)
	require.Truef(t, c.Close(), "Expecting Close to have succeeded")

	wg.Wait()

	require.Truef(t, resourceClosed, "Expecting resource to be closed")
}
