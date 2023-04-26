package server

import (
	"time"

	"github.com/labstack/echo/v4"
)

func (s *DatabaseServer) rpcGetNodeInfo(_ echo.Context) (any, error) {
	syncState := s.Database.LatestSyncState()

	return &GetNodeInfoResponse{
		AppName:                            s.AppInfo.Name,
		AppVersion:                         s.AppInfo.Version,
		LatestMilestone:                    syncState.LatestMilestone,
		LatestMilestoneIndex:               syncState.LatestMilestoneIndex,
		LatestSolidSubtangleMilestone:      syncState.LatestSolidSubtangleMilestone,
		LatestSolidSubtangleMilestoneIndex: syncState.LatestSolidSubtangleMilestoneIndex,
		IsSynced:                           true,
		Health:                             true,
		MilestoneStartIndex:                syncState.MilestoneStartIndex,
		LastSnapshottedMilestoneIndex:      syncState.LastSnapshottedMilestoneIndex,
		Neighbors:                          0,
		Time:                               time.Now().UnixMilli(),
		Tips:                               0,
		TransactionsToRequest:              0,
		Features:                           []string{},
		CoordinatorAddress:                 syncState.CoordinatorAddress,
	}, nil
}

//nolint:unparam // even if the error is never used, the structure of all routes should be the same
func (s *DatabaseServer) info() (*infoResponse, error) {

	syncState := s.Database.LatestSyncState()

	return &infoResponse{
		AppName:                            s.AppInfo.Name,
		AppVersion:                         s.AppInfo.Version,
		LatestMilestone:                    syncState.LatestMilestone,
		LatestMilestoneIndex:               syncState.LatestMilestoneIndex,
		LatestSolidSubtangleMilestone:      syncState.LatestSolidSubtangleMilestone,
		LatestSolidSubtangleMilestoneIndex: syncState.LatestSolidSubtangleMilestoneIndex,
		IsSynced:                           true,
		Health:                             true,
		MilestoneStartIndex:                syncState.MilestoneStartIndex,
		LastSnapshottedMilestoneIndex:      syncState.LastSnapshottedMilestoneIndex,
		Neighbors:                          0,
		Time:                               time.Now().UnixMilli(),
		Tips:                               0,
		TransactionsToRequest:              0,
		Features:                           []string{},
		CoordinatorAddress:                 syncState.CoordinatorAddress,
	}, nil
}
