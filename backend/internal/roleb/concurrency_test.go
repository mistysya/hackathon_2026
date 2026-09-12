package roleb

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func TestConcurrentSimulateCreatesExactlyOneTarget(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(ctx, "file:"+filepath.Join(t.TempDir(), "test.db")+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	seedApprovedCampaign(t, db)
	repo := NewRepository(db)
	start := make(chan struct{})
	results := make(chan error, 16)
	for range cap(results) {
		go func() {
			<-start
			_, err := repo.SimulateCampaign(ctx, "c_demo")
			results <- err
		}()
	}
	close(start)
	successes := 0
	for range cap(results) {
		if err := <-results; err == nil {
			successes++
		} else if !errors.Is(err, ErrConflict) {
			t.Errorf("concurrent simulation returned unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful simulations = %d, want 1", successes)
	}
	report, err := repo.CampaignReport(ctx, "c_demo")
	if err != nil || report.TargetCount != 1 {
		t.Fatalf("report = %+v, err = %v", report, err)
	}
}
