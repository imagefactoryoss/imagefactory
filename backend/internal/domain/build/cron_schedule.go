package build

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type cronField struct {
	any    bool
	values map[int]struct{}
}

func parseCronField(raw string, min, max int) (cronField, error) {
	raw = strings.TrimSpace(raw)
	if raw == "*" {
		return cronField{any: true}, nil
	}
	values := map[int]struct{}{}
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return cronField{}, fmt.Errorf("invalid empty cron field segment")
		}
		if strings.HasPrefix(part, "*/") {
			stepRaw := strings.TrimSpace(strings.TrimPrefix(part, "*/"))
			step, err := strconv.Atoi(stepRaw)
			if err != nil || step <= 0 {
				return cronField{}, fmt.Errorf("invalid cron step %q", part)
			}
			for i := min; i <= max; i += step {
				values[i] = struct{}{}
			}
			continue
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			return cronField{}, fmt.Errorf("invalid cron value %q", part)
		}
		if number < min || number > max {
			return cronField{}, fmt.Errorf("cron value %d out of range [%d,%d]", number, min, max)
		}
		values[number] = struct{}{}
	}
	if len(values) == 0 {
		return cronField{}, fmt.Errorf("cron field has no values")
	}
	return cronField{values: values}, nil
}

func (f cronField) matches(value int) bool {
	if f.any {
		return true
	}
	_, ok := f.values[value]
	return ok
}

func nextTriggerTimeFromCron(expr string, timezone string, from time.Time) (*time.Time, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must contain exactly 5 fields")
	}

	minuteField, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, err
	}
	hourField, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, err
	}
	dayOfMonthField, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, err
	}
	monthField, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, err
	}
	dayOfWeekField, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, err
	}

	loc := time.UTC
	if strings.TrimSpace(timezone) != "" {
		loaded, loadErr := time.LoadLocation(strings.TrimSpace(timezone))
		if loadErr != nil {
			return nil, fmt.Errorf("invalid timezone: %w", loadErr)
		}
		loc = loaded
	}

	cursor := from.In(loc).Truncate(time.Minute).Add(time.Minute)
	searchLimit := cursor.Add(366 * 24 * time.Hour)
	for !cursor.After(searchLimit) {
		dayMatch := dayOfMonthField.matches(cursor.Day())
		weekdayMatch := dayOfWeekField.matches(int(cursor.Weekday()))
		dayAllowed := dayMatch && weekdayMatch
		if !dayOfMonthField.any && !dayOfWeekField.any {
			dayAllowed = dayMatch || weekdayMatch
		}

		if monthField.matches(int(cursor.Month())) &&
			hourField.matches(cursor.Hour()) &&
			minuteField.matches(cursor.Minute()) &&
			dayAllowed {
			next := cursor.UTC()
			return &next, nil
		}
		cursor = cursor.Add(time.Minute)
	}
	return nil, fmt.Errorf("no valid next trigger time found within one year")
}
