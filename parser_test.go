package cron

import (
	"testing"
	"time"
)

func TestActivation(t *testing.T) {
	tests := []struct {
		time     string
		spec     string
		expected bool
	}{
		// Every 15 seconds.
		{"2012-07-09T15:00:00+08:00", "* * * * * * 0/15", true},
		{"2012-07-09T15:00:45+08:00", "* * * * * * 0/15", true},
		{"2012-07-09T15:00:40+08:00", "* * * * * * 0/15", false},

		// Every 15 seconds, starting at 5 seconds.
		{"2012-07-09T15:00:05+08:00", "* * * * * * 5/15", true},
		{"2012-07-09T15:00:20+08:00", "* * * * * * 5/15", true},
		{"2012-07-09T15:00:50+08:00", "* * * * * * 5/15", true},

		// Test interaction of DOW and DOM.
		// If both are restricted, then only one needs to match.
		// {"2012-07-15T00:00:00+08:00", "* * 1,15 0 * * *", true},
		// {"2012-06-15T00:00:00+08:00", "* * 1,15 0 * * *", true},
		// {"2012-08-01T00:00:00+08:00", "* * 1,15 0 * * *", true},
		// {"2012-07-15T00:00:00+08:00", "* * */10 0 * * *", true},

		// However, if one has a star, then both need to match.
		{"2012-07-15T00:00:00+08:00", "* * * 0 * * *", true},
		{"2012-07-09T00:00:00+08:00", "* * 1,15 * * * *", false},
		{"2012-07-15T00:00:00+08:00", "* * 1,15 * * * *", true},
		{"2012-07-15T00:00:00+08:00", "* * */2 0 * * *", true},
	}

	for _, test := range tests {
		sched, err := defaultParser.Parse(test.spec)
		if err != nil {
			t.Error(err)
			continue
		}
		actual := sched.Next(parseTime(test.time).Add(-1 * time.Second))
		// expected := getTime(test.time)
		expected := parseTime(test.time)
		if test.expected && expected != actual || !test.expected && expected == actual {
			t.Errorf("Fail evaluating %s on %s: (expected) %s != %s (actual)",
				test.spec, test.time, expected, actual)
		}
	}
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDomAndDow(t *testing.T) {
	start := "2024-11-06T00:00:00+08:00"

	expr1 := "* * */2 3 0 0 0"
	expectSet1 := []string{
		"2024-11-13T00:00:00+08:00",
		"2024-11-27T00:00:00+08:00",
		"2024-12-11T00:00:00+08:00",
		"2024-12-25T00:00:00+08:00",
		"2025-01-01T00:00:00+08:00",
	}
	sched1, err := defaultParser.Parse(expr1)
	if err != nil {
		panic(err)
	}

	next := parseTime(start)

	for _, item := range expectSet1 {
		actual := sched1.Next(next)
		expected := parseTime(item)
		if actual != expected {
			t.Errorf("Fail evaluating %s on %s: (expected) %s != %s (actual)",
				expr1, next, expected, actual)
		}

		next = actual
	}

	expr2 := "* * * 3 0 0 0"
	expectSet2 := []string{
		"2024-11-13T00:00:00+08:00",
		"2024-11-20T00:00:00+08:00",
		"2024-11-27T00:00:00+08:00",
		"2024-12-04T00:00:00+08:00",
		"2024-12-11T00:00:00+08:00",
	}
	sched2, err := defaultParser.Parse(expr2)
	if err != nil {
		panic(err)
	}

	next = parseTime(start)

	for _, item := range expectSet2 {
		actual := sched2.Next(next)
		expected := parseTime(item)
		if actual != expected {
			t.Errorf("Fail evaluating %s on %s: (expected) %s != %s (actual)",
				expr2, next, expected, actual)
		}

		next = actual
	}
}

func TestParserTimeZone(t *testing.T) {
	// TODO: implement
	// now := time.Now().Truncate(time.Second)
	// tm := now.Add(2 * time.Second)

	// fmt.Println("now     ", now)
	// fmt.Println("expected", tm)

	// expr := fmt.Sprintf("%d %d %d %d %d %d %d", tm.Year(), tm.Month(), tm.Day(), tm.Weekday(), tm.Hour(), tm.Minute(), tm.Second())

	// sched, err := defaultParser.Parse(expr)
	// if err != nil {
	// 	panic(err)
	// }

	// actual := sched.Next(now)

	// if tm != actual {
	// 	t.Errorf("Fail evaluating %s: (expected) %s != %s (actual)",
	// 		expr, tm, actual)
	// }
}
