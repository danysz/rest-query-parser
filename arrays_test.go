package rqp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArrayFilters(t *testing.T) {
	v := Validations{"ids:intarray": nil, "tags:stringarray": nil, "w:floatarray": nil}

	cases := []struct {
		url   string
		where string
		args  []interface{}
		err   string
	}{
		{"?ids=1,2", " WHERE (ids @> ARRAY[?,?]::int[] AND ids <@ ARRAY[?,?]::int[])", []interface{}{1, 2, 1, 2}, ""},
		{"?ids[ne]=3", " WHERE NOT (ids @> ARRAY[?]::int[] AND ids <@ ARRAY[?]::int[])", []interface{}{3, 3}, ""},
		{"?tags=a,b", " WHERE (tags @> ARRAY[?,?]::text[] AND tags <@ ARRAY[?,?]::text[])", []interface{}{"a", "b", "a", "b"}, ""},
		{"?tags=a') OR 1=1 --", " WHERE (tags @> ARRAY[?]::text[] AND tags <@ ARRAY[?]::text[])", []interface{}{"a') OR 1=1 --", "a') OR 1=1 --"}, ""},
		{"?w=1.5,2", " WHERE (w @> ARRAY[?,?]::float8[] AND w <@ ARRAY[?,?]::float8[])", []interface{}{1.5, 2.0, 1.5, 2.0}, ""},
		{"?ids[is]=NULL", " WHERE ids IS NULL", nil, ""},
		{"?ids=a", "", nil, "ids: bad format"},
		{"?ids[gt]=1", "", nil, "ids[gt]: method are not allowed"},
	}
	for _, c := range cases {
		u, _ := url.Parse("http://localhost/" + c.url)
		q, err := NewParse(u.Query(), v)
		if c.err != "" {
			assert.EqualError(t, err, c.err, c.url)
			continue
		}
		assert.NoError(t, err, c.url)
		assert.Equal(t, c.where, q.WHERE(), c.url)
		if c.args != nil {
			assert.Equal(t, c.args, q.Args(), c.url)
		}
	}
}
