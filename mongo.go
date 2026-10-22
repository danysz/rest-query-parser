package rqp

import (
	"strings"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// setMongoId parses values as bson.ObjectID (tag ":mongoid")
func (f *Filter) setMongoId(list []string) error {
	ids := make([]bson.ObjectID, len(list))
	for i, s := range list {
		id, err := bson.ObjectIDFromHex(s)
		if err != nil {
			return ErrBadFormat
		}
		ids[i] = id
	}

	switch f.Method {
	case EQ, NE:
		if len(ids) != 1 {
			return ErrMethodNotAllowed
		}
		f.Value = ids[0]
	case IN, NIN:
		f.Value = ids
	default:
		return ErrMethodNotAllowed
	}
	return nil
}

// WhereMongo returns the MongoDB condition for the filter value (without the field name)
func (f *Filter) WhereMongo() (interface{}, error) {
	switch f.Method {
	case EQ:
		return f.Value, nil
	case NE, GT, LT, GTE, LTE:
		return bson.M{"$" + strings.ToLower(string(f.Method)): f.Value}, nil
	case LIKE, ILIKE, NLIKE, NILIKE:
		pattern, ok := f.Value.(string)
		if !ok {
			return nil, ErrMethodNotAllowed
		}
		if strings.HasPrefix(pattern, "*") {
			pattern = pattern[1:]
		} else {
			pattern = "^" + pattern
		}
		if strings.HasSuffix(pattern, "*") {
			pattern = pattern[:len(pattern)-1]
		} else {
			pattern += "$"
		}

		re := bson.Regex{Pattern: pattern}
		if f.Method == ILIKE || f.Method == NILIKE {
			re.Options = "i"
		}
		if f.Method == NLIKE || f.Method == NILIKE {
			return bson.M{"$not": re}, nil
		}
		return bson.M{"$regex": re}, nil
	case IS, NOT:
		is := f.Method == IS
		switch f.Value {
		case NULL: // matches both null and missing fields, like SQL IS NULL
			if is {
				return nil, nil
			}
			return bson.M{"$ne": nil}, nil
		case EMPTY:
			if is {
				return "", nil
			}
			return bson.M{"$ne": ""}, nil
		case NULLOREMPTY:
			if is {
				return bson.M{"$in": bson.A{nil, ""}}, nil
			}
			return bson.M{"$nin": bson.A{nil, ""}}, nil
		}
		return nil, ErrMethodNotAllowed
	case IN, NIN:
		values, err := f.Args()
		if err != nil {
			return nil, err
		}
		return bson.M{"$" + strings.ToLower(string(f.Method)): bson.A(values)}, nil
	default:
		return nil, ErrUnknownMethod
	}
}

// MongoQueryFilters returns the filters as a MongoDB query document
func (q *Query) MongoQueryFilters() (bson.M, error) {
	if len(q.Filters) == 0 {
		return bson.M{}, nil
	}

	where := []bson.M{}
	orWhere := []bson.M{}

	for _, filter := range q.Filters {
		element, err := filter.WhereMongo()
		if err != nil {
			return nil, errors.Wrap(err, filter.Key)
		}

		if filter.OR == StartOR || filter.OR == InOR || filter.OR == EndOR {
			orWhere = append(orWhere, bson.M{filter.Name: element})
			if filter.OR == EndOR {
				where = append(where, bson.M{"$or": orWhere})
				orWhere = []bson.M{}
			}
		} else {
			where = append(where, bson.M{filter.Name: element})
		}
	}

	return bson.M{"$and": where}, nil
}

// MongoOrder returns the sorts as a MongoDB sort document
func (q *Query) MongoOrder() bson.D {
	s := bson.D{}
	for _, sort := range q.Sorts {
		direction := 1
		if sort.Desc {
			direction = -1
		}
		s = append(s, bson.E{Key: sort.By, Value: direction})
	}
	return s
}

// MongoAddFindOptions sets sort, limit and offset on the find options
func (q *Query) MongoAddFindOptions(o *options.FindOptionsBuilder) {
	if len(q.Sorts) > 0 {
		o.SetSort(q.MongoOrder())
	}
	o.SetLimit(int64(q.Limit))
	o.SetSkip(int64(q.Offset))
}
