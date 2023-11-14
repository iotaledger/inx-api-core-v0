package database

import (
	"context"
	"sync"
	"time"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/kvstore"
	hivedb "github.com/iotaledger/hive.go/kvstore/database"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/inx-api-core-v0/pkg/database/engine"
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
	"github.com/iotaledger/iota.go/trinary"
)

const (
	DBVersion = 3

	// printStatusInterval is the interval for printing status messages.
	printStatusInterval = 2 * time.Second
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
	StorePrefixSnapshotLedger          byte = 13 // unused
	StorePrefixUnconfirmedTransactions byte = 14 // unused
	StorePrefixSpentAddresses          byte = 15
	StorePrefixAutopeering             byte = 16 // unused
	StorePrefixWhiteFlag               byte = 17 // unused
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

func New(ctx context.Context, log *logger.Logger, tangleDatabasePath string, snapshotDatabasePath string, spentDatabasePath string, skipHealthCheck bool) (*Database, error) {

	type database struct {
		name        string
		path        string
		upgradeFunc func(context.Context, *logger.Logger, kvstore.KVStore, byte, byte) error
		store       kvstore.KVStore
	}

	tangleDatabase := &database{
		name:        "tangle",
		path:        tangleDatabasePath,
		upgradeFunc: migrateTangleDatabaseFunc,
		store:       nil,
	}

	snapshotDatabase := &database{
		name: "snapshot",
		path: snapshotDatabasePath,
		upgradeFunc: func(_ context.Context, _ *logger.Logger, _ kvstore.KVStore, oldVersion byte, newVersion byte) error {
			return supportedDatabaseVersionUpgradeFunc(oldVersion, newVersion)
		},
		store: nil,
	}

	spentDatabase := &database{
		name: "spent",
		path: spentDatabasePath,
		upgradeFunc: func(_ context.Context, _ *logger.Logger, _ kvstore.KVStore, oldVersion byte, newVersion byte) error {
			return supportedDatabaseVersionUpgradeFunc(oldVersion, newVersion)
		},
		store: nil,
	}

	checkDatabaseVersionAndHealth := func(store kvstore.KVStore, consumer func(healthTracker *kvstore.StoreHealthTracker) error, storeVersionUpdateFunc kvstore.StoreVersionUpdateFunc) error {
		healthTracker, err := kvstore.NewStoreHealthTracker(store, kvstore.KeyPrefix{StorePrefixHealth}, DBVersion, storeVersionUpdateFunc)
		if err != nil {
			return err
		}

		if !skipHealthCheck {
			if lo.PanicOnErr(healthTracker.IsCorrupted()) {
				return ierrors.New("database is corrupted")
			}

			if lo.PanicOnErr(healthTracker.IsTainted()) {
				return ierrors.New("database is tainted")
			}
		}

		return consumer(healthTracker)
	}

	for _, db := range []*database{tangleDatabase, snapshotDatabase, spentDatabase} {
		// open the database in readonly mode first
		store, err := engine.StoreWithDefaultSettings(db.path, false, hivedb.EngineAuto, true, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, ierrors.Wrapf(err, "failed to open %s database", db.name)
		}
		db.store = store

		// check if we need to upgrade the database
		upgradeNeeded := false
		if err := checkDatabaseVersionAndHealth(store, func(healthTracker *kvstore.StoreHealthTracker) error {
			correctVersion, err := healthTracker.CheckCorrectStoreVersion()
			if err != nil {
				return err
			}
			upgradeNeeded = !correctVersion

			return nil
		}, nil); err != nil {
			return nil, ierrors.Errorf("failed to check %s database health: %w", db.name, err)
		}

		if !upgradeNeeded {
			continue
		}

		log.Infof("Upgrading %s database...", db.name)

		// to be able to upgrade, we need to close the database and reopen it in read/write mode afterwards
		if err := store.Close(); err != nil {
			return nil, ierrors.Errorf("failed to close %s database: %w", db.name, err)
		}

		// open the database in read/write mode
		store, err = engine.StoreWithDefaultSettings(db.path, false, hivedb.EngineAuto, false, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, ierrors.Wrapf(err, "failed to open %s database", db.name)
		}

		// execute the upgrade function
		if err := checkDatabaseVersionAndHealth(store, func(healthTracker *kvstore.StoreHealthTracker) error {
			return lo.Return2(healthTracker.UpdateStoreVersion())
		}, func(oldVersion, newVersion byte) error {
			//nolint:scopelint
			return db.upgradeFunc(ctx, log, store, oldVersion, newVersion)
		}); err != nil {
			return nil, ierrors.Errorf("failed to upgrade %s database version: %w", db.name, err)
		}

		// close the database again
		if err := store.Close(); err != nil {
			return nil, ierrors.Errorf("failed to close %s database: %w", db.name, err)
		}

		// open the database in readonly mode again
		store, err = engine.StoreWithDefaultSettings(db.path, false, hivedb.EngineAuto, true, engine.AllowedEnginesStorageAuto...)
		if err != nil {
			return nil, ierrors.Wrapf(err, "failed to open %s database", db.name)
		}
		db.store = store

		log.Infof("Upgrading %s database...done!", db.name)
	}

	db := &Database{
		tangleDatabase:                 tangleDatabase.store,
		snapshotDatabase:               snapshotDatabase.store,
		spentDatabase:                  spentDatabase.store,
		txStore:                        lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixTransactions})),
		metadataStore:                  lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixTransactionMetadata})),
		addressesStore:                 lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixAddresses})),
		approversStore:                 lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixApprovers})),
		bundleStore:                    lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixBundles})),
		bundleTransactionsStore:        lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixBundleTransactions})),
		milestoneStore:                 lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixMilestones})),
		spentAddressesStore:            lo.PanicOnErr(spentDatabase.store.WithRealm([]byte{StorePrefixSpentAddresses})),
		tagsStore:                      lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixTags})),
		snapshotStore:                  lo.PanicOnErr(snapshotDatabase.store.WithRealm([]byte{StorePrefixSnapshot})),
		ledgerStore:                    lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixLedgerState})),
		ledgerBalanceStore:             lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixLedgerBalance})),
		ledgerDiffStore:                lo.PanicOnErr(tangleDatabase.store.WithRealm([]byte{StorePrefixLedgerDiff})),
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
