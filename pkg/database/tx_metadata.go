package database

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/ds/bitmask"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/kvstore"
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
	TransactionMetadataIsMilestone = 6

	// metadata, confirmationIndex, trunkHash, branchHash, bundleHash, milestoneIndex.
	TransactionMetadataSize = 1 + milestone.IndexByteSize + hornet.HashSize + hornet.HashSize + hornet.HashSize + milestone.IndexByteSize
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

	// The index of the milestone (in case the transaction is a milestone)
	milestoneIndex milestone.Index
}

func NewTransactionMetadata(txHash hornet.Hash) *TransactionMetadata {
	return &TransactionMetadata{
		txHash: txHash,
	}
}

func (m *TransactionMetadata) TxHash() hornet.Hash {
	return m.txHash
}

func (m *TransactionMetadata) TrunkHash() hornet.Hash {
	return m.trunkHash
}

func (m *TransactionMetadata) BranchHash() hornet.Hash {
	return m.branchHash
}

func (m *TransactionMetadata) BundleHash() hornet.Hash {
	return m.bundleHash
}

func (m *TransactionMetadata) IsTail() bool {
	return m.metadata.HasBit(TransactionMetadataIsTail)
}

func (m *TransactionMetadata) IsSolid() bool {
	return m.metadata.HasBit(TransactionMetadataSolid)
}

func (m *TransactionMetadata) IsConfirmed() bool {
	return m.metadata.HasBit(TransactionMetadataConfirmed)
}

func (m *TransactionMetadata) ConfirmedWithIndex() (bool, milestone.Index) {
	return m.metadata.HasBit(TransactionMetadataConfirmed), m.confirmationIndex
}

func (m *TransactionMetadata) IsConflicting() bool {
	return m.metadata.HasBit(TransactionMetadataConflicting)
}

func (m *TransactionMetadata) IsMilestone() bool {
	return m.metadata.HasBit(TransactionMetadataIsMilestone)
}

func (m *TransactionMetadata) MilestoneIndex() milestone.Index {
	return m.milestoneIndex
}

func (m *TransactionMetadata) Marshal() []byte {
	/*
		1 byte   metadata	bitmask
		4 bytes  uint32 	confirmationIndex
		49 bytes hash 		trunk
		49 bytes hash 		branch
		49 bytes hash 		bundle
		4 bytes  uint32 	milestoneIndex
	*/

	hashSize := hornet.HashSize
	msIndexSize := milestone.IndexByteSize

	value := make([]byte, TransactionMetadataSize)
	value[0] = byte(m.metadata)
	binary.LittleEndian.PutUint32(value[1:1+msIndexSize], uint32(m.confirmationIndex))
	copy(value[1+msIndexSize+0*hashSize:1+msIndexSize+1*hashSize], m.trunkHash)
	copy(value[1+msIndexSize+1*hashSize:1+msIndexSize+2*hashSize], m.branchHash)
	copy(value[1+msIndexSize+2*hashSize:1+msIndexSize+3*hashSize], m.bundleHash)
	binary.LittleEndian.PutUint32(value[1+msIndexSize+3*hashSize:], uint32(m.milestoneIndex))

	return value
}

func (m *TransactionMetadata) Unmarshal(data []byte) error {
	/*
		1 byte   metadata	bitmask
		4 bytes  uint32 	confirmationIndex
		49 bytes hash 		trunk
		49 bytes hash 		branch
		49 bytes hash 		bundle
		4 bytes  uint32 	milestoneIndex
	*/

	hashSize := hornet.HashSize
	msIndexSize := milestone.IndexByteSize

	m.metadata = bitmask.BitMask(data[0])
	m.confirmationIndex = milestone.Index(binary.LittleEndian.Uint32(data[1 : 1+msIndexSize]))
	m.trunkHash = hornet.Hash(data[1+msIndexSize+0*hashSize : 1+msIndexSize+1*hashSize])
	m.branchHash = hornet.Hash(data[1+msIndexSize+1*hashSize : 1+msIndexSize+2*hashSize])
	m.bundleHash = hornet.Hash(data[1+msIndexSize+2*hashSize : 1+msIndexSize+3*hashSize])
	m.milestoneIndex = milestone.Index(binary.LittleEndian.Uint32(data[1+msIndexSize+3*hashSize:]))

	return nil
}

func (db *Database) TxMetadataOrNil(txHash hornet.Hash) *TransactionMetadata {
	key := txHash

	data, err := db.metadataStore.Get(key)
	if err != nil {
		if !ierrors.Is(err, kvstore.ErrKeyNotFound) {
			panic(ierrors.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	txMeta, err := metadataFactory(key, data)
	if err != nil {
		panic(err)
	}

	return txMeta
}
