package server

import (
	"github.com/labstack/echo/v4"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota.go/guards"

	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
)

func (s *DatabaseServer) rpcGetInclusionStates(c echo.Context) (interface{}, error) {
	request := &GetInclusionStates{}
	if err := c.Bind(request); err != nil {
		return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	for _, tx := range request.Transactions {
		if !guards.IsTransactionHash(tx) {
			return nil, ierrors.Wrapf(httpserver.ErrInvalidParameter, "invalid reference hash provided: %s", tx)
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

func (s *DatabaseServer) transactionMetadata(c echo.Context) (interface{}, error) {
	txHash, err := parseTransactionHashParam(c)
	if err != nil {
		return nil, err
	}

	// get tx data
	txMeta := s.Database.TxMetadataOrNil(txHash)
	if txMeta == nil {
		// if tx is unknown, return false
		return &transactionMetadataResponse{
			TxHash:      txHash.Trytes(),
			Solid:       false,
			Included:    false,
			Confirmed:   false,
			Conflicting: false,
			LedgerIndex: s.Database.LedgerIndex(),
		}, nil
	}

	var referencedByMilestoneIndex milestone.Index
	var milestoneTimestampReferenced uint64
	confirmed, at := txMeta.ConfirmedWithIndex()
	if confirmed {
		referencedByMilestoneIndex = at

		timestamp, err := s.Database.MilestoneTimestamp(referencedByMilestoneIndex)
		if err == nil {
			milestoneTimestampReferenced = timestamp
		}
	}

	var milestoneIndex milestone.Index
	if txMeta.IsMilestone() {
		milestoneIndex = txMeta.MilestoneIndex()
	}

	return &transactionMetadataResponse{
		TxHash:                       txHash.Trytes(),
		Solid:                        txMeta.IsSolid(),
		Included:                     confirmed && !txMeta.IsConflicting(), // avoid passing true for conflicting tx to be backwards compatible
		Confirmed:                    confirmed,
		Conflicting:                  txMeta.IsConflicting(),
		ReferencedByMilestoneIndex:   referencedByMilestoneIndex,
		MilestoneTimestampReferenced: milestoneTimestampReferenced,
		MilestoneIndex:               milestoneIndex,
		LedgerIndex:                  s.Database.LedgerIndex(),
	}, nil
}
