package database

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/core/bitmask"
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
)

const (
	TransactionMetadataSolid       = 0
	TransactionMetadataConfirmed   = 1
	TransactionMetadataConflicting = 2
	TransactionMetadataIsHead      = 3
	TransactionMetadataIsTail      = 4
	TransactionMetadataIsValue     = 5
)

type TransactionMetadata struct {
	txHash hornet.Hash

	// Metadata
	metadata bitmask.BitMask

	// The index of the milestone which confirmed this tx
	confirmationIndex milestone.Index

	// trunkHash is the trunk of the transaction
	trunkHash hornet.Hash

	// branchHash is the branch of the transaction
	branchHash hornet.Hash

	// bundleHash is the bundle of the transaction
	bundleHash hornet.Hash
}

func NewTransactionMetadata(txHash hornet.Hash) *TransactionMetadata {
	return &TransactionMetadata{
		txHash: txHash,
	}
}

func (m *TransactionMetadata) GetTxHash() hornet.Hash {
	return m.txHash
}

func (m *TransactionMetadata) GetTrunkHash() hornet.Hash {
	return m.trunkHash
}

func (m *TransactionMetadata) GetBranchHash() hornet.Hash {
	return m.branchHash
}

func (m *TransactionMetadata) GetBundleHash() hornet.Hash {
	return m.bundleHash
}

func (m *TransactionMetadata) IsTail() bool {
	return m.metadata.HasBit(TransactionMetadataIsTail)
}

func (m *TransactionMetadata) IsConfirmed() bool {
	return m.metadata.HasBit(TransactionMetadataConfirmed)
}

func (m *TransactionMetadata) GetConfirmed() (bool, milestone.Index) {
	return m.metadata.HasBit(TransactionMetadataConfirmed), m.confirmationIndex
}

func (m *TransactionMetadata) IsConflicting() bool {
	return m.metadata.HasBit(TransactionMetadataConflicting)
}

func (m *TransactionMetadata) SetAdditionalTxInfo(trunkHash hornet.Hash, branchHash hornet.Hash, bundleHash hornet.Hash, isHead bool, isTail bool, isValue bool) {
	m.trunkHash = trunkHash
	m.branchHash = branchHash
	m.bundleHash = bundleHash
	m.metadata = m.metadata.ModifyBit(TransactionMetadataIsHead, isHead).ModifyBit(TransactionMetadataIsTail, isTail).ModifyBit(TransactionMetadataIsValue, isValue)
}

func (m *TransactionMetadata) Unmarshal(data []byte) error {
	/*
		1 byte  metadata bitmask
		4 bytes uint32 solidificationTimestamp
		4 bytes uint32 confirmationIndex
		4 bytes uint32 youngestRootSnapshotIndex
		4 bytes uint32 oldestRootSnapshotIndex
		4 bytes uint32 rootSnapshotCalculationIndex
		49 bytes hash trunk
		49 bytes hash branch
		49 bytes hash bundle
	*/

	m.metadata = bitmask.BitMask(data[0])
	m.confirmationIndex = milestone.Index(binary.LittleEndian.Uint32(data[5:9]))

	if len(data) > 17 {
		if len(data) == 21+49+49+49 {
			m.trunkHash = hornet.Hash(data[21 : 21+49])
			m.branchHash = hornet.Hash(data[21+49 : 21+49+49])
			m.bundleHash = hornet.Hash(data[21+49+49 : 21+49+49+49])
		}
	}

	return nil
}

func (db *Database) GetTxMetadataOrNil(txHash hornet.Hash) *TransactionMetadata {
	key := txHash

	data, err := db.metadataStore.Get(key)
	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			panic(fmt.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	txMeta, err := metadataFactory(key, data)
	if err != nil {
		panic(err)
	}

	db.addAdditionalTxInfoToMetadata(txMeta)

	return txMeta
}

func (db *Database) addAdditionalTxInfoToMetadata(metadata *TransactionMetadata) {
	trunkHash := metadata.GetTrunkHash()
	branchHash := metadata.GetTrunkHash()

	if len(trunkHash) == 0 || len(branchHash) == 0 {
		tx := db.GetTransactionOrNil(metadata.GetTxHash())
		if tx == nil {
			panic(fmt.Sprintf("transaction not found for metadata: %v", metadata.GetTxHash().Trytes()))
		}

		metadata.SetAdditionalTxInfo(tx.GetTrunkHash(), tx.GetBranchHash(), tx.GetBundleHash(), tx.IsHead(), tx.IsTail(), tx.IsValue())
	}
}
