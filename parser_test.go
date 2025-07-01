package beat

import (
	"fmt"
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
		{"2012-07-09T15:00:00+08:00", "* * * * * 0/15", true},
		{"2012-07-09T15:00:45+08:00", "* * * * * 0/15", true},
		{"2012-07-09T15:00:40+08:00", "* * * * * 0/15", false},

		// Every 15 seconds, starting at 5 seconds.
		{"2012-07-09T15:00:05+08:00", "* * * * * 5/15", true},
		{"2012-07-09T15:00:20+08:00", "* * * * * 5/15", true},
		{"2012-07-09T15:00:50+08:00", "* * * * * 5/15", true},

		// Test interaction of DOW and DOM.
		// If both are restricted, then both need to match.
		{"2012-07-15T00:00:00+08:00", "* 1,15 7 * * *", true},
		{"2012-06-15T00:00:00+08:00", "* 1,15 7 * * *", false},
		{"2012-08-01T00:00:00+08:00", "* 1,15 7 * * *", false},
		{"2012-07-15T00:00:00+08:00", "* */10 7 * * *", false},

		// However, if one has a star, then both need to match.
		{"2012-07-15T00:00:00+08:00", "* * 7 * * *", true},
		{"2012-07-09T00:00:00+08:00", "* 1,15 * * * *", false},
		{"2012-07-15T00:00:00+08:00", "* 1,15 * * * *", true},
		{"2012-07-15T00:00:00+08:00", "* */2 7 * * *", true},
	}

	for _, test := range tests {
		sched, err := defaultParser.Parse(test.spec)
		if err != nil {
			t.Error(err)
			continue
		}

		expected := parseTime(test.time)
		actual := sched.Next(expected.Add(-1 * time.Second))

		if test.expected && expected != actual || !test.expected && expected.Equal(actual) {
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

	expr1 := "* */2 3 0 0 0"
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

	expr2 := "* * 3 0 0 0"
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
	now := time.Now().Truncate(time.Second).In(time.UTC)
	expect := now.Add(5 * time.Second)

	shanghaiLoc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}

	tests := []struct {
		spec     string
		expected bool
	}{
		{
			fmt.Sprintf("%d %d %d %d %d %d",
				expect.Month(), expect.Day(), weekday(expect),
				expect.Hour(), expect.Minute(), expect.Second()),
			true,
		},
		{
			fmt.Sprintf("TZ=Asia/Shanghai %d %d %d %d %d %d",
				expect.In(shanghaiLoc).Month(), expect.In(shanghaiLoc).Day(), weekday(expect.In(shanghaiLoc)),
				expect.In(shanghaiLoc).Hour(), expect.In(shanghaiLoc).Minute(), expect.In(shanghaiLoc).Second()),
			true,
		},
		{
			fmt.Sprintf("TZ=Asia/Shanghai %d %d %d %d %d %d",
				expect.Month(), expect.Day(), weekday(expect),
				expect.Hour(), expect.Minute(), expect.Second()),
			false,
		},
		{
			fmt.Sprintf("TZ=Asia/Tokyo %d %d %d %d %d %d",
				expect.In(shanghaiLoc).Month(), expect.In(shanghaiLoc).Day(), weekday(expect.In(shanghaiLoc)),
				expect.In(shanghaiLoc).Hour(), expect.In(shanghaiLoc).Minute(), expect.In(shanghaiLoc).Second()),
			false,
		},
	}

	for _, test := range tests {
		sched, err := defaultParser.Parse(test.spec)
		if err != nil {
			t.Error(err)
			continue
		}
		actual := sched.Next(now)

		if (test.expected && expect != actual) || (!test.expected && expect.Equal(actual)) {
			t.Errorf("Fail evaluating %s on %s: (expected) %s != %s (actual)",
				test.spec, expect, expect, actual)
		}
	}
}
