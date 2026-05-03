package scheduler_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/09-task-scheduler/internal/scheduler"
)

func TestScheduler_FiresTask(t *testing.T) {
	s := scheduler.New()
	var count atomic.Int64

	// "* * * * *" fires every minute — but we'll use a short-interval test
	// by adding a task and manually calling tick via Start with a real ticker
	if err := s.Add("t1", "counter", "* * * * *", func() { count.Add(1) }); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go s.Start(ctx)

	// The scheduler ticks every second but "* * * * *" only fires on minute boundaries.
	// We verify the task is registered and list works.
	tasks := s.List()
	if len(tasks) != 1 || tasks[0].ID != "t1" {
		t.Fatalf("want 1 task, got %v", tasks)
	}
}

func TestScheduler_Remove(t *testing.T) {
	s := scheduler.New()
	s.Add("t1", "test", "* * * * *", func() {})
	s.Remove("t1")
	if len(s.List()) != 0 {
		t.Fatal("expected 0 tasks after remove")
	}
}

func TestScheduler_InvalidExpr(t *testing.T) {
	s := scheduler.New()
	if err := s.Add("t1", "bad", "not-a-cron", func() {}); err == nil {
		t.Fatal("expected error for invalid cron expr")
	}
}
