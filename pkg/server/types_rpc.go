package server

import (
	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
	"github.com/iotaledger/iota.go/trinary"
)

// Request struct.
type Request struct {
	Command string `json:"command"`
}

// ErrorReturn struct.
type ErrorReturn struct {
	Error string `json:"error"`
}

/////////////////////// getNodeInfo ///////////////////////////////

// GetNodeInfoResponse struct.
type GetNodeInfoResponse struct {
	AppName                            string          `json:"appName"`
	AppVersion                         string          `json:"appVersion"`
	LatestMilestone                    trinary.Hash    `json:"latestMilestone"`
	LatestMilestoneIndex               milestone.Index `json:"latestMilestoneIndex"`
	LatestSolidSubtangleMilestone      trinary.Hash    `json:"latestSolidSubtangleMilestone"`
	LatestSolidSubtangleMilestoneIndex milestone.Index `json:"latestSolidSubtangleMilestoneIndex"`
	IsSynced                           bool            `json:"isSynced"`
	Health                             bool            `json:"isHealthy"`
	MilestoneStartIndex                milestone.Index `json:"milestoneStartIndex"`
	LastSnapshottedMilestoneIndex      milestone.Index `json:"lastSnapshottedMilestoneIndex"`
	Neighbors                          uint            `json:"neighbors"`
	Time                               int64           `json:"time"`
	Tips                               uint32          `json:"tips"`
	TransactionsToRequest              int             `json:"transactionsToRequest"`
	Features                           []string        `json:"features"`
	CoordinatorAddress                 trinary.Hash    `json:"coordinatorAddress"`
	Duration                           int             `json:"duration"`
}

/////////////////// findTransactions //////////////////////////////

// FindTransactions struct.
type FindTransactions struct {
	Bundles    []trinary.Hash `json:"bundles"`
	Addresses  []trinary.Hash `json:"addresses"`
	Tags       []trinary.Hash `json:"tags"`
	Approvees  []trinary.Hash `json:"approvees"`
	MaxResults int            `json:"maxresults"`
	ValueOnly  bool           `json:"valueOnly"`
}

// FindTransactionsResponse struct.
type FindTransactionsResponse struct {
	Hashes   []trinary.Hash `json:"hashes"`
	Duration int            `json:"duration"`
}

//////////////////////// getTrytes ////////////////////////////////

// GetTrytes struct.
type GetTrytes struct {
	Hashes []trinary.Hash `json:"hashes"`
}

// GetTrytesResponse struct.
type GetTrytesResponse struct {
	Trytes     []trinary.Trytes `json:"trytes"`
	Milestones []uint32         `json:"milestones"`
	Duration   int              `json:"duration"`
}

/////////////////// getInclusionStates ////////////////////////////

// GetInclusionStates struct.
type GetInclusionStates struct {
	Transactions []trinary.Hash `json:"transactions"`
}

// GetInclusionStatesResponse struct.
type GetInclusionStatesResponse struct {
	States   []bool `json:"states"`
	Duration int    `json:"duration"`
}

///////////////////// getBalances /////////////////////////////////

// GetBalances struct.
type GetBalances struct {
	Addresses []trinary.Hash `json:"addresses"`
}

// GetBalancesResponse struct.
type GetBalancesResponse struct {
	Balances       []string        `json:"balances"`
	References     []trinary.Hash  `json:"references"`
	MilestoneIndex milestone.Index `json:"milestoneIndex"`
	Duration       int             `json:"duration"`
}

/////////////////// wereAddressesSpentFrom ////////////////////////

// WereAddressesSpentFrom struct.
type WereAddressesSpentFrom struct {
	Addresses []trinary.Hash `json:"addresses"`
}

// WereAddressesSpentFromResponse struct.
type WereAddressesSpentFromResponse struct {
	States   []bool `json:"states"`
	Duration int    `json:"duration"`
}

/////////////////// getLedgerState ////////////////////////

// GetLedgerState struct.
type GetLedgerState struct {
	TargetIndex milestone.Index `json:"targetIndex,omitempty"`
}

// GetLedgerStateResponse struct.
type GetLedgerStateResponse struct {
	Balances       map[trinary.Hash]uint64 `json:"balances"`
	MilestoneIndex milestone.Index         `json:"milestoneIndex"`
	Duration       int                     `json:"duration"`
}

/////////////////// getLedgerDiff ////////////////////////

// GetLedgerDiff struct.
type GetLedgerDiff struct {
	MilestoneIndex milestone.Index `json:"milestoneIndex"`
}

// GetLedgerDiffResponse struct.
type GetLedgerDiffResponse struct {
	Diff           map[trinary.Hash]int64 `json:"diff"`
	MilestoneIndex milestone.Index        `json:"milestoneIndex"`
	Duration       int                    `json:"duration"`
}

/////////////////// getLedgerDiffExt ////////////////////////

// GetLedgerDiffExt struct.
type GetLedgerDiffExt struct {
	MilestoneIndex milestone.Index `json:"milestoneIndex"`
}

// TxHashWithValue struct.
type TxHashWithValue struct {
	TxHash     trinary.Hash `json:"txHash"`
	TailTxHash trinary.Hash `json:"tailTxHash"`
	BundleHash trinary.Hash `json:"bundleHash"`
	Address    trinary.Hash `json:"address"`
	Value      int64        `json:"value"`
}

func (tx *TxHashWithValue) Item() Container {
	return tx
}

// TxWithValue struct.
type TxWithValue struct {
	TxHash  trinary.Hash `json:"txHash"`
	Address trinary.Hash `json:"address"`
	Index   uint64       `json:"index"`
	Value   int64        `json:"value"`
}

func (tx *TxWithValue) Item() Container {
	return tx
}

// BundleWithValue struct.
type BundleWithValue struct {
	BundleHash trinary.Hash   `json:"bundleHash"`
	TailTxHash trinary.Hash   `json:"tailTxHash"`
	LastIndex  uint64         `json:"lastIndex"`
	Txs        []*TxWithValue `json:"txs"`
}

func (b *BundleWithValue) Item() Container {
	return b
}

// GetLedgerDiffExtResponse struct.
type GetLedgerDiffExtResponse struct {
	ConfirmedTxWithValue      []*TxHashWithValue     `json:"confirmedTxWithValue"`
	ConfirmedBundlesWithValue []*BundleWithValue     `json:"confirmedBundlesWithValue"`
	Diff                      map[trinary.Hash]int64 `json:"diff"`
	MilestoneIndex            milestone.Index        `json:"milestoneIndex"`
	Duration                  int                    `json:"duration"`
}
