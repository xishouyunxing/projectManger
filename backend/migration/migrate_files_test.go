package migration

import (
	"sync"
	"testing"
	"time"
)

func TestGetMigrationStatusReturnsSnapshot(t *testing.T) {
	migrationMu.Lock()
	migrationStatus = &FileMigrationStatus{
		Status:    "running",
		StartTime: time.Now().Format(time.RFC3339),
	}
	migrationMu.Unlock()

	snapshot := GetMigrationStatus()
	snapshot.Status = "tampered"
	snapshot.ErrorMsg = "tampered"

	latest := GetMigrationStatus()
	if latest.Status != "running" {
		t.Fatalf("expected shared status unchanged, got %s", latest.Status)
	}
	if latest.ErrorMsg != "" {
		t.Fatalf("expected shared error unchanged, got %s", latest.ErrorMsg)
	}
}

func TestStartMigrationAllowsOnlyOneStarter(t *testing.T) {
	migrationMu.Lock()
	migrationStatus = &FileMigrationStatus{Status: "not_started"}
	migrationMu.Unlock()

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)

	results := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			results <- startMigration(time.Now())
		}()
	}
	wg.Wait()
	close(results)

	startedCount := 0
	for started := range results {
		if started {
			startedCount++
		}
	}

	if startedCount != 1 {
		t.Fatalf("expected exactly one starter, got %d", startedCount)
	}
}
