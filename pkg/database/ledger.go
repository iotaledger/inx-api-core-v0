package database

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
	"github.com/iotaledger/iota.go/consts"
)

const (
	ledgerMilestoneIndexKey = "ledgerMilestoneIndex"
)

func databaseKeyForAddress(address hornet.Hash) []byte {
	return address[:49]
}

func balanceFromBytes(bytes []byte) uint64 {
	return binary.LittleEndian.Uint64(bytes)
}

func diffFromBytes(bytes []byte) int64 {
	return int64(balanceFromBytes(bytes))
}

func milestoneIndexFromBytes(bytes []byte) milestone.Index {
	return milestone.Index(binary.LittleEndian.Uint32(bytes))
}

func (db *Database) BalanceForAddress(address hornet.Hash) (uint64, milestone.Index, error) {

	value, err := db.ledgerBalanceStore.Get(databaseKeyForAddress(address))
	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			return 0, db.LedgerIndex(), fmt.Errorf("%w: failed to retrieve balance", err)
		}

		return 0, db.LedgerIndex(), nil
	}

	return balanceFromBytes(value), db.LedgerIndex(), err
}

// LedgerDiffForMilestone returns the ledger changes of that specific milestone.
func (db *Database) LedgerDiffForMilestone(ctx context.Context, targetIndex milestone.Index) (map[string]int64, error) {

	solidMilestoneIndex := db.SolidMilestoneIndex()
	if targetIndex > solidMilestoneIndex {
		return nil, fmt.Errorf("target index is too new. maximum: %d, actual: %d", solidMilestoneIndex, targetIndex)
	}

	if targetIndex <= db.snapshot.PruningIndex {
		return nil, fmt.Errorf("target index is too old. minimum: %d, actual: %d", db.snapshot.PruningIndex+1, targetIndex)
	}

	diff := make(map[string]int64)

	keyPrefix := databaseKeyForMilestoneIndex(targetIndex)

	aborted := false
	err := db.ledgerDiffStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		select {
		case <-ctx.Done():
			aborted = true

			return false
		default:
		}
		// Remove prefix from key
		diff[string(key[len(keyPrefix):len(keyPrefix)+49])] = diffFromBytes(value)

		return true
	})

	if err != nil {
		return nil, err
	}

	if aborted {
		return nil, ErrOperationAborted
	}

	var diffSum int64
	for _, change := range diff {
		diffSum += change
	}

	if diffSum != 0 {
		panic(fmt.Sprintf("LedgerDiffForMilestone(): Ledger diff for milestone %d does not sum up to zero", targetIndex))
	}

	return diff, nil
}

func (db *Database) LedgerStateForMilestone(ctx context.Context, targetIndex milestone.Index) (map[string]uint64, milestone.Index, error) {

	solidMilestoneIndex := db.SolidMilestoneIndex()
	if targetIndex == 0 {
		targetIndex = solidMilestoneIndex
	}

	if targetIndex > solidMilestoneIndex {
		return nil, 0, fmt.Errorf("target index is too new. maximum: %d, actual: %d", solidMilestoneIndex, targetIndex)
	}

	if targetIndex <= db.snapshot.PruningIndex {
		return nil, 0, fmt.Errorf("target index is too old. minimum: %d, actual: %d", db.snapshot.PruningIndex+1, targetIndex)
	}

	balances, ledgerMilestone, err := db.LedgerStateForLSMI(ctx)
	if err != nil {
		if errors.Is(err, ErrOperationAborted) {
			return nil, 0, err
		}

		return nil, 0, fmt.Errorf("ledgerStateForLSMI failed! %w", err)
	}

	if ledgerMilestone != solidMilestoneIndex {
		return nil, 0, fmt.Errorf("ledgerMilestone wrong! %d/%d", ledgerMilestone, solidMilestoneIndex)
	}

	// Calculate balances for targetIndex
	for milestoneIndex := solidMilestoneIndex; milestoneIndex > targetIndex; milestoneIndex-- {
		diff, err := db.LedgerDiffForMilestone(ctx, milestoneIndex)
		if err != nil {
			if errors.Is(err, ErrOperationAborted) {
				return nil, 0, err
			}

			return nil, 0, fmt.Errorf("ledgerDiffForMilestone: %w", err)
		}

		for address, change := range diff {
			select {
			case <-ctx.Done():
				return nil, 0, ErrOperationAborted
			default:
			}

			newBalance := int64(balances[address]) - change

			switch {
			case newBalance < 0:
				return nil, 0, fmt.Errorf("ledger diff for milestone %d creates negative balance for address %s: current %d, diff %d", milestoneIndex, hornet.Hash(address).Trytes(), balances[address], change)
			case newBalance == 0:
				delete(balances, address)
			default:
				balances[address] = uint64(newBalance)
			}
		}
	}

	return balances, targetIndex, nil
}

// LedgerStateForLSMI returns all balances for the current solid milestone.
func (db *Database) LedgerStateForLSMI(ctx context.Context) (map[string]uint64, milestone.Index, error) {

	balances := make(map[string]uint64)

	aborted := false
	err := db.ledgerBalanceStore.Iterate(kvstore.EmptyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		select {
		case <-ctx.Done():
			aborted = true

			return false
		default:
		}

		balances[string(key[:49])] = balanceFromBytes(value)

		return true
	})
	if err != nil {
		return nil, db.LedgerIndex(), err
	}

	if aborted {
		return nil, db.LedgerIndex(), ErrOperationAborted
	}

	var total uint64
	for _, value := range balances {
		total += value
	}

	if total != consts.TotalSupply {
		panic(fmt.Sprintf("total does not match supply: %d != %d", total, consts.TotalSupply))
	}

	return balances, db.LedgerIndex(), err
}
