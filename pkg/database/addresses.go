package database

import (
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

const (
	AddressTxIsValue = 1
)

func databaseKeyPrefixForAddress(address hornet.Hash) []byte {
	return address
}

func databaseKeyPrefixForAddressTransaction(address hornet.Hash, txHash hornet.Hash, isValue bool) []byte {
	var isValueByte byte
	if isValue {
		isValueByte = AddressTxIsValue
	}

	result := append(databaseKeyPrefixForAddress(address), isValueByte)

	return append(result, txHash...)
}

func (db *Database) TransactionHashesForAddress(address hornet.Hash, valueOnly bool, maxFind ...int) hornet.Hashes {

	searchPrefix := databaseKeyPrefixForAddress(address)
	if valueOnly {
		var isValueByte byte = AddressTxIsValue
		searchPrefix = append(searchPrefix, isValueByte)
	}

	var txHashes hornet.Hashes

	i := 0
	_ = db.addressesStore.IterateKeys(searchPrefix, func(key []byte) bool {
		i++
		if (len(maxFind) > 0) && (i > maxFind[0]) {
			return false
		}

		txHashes = append(txHashes, key[50:99])

		return true
	})

	return txHashes
}

// ContainsAddress returns if the given address exists in the cache/persistence layer.
func (db *Database) ContainsAddress(address hornet.Hash, txHash hornet.Hash, valueOnly bool) bool {
	if valueOnly {
		return lo.PanicOnErr(db.addressesStore.Has(databaseKeyPrefixForAddressTransaction(address, txHash, true)))
	}

	return lo.PanicOnErr(db.addressesStore.Has(databaseKeyPrefixForAddressTransaction(address, txHash, false))) || lo.PanicOnErr(db.addressesStore.Has(databaseKeyPrefixForAddressTransaction(address, txHash, true)))
}
