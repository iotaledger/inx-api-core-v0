package server

import (
	"fmt"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/iota.go/trinary"

	"github.com/iotaledger/inx-api-core-v0/pkg/database"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
)

// Container holds an object.
type Container interface {
	Item() Container
}

type newTxWithValueFunc[T Container] func(txHash trinary.Hash, address trinary.Hash, index uint64, value int64) T
type newTxHashWithValueFunc[H Container] func(txHash trinary.Hash, tailTxHash trinary.Hash, bundleHash trinary.Hash, address trinary.Hash, value int64) H
type newBundleWithValueFunc[B Container, T Container] func(bundleHash trinary.Hash, tailTxHash trinary.Hash, transactions []T, lastIndex uint64) B

//nolint:nonamedreturns
func getMilestoneStateDiff[T Container, H Container, B Container](db *database.Database, milestoneIndex milestone.Index, newTxWithValue newTxWithValueFunc[T], newTxHashWithValue newTxHashWithValueFunc[H], newBundleWithValue newBundleWithValueFunc[B, T]) (confirmedTxWithValue []H, confirmedBundlesWithValue []B, totalLedgerChanges map[string]int64, err error) {

	msBndl := db.MilestoneBundleOrNil(milestoneIndex)
	if msBndl == nil {
		return nil, nil, nil, fmt.Errorf("milestone not found: %d", milestoneIndex)
	}

	txsToConfirm := make(map[string]struct{})
	txsToTraverse := make(map[string]struct{})
	totalLedgerChanges = make(map[string]int64)

	txsToTraverse[string(msBndl.TailHash())] = struct{}{}

	// Collect all tx to check by traversing the tangle
	// Loop as long as new transactions are added in every loop cycle
	for len(txsToTraverse) != 0 {

		for txHash := range txsToTraverse {
			delete(txsToTraverse, txHash)

			if _, checked := txsToConfirm[txHash]; checked {
				// Tx was already checked => ignore
				continue
			}

			if db.SolidEntryPointsContain(hornet.Hash(txHash)) {
				// Ignore solid entry points (snapshot milestone included)
				continue
			}

			txMeta := db.TxMetadataOrNil(hornet.Hash(txHash))
			if txMeta == nil {
				return nil, nil, nil, fmt.Errorf("getMilestoneStateDiff: transaction not found: %v", hornet.Hash(txHash).Trytes())
			}

			confirmed, at := txMeta.ConfirmedWithIndex()
			if confirmed {
				if at != milestoneIndex {
					// ignore all tx that were confirmed by another milestone
					continue
				}
			} else {
				return nil, nil, nil, fmt.Errorf("getMilestoneStateDiff: transaction not confirmed yet: %v", hornet.Hash(txHash).Trytes())
			}

			// Mark the approvees to be traversed
			txsToTraverse[string(txMeta.TrunkHash())] = struct{}{}
			txsToTraverse[string(txMeta.BranchHash())] = struct{}{}

			if !txMeta.IsTail() {
				continue
			}

			bndl := db.BundleOrNil(hornet.Hash(txHash))
			if bndl == nil {
				txBundle := txMeta.BundleHash()

				return nil, nil, nil, fmt.Errorf("getMilestoneStateDiff: Tx: %v, bundle not found: %v", hornet.Hash(txHash).Trytes(), txBundle.Trytes())
			}

			if !bndl.IsValid() {
				txBundle := txMeta.BundleHash()

				return nil, nil, nil, fmt.Errorf("getMilestoneStateDiff: Tx: %v, bundle not valid: %v", hornet.Hash(txHash).Trytes(), txBundle.Trytes())
			}

			if !bndl.IsValueSpam() {
				ledgerChanges := bndl.LedgerChanges()

				var txsWithValue []T

				txs := bndl.Transactions()
				for _, hornetTx := range txs {
					// hornetTx is being retained during the loop, so safe to use the pointer here
					if hornetTx.Tx.Value != 0 {
						confirmedTxWithValue = append(confirmedTxWithValue, newTxHashWithValue(hornetTx.Tx.Hash, bndl.TailHash().Trytes(), hornetTx.Tx.Bundle, hornetTx.Tx.Address, hornetTx.Tx.Value))
					}
					txsWithValue = append(txsWithValue, newTxWithValue(hornetTx.Tx.Hash, hornetTx.Tx.Address, hornetTx.Tx.CurrentIndex, hornetTx.Tx.Value))
				}
				for address, change := range ledgerChanges {
					totalLedgerChanges[address] += change
				}

				bundleHeadTx := bndl.Head()
				confirmedBundlesWithValue = append(confirmedBundlesWithValue, newBundleWithValue(txMeta.BundleHash().Trytes(), bndl.TailHash().Trytes(), txsWithValue, bundleHeadTx.Tx.CurrentIndex))
			}

			// we only add the tail transaction to the txsToConfirm set, in order to not
			// accidentally skip cones, in case the other transactions (non-tail) of the bundle do not
			// reference the same trunk transaction (as seen from the PoV of the bundle).
			// if we wouldn't do it like this, we have a high chance of computing an
			// inconsistent ledger state.
			txsToConfirm[txHash] = struct{}{}
		}
	}

	return confirmedTxWithValue, confirmedBundlesWithValue, totalLedgerChanges, nil
}

func (s *DatabaseServer) rpcGetLedgerState(c echo.Context) (interface{}, error) {
	request := &GetLedgerState{}
	if err := c.Bind(request); err != nil {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	balances, index, err := s.Database.LedgerStateForMilestone(c.Request().Context(), request.TargetIndex)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	balancesTrytes := make(map[trinary.Trytes]uint64)
	for address, balance := range balances {
		balancesTrytes[hornet.Hash(address).Trytes()] = balance
	}

	return &GetLedgerStateResponse{
		Balances:       balancesTrytes,
		MilestoneIndex: index,
	}, nil
}

func (s *DatabaseServer) rpcGetLedgerDiff(c echo.Context) (interface{}, error) {
	request := &GetLedgerDiff{}
	if err := c.Bind(request); err != nil {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	smi := s.Database.SolidMilestoneIndex()
	requestedIndex := request.MilestoneIndex
	if requestedIndex > smi {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid milestone index: %d, lsmi is %d", requestedIndex, smi)
	}

	diff, err := s.Database.LedgerDiffForMilestone(c.Request().Context(), requestedIndex)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	diffTrytes := make(map[trinary.Trytes]int64)
	for address, balance := range diff {
		diffTrytes[hornet.Hash(address).Trytes()] = balance
	}

	return &GetLedgerDiffResponse{
		Diff:           diffTrytes,
		MilestoneIndex: request.MilestoneIndex,
	}, nil
}

func (s *DatabaseServer) rpcGetLedgerDiffExt(c echo.Context) (interface{}, error) {
	request := &GetLedgerDiffExt{}
	if err := c.Bind(request); err != nil {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid request, error: %s", err)
	}

	smi := s.Database.SolidMilestoneIndex()
	requestedIndex := request.MilestoneIndex
	if requestedIndex > smi {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid milestone index: %d, lsmi is %d", requestedIndex, smi)
	}

	newTxWithValue := func(txHash trinary.Hash, address trinary.Hash, index uint64, value int64) *TxWithValue {
		return &TxWithValue{
			TxHash:  txHash,
			Address: address,
			Index:   index,
			Value:   value,
		}
	}

	newTxHashWithValue := func(txHash trinary.Hash, tailTxHash trinary.Hash, bundleHash trinary.Hash, address trinary.Hash, value int64) *TxHashWithValue {
		return &TxHashWithValue{
			TxHash:     txHash,
			TailTxHash: tailTxHash,
			BundleHash: bundleHash,
			Address:    address,
			Value:      value,
		}
	}

	newBundleWithValue := func(bundleHash trinary.Hash, tailTxHash trinary.Hash, transactions []*TxWithValue, lastIndex uint64) *BundleWithValue {
		return &BundleWithValue{
			BundleHash: bundleHash,
			TailTxHash: tailTxHash,
			Txs:        transactions,
			LastIndex:  lastIndex,
		}
	}

	confirmedTxWithValue, confirmedBundlesWithValue, ledgerChanges, err := getMilestoneStateDiff(s.Database, requestedIndex, newTxWithValue, newTxHashWithValue, newBundleWithValue)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	ledgerChangesTrytes := make(map[trinary.Trytes]int64)
	for address, balance := range ledgerChanges {
		ledgerChangesTrytes[hornet.Hash(address).Trytes()] = balance
	}

	result := &GetLedgerDiffExtResponse{}
	result.ConfirmedTxWithValue = confirmedTxWithValue
	result.ConfirmedBundlesWithValue = confirmedBundlesWithValue
	result.Diff = ledgerChangesTrytes
	result.MilestoneIndex = request.MilestoneIndex

	return result, nil
}

func (s *DatabaseServer) ledgerState(c echo.Context, targetIndex milestone.Index) (interface{}, error) {
	balances, index, err := s.Database.LedgerStateForMilestone(c.Request().Context(), targetIndex)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	addressesWithBalances := make(map[trinary.Trytes]string)
	for address, balance := range balances {
		addressesWithBalances[hornet.Hash(address).Trytes()] = strconv.FormatUint(balance, 10)
	}

	return &ledgerStateResponse{
		Balances:    addressesWithBalances,
		LedgerIndex: index,
	}, nil
}

func (s *DatabaseServer) ledgerStateByLatestSolidIndex(c echo.Context) (interface{}, error) {
	return s.ledgerState(c, 0)
}

func (s *DatabaseServer) ledgerStateByIndex(c echo.Context) (interface{}, error) {
	msIndex, err := httpserver.ParseMilestoneIndexParam(c, ParameterMilestoneIndex)
	if err != nil {
		return nil, err
	}

	return s.ledgerState(c, milestone.Index(msIndex))
}

func (s *DatabaseServer) ledgerDiff(c echo.Context) (interface{}, error) {
	msIndexIotaGo, err := httpserver.ParseMilestoneIndexParam(c, ParameterMilestoneIndex)
	if err != nil {
		return nil, err
	}
	msIndex := milestone.Index(msIndexIotaGo)

	smi := s.Database.SolidMilestoneIndex()
	if msIndex > smi {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid milestone index: %d, lsmi is %d", msIndex, smi)
	}

	diff, err := s.Database.LedgerDiffForMilestone(c.Request().Context(), msIndex)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	addressesWithDiffs := make(map[trinary.Trytes]string)
	for address, balance := range diff {
		addressesWithDiffs[hornet.Hash(address).Trytes()] = strconv.FormatInt(balance, 10)
	}

	return &ledgerDiffResponse{
		AddressDiffs: addressesWithDiffs,
		LedgerIndex:  msIndex,
	}, nil
}

func (s *DatabaseServer) ledgerDiffExtended(c echo.Context) (interface{}, error) {
	msIndexIotaGo, err := httpserver.ParseMilestoneIndexParam(c, ParameterMilestoneIndex)
	if err != nil {
		return nil, err
	}
	msIndex := milestone.Index(msIndexIotaGo)

	smi := s.Database.SolidMilestoneIndex()
	if msIndex > smi {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid milestone index: %d, lsmi is %d", msIndex, smi)
	}

	newTxWithValue := func(txHash trinary.Hash, address trinary.Hash, index uint64, value int64) *txWithValue {
		return &txWithValue{
			TxHash:  txHash,
			Address: address,
			Index:   uint32(index),
			Value:   strconv.FormatInt(value, 10),
		}
	}

	newTxHashWithValue := func(txHash trinary.Hash, tailTxHash trinary.Hash, bundleHash trinary.Hash, address trinary.Hash, value int64) *txHashWithValue {
		return &txHashWithValue{
			TxHash:     txHash,
			TailTxHash: tailTxHash,
			Bundle:     bundleHash,
			Address:    address,
			Value:      strconv.FormatInt(value, 10),
		}
	}

	newBundleWithValue := func(bundleHash trinary.Hash, tailTxHash trinary.Hash, transactions []*txWithValue, lastIndex uint64) *bundleWithValue {
		return &bundleWithValue{
			Bundle:     bundleHash,
			TailTxHash: tailTxHash,
			Txs:        transactions,
			LastIndex:  uint32(lastIndex),
		}
	}

	confirmedTxWithValue, confirmedBundlesWithValue, ledgerChanges, err := getMilestoneStateDiff(s.Database, msIndex, newTxWithValue, newTxHashWithValue, newBundleWithValue)
	if err != nil {
		return nil, errors.WithMessage(echo.ErrInternalServerError, err.Error())
	}

	addressesWithDiffs := make(map[trinary.Trytes]string)
	for address, balance := range ledgerChanges {
		addressesWithDiffs[hornet.Hash(address).Trytes()] = strconv.FormatInt(balance, 10)
	}

	return ledgerDiffExtendedResponse{
		ConfirmedTxWithValue:      confirmedTxWithValue,
		ConfirmedBundlesWithValue: confirmedBundlesWithValue,
		AddressDiffs:              addressesWithDiffs,
		LedgerIndex:               msIndex,
	}, nil
}
