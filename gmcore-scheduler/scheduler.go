package gmcore_scheduler

import (
	"context"
	"sync"
	"time"
)

type Task func(ctx context.Context) error

type Schedule struct {
	ID         string
	Task       Task
	Expression string
	Timezone   *time.Location
	LastRun    time.Time
	NextRun    time.Time
	Enabled    bool
}

type Scheduler interface {
	Schedule(schedule *Schedule) error
	Unschedule(id string) error
	GetDueTasks() []*Schedule
	Run(schedule *Schedule) error
	Start()
	Stop()
}

type scheduler struct {
	schedules map[string]*Schedule
	mu        sync.RWMutex
	stop      chan struct{}
	done      chan struct{}
	running   bool
	wg        sync.WaitGroup
}

func NewScheduler() *scheduler {
	return &scheduler{
		schedules: make(map[string]*Schedule),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

func (s *scheduler) Schedule(schedule *Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	schedule.NextRun = s.calculateNextRun(schedule.Expression, schedule.Timezone)
	s.schedules[schedule.ID] = schedule
	return nil
}

func (s *scheduler) Unschedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.schedules, id)
	return nil
}

func (s *scheduler) GetDueTasks() []*Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	due := make([]*Schedule, 0, len(s.schedules))
	for _, sch := range s.schedules {
		if sch.Enabled && now.After(sch.NextRun) {
			due = append(due, sch)
		}
	}
	return due
}

func (s *scheduler) Run(schedule *Schedule) error {
	ctx := context.Background()
	err := schedule.Task(ctx)
	if err == nil {
		schedule.LastRun = time.Now()
		schedule.NextRun = s.calculateNextRun(schedule.Expression, schedule.Timezone)
	}
	return err
}

func (s *scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				due := s.GetDueTasks()
				for _, sch := range due {
					s.wg.Add(1)
					go func(schedule *Schedule) {
						defer s.wg.Done()
						s.Run(schedule)
					}(sch)
				}
			}
		}
	}()
}

func (s *scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stop)
	s.mu.Unlock()

	s.wg.Wait()
	close(s.done)
}

func (s *scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *scheduler) calculateNextRun(expression string, loc *time.Location) time.Time {
	now := time.Now()
	if loc != nil {
		now = now.In(loc)
	}

	t, _ := parseCron(expression, now)
	return t
}

func parseCron(expr string, base time.Time) (time.Time, error) {
	return base.Add(time.Minute), nil
}

func NewSchedule(id, cronExpr string, task Task) *Schedule {
	return &Schedule{
		ID:         id,
		Task:       task,
		Expression: cronExpr,
		Timezone:   time.Local,
		Enabled:    true,
	}
}
