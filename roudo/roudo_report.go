package roudo

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/alexflint/go-filemutex"
)

type RoudoReporter interface {
	HandleRoudoEvent() error
	Kansi() error
	SaveRoudoReport(date Date, rs []Roudo) error
}

func NewRoudoReporter(repo RoudoReportRepository, logger *slog.Logger, notificator Notificator, fm *filemutex.FileMutex) RoudoReporter {
	return &roudoReport{
		repo:                  repo,
		mux:                   fm,
		notificator:           notificator,
		shiftDuration:         5 * time.Hour,
		startBreakInterval:    35 * time.Minute,
		finishWorkingInterval: 4 * time.Hour,
		logger:                logger,
	}
}

type roudoReport struct {
	repo                  RoudoReportRepository
	mux                   *filemutex.FileMutex
	notificator           Notificator
	shiftDuration         time.Duration
	startBreakInterval    time.Duration
	finishWorkingInterval time.Duration
	logger                *slog.Logger
}

func (r *roudoReport) HandleRoudoEvent() error {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.logger.Debug("handle roudo event!")

	if err := r.repo.SaveLastEventAt(NewRoudoTime(time.Now(), r.shiftDuration)); err != nil {
		return err
	}

	s, err := r.repo.GetCurrentState()
	if err != nil {
		return err
	}

	t := NewRoudoTime(time.Now(), r.shiftDuration)
	switch s {
	case RoudoStateOff:
		return r.startNewWorking(t)
	case RoudoStateBreaking:
		return r.finishBreaking()
	}

	return nil
}

func (r *roudoReport) Kansi() error {
	r.mux.Lock()
	defer r.mux.Unlock()

	s, err := r.repo.GetCurrentState()
	if err != nil {
		return err
	}

	r.logger.Debug("kansi", slog.String("state", string(s)))

	t := NewRoudoTime(time.Now(), r.shiftDuration)
	switch s {
	case RoudoStateWorking:
		return r.kansiWorking(t)
	case RoudoStateBreaking:
		return r.kansiBreaking(t)
	}

	return nil
}

func (r *roudoReport) SaveRoudoReport(date Date, rs []Roudo) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	return r.repo.SaveRoudoReport(date, rs)
}

func (r *roudoReport) kansiWorking(now RoudoTime) error {
	lastEventAt, err := r.repo.GetLastEventAt()
	if err != nil {
		return err
	}
	if lastEventAt == nil {
		return fmt.Errorf("current_state: working なのに lastEventAt が nil です")
	}

	// 労働中に日跨ぎした場合は、最終イベント時刻を前日の労働終了時刻とし、労働を終了する
	if now.IsOvernight(*lastEventAt) {
		yesterdayReport, err := r.repo.GetRoudoReport(lastEventAt.ShiftedDate())
		if err != nil {
			return err
		}
		yesterdayReport[len(yesterdayReport)-1].EndAt = lastEventAt.Time()
		if err := r.repo.SaveRoudoReport(lastEventAt.ShiftedDate(), yesterdayReport); err != nil {
			return err
		}
		return r.finishWorking(*lastEventAt)
	}

	if now.Time().After(lastEventAt.Time().Add(r.startBreakInterval)) {
		return r.startBreaking(*lastEventAt)
	}

	return nil
}

func (r *roudoReport) kansiBreaking(now RoudoTime) error {
	lastEventAt, err := r.repo.GetLastEventAt()
	if err != nil {
		return err
	}
	if lastEventAt == nil {
		return fmt.Errorf("current_state: breaking なのに lastEventAt が nil です")
	}

	// 休憩時間中に日跨ぎをした場合、最終イベント時刻を前日の労働終了時刻とし、労働を終了する
	if now.IsOvernight(*lastEventAt) {
		yesterdayReport, err := r.repo.GetRoudoReport(lastEventAt.ShiftedDate())
		if err != nil {
			return err
		}
		yesterdayReport[len(yesterdayReport)-1].EndAt = lastEventAt.Time()
		if err := r.repo.SaveRoudoReport(lastEventAt.ShiftedDate(), yesterdayReport); err != nil {
			return err
		}
		return r.finishWorking(*lastEventAt)
	}

	if now.Time().After(lastEventAt.Time().Add(r.finishWorkingInterval)) {
		return r.finishWorking(*lastEventAt)
	}

	return nil
}

func (r *roudoReport) startNewWorking(t RoudoTime) error {
	r.logger.Debug("start new working")
	r.notificator.Notify("労働開始", "よろしくお願いします")
	if err := r.repo.SaveCurrentState(RoudoStateWorking); err != nil {
		return err
	}
	rs, err := r.repo.GetRoudoReport(t.ShiftedDate())
	if err != nil {
		return err
	}
	rs = append(rs, Roudo{StartAt: t.Time()})
	return r.repo.SaveRoudoReport(t.ShiftedDate(), rs)
}

func (r *roudoReport) finishWorking(endAt RoudoTime) error {
	r.logger.Debug("finish working")
	r.notificator.Notify("労働終了", "お疲れ様でした")
	if err := r.repo.SaveCurrentState(RoudoStateOff); err != nil {
		return err
	}
	report, err := r.repo.GetRoudoReport(endAt.ShiftedDate())
	if err != nil {
		return err
	}
	report[len(report)-1].EndAt = endAt.Time()

	if len(report[len(report)-1].Breaks) != 0 {
		lastBreak := report[len(report)-1].Breaks[len(report[len(report)-1].Breaks)-1]
		if lastBreak.EndAt == nil {
			report[len(report)-1].Breaks = report[len(report)-1].Breaks[:len(report[len(report)-1].Breaks)-1]
		}
	}

	return r.repo.SaveRoudoReport(endAt.ShiftedDate(), report)
}

func (r *roudoReport) startBreaking(startAt RoudoTime) error {
	r.logger.Debug("start breaking")
	r.notificator.Notify("休憩開始", "ゆっくり休んでください")

	rs, err := r.repo.GetRoudoReport(NewRoudoTime(time.Now(), r.shiftDuration).ShiftedDate())
	if err != nil {
		return err
	}
	if len(rs) == 0 {
		return nil
	}
	rs[len(rs)-1].Breaks = append(rs[len(rs)-1].Breaks, Break{StartAt: *startAt.Time()})
	if err := r.repo.SaveRoudoReport(NewRoudoTime(time.Now(), r.shiftDuration).ShiftedDate(), rs); err != nil {
		return err
	}

	return r.repo.SaveCurrentState(RoudoStateBreaking)
}

func (r *roudoReport) finishBreaking() error {
	r.logger.Debug("finish breaking")
	r.notificator.Notify("休憩終了", "がんばりましょう")
	if err := r.repo.SaveCurrentState(RoudoStateWorking); err != nil {
		return err
	}
	t := NewRoudoTime(time.Now(), r.shiftDuration)
	rs, err := r.repo.GetRoudoReport(t.ShiftedDate())
	if err != nil {
		return err
	}

	if len(rs) == 0 {
		return nil
	}
	if len(rs[len(rs)-1].Breaks) == 0 {
		return nil
	}
	rs[len(rs)-1].Breaks[len(rs[len(rs)-1].Breaks)-1].EndAt = t.Time()

	return r.repo.SaveRoudoReport(t.ShiftedDate(), rs)
}
