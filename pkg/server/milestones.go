package server

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/iotaledger/inx-api-core-v0/pkg/milestone"
	"github.com/iotaledger/inx-app/pkg/httpserver"
)

func (s *DatabaseServer) milestone(c echo.Context) (interface{}, error) {
	msIndexIotaGo, err := httpserver.ParseMilestoneIndexParam(c, ParameterMilestoneIndex)
	if err != nil {
		return nil, err
	}
	msIndex := milestone.Index(msIndexIotaGo)

	smi := s.Database.SolidMilestoneIndex()
	if msIndex > smi {
		return nil, errors.WithMessagef(httpserver.ErrInvalidParameter, "invalid milestone index: %d, lsmi is %d", msIndex, smi)
	}

	msBndl := s.Database.MilestoneBundleOrNil(msIndex)
	if msBndl == nil {
		return nil, fmt.Errorf("milestone not found: %d", msIndex)
	}

	return milestoneResponse{
		MilestoneIndex:     msIndex,
		MilestoneHash:      msBndl.Tail().Tx.Hash,
		MilestoneTimestamp: msBndl.Tail().Tx.Timestamp,
	}, nil
}
