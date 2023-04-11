package database

import (
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

func (db *Database) TagHashes(txTag hornet.Hash, forceRelease bool, maxFind ...int) hornet.Hashes {
	var tagHashes hornet.Hashes

	i := 0
	_ = db.tagsStore.IterateKeys(txTag, func(key []byte) bool {
		i++
		if (len(maxFind) > 0) && (i > maxFind[0]) {
			return false
		}

		tagHashes = append(tagHashes, hornet.Hash(key[17:66]))

		return true
	})

	return tagHashes
}

// ContainsTag returns if the given tag exists in the cache/persistence layer.
func (db *Database) ContainsTag(txTag hornet.Hash, txHash hornet.Hash) bool {
	return lo.PanicOnErr(db.tagsStore.Has(append(txTag, txHash...)))
}
