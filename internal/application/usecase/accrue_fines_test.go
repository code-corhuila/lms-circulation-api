package usecase_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
)

func TestAccrueFines_IsANoOpStub(t *testing.T) {
	uc := usecase.NewAccrueFines(zap.NewNop().Sugar())

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
