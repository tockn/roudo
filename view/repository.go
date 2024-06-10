package view

import (
	"fmt"
	"roudo/roudo"
	"time"
)

type ViewRepository interface {
	ListReports(yearMonth string) (roudoReportForView, error)
}

type viewRepository struct {
	roudoRepo roudo.RoudoReportRepository
}

func NewViewRepository(roudoRepo roudo.RoudoReportRepository) ViewRepository {
	return &viewRepository{roudoRepo}
}

func (r *viewRepository) ListReports(yearMonth string) (roudoReportForView, error) {
	monthStart, monthEnd, err := getMonthStartEnd(yearMonth)
	if err != nil {
		return nil, err
	}

	rsByDate := make(map[roudo.Date][]roudo.Roudo)
	for d := monthStart; !d.After(monthEnd); d = d.AddDate(0, 0, 1) {
		date := roudo.Date(d.Format("2006-01-02"))
		rs, err := r.roudoRepo.GetRoudoReport(date)
		if err != nil {
			return nil, err
		}
		rsByDate[date] = rs
	}

	var reports roudoReportForView
	for d := monthStart; !d.After(monthEnd); d = d.AddDate(0, 0, 1) {
		date := roudo.Date(d.Format("2006-01-02"))
		reports = append(reports, struct {
			Date   roudo.Date
			Roudos []roudo.Roudo
		}{Date: date, Roudos: rsByDate[date]})
	}

	return reports, nil
}

func getMonthStartEnd(yearMonth string) (time.Time, time.Time, error) {
	monthStart, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("月の指定が不正です ex: 2024-03")
	}
	monthEnd := monthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)
	return monthStart, monthEnd, nil
}

type roudoReportForView []struct {
	Date   roudo.Date
	Roudos []roudo.Roudo
}

type flattenRoudoReportForView struct {
	Date       roudo.Date
	Roudo      roudo.Roudo
	RoudoIndex int
	Break      *roudo.Break
	BreakIndex int
}

func (r roudoReportForView) FindByDate(date roudo.Date) []roudo.Roudo {
	for _, report := range r {
		if report.Date == date {
			return report.Roudos
		}
	}
	return nil
}

func (r roudoReportForView) Flatten() []flattenRoudoReportForView {
	flattenRoudos := make([]flattenRoudoReportForView, 0)
	for _, report := range r {
		if len(report.Roudos) == 0 {
			flattenRoudos = append(flattenRoudos, flattenRoudoReportForView{report.Date, roudo.Roudo{}, 0, nil, 0})
		}
		for roudoIndex, roudo := range report.Roudos {
			if len(roudo.Breaks) == 0 {
				flattenRoudos = append(flattenRoudos, flattenRoudoReportForView{report.Date, roudo, roudoIndex, nil, 0})
			}

			for breakIndex, b := range roudo.Breaks {
				b := b
				flattenRoudos = append(flattenRoudos, flattenRoudoReportForView{report.Date, roudo, roudoIndex, &b, breakIndex})
			}
		}
	}
	return flattenRoudos
}
