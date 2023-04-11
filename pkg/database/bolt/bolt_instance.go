package bolt

import (
	"fmt"
	"path/filepath"

	"go.etcd.io/bbolt"

	"github.com/iotaledger/hive.go/runtime/ioutils"
)

func CreateDB(path string, optionalOptions ...*bbolt.Options) (*bbolt.DB, error) {
	if err := ioutils.CreateDirectory(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("could not create directory: %w", err)
	}

	options := bbolt.DefaultOptions
	if len(optionalOptions) > 0 {
		options = optionalOptions[0]
	}

	db, err := bbolt.Open(path, 0666, options)
	if err != nil {
		return nil, fmt.Errorf("could not open new DB: %w", err)
	}

	return db, nil
}
