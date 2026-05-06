package gmcore_scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	s := NewScheduler()
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if s.schedules == nil {
		t.Fatal("schedules map should be initialized")
	}
}

func TestNewSchedule(t *testing.T) {
	task := func(ctx context.Context) error { return nil }
	s := NewSchedule("job-1", "*/5", task)

	if s.ID != "job-1" {
		t.Fatalf("expected ID 'job-1', got %s", s.ID)
	}
	if s.Expression != "*/5" {
		t.Fatalf("expected expression '*/5', got %s", s.Expression)
	}
	if s.Task == nil {
		t.Fatal("task should not be nil")
	}
	if !s.Enabled {
		t.Fatal("schedule should be enabled by default")
	}
	if s.Timezone != time.Local {
		t.Fatal("timezone should default to local")
	}
}

func TestScheduler_Schedule(t *testing.T) {
	s := NewScheduler()
	task := func(ctx context.Context) error { return nil }
	sch := NewSchedule("job-1", "*/10", task)

	err := s.Schedule(sch)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}

	if len(s.schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(s.schedules))
	}
	if s.schedules["job-1"].NextRun.IsZero() {
		t.Fatal("NextRun should be set")
	}
}

func TestScheduler_Unschedule(t *testing.T) {
	s := NewScheduler()
	task := func(ctx context.Context) error { return nil }
	s.Schedule(NewSchedule("job-1", "*/10", task))

	err := s.Unschedule("job-1")
	if err != nil {
		t.Fatalf("Unschedule failed: %v", err)
	}

	if len(s.schedules) != 0 {
		t.Fatalf("expected 0 schedules, got %d", len(s.schedules))
	}

	err = s.Unschedule("nonexistent")
	if err != nil {
		t.Fatalf("Unschedule of nonexistent should not error: %v", err)
	}
}

func TestScheduler_GetDueTasks_NoTasks(t *testing.T) {
	s := NewScheduler()
	due := s.GetDueTasks()
	if len(due) != 0 {
		t.Fatalf("expected 0 due tasks, got %d", len(due))
	}
}

func TestScheduler_GetDueTasks_NotEnabled(t *testing.T) {
	s := NewScheduler()
	task := func(ctx context.Context) error { return nil }
	sch := NewSchedule("job-1", "*/1", task)
	sch.Enabled = false
	sch.NextRun = time.Now().Add(-time.Hour)
	s.Schedule(sch)

	due := s.GetDueTasks()
	if len(due) != 0 {
		t.Fatalf("expected 0 due tasks when disabled, got %d", len(due))
	}
}

func TestScheduler_GetDueTasks_FutureTask(t *testing.T) {
	s := NewScheduler()
	task := func(ctx context.Context) error { return nil }
	sch := NewSchedule("job-1", "*/1", task)
	sch.NextRun = time.Now().Add(time.Hour)
	s.Schedule(sch)

	due := s.GetDueTasks()
	if len(due) != 0 {
		t.Fatalf("expected 0 due tasks for future task, got %d", len(due))
	}
}

func TestScheduler_GetDueTasks_PastTask(t *testing.T) {
	s := NewScheduler()
	task := func(ctx context.Context) error { return nil }
	sch := NewSchedule("job-1", "*/1", task)
	sch.Enabled = true
	s.Schedule(sch)
	sch.NextRun = time.Now().Add(-time.Hour)

	due := s.GetDueTasks()
	if len(due) != 1 {
		t.Fatalf("expected 1 due task, got %d", len(due))
	}
}

func TestScheduler_Run(t *testing.T) {
	s := NewScheduler()
	executed := false
	task := func(ctx context.Context) error {
		executed = true
		return nil
	}
	sch := NewSchedule("job-1", "*/1", task)
	s.Schedule(sch)

	err := s.Run(sch)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !executed {
		t.Fatal("task should have been executed")
	}
	if sch.LastRun.IsZero() {
		t.Fatal("LastRun should be updated")
	}
}

func TestScheduler_Run_Error(t *testing.T) {
	s := NewScheduler()
	task := func(ctx context.Context) error {
		return errors.New("task failed")
	}
	sch := NewSchedule("job-1", "*/1", task)
	s.Schedule(sch)

	err := s.Run(sch)
	if err == nil {
		t.Fatal("expected error from task")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	s := NewScheduler()

	if s.IsRunning() {
		t.Fatal("should not be running initially")
	}

	s.Start()
	if !s.IsRunning() {
		t.Fatal("should be running after Start")
	}

	s.Stop()
	if s.IsRunning() {
		t.Fatal("should not be running after Stop")
	}

	s.Start()
	if !s.IsRunning() {
		t.Fatal("should be running after restart")
	}
	s.Stop()
}

func TestParseCron(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"@every 5s", false},
		{"@every 1m", false},
		{"@every 1h", false},
		{"*/5", false},
		{"*/10", false},
		{"0 0 * * *", true},
		{"invalid", true},
	}

	for _, tt := range tests {
		_, err := parseCron(tt.expr, time.Now())
		if tt.wantErr && err == nil {
			t.Fatalf("parseCron(%q) should have errored", tt.expr)
		}
		if !tt.wantErr && err != nil {
			t.Fatalf("parseCron(%q) should not have errored: %v", tt.expr, err)
		}
	}
}

func TestParseCron_AtEvery(t *testing.T) {
	base := time.Now()
	result, err := parseCron("@every 30s", base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := base.Add(30 * time.Second)
	if result.Unix() != expected.Unix() {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestParseCron_StarSlash(t *testing.T) {
	base := time.Now()
	result, err := parseCron("*/15", base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := base.Add(15 * time.Minute)
	if result.Unix() != expected.Unix() {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestScheduler_StartStop_MultipleStops(t *testing.T) {
	s := NewScheduler()
	s.Start()
	s.Stop()

	s.Stop()
}

func TestScheduler_DoubleStart(t *testing.T) {
	s := NewScheduler()
	s.Start()
	s.Start()

	if !s.IsRunning() {
		t.Fatal("should still be running")
	}
	s.Stop()
}
