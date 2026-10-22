package rqp

import (
	"math"
	"time"

	"github.com/pkg/errors"
)

// ValidationFunc represents validator for Filters
type ValidationFunc func(value interface{}) error

// Validations type replacement for map.
// Used in NewParse(), NewQV(), SetValidations()
type Validations map[string]ValidationFunc

// Multi multiple validation func
// usage: Multi(Min(10), Max(100))
func Multi(values ...ValidationFunc) ValidationFunc {
	return func(value interface{}) error {
		for _, v := range values {
			if err := v(value); err != nil {
				return err
			}
		}
		return nil
	}
}

// In validation if values contatin value
func In(values ...interface{}) ValidationFunc {
	return func(value interface{}) error {

		var (
			v  interface{}
			in bool = false
		)

		for _, v = range values {
			if v == value {
				in = true
				break
			}
		}

		if !in {
			return errors.Wrapf(ErrNotInScope, "%v", value)
		}

		return nil
	}
}

// Min validation if value greater or equal then min
func Min(min int) ValidationFunc {
	return func(value interface{}) error {
		if limit, ok := value.(int); ok {
			if limit >= min {
				return nil
			}
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}

// Max validation if value lower or equal then max
func Max(max int) ValidationFunc {
	return func(value interface{}) error {
		if limit, ok := value.(int); ok {
			if limit <= max {
				return nil
			}
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}

// MinMax validation if value between or equal min and max
func MinMax(min, max int) ValidationFunc {
	return func(value interface{}) error {
		if limit, ok := value.(int); ok {
			if min <= limit && limit <= max {
				return nil
			}
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}

// NotEmpty validation if string value length more then 0
func NotEmpty() ValidationFunc {
	return func(value interface{}) error {
		if s, ok := value.(string); ok {
			if len(s) > 0 {
				return nil
			}
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}

// MinFloat validation if value greater or equal then min
func MinFloat(min float32) ValidationFunc {
	return MinMaxFloat(min, float32(math.MaxFloat32))
}

// MaxFloat validation if value lower or equal then max
func MaxFloat(max float32) ValidationFunc {
	return MinMaxFloat(-float32(math.MaxFloat32), max)
}

// MinMaxFloat validation if value between or equal min and max
func MinMaxFloat(min, max float32) ValidationFunc {
	return func(value interface{}) error {
		if v, ok := value.(float32); ok && min <= v && v <= max {
			return nil
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}

// MinTime validation if value is not before min
func MinTime(min time.Time) ValidationFunc {
	return func(value interface{}) error {
		if v, ok := value.(time.Time); ok && !v.Before(min) {
			return nil
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}

// MaxTime validation if value is not after max
func MaxTime(max time.Time) ValidationFunc {
	return func(value interface{}) error {
		if v, ok := value.(time.Time); ok && !v.After(max) {
			return nil
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}

// MinMaxTime validation if value between or equal min and max
func MinMaxTime(min, max time.Time) ValidationFunc {
	return func(value interface{}) error {
		if v, ok := value.(time.Time); ok && !v.Before(min) && !v.After(max) {
			return nil
		}
		return errors.Wrapf(ErrNotInScope, "%v", value)
	}
}
