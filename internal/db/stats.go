package db

import (
	"context"
	"fmt"
	"time"
)

type StatsStore struct {
	pool *Pool
}

func NewStatsStore(pool *Pool) *StatsStore {
	return &StatsStore{pool: pool}
}

type DailyStat struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type NamedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Stats struct {
	EventsPerDay []DailyStat  `json:"events_per_day"`
	TopKeywords  []NamedCount `json:"top_keywords"`
	TopSources   []NamedCount `json:"top_sources"`
	ChannelUsage []NamedCount `json:"channel_usage"`
	TotalEvents  int          `json:"total_events"`
	AckedEvents  int          `json:"acked_events"`
}

func (s *StatsStore) Get(ctx context.Context) (Stats, error) {
	var st Stats
	since := time.Now().AddDate(0, 0, -30)

	rows, err := s.pool.Query(ctx, `
		SELECT to_char(received_at AT TIME ZONE 'Europe/Berlin', 'YYYY-MM-DD'), COUNT(*)
		FROM event_log WHERE received_at >= $1 AND status != 'NO_MATCH'
		GROUP BY 1 ORDER BY 1
	`, since)
	if err != nil {
		return st, fmt.Errorf("events per day: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d DailyStat
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return st, err
		}
		st.EventsPerDay = append(st.EventsPerDay, d)
		st.TotalEvents += d.Count
	}
	if err := rows.Err(); err != nil {
		return st, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT keyword, COUNT(*) c FROM event_log
		WHERE received_at >= $1 AND status != 'NO_MATCH'
		GROUP BY keyword ORDER BY c DESC LIMIT 10
	`, since)
	if err != nil {
		return st, fmt.Errorf("top keywords: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n NamedCount
		if err := rows.Scan(&n.Name, &n.Count); err != nil {
			return st, err
		}
		st.TopKeywords = append(st.TopKeywords, n)
	}
	if err := rows.Err(); err != nil {
		return st, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT source, COUNT(*) c FROM event_log
		WHERE received_at >= $1 AND status != 'NO_MATCH'
		GROUP BY source ORDER BY c DESC LIMIT 10
	`, since)
	if err != nil {
		return st, fmt.Errorf("top sources: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n NamedCount
		if err := rows.Scan(&n.Name, &n.Count); err != nil {
			return st, err
		}
		st.TopSources = append(st.TopSources, n)
	}
	if err := rows.Err(); err != nil {
		return st, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT ch, COUNT(*) c FROM event_log, UNNEST(channels) AS ch
		WHERE received_at >= $1 AND status != 'NO_MATCH'
		GROUP BY ch ORDER BY c DESC LIMIT 10
	`, since)
	if err != nil {
		return st, fmt.Errorf("channel usage: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n NamedCount
		if err := rows.Scan(&n.Name, &n.Count); err != nil {
			return st, err
		}
		st.ChannelUsage = append(st.ChannelUsage, n)
	}
	if err := rows.Err(); err != nil {
		return st, err
	}
	rows.Close()

	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM acknowledgements a
		JOIN event_log e ON e.id = a.event_log_id
		WHERE e.received_at >= $1
	`, since).Scan(&st.AckedEvents)
	if err != nil {
		return st, fmt.Errorf("acked events: %w", err)
	}

	return st, nil
}
