package database

import (
	"bytes"
	"encoding/binary"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/inx-api-core-v0/pkg/hornet"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
)

type SolidEntryPoints struct {
	entryPointsMap map[string]milestone.Index
}

func newSolidEntryPoints() *SolidEntryPoints {
	return &SolidEntryPoints{
		entryPointsMap: make(map[string]milestone.Index),
	}
}

func (s *SolidEntryPoints) Add(txHash hornet.Hash, milestoneIndex milestone.Index) {
	if _, exists := s.entryPointsMap[string(txHash)]; !exists {
		s.entryPointsMap[string(txHash)] = milestoneIndex
	}
}

func (s *SolidEntryPoints) contains(txHash hornet.Hash) bool {
	_, exists := s.entryPointsMap[string(txHash)]

	return exists
}

func solidEntryPointsFromBytes(solidEntryPointsBytes []byte) (*SolidEntryPoints, error) {
	s := newSolidEntryPoints()

	bytesReader := bytes.NewReader(solidEntryPointsBytes)

	var err error

	solidEntryPointsCount := len(solidEntryPointsBytes) / (hornet.HashSize + 4)
	for i := 0; i < solidEntryPointsCount; i++ {
		hashBuf := make([]byte, hornet.HashSize)
		var msIndex milestone.Index

		err = binary.Read(bytesReader, binary.BigEndian, hashBuf)
		if err != nil {
			return nil, ierrors.Errorf("solidEntryPoints: %w", err)
		}

		err = binary.Read(bytesReader, binary.BigEndian, &msIndex)
		if err != nil {
			return nil, ierrors.Errorf("solidEntryPoints: %w", err)
		}

		s.Add(hornet.Hash(hashBuf), msIndex)
	}

	return s, nil
}

func (db *Database) readSolidEntryPoints() (*SolidEntryPoints, error) {
	value, err := db.snapshotStore.Get([]byte("solidEntryPoints"))
	if err != nil {
		return nil, ierrors.Errorf("%w: failed to retrieve solid entry points", err)
	}

	points, err := solidEntryPointsFromBytes(value)
	if err != nil {
		return nil, ierrors.Errorf("%w: failed to convert solid entry points", err)
	}

	return points, nil
}

func (db *Database) loadSolidEntryPoints() error {
	solidEntryPoints, err := db.readSolidEntryPoints()
	if err != nil {
		return err
	}
	db.solidEntryPoints = solidEntryPoints

	return nil
}

func (db *Database) SolidEntryPointsContain(txHash hornet.Hash) bool {
	return db.solidEntryPoints.contains(txHash)
}
