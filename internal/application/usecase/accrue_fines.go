package usecase

import (
	"context"

	"go.uber.org/zap"
)

// AccrueFines is a deliberate stub for ADR-009's second responsibility.
// There is no fine/monetary-penalty domain rule in this project yet
// (02-domain/entities-and-rules.md has none) — inventing one here would be
// hidden scope creep, not a real implementation. This only confirms the
// route is reachable and safe to call on lms-worker's schedule, so that job
// gets a real 200 instead of a 404. Replace this body once a fine-accrual
// rule is actually decided.
type AccrueFines struct {
	Log *zap.SugaredLogger
}

func NewAccrueFines(log *zap.SugaredLogger) *AccrueFines {
	return &AccrueFines{Log: log}
}

func (uc *AccrueFines) Execute(_ context.Context) error {
	uc.Log.Info("accrue-fines called — stub, no fine domain model exists yet")
	return nil
}
