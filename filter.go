package rqp

import (
	"fmt"
	"strconv"
	"strings"
)

type StateOR byte

const (
	NoOR StateOR = iota
	StartOR
	InOR
	EndOR
)

// Filter represents a filter defined in the query part of URL
type Filter struct {
	Key       string // key from URL (eg. "id[eq]")
	Name      string // name of filter, takes from Key (eg. "id")
	Method    Method // compare method, takes from Key (eg. EQ)
	Value     interface{}
	ValueType string // type of Value, can be "int", "bool", "date" or "string"
	OR        StateOR
}

// detectValidation
// name - only name without method
// validations - must be q.validations
func detectValidation(name string, validations Validations) (ValidationFunc, bool) {

	for k, v := range validations {
		if strings.Contains(k, ":") {
			split := strings.Split(k, ":")
			if split[0] == name {
				return v, true
			}
		} else if k == name {
			return v, true
		}
	}

	return nil, false
}

// detectType
func detectType(name string, validations Validations) string {

	for k := range validations {
		if strings.Contains(k, ":") {
			split := strings.Split(k, ":")
			if split[0] == name {
				switch split[1] {
				case "int", "i":
					return "int"
				case "bool", "b":
					return "bool"
				case "date":
					return "date"
				default:
					return "string"
				}
			}
		}
	}

	return "string"
}

func isNotNull(f *Filter) bool {
	s, ok := f.Value.(string)
	if !ok {
		return false
	}
	return f.Method == NOT && strings.ToUpper(s) == NULL
}

// rawKey - url key
// value - must be one value (if need IN method then values must be separated by comma (,))
func newFilter(rawKey string, value string, delimiter string, validations Validations) (*Filter, error) {
	f := &Filter{
		Key: rawKey,
	}

	// set Key, Name, Method
	if err := f.parseKey(rawKey); err != nil {
		return nil, err
	}

	// detect have we validator func definition on this parameter or not
	validate, ok := detectValidation(f.Name, validations)
	if !ok {
		return nil, ErrValidationNotFound
	}

	// detect type by key names in validations
	valueType := detectType(f.Name, validations)
	f.ValueType = valueType

	if err := f.parseValue(valueType, value, delimiter); err != nil {
		return nil, err
	}

	if !isNotNull(f) && validate != nil {
		if err := f.validate(validate); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *Filter) validate(validate ValidationFunc) error {

	switch f.Value.(type) {
	case []int:
		for _, v := range f.Value.([]int) {
			err := validate(v)
			if err != nil {
				return err
			}
		}
	case []string:
		for _, v := range f.Value.([]string) {
			err := validate(v)
			if err != nil {
				return err
			}
		}
	case int, bool, string:
		// Skip validation for NULL values as they are handled specially
		if IsNullOrEmpty(f.Value) {
			return nil
		}
		err := validate(f.Value)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseKey parses key to set f.Name and f.Method
//
//	id[eq] -> f.Name = "id", f.Method = EQ
func (f *Filter) parseKey(key string) error {

	// default Method is EQ
	f.Method = EQ

	spos := strings.Index(key, "[")
	if spos != -1 {
		f.Name = key[:spos]
		epos := strings.Index(key[spos:], "]")
		if epos != -1 {
			// go inside brekets
			spos = spos + 1
			epos = spos + epos - 1

			if epos-spos > 0 {
				f.Method = Method(strings.ToUpper(string(key[spos:epos])))
				if _, ok := translateMethods[f.Method]; !ok {
					return ErrUnknownMethod
				}
			}
		}
	} else {
		f.Name = key
	}

	return nil
}

// parseValue parses value depends on its type
func (f *Filter) parseValue(valueType string, value string, delimiter string) error {

	var list []string

	if strings.Contains(value, delimiter) && (f.Method == IN || f.Method == NIN || valueType == "int") {
		list = strings.Split(value, delimiter)
	} else {
		list = append(list, value)
	}

	switch valueType {
	case "int":
		err := f.setInt(list)
		if err != nil {
			return err
		}
	case "bool":
		err := f.setBool(list)
		if err != nil {
			return err
		}
	case "date":
		err := f.setDate(list)
		if err != nil {
			return err
		}
	default: // str, string and all other unknown types will handle as string
		err := f.setString(list)
		if err != nil {
			return err
		}
	}

	return nil
}

// Where returns condition expression
func (f *Filter) Where() (string, error) {
	var exp string

	switch f.Method {
	case EQ, NE, GT, LT, GTE, LTE, LIKE, ILIKE, NLIKE, NILIKE:
		exp = fmt.Sprintf("%s %s ?", f.Name, translateMethods[f.Method])
		return exp, nil
	case IS, NOT:
		if !IsNullOrEmpty(f.Value) {
			return exp, ErrUnknownMethod
		}

		// Handling for anything else than text types
		if f.ValueType != "string" && f.ValueType != "date" {
			if f.Value == EMPTY {
				// bools and int's can not be empty
				return exp, ErrUnknownMethod
			}
			return fmt.Sprintf("%s %s NULL", f.Name, translateMethods[f.Method]), nil
		}

		if f.Value == NULL {
			return fmt.Sprintf("%s %s NULL", f.Name, translateMethods[f.Method]), nil
		}

		if f.Value == EMPTY {
			return fmt.Sprintf("%s %s ''", f.Name, translateMethods[f.Method]), nil
		}

		if f.Value == NULLOREMPTY {
			connector := "OR"
			if f.Method == NOT {
				connector = "AND"
			}

			return fmt.Sprintf("(%s %s NULL %s %s %s '')", f.Name, translateMethods[f.Method], connector, f.Name, translateMethods[f.Method]), nil
		}

		return exp, ErrUnknownMethod

	case IN, NIN:
		exp = fmt.Sprintf("%s %s (?)", f.Name, translateMethods[f.Method])
		exp, _, _ = in(exp, f.Value)
		return exp, nil
	case raw:
		return f.Name, nil
	default:
		return exp, ErrUnknownMethod
	}
}

// Args returns arguments slice depending on filter condition
func (f *Filter) Args() ([]interface{}, error) {

	args := make([]interface{}, 0)

	switch f.Method {
	case EQ, NE, GT, LT, GTE, LTE:
		args = append(args, f.Value)
		return args, nil
	case IS, NOT:
		if IsNullOrEmpty(f.Value) {
			args = append(args, f.Value)
			return args, nil
		}
		return nil, ErrUnknownMethod
	case LIKE, ILIKE, NLIKE, NILIKE:
		value := f.Value.(string)
		if len(value) >= 2 && strings.HasPrefix(value, "*") {
			value = "%" + value[1:]
		}
		if len(value) >= 2 && strings.HasSuffix(value, "*") {
			value = value[:len(value)-1] + "%"
		}
		args = append(args, value)
		return args, nil
	case IN, NIN:
		_, params, _ := in("?", f.Value)
		args = append(args, params...)
		return args, nil
	case raw:
		return args, nil
	default:
		return nil, ErrUnknownMethod
	}
}

func (f *Filter) setInt(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, GT, LT, GTE, LTE, IN, NIN:
			i, err := strconv.Atoi(list[0])
			if err != nil {
				return ErrBadFormat
			}
			f.Value = i
			return nil
		case IS, NOT:
			if IsNullOrEmpty(strings.ToUpper(list[0])) {
				// We do handle all variants as null
				f.Value = NULL
				return nil
			}
			return ErrBadFormat
		default:
			return ErrMethodNotAllowed
		}
	} else {
		// Handle multiple values (IN/NIN)
		switch f.Method {
		case IN, NIN:
			intSlice := make([]int, len(list))
			for i, v := range list {
				val, err := strconv.Atoi(v)
				if err != nil {
					return ErrBadFormat
				}
				intSlice[i] = val
			}
			f.Value = intSlice
			return nil
		default:
			return ErrMethodNotAllowed
		}
	}
}

func (f *Filter) setBool(list []string) error {
	if len(list) == 1 {
		if f.Method == IS || f.Method == NOT {
			if IsNullOrEmpty(strings.ToUpper(list[0])) {
				// We do handle all variants as null
				f.Value = NULL
				return nil
			}
			return ErrBadFormat
		}
		if f.Method != EQ {
			return ErrMethodNotAllowed
		}

		i, err := strconv.ParseBool(list[0])
		if err != nil {
			return ErrBadFormat
		}
		f.Value = i
	} else {
		return ErrMethodNotAllowed
	}
	return nil
}

func (f *Filter) setDate(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, GT, LT, GTE, LTE, IN, NIN:
			date, err := ParseDate(list[0])
			if err != nil {
				return err
			}
			f.Value = date.Format(DateLayout)
			return nil
		case IS, NOT:
			v := strings.ToUpper(list[0])
			if IsNullOrEmpty(v) {
				f.Value = v
				return nil
			}
			return ErrBadFormat
		default:
			return ErrMethodNotAllowed
		}
	}

	if f.Method != IN && f.Method != NIN {
		return ErrMethodNotAllowed
	}

	dates := make([]string, len(list))
	for i, v := range list {
		date, err := ParseDate(v)
		if err != nil {
			return err
		}
		dates[i] = date.Format(DateLayout)
	}
	f.Value = dates

	return nil
}

func (f *Filter) setString(list []string) error {
	if len(list) == 1 {
		switch f.Method {
		case EQ, NE, GT, LT, GTE, LTE, LIKE, ILIKE, NLIKE, NILIKE, IN, NIN:
			f.Value = list[0]
			return nil
		case IS, NOT:
			v := strings.ToUpper(list[0])
			if IsNullOrEmpty(v) {
				switch v {
				case NULL:
					f.Value = NULL
				case EMPTY:
					f.Value = EMPTY
				case NULLOREMPTY:
					f.Value = NULLOREMPTY
				default:
					f.Value = NULL
				}
				return nil
			}
		default:
			return ErrMethodNotAllowed
		}
	} else {
		// Handle multiple values (IN/NIN)
		switch f.Method {
		case IN, NIN:
			f.Value = list
			return nil
		default:
			return ErrBadFormat
		}
	}
	return ErrMethodNotAllowed
}
