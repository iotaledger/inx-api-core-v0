package database

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
)

func databaseKeyForMilestoneIndex(milestoneIndex milestone.Index) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(milestoneIndex))

	return bytes
}

func milestoneIndexFromDatabaseKey(key []byte) milestone.Index {
	return milestone.Index(binary.LittleEndian.Uint32(key))
}

func milestoneFactory(key []byte, data []byte) *Milestone {
	return &Milestone{
		Index: milestoneIndexFromDatabaseKey(key),
		Hash:  hornet.Hash(data[:49]),
	}
}

type Milestone struct {
	Index milestone.Index
	Hash  hornet.Hash
}

func (db *Database) GetMilestoneOrNil(milestoneIndex milestone.Index) *Milestone {
	key := databaseKeyForMilestoneIndex(milestoneIndex)

	data, err := db.milestoneStore.Get(key)
	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			panic(fmt.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	milestone := milestoneFactory(key, data)

	return milestone
}

// GetMilestoneBundleOrNil returns the Bundle of a milestone index or nil if it doesn't exist.
func (db *Database) GetMilestoneBundleOrNil(milestoneIndex milestone.Index) *Bundle {

	milestone := db.GetMilestoneOrNil(milestoneIndex)
	if milestone == nil {
		return nil
	}

	return db.GetBundleOrNil(milestone.Hash)
}

func (db *Database) GetLedgerIndex() milestone.Index {
	db.ledgerMilestoneIndexOnce.Do(func() {
		value, err := db.ledgerStore.Get([]byte(ledgerMilestoneIndexKey))
		if err != nil {
			panic(fmt.Errorf("%w: failed to load ledger milestone index", err))
		}
		db.ledgerMilestoneIndex = milestoneIndexFromBytes(value)
	})

	return db.ledgerMilestoneIndex
}

// GetSolidMilestoneIndex returns the latest solid milestone index.
func (db *Database) GetSolidMilestoneIndex() milestone.Index {
	// the solid milestone index is always equal to the ledgerMilestoneIndex in "readonly" mode
	return db.GetLedgerIndex()
}

// GetLatestSolidMilestoneBundle returns the latest solid milestone bundle.
func (db *Database) GetLatestSolidMilestoneBundle() *Bundle {
	db.latestSolidMilestoneBundleOnce.Do(func() {
		latestSolidMilestoneIndex := db.GetSolidMilestoneIndex()
		latestSolidMilestoneBundle := db.GetMilestoneBundleOrNil(latestSolidMilestoneIndex)
		if latestSolidMilestoneBundle == nil {
			panic(fmt.Errorf("latest solid milestone bundle not found: %d", latestSolidMilestoneIndex))
		}
		db.latestSolidMilestoneBundle = latestSolidMilestoneBundle
	})

	return db.latestSolidMilestoneBundle
}
