package database

import (
	"sync"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
	"github.com/iotaledger/iota.go/trinary"
)

const (
	DBVersion = 2
)

const (
	StorePrefixHealth                  byte = 0
	StorePrefixTransactions            byte = 1
	StorePrefixTransactionMetadata     byte = 2
	StorePrefixBundleTransactions      byte = 3
	StorePrefixBundles                 byte = 4
	StorePrefixAddresses               byte = 5
	StorePrefixMilestones              byte = 6
	StorePrefixLedgerState             byte = 7
	StorePrefixLedgerBalance           byte = 8
	StorePrefixLedgerDiff              byte = 9
	StorePrefixApprovers               byte = 10
	StorePrefixTags                    byte = 11
	StorePrefixSnapshot                byte = 12
	StorePrefixSnapshotLedger          byte = 13
	StorePrefixUnconfirmedTransactions byte = 14
	StorePrefixSpentAddresses          byte = 15
	StorePrefixAutopeering             byte = 16
	StorePrefixWhiteFlag               byte = 17
)

var (
	// ErrOperationAborted is returned when the operation was aborted e.g. by a shutdown signal.
	ErrOperationAborted = ierrors.New("operation was aborted")
)

type Database struct {
	// databases
	tangleDatabase   kvstore.KVStore
	snapshotDatabase kvstore.KVStore
	spentDatabase    kvstore.KVStore

	// kv stores
	txStore                 kvstore.KVStore
	metadataStore           kvstore.KVStore
	bundleTransactionsStore kvstore.KVStore
	addressesStore          kvstore.KVStore
	tagsStore               kvstore.KVStore
	milestoneStore          kvstore.KVStore
	approversStore          kvstore.KVStore
	spentAddressesStore     kvstore.KVStore
	bundleStore             kvstore.KVStore
	snapshotStore           kvstore.KVStore
	ledgerStore             kvstore.KVStore
	ledgerBalanceStore      kvstore.KVStore
	ledgerDiffStore         kvstore.KVStore

	// solid entry points
	solidEntryPoints *SolidEntryPoints

	// snapshot info
	snapshot *SnapshotInfo

	// syncstate
	syncState     *SyncState
	syncStateOnce sync.Once

	ledgerMilestoneIndex     milestone.Index
	ledgerMilestoneIndexOnce sync.Once

	latestSolidMilestoneBundle     *Bundle
	latestSolidMilestoneBundleOnce sync.Once
}

func New(tangleDatabase, snapshotDatabase, spentDatabase kvstore.KVStore, skipHealthCheck bool) (*Database, error) {

	checkDatabaseHealth := func(store kvstore.KVStore) error {
		healthTracker, err := kvstore.NewStoreHealthTracker(store, kvstore.KeyPrefix{StorePrefixHealth}, DBVersion, nil)
		if err != nil {
			return err
		}

		if lo.PanicOnErr(healthTracker.IsCorrupted()) {
			return ierrors.New("database is corrupted")
		}

		if lo.PanicOnErr(healthTracker.IsTainted()) {
			return ierrors.New("database is tainted")
		}

		return nil
	}

	if !skipHealthCheck {
		if err := checkDatabaseHealth(tangleDatabase); err != nil {
			return nil, ierrors.Errorf("opening tangle database failed: %w", err)
		}
		if err := checkDatabaseHealth(snapshotDatabase); err != nil {
			return nil, ierrors.Errorf("opening snapshot database failed: %w", err)
		}
		if err := checkDatabaseHealth(spentDatabase); err != nil {
			return nil, ierrors.Errorf("opening spent database failed: %w", err)
		}
	}

	db := &Database{
		tangleDatabase:                 tangleDatabase,
		snapshotDatabase:               snapshotDatabase,
		spentDatabase:                  spentDatabase,
		txStore:                        lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixTransactions})),
		metadataStore:                  lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixTransactionMetadata})),
		addressesStore:                 lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixAddresses})),
		approversStore:                 lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixApprovers})),
		bundleStore:                    lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixBundles})),
		bundleTransactionsStore:        lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixBundleTransactions})),
		milestoneStore:                 lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixMilestones})),
		spentAddressesStore:            lo.PanicOnErr(spentDatabase.WithRealm([]byte{StorePrefixSpentAddresses})),
		tagsStore:                      lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixTags})),
		snapshotStore:                  lo.PanicOnErr(snapshotDatabase.WithRealm([]byte{StorePrefixSnapshot})),
		ledgerStore:                    lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixLedgerState})),
		ledgerBalanceStore:             lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixLedgerBalance})),
		ledgerDiffStore:                lo.PanicOnErr(tangleDatabase.WithRealm([]byte{StorePrefixLedgerDiff})),
		solidEntryPoints:               nil,
		snapshot:                       nil,
		syncState:                      nil,
		syncStateOnce:                  sync.Once{},
		ledgerMilestoneIndex:           0,
		ledgerMilestoneIndexOnce:       sync.Once{},
		latestSolidMilestoneBundle:     nil,
		latestSolidMilestoneBundleOnce: sync.Once{},
	}

	if err := db.loadSnapshotInfo(); err != nil {
		return nil, err
	}
	if err := db.loadSolidEntryPoints(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *Database) CloseDatabases() error {
	var closeError error
	if err := db.tangleDatabase.Close(); err != nil {
		closeError = err
	}
	if err := db.snapshotDatabase.Close(); err != nil {
		closeError = err
	}
	if err := db.spentDatabase.Close(); err != nil {
		closeError = err
	}

	return closeError
}

type SyncState struct {
	LatestMilestone                    trinary.Hash
	LatestMilestoneIndex               milestone.Index
	LatestSolidSubtangleMilestone      trinary.Hash
	LatestSolidSubtangleMilestoneIndex milestone.Index
	MilestoneStartIndex                milestone.Index
	LastSnapshottedMilestoneIndex      milestone.Index
	CoordinatorAddress                 trinary.Hash
}

func (db *Database) LatestSyncState() *SyncState {
	db.syncStateOnce.Do(func() {
		ledgerIndex := db.LedgerIndex()
		latestMilestoneHash := db.LatestSolidMilestoneBundle().MilestoneHash().Trytes()

		db.syncState = &SyncState{
			LatestMilestone:                    latestMilestoneHash,
			LatestMilestoneIndex:               ledgerIndex,
			LatestSolidSubtangleMilestone:      latestMilestoneHash,
			LatestSolidSubtangleMilestoneIndex: ledgerIndex,
			MilestoneStartIndex:                db.snapshot.PruningIndex,
			LastSnapshottedMilestoneIndex:      db.snapshot.PruningIndex,
			CoordinatorAddress:                 db.snapshot.CoordinatorAddress.Trytes(),
		}
	})

	return db.syncState
}
