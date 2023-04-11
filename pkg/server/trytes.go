package server

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota.go/guards"
	"github.com/iotaledger/iota.go/transaction"

	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

func (s *DatabaseServer) rpcGetTrytes(c echo.Context) (interface{}, error) {
	request := &GetTrytes{}
	if err := c.Bind(request); err != nil {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	maxResults := s.RestAPILimitsMaxResults
	if len(request.Hashes) > maxResults {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "too many hashes. maximum allowed: %d", maxResults)
	}

	trytes := []string{}
	for _, hash := range request.Hashes {
		if !guards.IsTransactionHash(hash) {
			return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid hash provided: %s", hash)
		}
	}

	for _, hash := range request.Hashes {
		tx := s.Database.TransactionOrNil(hornet.HashFromHashTrytes(hash))
		if tx == nil {
			trytes = append(trytes, strings.Repeat("9", 2673))

			continue
		}

		txTrytes, err := transaction.TransactionToTrytes(tx.Tx)
		if err != nil {
			return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
		}

		trytes = append(trytes, txTrytes)
	}

	return &GetTrytesResponse{
		Trytes: trytes,
	}, nil
}

func (s *DatabaseServer) transactionTrytes(c echo.Context) (interface{}, error) {
	txHash, err := parseTransactionHashParam(c)
	if err != nil {
		return nil, err
	}

	tx := s.Database.TransactionOrNil(txHash)
	if tx == nil {
		return nil, errors.WithMessagef(echo.ErrNotFound, "transaction not found: %s", txHash.Trytes())
	}

	txTrytes, err := transaction.TransactionToTrytes(tx.Tx)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	return &trytesResponse{
		TxHash: txHash.Trytes(),
		Trytes: txTrytes,
	}, nil
}
