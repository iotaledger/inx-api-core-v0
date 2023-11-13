package engine

import (
	"fmt"

	"github.com/iotaledger/hive.go/kvstore"
	hivedb "github.com/iotaledger/hive.go/kvstore/database"
	"github.com/iotaledger/hive.go/kvstore/rocksdb"
)

var (
	AllowedEnginesDefault = []hivedb.Engine{
		hivedb.EngineAuto,
		hivedb.EngineRocksDB,
	}

	AllowedEnginesStorage = []hivedb.Engine{
		hivedb.EngineRocksDB,
	}

	AllowedEnginesStorageAuto = append(AllowedEnginesStorage, hivedb.EngineAuto)
)

// StoreWithDefaultSettings returns a kvstore with default settings.
// It also checks if the database engine is correct.
func StoreWithDefaultSettings(directory string, createDatabaseIfNotExists bool, dbEngine hivedb.Engine, allowedEngines ...hivedb.Engine) (kvstore.KVStore, error) {

	tmpAllowedEngines := AllowedEnginesDefault
	if len(allowedEngines) > 0 {
		tmpAllowedEngines = allowedEngines
	}

	targetEngine, err := hivedb.CheckEngine(directory, createDatabaseIfNotExists, dbEngine, tmpAllowedEngines)
	if err != nil {
		return nil, err
	}

	//nolint:exhaustive
	switch targetEngine {
	case hivedb.EngineRocksDB:
		db, err := NewRocksDB(directory)
		if err != nil {
			return nil, err
		}

		return rocksdb.New(db), nil

	default:
		return nil, fmt.Errorf("unknown database engine: %s, supported engines: rocksdb", dbEngine)
	}
}
