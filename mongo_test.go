package rqp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMongoQueryFilters(t *testing.T) {
	id := primitive.NewObjectID()
	v := Validations{"id:mongoid": nil, "i:int": nil, "s": nil, "limit": nil, "sort": In("i")}

	cases := []struct {
		url      string
		expected bson.M
		err      string
	}{
		{"", bson.M{}, ""},
		{"?id=" + id.Hex(), bson.M{"$and": []bson.M{{"_id": id}}}, ""},
		{"?id[in]=" + id.Hex() + "," + id.Hex(), bson.M{"$and": []bson.M{{"_id": bson.M{"$in": bson.A{id, id}}}}}, ""},
		{"?id=zzz", nil, "id: bad format"},
		{"?id[gt]=" + id.Hex(), nil, "id[gt]: method are not allowed"},
		{"?i[gte]=5", bson.M{"$and": []bson.M{{"i": bson.M{"$gte": 5}}}}, ""},
		{"?s[is]=NULL", bson.M{"$and": []bson.M{{"s": nil}}}, ""},
		{"?s[not]=NULL", bson.M{"$and": []bson.M{{"s": bson.M{"$ne": nil}}}}, ""},
		{"?s[like]=*a*", bson.M{"$and": []bson.M{{"s": bson.M{"$regex": primitive.Regex{Pattern: "a"}}}}}, ""},
		{"?s[nilike]=a", bson.M{"$and": []bson.M{{"s": bson.M{"$not": primitive.Regex{Pattern: "^a$", Options: "i"}}}}}, ""},
		{"?i=1|i=2", bson.M{"$and": []bson.M{{"$or": []bson.M{{"i": 1}, {"i": 2}}}}}, ""},
	}
	for _, c := range cases {
		u, _ := url.Parse("http://localhost/" + c.url)
		q, err := NewParse(u.Query(), v)
		if c.err != "" {
			assert.EqualError(t, err, c.err, c.url)
			continue
		}
		assert.NoError(t, err, c.url)
		got, err := q.MongoQueryFilters()
		assert.NoError(t, err, c.url)
		assert.Equal(t, c.expected, got, c.url)
	}
}

func TestMongoOrder(t *testing.T) {
	u, _ := url.Parse("http://localhost/?sort=i,-s")
	q, err := NewParse(u.Query(), Validations{"sort": In("i", "s")})
	assert.NoError(t, err)
	assert.Equal(t, bson.D{{Key: "i", Value: 1}, {Key: "s", Value: -1}}, q.MongoOrder())
}
