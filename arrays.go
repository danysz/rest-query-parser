package rqp

import (
	"fmt"
	"strconv"
	"strings"
)

// Array column types (PostgreSQL), set with the tags ":intarray", ":stringarray" and ":floatarray".
// "eq" matches columns holding exactly the given set of values, "ne" is its negation.
var arrayCasts = map[string]string{
	"intarray":    "int[]",
	"stringarray": "text[]",
	"floatarray":  "float8[]",
}

func isArrayType(valueType string) bool {
	_, ok := arrayCasts[valueType]
	return ok
}

func (f *Filter) setArray(list []string) error {
	switch f.Method {
	case EQ, NE:
		switch f.ValueType {
		case "intarray":
			ints := make([]int, len(list))
			for i, s := range list {
				v, err := strconv.Atoi(s)
				if err != nil {
					return ErrBadFormat
				}
				ints[i] = v
			}
			f.Value = ints
		case "floatarray":
			floats := make([]float64, len(list))
			for i, s := range list {
				v, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return ErrBadFormat
				}
				floats[i] = v
			}
			f.Value = floats
		default:
			f.Value = list
		}
		return nil
	case IS, NOT:
		if len(list) == 1 && strings.ToUpper(list[0]) == NULL {
			f.Value = NULL
			return nil
		}
		return ErrBadFormat
	default:
		return ErrMethodNotAllowed
	}
}

// whereArray returns the condition for eq/ne on array columns, ok is false for any other filter
func (f *Filter) whereArray() (exp string, ok bool) {
	if !isArrayType(f.ValueType) || (f.Method != EQ && f.Method != NE) {
		return "", false
	}
	args, _ := f.Args()
	marks := strings.TrimSuffix(strings.Repeat("?,", len(args)/2), ",")
	arr := fmt.Sprintf("ARRAY[%s]::%s", marks, arrayCasts[f.ValueType])
	exp = fmt.Sprintf("(%s @> %s AND %s <@ %s)", f.Name, arr, f.Name, arr)
	if f.Method == NE {
		exp = "NOT " + exp
	}
	return exp, true
}

// argsArray returns the values twice (once per containment check), ok is false for any other filter
func (f *Filter) argsArray() (args []interface{}, ok bool) {
	if !isArrayType(f.ValueType) || (f.Method != EQ && f.Method != NE) {
		return nil, false
	}
	var values []interface{}
	switch v := f.Value.(type) {
	case []int:
		for _, e := range v {
			values = append(values, e)
		}
	case []float64:
		for _, e := range v {
			values = append(values, e)
		}
	case []string:
		for _, e := range v {
			values = append(values, e)
		}
	}
	return append(values, values...), true
}
