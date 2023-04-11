package engine

import (
	"github.com/iotaledger/hive.go/kvstore/rocksdb"
)

// NewRocksDB creates a new RocksDB instance.
func NewRocksDB(path string) (*rocksdb.RocksDB, error) {
	return rocksdb.OpenDBReadOnly(path)
}
