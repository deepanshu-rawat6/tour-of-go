// Package cron parses cron expressions and checks if they match a given time.
// Supports 5-field cron: minute hour day month weekday
// Supports: * (any), */n (every n), n (exact), n-m (range)
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Schedule represents a parsed cron expression.
type Schedule struct {
	expr string
	mins    []bool // 0-59
	hours   []bool // 0-23
	days    []bool // 1-31
	months  []bool // 1-12
	weekdays []bool // 0-6 (Sun=0)
}

// Parse parses a 5-field cron expression.
func Parse(expr string) (*Schedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron: expected 5 fields, got %d", len(fields))
	}
	s := &Schedule{expr: expr}
	var err error
	if s.mins, err = parseField(fields[0], 0, 59); err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	if s.hours, err = parseField(fields[1], 0, 23); err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	if s.days, err = parseField(fields[2], 1, 31); err != nil {
		return nil, fmt.Errorf("day: %w", err)
	}
	if s.months, err = parseField(fields[3], 1, 12); err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	if s.weekdays, err = parseField(fields[4], 0, 6); err != nil {
		return nil, fmt.Errorf("weekday: %w", err)
	}
	return s, nil
}

// Match returns true if t matches the schedule.
func (s *Schedule) Match(t time.Time) bool {
	return s.mins[t.Minute()] &&
		s.hours[t.Hour()] &&
		s.days[t.Day()] &&
		s.months[int(t.Month())] &&
		s.weekdays[int(t.Weekday())]
}

func (s *Schedule) String() string { return s.expr }

func parseField(field string, min, max int) ([]bool, error) {
	bits := make([]bool, max+1)
	for _, part := range strings.Split(field, ",") {
		if err := applyPart(part, min, max, bits); err != nil {
			return nil, err
		}
	}
	return bits, nil
}

func applyPart(part string, min, max int, bits []bool) error {
	if part == "*" {
		for i := min; i <= max; i++ {
			bits[i] = true
		}
		return nil
	}
	if strings.HasPrefix(part, "*/") {
		step, err := strconv.Atoi(part[2:])
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step %q", part)
		}
		for i := min; i <= max; i += step {
			bits[i] = true
		}
		return nil
	}
	if idx := strings.Index(part, "-"); idx > 0 {
		lo, err1 := strconv.Atoi(part[:idx])
		hi, err2 := strconv.Atoi(part[idx+1:])
		if err1 != nil || err2 != nil {
			return fmt.Errorf("invalid range %q", part)
		}
		for i := lo; i <= hi; i++ {
			bits[i] = true
		}
		return nil
	}
	n, err := strconv.Atoi(part)
	if err != nil {
		return fmt.Errorf("invalid value %q", part)
	}
	if n < min || n > max {
		return fmt.Errorf("value %d out of range [%d,%d]", n, min, max)
	}
	bits[n] = true
	return nil
}
