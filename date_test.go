package rqp

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fixNow pins NowFunc to 2026-08-19 (a Wednesday) for the duration of a test.
func fixNow(t *testing.T) {
	t.Helper()

	original := NowFunc
	NowFunc = func() time.Time {
		return time.Date(2026, 8, 19, 14, 35, 12, 0, time.UTC)
	}
	t.Cleanup(func() {
		NowFunc = original
	})
}

func Test_ParseDate(t *testing.T) {
	fixNow(t)

	t.Run("absolute", func(t *testing.T) {
		date, err := ParseDate("2020-10-02")
		assert.NoError(t, err)
		assert.Equal(t, "2020-10-02", date.Format(DateLayout))
	})

	t.Run("keywords", func(t *testing.T) {
		tests := map[string]string{
			"today":     "2026-08-19",
			"yesterday": "2026-08-18",
			"tomorrow":  "2026-08-20",
			"TODAY":     "2026-08-19",
		}

		for value, expected := range tests {
			date, err := ParseDate(value)
			assert.NoError(t, err, value)
			assert.Equal(t, expected, date.Format(DateLayout), value)
		}
	})

	t.Run("relative", func(t *testing.T) {
		tests := map[string]string{
			"-90d":     "2026-05-21",
			"-90days":  "2026-05-21",
			"-1day":    "2026-08-18",
			"0d":       "2026-08-19",
			"30d":      "2026-09-18",
			"+30d":     "2026-09-18",
			"-2w":      "2026-08-05",
			"3weeks":   "2026-09-09",
			"-1m":      "2026-07-19",
			"-6months": "2026-02-19",
			"18months": "2028-02-19",
			"-1y":      "2025-08-19",
			"-10years": "2016-08-19",
			"1YEAR":    "2027-08-19",
		}

		for value, expected := range tests {
			date, err := ParseDate(value)
			assert.NoError(t, err, value)
			assert.Equal(t, expected, date.Format(DateLayout), value)
		}
	})

	t.Run("month overflow normalizes", func(t *testing.T) {
		original := NowFunc
		NowFunc = func() time.Time {
			return time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC)
		}
		defer func() { NowFunc = original }()

		date, err := ParseDate("-1m")
		assert.NoError(t, err)
		assert.Equal(t, "2026-03-03", date.Format(DateLayout))
	})

	t.Run("leap day", func(t *testing.T) {
		original := NowFunc
		NowFunc = func() time.Time {
			return time.Date(2024, 2, 29, 8, 0, 0, 0, time.UTC)
		}
		defer func() { NowFunc = original }()

		date, err := ParseDate("1y")
		assert.NoError(t, err)
		assert.Equal(t, "2025-03-01", date.Format(DateLayout))
	})

	t.Run("start of day in location of NowFunc", func(t *testing.T) {
		original := NowFunc
		location := time.FixedZone("CEST", 2*60*60)
		NowFunc = func() time.Time {
			return time.Date(2026, 8, 19, 23, 59, 59, 0, location)
		}
		defer func() { NowFunc = original }()

		date, err := ParseDate("today")
		assert.NoError(t, err)
		assert.Equal(t, "2026-08-19 00:00:00 +0200 CEST", date.String())
	})

	t.Run("bad format", func(t *testing.T) {
		values := []string{
			"", " ", "-90", "90x", "d90", "-d", "--1d", "-1 d", "2026-13-01",
			"19.08.2026", "now", "-90dd", "yesterdays", "-99999999999999999999d",
		}

		for _, value := range values {
			_, err := ParseDate(value)
			assert.Equal(t, ErrBadFormat, err, value)
		}
	})
}

func Test_IsRelativeDate(t *testing.T) {
	relative := []string{"-90d", "30days", "+1y", "today", "YESTERDAY", "tomorrow", " -1w "}
	for _, value := range relative {
		assert.True(t, IsRelativeDate(value), value)
	}

	absolute := []string{"2026-08-19", "", "now", "-90", "days"}
	for _, value := range absolute {
		assert.False(t, IsRelativeDate(value), value)
	}
}

func Test_DateFilter(t *testing.T) {
	fixNow(t)

	validations := Validations{
		"created_at:date": nil,
	}

	t.Run("relative value is resolved", func(t *testing.T) {
		filter, err := newFilter("created_at[gte]", "-90d", ",", validations, false)
		assert.NoError(t, err)
		assert.Equal(t, "date", filter.ValueType)
		assert.Equal(t, "2026-05-21", filter.Value)

		where, err := filter.Where(0)
		assert.NoError(t, err)
		assert.Equal(t, "created_at >= ?", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{"2026-05-21"}, args)
	})

	t.Run("absolute value is normalized", func(t *testing.T) {
		filter, err := newFilter("created_at", "2020-10-02", ",", validations, false)
		assert.NoError(t, err)
		assert.Equal(t, "2020-10-02", filter.Value)
	})

	t.Run("list values", func(t *testing.T) {
		filter, err := newFilter("created_at[in]", "today,-1d,2020-10-02", ",", validations, false)
		assert.NoError(t, err)
		assert.Equal(t, []string{"2026-08-19", "2026-08-18", "2020-10-02"}, filter.Value)

		where, err := filter.Where(0)
		assert.NoError(t, err)
		assert.Equal(t, "created_at IN (?, ?, ?)", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{"2026-08-19", "2026-08-18", "2020-10-02"}, args)
	})

	t.Run("null and empty", func(t *testing.T) {
		tests := map[string]string{
			"null":        "created_at IS NULL",
			"empty":       "created_at IS ''",
			"nullorempty": "(created_at IS NULL OR created_at IS '')",
		}

		for value, expected := range tests {
			filter, err := newFilter("created_at[is]", value, ",", validations, false)
			assert.NoError(t, err, value)

			where, err := filter.Where(0)
			assert.NoError(t, err, value)
			assert.Equal(t, expected, where, value)
		}

		filter, err := newFilter("created_at[not]", "null", ",", validations, false)
		assert.NoError(t, err)

		where, err := filter.Where(0)
		assert.NoError(t, err)
		assert.Equal(t, "created_at IS NOT NULL", where)
	})

	t.Run("errors", func(t *testing.T) {
		_, err := newFilter("created_at[gte]", "-90x", ",", validations, false)
		assert.Equal(t, ErrBadFormat, err)

		_, err = newFilter("created_at[in]", "-90d,broken", ",", validations, false)
		assert.Equal(t, ErrBadFormat, err)

		_, err = newFilter("created_at[is]", "-90d", ",", validations, false)
		assert.Equal(t, ErrBadFormat, err)

		_, err = newFilter("created_at[like]", "-90d", ",", validations, false)
		assert.Equal(t, ErrMethodNotAllowed, err)

		// a delimited value is only split for IN/NIN, so it stays one bad date
		_, err = newFilter("created_at[gte]", "-90d,-80d", ",", validations, false)
		assert.Equal(t, ErrBadFormat, err)

		filter := &Filter{Name: "created_at", Method: GTE}
		assert.Equal(t, ErrMethodNotAllowed, filter.setDate([]string{"-90d", "-80d"}))
	})

	t.Run("validation func receives resolved value", func(t *testing.T) {
		var seen interface{}

		q, err := NewParse(url.Values{"created_at[gte]": []string{"-90d"}}, Validations{
			"created_at:date": func(value interface{}) error {
				seen = value
				return nil
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, "2026-05-21", seen)
		assert.Equal(t, " WHERE created_at >= ?", q.WHERE())
		assert.Equal(t, []interface{}{"2026-05-21"}, q.Args())
	})

	t.Run("required tag", func(t *testing.T) {
		_, err := NewParse(url.Values{}, Validations{
			"created_at:date:required": nil,
		})
		assert.Error(t, err)

		q, err := NewParse(url.Values{"created_at[lt]": []string{"today"}}, Validations{
			"created_at:date:required": nil,
		})
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{"2026-08-19"}, q.Args())
	})
}
