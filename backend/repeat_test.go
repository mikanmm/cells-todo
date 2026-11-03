package main

import (
	"reflect"
	"testing"
)

func TestRepeatValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    string
		days    []int64
		want    []int64
		invalid bool
	}{
		{"default", "", nil, []int64{}, false},
		{"none clears days", "none", []int64{1}, []int64{}, false},
		{"weekdays locks preset", "weekdays", []int64{6}, []int64{1, 2, 3, 4, 5}, false},
		{"daily locks preset", "daily", nil, []int64{1, 2, 3, 4, 5, 6, 7}, false},
		{"custom sorted", "custom", []int64{6, 1, 5}, []int64{1, 5, 6}, false},
		{"empty custom", "custom", nil, nil, true},
		{"invalid day", "custom", []int64{8}, nil, true},
		{"zero day", "custom", []int64{0}, nil, true},
		{"duplicate", "custom", []int64{1, 1}, nil, true},
		{"unknown type", "weekly", nil, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := todoInput{Title: "Task", RepeatType: tc.kind, RepeatDays: tc.days}
			_, msg := in.validate()
			if (msg != "") != tc.invalid {
				t.Fatalf("unexpected validation: %q", msg)
			}
			if !tc.invalid && !reflect.DeepEqual([]int64(in.RepeatDays), tc.want) {
				t.Fatalf("got days %v, want %v", in.RepeatDays, tc.want)
			}
			if in.Notify {
				t.Fatal("notify must default to false")
			}
		})
	}
}
