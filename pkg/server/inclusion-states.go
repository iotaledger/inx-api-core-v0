package server

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota.go/guards"

	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

func (s *DatabaseServer) rpcGetInclusionStates(c echo.Context) (interface{}, error) {
	request := &GetInclusionStates{}
	if err := c.Bind(request); err != nil {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	for _, tx := range request.Transactions {
		if !guards.IsTransactionHash(tx) {
			return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid reference hash provided: %s", tx)
		}
	}

	inclusionStates := []bool{}

	for _, tx := range request.Transactions {
		// get tx data
		txMeta := s.Database.TxMetadataOrNil(hornet.HashFromHashTrytes(tx))
		if txMeta == nil {
			// if tx is unknown, return false
			inclusionStates = append(inclusionStates, false)

			continue
		}
		// check if tx is set as confirmed. Avoid passing true for conflicting tx to be backwards compatible
		confirmed := txMeta.IsConfirmed() && !txMeta.IsConflicting()

		inclusionStates = append(inclusionStates, confirmed)
	}

	return &GetInclusionStatesResponse{
		States: inclusionStates,
	}, nil
}

func (s *DatabaseServer) transactionInclusionState(c echo.Context) (interface{}, error) {
	txHash, err := parseTransactionHashParam(c)
	if err != nil {
		return nil, err
	}

	// get tx data
	txMeta := s.Database.TxMetadataOrNil(txHash)
	if txMeta == nil {
		// if tx is unknown, return false
		return &transactionInclusionStateResponse{
			TxHash:      txHash.Trytes(),
			Included:    false,
			Confirmed:   false,
			Conflicting: false,
			LedgerIndex: s.Database.GetLedgerIndex(),
		}, nil
	}

	return &transactionInclusionStateResponse{
		TxHash: txHash.Trytes(),
		// avoid passing true for conflicting tx to be backwards compatible
		Included:    txMeta.IsConfirmed() && !txMeta.IsConflicting(),
		Confirmed:   txMeta.IsConfirmed(),
		Conflicting: txMeta.IsConflicting(),
		LedgerIndex: s.Database.GetLedgerIndex(),
	}, nil
}
