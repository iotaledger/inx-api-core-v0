package server

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota.go/address"

	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

func (s *DatabaseServer) rpcWereAddressesSpentFrom(c echo.Context) (interface{}, error) {
	request := &WereAddressesSpentFrom{}
	if err := c.Bind(request); err != nil {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	if len(request.Addresses) == 0 {
		return nil, errors.WithMessage(httpserver.ErrInvalidParameter, "invalid request, error: no addresses provided")
	}

	result := &WereAddressesSpentFromResponse{}

	for _, addr := range request.Addresses {
		if err := address.ValidAddress(addr); err != nil {
			return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid address hash provided: %s", addr)
		}

		// State
		result.States = append(result.States, s.Database.WasAddressSpentFrom(hornet.HashFromAddressTrytes(addr[:81])))
	}

	return result, nil
}

func (s *DatabaseServer) addressWasSpent(c echo.Context) (interface{}, error) {
	addr, err := parseAddressParam(c)
	if err != nil {
		return nil, err
	}

	return &addressWasSpentResponse{
		Address:     addr.Trytes(),
		WasSpent:    s.Database.WasAddressSpentFrom(addr),
		LedgerIndex: s.Database.LedgerIndex(),
	}, nil
}
