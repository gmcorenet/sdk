package gmcore_scheduler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	schedules  map[string]*Schedule
	mu         sync.RWMutex
	stop       chan struct{}
	done       chan struct{}
	running    bool
	wg         sync.WaitGroup
	stopOnce   sync.Once
	ctx        context.Context
	cancel     context.CancelFunc
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
	s.mu.RLock()
	ctx := s.ctx
	s.mu.RUnlock()

	if ctx == nil {
		ctx = context.Background()
	}
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
	s.ctx, s.cancel = context.WithCancel(context.Background())
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
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.running = false
		if s.cancel != nil {
			s.cancel()
		}
		close(s.stop)
		s.mu.Unlock()

		s.wg.Wait()
		close(s.done)
	})
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
	if strings.HasPrefix(expr, "@every ") {
		durationStr := strings.TrimPrefix(expr, "@every ")
		d, err := time.ParseDuration(durationStr)
		if err != nil {
			return base.Add(time.Minute), fmt.Errorf("invalid duration: %w", err)
		}
		return base.Add(d), nil
	}
	if strings.HasPrefix(expr, "*/") {
		intervalStr := strings.TrimPrefix(expr, "*/")
		interval, err := strconv.Atoi(intervalStr)
		if err != nil || interval <= 0 {
			return base.Add(time.Minute), nil
		}
		return base.Add(time.Duration(interval)*time.Minute), nil
	}
	return base.Add(time.Minute), fmt.Errorf("unsupported cron expression: %s (only @every and */N formats supported)", expr)
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
