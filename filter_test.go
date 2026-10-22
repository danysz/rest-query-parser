package rqp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Args(t *testing.T) {
	t.Run("ErrUnknownMethod", func(t *testing.T) {
		filter := Filter{
			Key:    "id[not]",
			Name:   "id",
			Method: NOT,
			Value:  "id",
		}
		_, err := filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)

		filter = Filter{
			Key:    "id[fake]",
			Name:   "id",
			Method: "fake",
		}
		_, err = filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)
	})

	t.Run("String field with IS empty string should fail", func(t *testing.T) {
		filter := Filter{
			Key:    "name[is]",
			Name:   "name",
			Method: IS,
			Value:  "",
		}
		_, err := filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)

		_, err = filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)
	})

	t.Run("Integer field with IS empty string should fail", func(t *testing.T) {
		filter := Filter{
			Key:    "id[is]",
			Name:   "id",
			Method: IS,
			Value:  "",
		}
		_, err := filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)

		_, err = filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)
	})
}

func Test_NullIntegerHandling(t *testing.T) {
	t.Run("Integer with IS NULL method", func(t *testing.T) {
		filter := Filter{
			Key:    "id[is]",
			Name:   "id",
			Method: IS,
			Value:  NULL,
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "id IS NULL", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{NULL}, args)
	})

	t.Run("Integer with NOT NULL method", func(t *testing.T) {
		filter := Filter{
			Key:    "id[not]",
			Name:   "id",
			Method: NOT,
			Value:  NULL,
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "id IS NOT NULL", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{NULL}, args)
	})

	t.Run("Integer with IS method but non-NULL value should fail", func(t *testing.T) {
		filter := Filter{
			Key:    "id[is]",
			Name:   "id",
			Method: IS,
			Value:  123,
		}
		_, err := filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)

		_, err = filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)
	})

	t.Run("Integer with NOT method but non-NULL value should fail", func(t *testing.T) {
		filter := Filter{
			Key:    "id[not]",
			Name:   "id",
			Method: NOT,
			Value:  123,
		}
		_, err := filter.Where()
		assert.Equal(t, err, ErrUnknownMethod)

		_, err = filter.Args()
		assert.Equal(t, err, ErrUnknownMethod)
	})
}

func Test_IntegerNullParsing(t *testing.T) {
	validations := Validations{
		"color_id:int": nil,
	}

	t.Run("Parse color_id[is]=null from URL", func(t *testing.T) {
		filter, err := newFilter("color_id[is]", "null", ",", validations)
		assert.NoError(t, err)
		assert.Equal(t, "color_id[is]", filter.Key)
		assert.Equal(t, "color_id", filter.Name)
		assert.Equal(t, IS, filter.Method)
		assert.Equal(t, NULL, filter.Value)

		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "color_id IS NULL", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{NULL}, args)
	})

	t.Run("Parse color_id[not]=NULL from URL", func(t *testing.T) {
		filter, err := newFilter("color_id[not]", "NULL", ",", validations)
		assert.NoError(t, err)
		assert.Equal(t, "color_id[not]", filter.Key)
		assert.Equal(t, "color_id", filter.Name)
		assert.Equal(t, NOT, filter.Method)
		assert.Equal(t, NULL, filter.Value)

		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "color_id IS NOT NULL", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{NULL}, args)
	})

	t.Run("Parse color_id[eq]=123 still works", func(t *testing.T) {
		filter, err := newFilter("color_id[eq]", "123", ",", validations)
		assert.NoError(t, err)
		assert.Equal(t, "color_id[eq]", filter.Key)
		assert.Equal(t, "color_id", filter.Name)
		assert.Equal(t, EQ, filter.Method)
		assert.Equal(t, 123, filter.Value)

		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "color_id = ?", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{123}, args)
	})
}

func Test_EmptyStringHandling(t *testing.T) {
	t.Run("Empty string with EQ method", func(t *testing.T) {
		filter := Filter{
			Key:    "name[eq]",
			Name:   "name",
			Method: EQ,
			Value:  "",
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "name = ?", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{""}, args)
	})

	t.Run("Empty string with NE method", func(t *testing.T) {
		filter := Filter{
			Key:    "name[ne]",
			Name:   "name",
			Method: NE,
			Value:  "",
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "name != ?", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{""}, args)
	})

	t.Run("Empty string with LIKE method", func(t *testing.T) {
		filter := Filter{
			Key:    "name[like]",
			Name:   "name",
			Method: LIKE,
			Value:  "",
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "name LIKE ?", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{""}, args)
	})

	t.Run("Empty string with ILIKE method", func(t *testing.T) {
		filter := Filter{
			Key:    "name[ilike]",
			Name:   "name",
			Method: ILIKE,
			Value:  "",
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "name ILIKE ?", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{""}, args)
	})

	t.Run("Empty string with IN method", func(t *testing.T) {
		filter := Filter{
			Key:    "name[in]",
			Name:   "name",
			Method: IN,
			Value:  []string{"", "test", ""},
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "name IN (?, ?, ?)", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{"", "test", ""}, args)
	})

	t.Run("Empty string with NIN method", func(t *testing.T) {
		filter := Filter{
			Key:    "name[nin]",
			Name:   "name",
			Method: NIN,
			Value:  []string{"", "excluded"},
		}
		where, err := filter.Where()
		assert.NoError(t, err)
		assert.Equal(t, "name NOT IN (?, ?)", where)

		args, err := filter.Args()
		assert.NoError(t, err)
		assert.Equal(t, []interface{}{"", "excluded"}, args)
	})
}

func Test_RemoveOrEntries(t *testing.T) {
	type testCase struct {
		name           string
		urlQuery       string
		filterToRemove string
		wantWhere      string
	}
	tests := []testCase{
		{
			name:           "should fix OR statements after removing EndOR filter with 2 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10",
			filterToRemove: "test2",
			wantWhere:      " WHERE test1 = ?",
		},
		{
			name:           "should fix OR statements after removing StartOR filter with 2 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10",
			filterToRemove: "test1",
			wantWhere:      " WHERE test2 = ?",
		},
		{
			name:           "should fix OR statements after removing StartOR filter with 3 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10|test3[eq]=test10",
			filterToRemove: "test1",
			wantWhere:      " WHERE (test2 = ? OR test3 = ?)",
		},
		{
			name:           "should fix OR statements after removing EndOR filter with 3 items",
			urlQuery:       "?test1[eq]=test10|test2[eq]=test10|test3[eq]=test10",
			filterToRemove: "test3",
			wantWhere:      " WHERE (test1 = ? OR test2 = ?)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			URL, _ := url.Parse(tt.urlQuery)
			q := NewQV(nil, Validations{
				"test1": nil,
				"test2": nil,
				"test3": nil,
			})
			_ = q.SetUrlQuery(URL.Query()).Parse()

			// Act
			_ = q.RemoveFilter(tt.filterToRemove)

			// Assert
			assert.Equal(t, tt.wantWhere, q.WHERE())
		})
	}
}
