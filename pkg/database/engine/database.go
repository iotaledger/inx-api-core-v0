package engine

import (
	"fmt"
	"path"

	"github.com/iotaledger/hive.go/kvstore"
	hivedb "github.com/iotaledger/hive.go/kvstore/database"
	"github.com/iotaledger/hive.go/kvstore/pebble"
	"github.com/iotaledger/hive.go/kvstore/rocksdb"
	"github.com/iotaledger/inx-api-core-v0/pkg/database/bolt"
)

const (
	EngineBolt = "bolt"
)

var (
	AllowedEnginesDefault = []hivedb.Engine{
		hivedb.EngineAuto,
		hivedb.EnginePebble,
		hivedb.EngineRocksDB,
		EngineBolt,
	}

	AllowedEnginesStorage = []hivedb.Engine{
		hivedb.EnginePebble,
		hivedb.EngineRocksDB,
		EngineBolt,
	}

	AllowedEnginesStorageAuto = append(AllowedEnginesStorage, hivedb.EngineAuto)
)

// StoreWithDefaultSettings returns a kvstore with default settings.
// It also checks if the database engine is correct.
func StoreWithDefaultSettings(directory string, createDatabaseIfNotExists bool, dbEngine hivedb.Engine, boltFileName string, allowedEngines ...hivedb.Engine) (kvstore.KVStore, error) {

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
	case hivedb.EnginePebble:
		db, err := NewPebbleDB(directory, nil, false)
		if err != nil {
			return nil, err
		}

		return pebble.New(db), nil

	case hivedb.EngineRocksDB:
		db, err := NewRocksDB(directory)
		if err != nil {
			return nil, err
		}

		return rocksdb.New(db), nil

	case EngineBolt:
		db, err := NewBoltDB(path.Join(directory, boltFileName))
		if err != nil {
			return nil, err
		}

		return bolt.New(db), nil

	default:
		return nil, fmt.Errorf("unknown database engine: %s, supported engines: pebble/rocksdb/bolt", dbEngine)
	}
}
