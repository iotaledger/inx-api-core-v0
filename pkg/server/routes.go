package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pangpanglabs/echoswagger/v2"

	"github.com/iotaledger/inx-app/pkg/httpserver"
)

const (
	ParameterAddress         = "address"
	ParameterTransactionHash = "txHash"
	ParameterMilestoneIndex  = "index"

	QueryParameterBundle     = "bundle"
	QueryParameterAddress    = "address"
	QueryParameterTag        = "tag"
	QueryParameterApprovee   = "approvee"
	QueryParameterMaxResults = "maxResults"
)

const (
	// RouteRPCEndpoint is the route for sending RPC requests to the API.
	// POST sends an IOTA legacy API request and returns the results.
	RouteRPCEndpoint = "/"

	// RouteInfo is the route for getting the node info.
	// GET returns the node info.
	RouteInfo = "/info"

	// RouteTransactions is the route for getting transactions filtered by the given parameters.
	// GET with query parameter returns all txHashes that fit these filter criteria.
	// Query parameters: "bundle", "address", "tag", "approvee", "maxResults"
	// Returns an empty list if no results are found.
	RouteTransactions = "/transactions" // former findTransactions

	// RouteTransactionTrytes is the route for getting the trytes of a transaction.
	// GET will return the transaction trytes.
	RouteTransactionTrytes = "/transactions/:" + ParameterTransactionHash + "/trytes" // former getTrytes

	// RouteTransactionInclusionState is the route for getting the inclusion state of a transaction.
	// GET will return the inclusion state.
	RouteTransactionInclusionState = "/transactions/:" + ParameterTransactionHash + "/inclusion-state" // former getInclusionStates

	// RouteAddressBalance is the route for getting the balance of an address.
	// GET will return the balance.
	RouteAddressBalance = "/addresses/:" + ParameterAddress + "/balance" // former getBalances

	// RouteAddressBalance is the route to check whether an address was already spent or not.
	// GET will return true if the address was already spent.
	RouteAddressWasSpent = "/addresses/:" + ParameterAddress + "/was-spent" // former wereAddressesSpentFrom

	// RouteLedgerState is the route to return the current ledger state.
	// GET will return all addresses with their balances.
	RouteLedgerState = "/ledger/state" // former getLedgerState

	// RouteLedgerStateByIndex is the route to return the ledger state of a given ledger index.
	// GET will return all addresses with their balances.
	RouteLedgerStateByIndex = "/ledger/state/by-index/:" + ParameterMilestoneIndex // former getLedgerState

	// RouteLedgerDiffByIndex is the route to return the ledger diff of a given ledger index.
	// GET will return all addresses with their diffs.
	RouteLedgerDiffByIndex = "/ledger/diff/by-index/:" + ParameterMilestoneIndex // former getLedgerDiff

	// RouteLedgerDiffExtendedByIndex is the route to return the ledger diff of a given ledger index with extended informations.
	// GET will return all addresses with their diffs, the confirmed transactions and the confirmed bundles.
	RouteLedgerDiffExtendedByIndex = "/ledger/diff-extended/by-index/:" + ParameterMilestoneIndex // former getLedgerDiffExt
)

func (s *DatabaseServer) configureRoutes(routeGroup echoswagger.ApiGroup) {

	s.configureRPCEndpoints()

	routeGroup.POST(RouteRPCEndpoint, func(c echo.Context) error {
		resp, err := rpc(c, s.RPCEndpoints)
		if err != nil {
			// the RPC endpoint has custom error handling for compatibility reasons
			var e *echo.HTTPError

			var statusCode int
			var message string
			if errors.As(err, &e) {
				statusCode = e.Code
				message = fmt.Sprintf("%s, error: %s", e.Message, err)
			} else {
				statusCode = http.StatusInternalServerError
				message = fmt.Sprintf("internal server error. error: %s", err)
			}

			return httpserver.JSONResponse(c, statusCode, &ErrorReturn{Error: message})
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route for sending RPC requests to the API").
		SetOperationId("rpc").
		AddParamBody(Request{}, "", "the command of the request", true)

	routeGroup.GET(RouteInfo, func(c echo.Context) error {
		resp, err := s.info()
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route for getting the node info").
		SetOperationId("info")

	routeGroup.GET(RouteTransactions, func(c echo.Context) error {
		resp, err := s.transactions(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route for getting transactions filtered by the given parameters. Returns an empty list if no results are found.").
		SetOperationId("transactions").
		AddParamQuery("", QueryParameterBundle, "filter for transactions with a specific bundle hash", false).
		AddParamQuery("", QueryParameterAddress, "filter for transactions with a specific address", false).
		AddParamQuery("", QueryParameterTag, "filter for transactions with a specific tag", false).
		AddParamQuery("", QueryParameterApprovee, "filter for transactions with a specific approvee hash", false).
		AddParamQuery("", QueryParameterMaxResults, "limit the maximum number of results", false)

	routeGroup.GET(RouteTransactionTrytes, func(c echo.Context) error {
		resp, err := s.transactionTrytes(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route for getting the trytes of a transaction").
		SetOperationId("transactionTrytes").
		AddParamPath("", ParameterTransactionHash, "the hash of the transaction")

	routeGroup.GET(RouteTransactionInclusionState, func(c echo.Context) error {
		resp, err := s.transactionInclusionState(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route for getting the inclusion state of a transaction").
		SetOperationId("transactionInclusionState").
		AddParamPath("", ParameterTransactionHash, "the hash of the transaction")

	routeGroup.GET(RouteAddressBalance, func(c echo.Context) error {
		resp, err := s.addressBalance(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route for getting the balance of an address").
		SetOperationId("addressBalance").
		AddParamPath("", ParameterAddress, "the hash of the address")

	routeGroup.GET(RouteAddressWasSpent, func(c echo.Context) error {
		resp, err := s.addressWasSpent(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route to check whether an address was already spent or not").
		SetOperationId("addressWasSpent").
		AddParamPath("", ParameterAddress, "the hash of the address")

	routeGroup.GET(RouteLedgerState, func(c echo.Context) error {
		resp, err := s.ledgerStateByLatestSolidIndex(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route to return the current ledger state").
		SetOperationId("ledgerStateByLatestSolidIndex")

	routeGroup.GET(RouteLedgerStateByIndex, func(c echo.Context) error {
		resp, err := s.ledgerStateByIndex(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route to return the ledger state of a given ledger index").
		SetOperationId("ledgerStateByIndex").
		AddParamPath("", ParameterMilestoneIndex, "the index of the milestone")

	routeGroup.GET(RouteLedgerDiffByIndex, func(c echo.Context) error {
		resp, err := s.ledgerDiff(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route to return the ledger diff of a given ledger index").
		SetOperationId("ledgerDiff").
		AddParamPath("", ParameterMilestoneIndex, "the index of the milestone")

	routeGroup.GET(RouteLedgerDiffExtendedByIndex, func(c echo.Context) error {
		resp, err := s.ledgerDiffExtended(c)
		if err != nil {
			return err
		}

		return httpserver.JSONResponse(c, http.StatusOK, resp)
	}).
		SetDescription("the route to return the ledger diff of a given ledger index with extended informations").
		SetOperationId("ledgerDiffExtended").
		AddParamPath("", ParameterMilestoneIndex, "the index of the milestone")
}
