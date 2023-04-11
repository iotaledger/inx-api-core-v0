package database

import (
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
)

func (db *Database) ApproverHashes(txHash hornet.Hash, maxFind ...int) hornet.Hashes {
	var approverHashes hornet.Hashes

	i := 0
	_ = db.approversStore.IterateKeys(txHash, func(key []byte) bool {
		i++
		if (len(maxFind) > 0) && (i > maxFind[0]) {
			return false
		}

		approverHashes = append(approverHashes, key[49:98])

		return true
	})

	return approverHashes
}

// ContainsApprover returns if the given approver exists in the cache/persistence layer.
func (db *Database) ContainsApprover(txHash hornet.Hash, approverHash hornet.Hash) bool {
	return lo.PanicOnErr(db.approversStore.Has(append(txHash, approverHash...)))
}
