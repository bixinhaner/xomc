package regularreport

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type EnsureRunRequest struct {
	Template     Template
	Period       Period
	ScheduledAt  time.Time
	WindowStart  time.Time
	WindowEnd    time.Time
	ExportParams []byte
	Recipients   []string
}

type SchedulerRepository interface {
	ListEnabledTemplates(ctx context.Context) ([]Template, error)
	EnsureRun(ctx context.Context, request EnsureRunRequest) (bool, error)
}

type Scheduler struct {
	repo             SchedulerRepository
	locationProvider func() *time.Location
	logger           *zap.Logger
	shortPeriodGrace time.Duration
	dailyGrace       time.Duration
}

func NewScheduler(repo SchedulerRepository, locationProvider func() *time.Location, logger *zap.Logger) *Scheduler {
	if locationProvider == nil {
		locationProvider = func() *time.Location { return time.UTC }
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Scheduler{
		repo: repo, locationProvider: locationProvider, logger: logger.Named("kpi-regular-report-scheduler"),
		shortPeriodGrace: 12 * time.Minute,
		dailyGrace:       15 * time.Minute,
	}
}

func (s *Scheduler) SetReadinessGrace(shortPeriodGrace, dailyGrace time.Duration) *Scheduler {
	if shortPeriodGrace > 0 {
		s.shortPeriodGrace = shortPeriodGrace
	}
	if dailyGrace > 0 {
		s.dailyGrace = dailyGrace
	}
	return s
}

func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if err := s.ScheduleOnce(ctx, time.Now()); err != nil {
		s.logger.Warn("schedule KPI regular reports", zap.Error(err))
	}
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := s.ScheduleOnce(ctx, now); err != nil {
				s.logger.Warn("schedule KPI regular reports", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) ScheduleOnce(ctx context.Context, now time.Time) error {
	templates, err := s.repo.ListEnabledTemplates(ctx)
	if err != nil {
		return fmt.Errorf("list enabled KPI report templates: %w", err)
	}
	location := s.locationProvider()
	if location == nil {
		location = time.UTC
	}
	for _, template := range templates {
		scheduledAt, err := ScheduledAt(now, template.Config.SendTime, location)
		if err != nil {
			s.logger.Warn("skip invalid KPI report schedule", zap.String("template_id", template.ID.String()), zap.Error(err))
			continue
		}
		// 最多补当天一次。超过 24 小时的历史任务不在启动时洪泛补跑。
		if now.Before(scheduledAt) || now.Sub(scheduledAt) >= 24*time.Hour {
			continue
		}
		for _, period := range template.Config.Periods {
			start, end, err := CompleteWindow(scheduledAt, period, location)
			if err != nil {
				s.logger.Warn("skip invalid KPI report period", zap.String("template_id", template.ID.String()), zap.Error(err))
				continue
			}
			grace := s.shortPeriodGrace
			if period == PeriodDaily {
				grace = s.dailyGrace
			}
			if now.Before(end.Add(grace)) {
				continue
			}
			params, err := BuildExportParams(template.Payload, period, start, end)
			if err != nil {
				s.logger.Warn("skip invalid KPI report template payload", zap.String("template_id", template.ID.String()), zap.Error(err))
				continue
			}
			created, err := s.repo.EnsureRun(ctx, EnsureRunRequest{
				Template: template, Period: period, ScheduledAt: scheduledAt,
				WindowStart: start, WindowEnd: end, ExportParams: params,
				Recipients: template.Config.Recipients,
			})
			if err != nil {
				return fmt.Errorf("ensure KPI report run for template %s period %s: %w", template.ID, period, err)
			}
			if created {
				s.logger.Info("scheduled KPI regular report", zap.String("template_id", template.ID.String()), zap.String("period", string(period)), zap.Time("scheduled_at", scheduledAt))
			}
		}
	}
	return nil
}
