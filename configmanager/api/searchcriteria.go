/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"fmt"

	"github.com/pkg/errors"
)

// SearchCriteria is used for querying configuration
type SearchCriteria interface {
	// GetSearchType returns the search type
	GetSearchType() SearchType
	// GetMspID returns the MspID
	GetMspID() string
}

// SearchType specifies the type of search to perform
type SearchType int32

const (
	// SearchByMspID (org) searches for config messages by MspID
	SearchByMspID SearchType = iota
)

// NewSearchCriteriaByMspID returns search criteria that queries by MSP ID
func NewSearchCriteriaByMspID(mspID string) (SearchCriteria, error) {
	if mspID == "" {
		return nil, errors.New("must specify message ID for search criteria")
	}
	return &searchCriteriaImpl{
		searchType: SearchByMspID,
		mspID:      mspID,
	}, nil
}

var searchTypeName = map[SearchType]string{
	SearchByMspID: "ByMspID",
}

var searchTypeValue = map[string]SearchType{
	"ByMspID": SearchByMspID,
}

type searchCriteriaImpl struct {
	searchType SearchType
	mspID      string
}

func (c SearchType) String() string {
	return searchTypeName[c]
}

//GetSearchType returns searchType
func (c *searchCriteriaImpl) GetSearchType() SearchType {
	return c.searchType
}

//GetMspID returns MspId
func (c *searchCriteriaImpl) GetMspID() string {
	return c.mspID
}

func (c *searchCriteriaImpl) String() string {
	switch c.searchType {
	case SearchByMspID:
		return fmt.Sprintf("%s[msgID='%s']", c.GetSearchType(), c.GetMspID())
	default:
		return c.searchType.String()
	}
}
