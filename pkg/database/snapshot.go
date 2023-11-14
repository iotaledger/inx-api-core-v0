package database

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/iotaledger/hive.go/ds/bitmask"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
)

const (
	SnapshotMetadataSpentAddressesEnabled = 0
)

type SnapshotInfo struct {
	CoordinatorAddress hornet.Hash
	Hash               hornet.Hash
	SnapshotIndex      milestone.Index
	EntryPointIndex    milestone.Index
	PruningIndex       milestone.Index
	Timestamp          int64
	Metadata           bitmask.BitMask
}

func (db *Database) readSnapshotInfo() (*SnapshotInfo, error) {
	value, err := db.snapshotStore.Get([]byte("snapshotInfo"))
	if err != nil {
		return nil, ierrors.Errorf("%w: failed to retrieve snapshot info", err)
	}

	info, err := snapshotInfoFromBytes(value)
	if err != nil {
		return nil, ierrors.Errorf("%w: failed to convert snapshot info", err)
	}

	return info, nil
}

func (db *Database) loadSnapshotInfo() error {
	info, err := db.readSnapshotInfo()
	if err != nil {
		return err
	}
	db.snapshot = info

	if info != nil {
		println(fmt.Sprintf(`SnapshotInfo:
	CooAddr: %v
	SnapshotIndex: %d (%v)
	EntryPointIndex: %d
	PruningIndex: %d
	Timestamp: %v
	SpentAddressesEnabled: %v`, info.CoordinatorAddress.Trytes(), info.SnapshotIndex, info.Hash.Trytes(), info.EntryPointIndex, info.PruningIndex, time.Unix(info.Timestamp, 0).Truncate(time.Second), info.IsSpentAddressesEnabled()))
	}

	return nil
}

func snapshotInfoFromBytes(bytes []byte) (*SnapshotInfo, error) {

	if len(bytes) != 119 {
		return nil, ierrors.Errorf("parsing of snapshot info failed, error: invalid length %d != 119", len(bytes))
	}

	cooAddr := hornet.Hash(bytes[:49])
	hash := hornet.Hash(bytes[49:98])
	snapshotIndex := binary.LittleEndian.Uint32(bytes[98:102])
	entryPointIndex := binary.LittleEndian.Uint32(bytes[102:106])
	pruningIndex := binary.LittleEndian.Uint32(bytes[106:110])
	timestamp := int64(binary.LittleEndian.Uint64(bytes[110:118]))
	metadata := bitmask.BitMask(bytes[118])

	return &SnapshotInfo{
		CoordinatorAddress: cooAddr,
		Hash:               hash,
		SnapshotIndex:      milestone.Index(snapshotIndex),
		EntryPointIndex:    milestone.Index(entryPointIndex),
		PruningIndex:       milestone.Index(pruningIndex),
		Timestamp:          timestamp,
		Metadata:           metadata,
	}, nil
}

func (i *SnapshotInfo) IsSpentAddressesEnabled() bool {
	return i.Metadata.HasBit(SnapshotMetadataSpentAddressesEnabled)
}
