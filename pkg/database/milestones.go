package database

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/ierrors"
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
		Hash:  hornet.Hash(data[:hornet.HashSize]),
	}
}

type Milestone struct {
	Index milestone.Index
	Hash  hornet.Hash
}

func (db *Database) MilestoneOrNil(milestoneIndex milestone.Index) *Milestone {
	key := databaseKeyForMilestoneIndex(milestoneIndex)

	data, err := db.milestoneStore.Get(key)
	if err != nil {
		if !ierrors.Is(err, kvstore.ErrKeyNotFound) {
			panic(ierrors.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	milestone := milestoneFactory(key, data)

	return milestone
}

// MilestoneBundleOrNil returns the Bundle of a milestone index or nil if it doesn't exist.
func (db *Database) MilestoneBundleOrNil(milestoneIndex milestone.Index) *Bundle {

	milestone := db.MilestoneOrNil(milestoneIndex)
	if milestone == nil {
		return nil
	}

	return db.BundleOrNil(milestone.Hash)
}

// MilestoneTimestamp returns the timestamp of a milestone.
func (db *Database) MilestoneTimestamp(milestoneIndex milestone.Index) (uint64, error) {

	milestone := db.MilestoneOrNil(milestoneIndex)
	if milestone == nil {
		return 0, ierrors.Errorf("milestone %d not found", milestoneIndex)
	}

	tx := db.TransactionOrNil(milestone.Hash)
	if tx == nil {
		return 0, ierrors.Errorf("milestone %d tail transaction not found", milestoneIndex)
	}

	return tx.Tx.Timestamp, nil
}

func (db *Database) LedgerIndex() milestone.Index {
	db.ledgerMilestoneIndexOnce.Do(func() {
		value, err := db.ledgerStore.Get([]byte(ledgerMilestoneIndexKey))
		if err != nil {
			panic(ierrors.Errorf("%w: failed to load ledger milestone index", err))
		}
		db.ledgerMilestoneIndex = milestoneIndexFromBytes(value)
	})

	return db.ledgerMilestoneIndex
}

// SolidMilestoneIndex returns the latest solid milestone index.
func (db *Database) SolidMilestoneIndex() milestone.Index {
	// the solid milestone index is always equal to the ledgerMilestoneIndex in "readonly" mode
	return db.LedgerIndex()
}

// LatestSolidMilestoneBundle returns the latest solid milestone bundle.
func (db *Database) LatestSolidMilestoneBundle() *Bundle {
	db.latestSolidMilestoneBundleOnce.Do(func() {
		latestSolidMilestoneIndex := db.SolidMilestoneIndex()
		latestSolidMilestoneBundle := db.MilestoneBundleOrNil(latestSolidMilestoneIndex)
		if latestSolidMilestoneBundle == nil {
			panic(ierrors.Errorf("latest solid milestone bundle not found: %d", latestSolidMilestoneIndex))
		}
		db.latestSolidMilestoneBundle = latestSolidMilestoneBundle
	})

	return db.latestSolidMilestoneBundle
}
