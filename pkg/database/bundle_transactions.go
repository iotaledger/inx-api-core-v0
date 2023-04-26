package database

import (
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

const (
	BundleTxIsTail = 1
)

func databaseKeyPrefixForBundleHash(bundleHash hornet.Hash) []byte {
	return bundleHash
}

func (db *Database) BundleTransactionHashes(bundleHash hornet.Hash, maxFind ...int) hornet.Hashes {
	var bundleTransactionHashes hornet.Hashes

	/*
		49 bytes					bundleHash
		1 byte  	   				isTail
		49 bytes                 	txHash
	*/

	i := 0
	_ = db.bundleTransactionsStore.IterateKeys(databaseKeyPrefixForBundleHash(bundleHash), func(key []byte) bool {
		i++
		if (len(maxFind) > 0) && (i > maxFind[0]) {
			return false
		}

		bundleTransactionHashes = append(bundleTransactionHashes, key[50:99])

		return true
	})

	return bundleTransactionHashes
}

func (db *Database) ForEachBundleTailTransactionHash(bundleHash hornet.Hash, consumer func(txTailHash hornet.Hash) bool, maxFind ...int) {
	i := 0
	_ = db.bundleTransactionsStore.IterateKeys(append(databaseKeyPrefixForBundleHash(bundleHash), BundleTxIsTail), func(key []byte) bool {
		i++
		if (len(maxFind) > 0) && (i > maxFind[0]) {
			return false
		}

		return consumer(key[50:99])
	})
}
