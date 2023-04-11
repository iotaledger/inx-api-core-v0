package database

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/inx-api-core-v0/pkg/compressed"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/iota.go/transaction"
)

type Transaction struct {
	trunkHashOnce  sync.Once
	branchHashOnce sync.Once
	bundleHashOnce sync.Once

	txHash     hornet.Hash
	trunkHash  hornet.Hash
	branchHash hornet.Hash
	bundleHash hornet.Hash

	// Decompressed iota.go Transaction containing Hash
	Tx *transaction.Transaction

	// TxTimestamp or, if available, AttachmentTimestamp
	timestamp int64
}

func NewTransaction(txHash hornet.Hash) *Transaction {
	return &Transaction{
		txHash: txHash,
	}
}

func getTimestampFromTx(transaction *transaction.Transaction) int64 {
	// Timestamp = Seconds elapsed since 00:00:00 UTC 1 January 1970
	timestamp := int64(transaction.Timestamp)
	if transaction.AttachmentTimestamp != 0 {
		// AttachmentTimestamp = Milliseconds elapsed since 00:00:00 UTC 1 January 1970
		timestamp = transaction.AttachmentTimestamp / 1000
	}

	return timestamp
}

func (tx *Transaction) GetTrunkHash() hornet.Hash {
	tx.trunkHashOnce.Do(func() {
		tx.trunkHash = hornet.HashFromHashTrytes(tx.Tx.TrunkTransaction)
	})

	return tx.trunkHash
}

func (tx *Transaction) GetBranchHash() hornet.Hash {
	tx.branchHashOnce.Do(func() {
		tx.branchHash = hornet.HashFromHashTrytes(tx.Tx.BranchTransaction)
	})

	return tx.branchHash
}

func (tx *Transaction) GetBundleHash() hornet.Hash {
	tx.bundleHashOnce.Do(func() {
		tx.bundleHash = hornet.HashFromHashTrytes(tx.Tx.Bundle)
	})

	return tx.bundleHash
}

func (tx *Transaction) IsTail() bool {
	return tx.Tx.CurrentIndex == 0
}

func (tx *Transaction) IsHead() bool {
	return tx.Tx.CurrentIndex == tx.Tx.LastIndex
}

func (tx *Transaction) IsValue() bool {
	return tx.Tx.Value != 0
}

func (tx *Transaction) Unmarshal(data []byte) error {

	/*
		x bytes RawBytes
	*/

	transactionHash := tx.txHash.Trytes()

	transaction, err := compressed.TransactionFromCompressedBytes(data, transactionHash)
	if err != nil {
		panic(err)
	}
	tx.Tx = transaction

	tx.timestamp = getTimestampFromTx(transaction)

	return nil
}

func transactionFactory(key []byte, data []byte) (*Transaction, error) {
	tx := NewTransaction(key[:49])

	if err := tx.Unmarshal(data); err != nil {
		return nil, err
	}

	return tx, nil
}

func metadataFactory(key []byte, data []byte) (*TransactionMetadata, error) {
	txMeta := NewTransactionMetadata(key[:49])

	if err := txMeta.Unmarshal(data); err != nil {
		return nil, err
	}

	return txMeta, nil
}

func (db *Database) GetTransactionOrNil(txHash hornet.Hash) *Transaction {
	key := txHash

	data, err := db.txStore.Get(key)
	if err != nil {
		if !errors.Is(err, kvstore.ErrKeyNotFound) {
			panic(fmt.Errorf("failed to get value from database: %w", err))
		}

		return nil
	}

	tx, err := transactionFactory(key, data)
	if err != nil {
		panic(err)
	}

	return tx
}
