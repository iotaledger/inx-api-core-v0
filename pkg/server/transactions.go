package server

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/guards"
	"github.com/iotaledger/iota.go/trinary"

	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

func (s *DatabaseServer) findTransactions(maxResults int, valueOnly bool, queryBundleHashes, queryApproveeHashes, queryAddressHashes, queryTagHashes map[string]struct{}) []string {

	results := make(map[string]struct{})
	searchedBefore := false

	// check if bundle hash search criteria was given
	if len(queryBundleHashes) > 0 {
		// search txs by bundle hash
		for bundleHash := range queryBundleHashes {
			for _, r := range s.Database.BundleTransactionHashes(hornet.Hash(bundleHash), maxResults-len(results)) {
				results[string(r)] = struct{}{}
			}
		}
		searchedBefore = true
	}

	// check if approvee search criteria was given
	if len(queryApproveeHashes) > 0 {
		if !searchedBefore {
			// search txs by approvees
			for approveeHash := range queryApproveeHashes {
				for _, r := range s.Database.ApproverHashes(hornet.Hash(approveeHash), maxResults-len(results)) {
					results[string(r)] = struct{}{}
				}
			}
			searchedBefore = true
		} else {
			// check if results match at least one of the approvee search criteria
			for txHash := range results {
				contains := false
				for approveeHash := range queryApproveeHashes {
					if s.Database.ContainsApprover(hornet.Hash(approveeHash), hornet.Hash(txHash)) {
						contains = true

						break
					}
				}
				if !contains {
					delete(results, txHash)
				}
			}
		}
	}

	// check if address search criteria was given
	if len(queryAddressHashes) > 0 {
		if !searchedBefore {
			// search txs by address
			for addressHash := range queryAddressHashes {
				for _, r := range s.Database.TransactionHashesForAddress(hornet.Hash(addressHash), valueOnly, maxResults-len(results)) {
					results[string(r)] = struct{}{}
				}
			}
			searchedBefore = true
		} else {
			// check if results match at least one of the address search criteria
			for txHash := range results {
				contains := false
				for addressHash := range queryAddressHashes {
					if s.Database.ContainsAddress(hornet.Hash(addressHash), hornet.Hash(txHash), valueOnly) {
						contains = true

						break
					}
				}
				if !contains {
					delete(results, txHash)
				}
			}
		}
	}

	// check if tag search criteria was given
	if len(queryTagHashes) > 0 {
		if !searchedBefore {
			// search txs by tags
			for tagHash := range queryTagHashes {
				for _, r := range s.Database.TagHashes(hornet.Hash(tagHash), maxResults-len(results)) {
					results[string(r)] = struct{}{}
				}
			}
		} else {
			// check if results match at least one of the tag search criteria
			for txHash := range results {
				contains := false
				for tagHash := range queryTagHashes {
					if s.Database.ContainsTag(hornet.Hash(tagHash), hornet.Hash(txHash)) {
						contains = true

						break
					}
				}
				if !contains {
					delete(results, txHash)
				}
			}
		}
	}

	// convert to slice
	txHashes := make([]string, 0, len(results))
	for r := range results {
		txHashes = append(txHashes, hornet.Hash(r).Trytes())
	}

	return txHashes
}

func (s *DatabaseServer) rpcFindTransactions(c echo.Context) (interface{}, error) {
	request := &FindTransactions{}
	if err := c.Bind(request); err != nil {
		return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	maxResults := s.RestAPILimitsMaxResults
	if (request.MaxResults > 0) && (request.MaxResults < maxResults) {
		maxResults = request.MaxResults
	}

	if len(request.Bundles) == 0 && len(request.Addresses) == 0 && len(request.Approvees) == 0 && len(request.Tags) == 0 {
		return nil, ierrors.Wrap(httpserver.ErrInvalidParameter, "no search criteria was given")
	}

	queryBundleHashes := make(map[string]struct{})
	queryApproveeHashes := make(map[string]struct{})
	queryAddressHashes := make(map[string]struct{})
	queryTagHashes := make(map[string]struct{})

	// check all queries first
	for _, bundleTrytes := range request.Bundles {
		if !guards.IsTransactionHash(bundleTrytes) {
			return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid bundle hash provided: %s", bundleTrytes)
		}
		queryBundleHashes[string(hornet.HashFromHashTrytes(bundleTrytes))] = struct{}{}
	}

	for _, approveeTrytes := range request.Approvees {
		if !guards.IsTransactionHash(approveeTrytes) {
			return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid aprovee hash provided: %s", approveeTrytes)
		}
		queryApproveeHashes[string(hornet.HashFromHashTrytes(approveeTrytes))] = struct{}{}
	}

	for _, addressTrytes := range request.Addresses {
		if err := address.ValidAddress(addressTrytes); err != nil {
			return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid address hash provided: %s", addressTrytes)
		}
		queryAddressHashes[string(hornet.HashFromAddressTrytes(addressTrytes[:81]))] = struct{}{}
	}

	for _, tagTrytes := range request.Tags {
		if err := trinary.ValidTrytes(tagTrytes); err != nil {
			return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid tag trytes provided: %s", tagTrytes)
		}
		if len(tagTrytes) > 27 {
			return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid tag length: %s", tagTrytes)
		}
		if len(tagTrytes) < 27 {
			tagTrytes = trinary.MustPad(tagTrytes, 27)
		}
		queryTagHashes[string(hornet.HashFromTagTrytes(tagTrytes))] = struct{}{}
	}

	txHashes := s.findTransactions(maxResults, request.ValueOnly, queryBundleHashes, queryApproveeHashes, queryAddressHashes, queryTagHashes)

	return &FindTransactionsResponse{
		Hashes: txHashes,
	}, nil
}

func (s *DatabaseServer) transactions(c echo.Context) (interface{}, error) {
	valueOnly := false
	for query := range c.QueryParams() {
		if strings.ToLower(query) == "valueonly" {
			valueOnly = true

			break
		}
	}

	maxResults, err := parseMaxResultsQueryParam(c, s.RestAPILimitsMaxResults)
	if err != nil {
		return nil, err
	}

	requestBundleHash, err := parseBundleQueryParam(c)
	if err != nil {
		return nil, err
	}
	requestApproveeHash, err := parseApproveeQueryParam(c)
	if err != nil {
		return nil, err
	}
	requestAddressHash, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	requestTagHash, err := parseTagQueryParam(c)
	if err != nil {
		return nil, err
	}

	if requestBundleHash == nil && requestApproveeHash == nil && requestAddressHash == nil && requestTagHash == nil {
		return nil, ierrors.Wrap(httpserver.ErrInvalidParameter, "no search criteria was given")
	}

	queryBundleHashes := make(map[string]struct{})
	queryApproveeHashes := make(map[string]struct{})
	queryAddressHashes := make(map[string]struct{})
	queryTagHashes := make(map[string]struct{})

	if requestBundleHash != nil {
		queryBundleHashes[string(requestBundleHash)] = struct{}{}
	}
	if requestApproveeHash != nil {
		queryApproveeHashes[string(requestApproveeHash)] = struct{}{}
	}
	if requestAddressHash != nil {
		queryAddressHashes[string(requestAddressHash)] = struct{}{}
	}
	if requestTagHash != nil {
		queryTagHashes[string(requestTagHash)] = struct{}{}
	}

	txHashes := s.findTransactions(maxResults, valueOnly, queryBundleHashes, queryApproveeHashes, queryAddressHashes, queryTagHashes)

	return &transactionsResponse{
		Bundle: func() string {
			if requestBundleHash != nil {
				return requestBundleHash.Trytes()
			}

			return ""
		}(),
		Approvee: func() string {
			if requestApproveeHash != nil {
				return requestApproveeHash.Trytes()
			}

			return ""
		}(),
		Address: func() string {
			if requestAddressHash != nil {
				return requestAddressHash.Trytes()
			}

			return ""
		}(),
		Tag: func() string {
			if requestTagHash != nil {
				return requestTagHash.Trytes()
			}

			return ""
		}(),
		TransactionHashes: txHashes,
		LedgerIndex:       s.Database.LedgerIndex(),
	}, nil
}
