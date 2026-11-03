package main

import "sort"

// Reopen completed recurring tasks once on a selected local calendar day.
// A missed occurrence does not create duplicate tasks after downtime.
func (a *App) refreshRepeatingTodos() error {
	_, err := a.db.Exec(`UPDATE todos SET done=FALSE, done_at=NULL,
		due_date=(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Jakarta')::date,
		updated_at=CURRENT_TIMESTAMP
		WHERE done=TRUE AND repeat_type<>'none'
		AND (done_at AT TIME ZONE 'Asia/Jakarta')::date < (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Jakarta')::date
		AND (due_date IS NULL OR due_date <= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Jakarta')::date)
		AND EXTRACT(ISODOW FROM CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Jakarta')::integer = ANY(repeat_days)`)
	return err
}

func (in *todoInput) validateRepeat() string {
	switch in.RepeatType {
	case "", "none":
		in.RepeatType = "none"
		in.RepeatDays = []int64{}
	case "weekdays":
		in.RepeatDays = []int64{1, 2, 3, 4, 5}
	case "daily":
		in.RepeatDays = []int64{1, 2, 3, 4, 5, 6, 7}
	case "custom":
		if len(in.RepeatDays) == 0 {
			return "Select at least one repeat day."
		}
		seen := map[int64]bool{}
		for _, day := range in.RepeatDays {
			if day < 1 || day > 7 || seen[day] {
				return "Repeat days must be unique days from Monday (1) to Sunday (7)."
			}
			seen[day] = true
		}
		sort.Slice(in.RepeatDays, func(i, j int) bool { return in.RepeatDays[i] < in.RepeatDays[j] })
	default:
		return "Unknown repeat type."
	}
	return ""
}
