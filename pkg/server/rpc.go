package server

import (
	"io"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/inx-app/pkg/httpserver"
)

/*
IOTA core-api-v0 endpoints:

implemented by this plugin:
- getNodeInfo
- getBalances
- findTransactions
- getTrytes
- getLedgerDiff
- getLedgerState
- getInclusionStates
- wereAddressesSpentFrom

useless in "read-only" mode:
- checkConsistency
- getRequests
- searchConfirmedApprover
- searchEntryPoints
- triggerSolidifier
- getFundsOnSpentAddresses
- getNodeAPIConfiguration
- getLedgerDiffExt
- addNeighbors
- removeNeighbors
- getNeighbors
- attachToTangle
- pruneDatabase
- createSnapshotFile
- getTransactionsToApprove
- getTipInfo
- getSpammerTips
- broadcastTransactions
- storeTransactions
- getWhiteFlagConfirmation
*/

type rpcEndpoint func(c echo.Context) (any, error)

func (s *DatabaseServer) configureRPCEndpoints() {
	addEndpoint := func(endpointName string, implementation rpcEndpoint) {
		s.RPCEndpoints[strings.ToLower(endpointName)] = func(c echo.Context) (any, error) {
			return implementation(c)
		}
	}

	addEndpoint("getNodeInfo", s.rpcGetNodeInfo)
	addEndpoint("findTransactions", s.rpcFindTransactions)
	addEndpoint("getTrytes", s.rpcGetTrytes)
	addEndpoint("getInclusionStates", s.rpcGetInclusionStates)
	addEndpoint("getBalances", s.rpcGetBalances)
	addEndpoint("wereAddressesSpentFrom", s.rpcWereAddressesSpentFrom)
	addEndpoint("getLedgerState", s.rpcGetLedgerState)
	addEndpoint("getLedgerDiff", s.rpcGetLedgerDiff)
	addEndpoint("getLedgerDiffExt", s.rpcGetLedgerDiffExt)
}

func rpc(c echo.Context, implementedAPIcalls map[string]rpcEndpoint) (interface{}, error) {

	request := &Request{}

	// Read the content of the body
	var bodyBytes []byte
	if c.Request().Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(c.Request().Body)
		if err != nil {
			return nil, ierrors.Wrap(echo.ErrInternalServerError, err.Error())
		}
	}

	// we need to restore the body after reading it
	restoreBody(c, bodyBytes)

	if err := c.Bind(request); err != nil {
		return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	// we need to restore the body after reading it
	restoreBody(c, bodyBytes)

	implementation, exists := implementedAPIcalls[strings.ToLower(request.Command)]
	if !exists {
		return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "command is unknown: %s", request.Command)
	}

	return implementation(c)
}
