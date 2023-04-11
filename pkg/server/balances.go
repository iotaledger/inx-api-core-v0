package server

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota.go/address"

	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

func (s *DatabaseServer) rpcGetBalances(c echo.Context) (interface{}, error) {
	request := &GetBalances{}
	if err := c.Bind(request); err != nil {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	if len(request.Addresses) == 0 {
		return nil, errors.WithMessage(httpserver.ErrInvalidParameter, "invalid request, error: no addresses provided")
	}

	for _, addr := range request.Addresses {
		// Check if address is valid
		if err := address.ValidAddress(addr); err != nil {
			return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid address hash provided: %s", addr)
		}
	}

	result := &GetBalancesResponse{}

	for _, addr := range request.Addresses {

		balance, _, err := s.Database.BalanceForAddress(hornet.HashFromAddressTrytes(addr))
		if err != nil {
			return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
		}

		// Address balance
		result.Balances = append(result.Balances, strconv.FormatUint(balance, 10))
	}

	latestSolidMilestoneBundle := s.Database.LatestSolidMilestoneBundle()

	// The index of the milestone that confirmed the most recent balance
	result.MilestoneIndex = latestSolidMilestoneBundle.MilestoneIndex()
	result.References = []string{latestSolidMilestoneBundle.MilestoneHash().Trytes()}

	return result, nil
}

func (s *DatabaseServer) addressBalance(c echo.Context) (interface{}, error) {
	addr, err := parseAddressParam(c)
	if err != nil {
		return nil, err
	}

	balance, _, err := s.Database.BalanceForAddress(addr)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	return &balanceResponse{
		Address:     addr.Trytes(),
		Balance:     strconv.FormatUint(balance, 10),
		LedgerIndex: s.Database.LedgerIndex(),
	}, nil
}
