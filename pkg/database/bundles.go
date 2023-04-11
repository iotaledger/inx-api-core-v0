package database

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/ds/bitmask"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
	"github.com/iotaledger/iota.go/trinary"
)

func databaseKeyForBundle(tailTxHash hornet.Hash) []byte {
	return tailTxHash
}

func bundleFactory(db *Database, key []byte, data []byte) (*Bundle, error) {
	bndl := &Bundle{
		db:     db,
		tailTx: key[:49],
		txs:    make(map[string]struct{}),
	}

	if err := bndl.Unmarshal(data); err != nil {
		return nil, err
	}

	return bndl, nil
}

const (
	MetadataSolid                = 0
	MetadataValid                = 1
	MetadataConfirmed            = 2
	MetadataIsMilestone          = 3
	MetadataIsValueSpam          = 4
	MetadataValidStrictSemantics = 5
	MetadataConflicting          = 6
	MetadataInvalidPastCone      = 7
)

type Bundle struct {
	db *Database

	// Key
	tailTx hornet.Hash

	// Value
	metadata      bitmask.BitMask
	lastIndex     uint64
	hash          hornet.Hash
	headTx        hornet.Hash
	txs           map[string]struct{}
	ledgerChanges map[string]int64

	milestoneIndexOnce sync.Once
	milestoneIndex     milestone.Index
}

func (bundle *Bundle) Unmarshal(data []byte) error {

	/*
		 1 byte  	   				metadata
		 8 bytes uint64 			lastIndex
		 8 bytes uint64 			txCount
		 8 bytes uint64 			ledgerChangesCount
		49 bytes					bundleHash
		49 bytes					headTx
		49 bytes                 	txHashes		(x txCount)
		49 bytes + 8 bytes uint64 	ledgerChanges	(x ledgerChangesCount)
	*/

	bundle.metadata = bitmask.BitMask(data[0])
	bundle.lastIndex = binary.LittleEndian.Uint64(data[1:9])
	txCount := int(binary.LittleEndian.Uint64(data[9:17]))
	ledgerChangesCount := int(binary.LittleEndian.Uint64(data[17:25]))
	bundle.hash = data[25:74]
	bundle.headTx = data[74:123]

	offset := 123
	for i := 0; i < txCount; i++ {
		bundle.txs[string(data[offset:offset+49])] = struct{}{}
		offset += 49
	}

	if ledgerChangesCount > 0 {
		bundle.ledgerChanges = make(map[string]int64, ledgerChangesCount)
	}

	for i := 0; i < ledgerChangesCount; i++ {
		address := data[offset : offset+49]
		offset += 49
		balance := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
		offset += 8
		bundle.ledgerChanges[string(address)] = balance
	}

	return nil
}

func (bundle *Bundle) GetLedgerChanges() map[string]int64 {
	return bundle.ledgerChanges
}

func (bundle *Bundle) GetHead() *Transaction {
	if len(bundle.headTx) == 0 {
		panic("head hash can never be empty")
	}

	return bundle.db.loadBundleTxIfExistsOrPanic(bundle.headTx, bundle.hash)
}

func (bundle *Bundle) GetTailHash() hornet.Hash {
	if len(bundle.tailTx) == 0 {
		panic("tail hash can never be empty")
	}

	return bundle.tailTx
}

func (bundle *Bundle) GetTail() *Transaction {
	if len(bundle.tailTx) == 0 {
		panic("tail hash can never be empty")
	}

	return bundle.db.loadBundleTxIfExistsOrPanic(bundle.tailTx, bundle.hash)
}

func (bundle *Bundle) GetTransactions() []*Transaction {

	txs := make([]*Transaction, 0, len(bundle.txs))
	for txHash := range bundle.txs {
		tx := bundle.db.loadBundleTxIfExistsOrPanic(hornet.Hash(txHash), bundle.hash)
		txs = append(txs, tx)
	}

	return txs
}

func (bundle *Bundle) IsValid() bool {
	return bundle.metadata.HasBit(MetadataValid)
}

func (bundle *Bundle) IsValueSpam() bool {
	return bundle.metadata.HasBit(MetadataIsValueSpam)
}

func (bundle *Bundle) GetMilestoneIndex() milestone.Index {
	bundle.milestoneIndexOnce.Do(func() {
		tailTx := bundle.GetTail()
		bundle.milestoneIndex = milestone.Index(trinary.TrytesToInt(tailTx.Tx.ObsoleteTag))
	})

	return bundle.milestoneIndex
}

func (bundle *Bundle) GetMilestoneHash() hornet.Hash {
	return bundle.tailTx
}

func (db *Database) loadBundleTxIfExistsOrPanic(txHash hornet.Hash, bundleHash hornet.Hash) *Transaction {
	tx := db.GetTransactionOrNil(txHash)
	if tx == nil {
		log.Panicf("bundle %s has a reference to a non persisted transaction: %s", bundleHash.Trytes(), txHash.Trytes())
	}

	return tx
}

func (db *Database) GetBundleOrNil(tailTxHash hornet.Hash) *Bundle {
	key := databaseKeyForBundle(tailTxHash)

	data, err := db.bundleStore.Get(key)
	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			panic(fmt.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	bundle, err := bundleFactory(db, key, data)
	if err != nil {
		panic(err)
	}

	return bundle
}
