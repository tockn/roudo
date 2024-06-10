package roudo

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/tidwall/buntdb"
)

type RoudoReportRepository interface {
	SaveCurrentState(s RoudoState) error
	GetCurrentState() (RoudoState, error)
	GetLastEventAt() (*RoudoTime, error)
	SaveLastEventAt(rt RoudoTime) error

	SaveRoudoReport(date Date, rs []Roudo) error
	GetRoudoReport(date Date) ([]Roudo, error)
}

func NewRoudoReportRepository(db *buntdb.DB) RoudoReportRepository {
	return &roudoRepository{db: db}
}

type roudoRepository struct {
	db *buntdb.DB
}

const (
	CurrentStateKey = "current_state"
	LastEventAtKey  = "last_event_at"
)

func (r *roudoRepository) SaveCurrentState(s RoudoState) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(CurrentStateKey, string(s), nil)
		return err
	})
}

func (r *roudoRepository) GetCurrentState() (RoudoState, error) {
	var s RoudoState
	err := r.db.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(CurrentStateKey)
		if err != nil {
			return err
		}
		s = RoudoState(v)
		return nil
	})
	if errors.Is(err, buntdb.ErrNotFound) {
		return RoudoStateOff, nil
	} else if err != nil {
		return "", err
	}
	return s, nil
}

func (r *roudoRepository) SaveLastEventAt(rt RoudoTime) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(LastEventAtKey, rt.t.Format(time.RFC3339), nil)
		return err
	})
}

func (r *roudoRepository) GetLastEventAt() (*RoudoTime, error) {
	var rt *RoudoTime
	err := r.db.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(LastEventAtKey)
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil
		} else if err != nil {
			return err
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		tt := NewRoudoTime(t, 0)
		rt = &tt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (r *roudoRepository) SaveRoudoReport(date Date, rs []Roudo) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		bs, err := json.Marshal(rs)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(string(date), string(bs), nil)
		return err
	})
}

func (r *roudoRepository) GetRoudoReport(date Date) ([]Roudo, error) {
	var rs []Roudo
	err := r.db.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(string(date))
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil
		} else if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(v), &rs); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rs, nil
}
