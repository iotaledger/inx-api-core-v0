package database

import (
	"context"
	"encoding/binary"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/iotaledger/hive.go/ds/bitmask"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/runtime/contextutils"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
	"github.com/iotaledger/iota.go/trinary"
)

func supportedDatabaseVersionUpgradeFunc(oldVersion, newVersion byte) error {
	if !(oldVersion == 2 && newVersion == 3) {
		return ierrors.Errorf("unsupported database version migration from %d to %d", oldVersion, newVersion)
	}

	return nil
}

func migrateTangleDatabaseFunc(ctx context.Context, logger *logger.Logger, tangleDatabase kvstore.KVStore, oldVersion, newVersion byte) error {
	if err := supportedDatabaseVersionUpgradeFunc(oldVersion, newVersion); err != nil {
		return err
	}

	// in version 3, we add the information about milestones to the transaction metadata.
	// we also determine all missing information once and store it in the metadata.
	// this is only possible because the legacy database is complete and we will never add new transactions to it.
	txStore := lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixTransactions}))
	metadataStore := lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixTransactionMetadata}))
	bundleStore := lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixBundles}))
	bundleTransactionsStore := lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixBundleTransactions}))

	getBundleMilestoneIndex := func(bundle *Bundle) (milestone.Index, error) {
		if len(bundle.tailTx) == 0 {
			return 0, ierrors.Errorf("tail hash can never be empty, bundle: %s", bundle.hash.Trytes())
		}

		tailTxData, err := txStore.Get(bundle.tailTx)
		if err != nil {
			if !ierrors.Is(err, kvstore.ErrKeyNotFound) {
				return 0, ierrors.Wrapf(err, "failed to get transaction %s from database", bundle.tailTx.Trytes())
			}

			return 0, ierrors.Errorf("bundle %s has a reference to a non persisted transaction: %s", bundle.hash.Trytes(), bundle.tailTx.Trytes())
		}

		tailTx, err := transactionFactory(bundle.tailTx, tailTxData)
		if err != nil {
			return 0, ierrors.Wrapf(err, "failed to parse data for transaction %s", bundle.tailTx.Trytes())
		}

		return milestone.Index(trinary.TrytesToInt(tailTx.Tx.ObsoleteTag)), nil
	}

	// we add a cache to speed up the migration
	milestoneCache := lo.PanicOnErr(lru.New[[49]byte, milestone.Index](10_000_000))
	defer milestoneCache.Purge()

	lastStatusTime := time.Now()
	var transactionsCounter int64

	var innerErr error

	// iterate over all transaction metadata
	if err := metadataStore.Iterate(kvstore.EmptyPrefix, func(key kvstore.Key, data kvstore.Value) bool {
		transactionsCounter++

		txHash := hornet.Hash(key[:hornet.HashSize])

		// print status to show progress
		if time.Since(lastStatusTime) >= printStatusInterval {
			lastStatusTime = time.Now()

			// check if the context was already canceled
			if err := contextutils.ReturnErrIfCtxDone(ctx, ErrOperationAborted); err != nil {
				innerErr = err
				return false
			}

			logger.Infof("	analyzed %d transactions", transactionsCounter)
		}

		txMeta := NewTransactionMetadata(txHash)
		// we only care about the metadata and the confirmation index in the migration,
		// the rest is set by the migration logic itself.
		/*
			Old database version v2 format:
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
		txMeta.metadata = bitmask.BitMask(data[0])
		txMeta.confirmationIndex = milestone.Index(binary.LittleEndian.Uint32(data[1+milestone.IndexByteSize:]))

		// get the transaction data
		txData, err := txStore.Get(txHash)
		if err != nil {
			if !ierrors.Is(err, kvstore.ErrKeyNotFound) {
				innerErr = ierrors.Wrapf(err, "failed to get transaction %s from database", txHash.Trytes())
				return false
			}
			innerErr = ierrors.Errorf("transaction %s not found", txHash.Trytes())

			return false
		}

		tx, err := transactionFactory(txHash, txData)
		if err != nil {
			innerErr = ierrors.Wrapf(err, "failed to parse data for transaction %s", txHash.Trytes())
			return false
		}

		milestoneCacheKey := [49]byte(tx.BundleHash())
		milestoneIndex, ok := milestoneCache.Get(milestoneCacheKey)
		if !ok {
			lastBundleStatusTime := time.Now()
			var bundleTransactionsCounter int64

			// the entry for the bundle doesn't exist yet in the cache
			if err := bundleTransactionsStore.IterateKeys(append(databaseKeyPrefixForBundleHash(tx.BundleHash()), BundleTxIsTail), func(key []byte) bool {
				bundleTransactionsCounter++

				// print status to show progress
				if time.Since(lastBundleStatusTime) >= printStatusInterval {
					lastBundleStatusTime = time.Now()

					// check if the context was already canceled
					if err := contextutils.ReturnErrIfCtxDone(ctx, ErrOperationAborted); err != nil {
						innerErr = err
						return false
					}

					logger.Infof("		analyzed %d bundle transactions for bundle %s", bundleTransactionsCounter, tx.BundleHash().Trytes())
				}

				bundleKey := databaseKeyForBundle(key[50:99]) // tailTxHash in that case, because we only search for "BundleTxIsTail"

				bundleData, err := bundleStore.Get(bundleKey)
				if err != nil {
					if !ierrors.Is(err, kvstore.ErrKeyNotFound) {
						innerErr = ierrors.Wrapf(err, "failed to get bundle %s from database", hornet.Hash(bundleKey).Trytes())
						return false
					}

					// bundle doesn't exist for this tailTxHash, keep searching
					return true
				}

				bundle, err := bundleFactory(nil, bundleKey, bundleData)
				if err != nil {
					innerErr = ierrors.Wrapf(err, "failed to get bundle %s from database", hornet.Hash(bundleKey).Trytes())
					return false
				}

				if bundle.IsMilestone() {
					msIndex, err := getBundleMilestoneIndex(bundle)
					if err != nil {
						innerErr = ierrors.Wrapf(err, "failed to get milestone index for bundle %s", hornet.Hash(bundleKey).Trytes())
						return false
					}

					milestoneIndex = msIndex

					// we found the milestone, stop searching
					return false
				}

				// another attachment could be a milestone, keep searching
				return true
			}); err != nil {
				innerErr = ierrors.Wrapf(err, "failed to iterate over all bundle transactions, bundle: %s", tx.BundleHash().Trytes())
				return false
			}

			// set the entry in the cache, so we don't need to search for it again
			milestoneCache.Add(milestoneCacheKey, milestoneIndex)
		}

		// set the metadata
		txMeta.metadata = txMeta.metadata.
			ModifyBit(TransactionMetadataIsHead, tx.IsHead()).
			ModifyBit(TransactionMetadataIsTail, tx.IsTail()).
			ModifyBit(TransactionMetadataIsValue, tx.IsValue()).
			ModifyBit(TransactionMetadataIsMilestone, milestoneIndex != 0)
		txMeta.trunkHash = tx.TrunkHash()
		txMeta.branchHash = tx.BranchHash()
		txMeta.bundleHash = tx.BundleHash()
		txMeta.milestoneIndex = milestoneIndex

		if err := metadataStore.Set(key, txMeta.Marshal()); err != nil {
			innerErr = ierrors.Wrapf(err, "failed to set updated transaction metadata, txHash: %s", txHash.Trytes())
			return false
		}

		return true
	}); err != nil {
		return ierrors.Wrap(err, "failed to iterate over all transaction metadata")
	}

	if innerErr != nil {
		return innerErr
	}

	return nil
}
