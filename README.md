# Query Parser for REST

<a href="https://awesome-go.com/"><img align="right" src="https://github.com/avelino/awesome-go/raw/main/tmpl/assets/logo.png" alt="awesome-go" title="awesome-go" /></a>

[![GoDoc](https://godoc.org/github.com/qrs-source/rest-query-parser?status.png)](https://godoc.org/github.com/qrs-source/rest-query-parser)
[![Coverage Status](https://coveralls.io/repos/github/timsolov/rest-query-parser/badge.svg?branch=master)](https://coveralls.io/github/timsolov/rest-query-parser?branch=master)
[![Awesome](https://cdn.rawgit.com/sindresorhus/awesome/d7305f38d29fed78fa85652e3a63e154dd8e8829/media/badge.svg)](https://github.com/sindresorhus/awesome)

Query Parser is a library for easy building dynamic SQL queries to Database. It provides a simple API for web-applications which needs to do some filtering throught GET queries. It is a connector between the HTTP handler and the DB engine, and manages validations and translations for user inputs.



## Installation
    go get -u github.com/qrs-source/rest-query-parser

## Idea

The idia to write this library comes to me after reading this article: 
[REST API Design: Filtering, Sorting, and Pagination](https://www.moesif.com/blog/technical/api-design/REST-API-Design-Filtering-Sorting-and-Pagination/).

And principles enumerated in article I considered very useful and practical to use in our project with amount of listings with different filtering.

## Fast start
See cmd/main.go and tests for more examples.

```go
    package main
    
    import (
        "errors"
        "fmt"
        "net/url"
    
        rqp "github.com/qrs-source/rest-query-parser"
    )
    
    func main() {
        url, _ := url.Parse("http://localhost/?sort=name,-id&limit=10&id=1&i[eq]=5&s[eq]=one&email[like]=*tim*|name[like]=*tim*")
        q, _ := rqp.NewParse(url.Query(), rqp.Validations{
            "limit:required": rqp.MinMax(10, 100),  // limit must present in the Query part and must be between 10 and 100 (default: Min(1))
            "sort":           rqp.In("id", "name"), // sort could be or not in the query but if it is present it must be equal to "in" or "name"
            "s":      rqp.In("one", "two"), // filter: s - string and equal
            "id:int": nil,                  // filter: id is integer without additional validation
            "i:int": func(value interface{}) error { // filter: custom func for validating
                if value.(int) > 1 && value.(int) < 10 {
                    return nil
                }
                return errors.New("i: must be greater then 1 and lower then 10")
            },
            "email": nil,
            "name":  nil,
        })

        fmt.Println(q.SQL("table")) // SELECT * FROM table WHERE id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?) ORDER BY name, id DESC LIMIT 10
        fmt.Println(q.Where())      // id = ? AND i = ? AND s = ? AND (email LIKE ? OR name LIKE ?)
        fmt.Println(q.Args())       // [1 5 one %tim% %tim%]

        q.AddValidation("fields", rqp.In("id", "name"))
        q.SetUrlString("http://localhost/?fields=id,name&limit=10")
        q.Parse()

        fmt.Println(q.SQL("table")) // SELECT id, name FROM table ORDER BY id LIMIT 10
        fmt.Println(q.Select())     // id, name
        fmt.Println(q.Args())       // []
    }
```

## Top level fields:
* `fields` - fields for SELECT clause separated by comma (",") Eg. `&fields=id,name`. If nothing provided will use "\*" by default. Attention! If you want to use this filter you have to define validation func for it. Use `rqp.In("id", "name")` func for limit fields for your query.
* `sort` - sorting fields list separated by comma (","). Must be validated too. Could include prefix +/- which means ASC/DESC sorting. Eg. `&sort=+id,-name` will print `ORDER BY id, name  DESC`. You have to filter fields in this parameter by adding `rqp.In("id", "name")`.
* `limit` - is limit for LIMIT clause. Should be greater then 0 by default. Definition of the validation for `limit` is not required. But you may use `rqp.Max(100)` to limit top threshold.
* `offset` - is offset for OFFSET clause. Should be greater then or equal to 0 by default. Definition of the validation for `offset` is not required.

## Validation modificators:
* `:required` - parameter is required. Must present in the query string. Raise error if not.
* `:int` - parameter must be convertable to int type. Raise error if not.
* `:bool` - parameter must be convertable to bool type. Raise error if not.
* `:date` - parameter must be a date. Absolute (`2006-01-02`) and relative (`-90d`) values are accepted. Raise error if not.

## Supported types
- `string` - the default type for all provided filters if not specified another. Could be compared by `eq, ne, gt, lt, gte, lte, like, ilike, nlike, nilike, in, nin, is, not` methods (`nlike, nilike` means `NOT LIKE, NOT ILIKE` respectively, `in, nin` means `IN, NOT IN` respectively, `is, not` for comparison to NULL `IS NULL, IS NOT NULL`).
- `int` - integer type. Must be specified with tag ":int". Could be compared by `eq, ne, gt, lt, gte, lte, in, nin, is, not` methods.
- `float` - float32 type. Must be specified with tag ":float". Could be compared by `eq, ne, gt, lt, gte, lte, in, nin` methods (avoid `eq, ne, in, nin` because of float precision). Validators: `MinFloat, MaxFloat, MinMaxFloat`.
- `timestamp` - `time.Time` type in RFC3339 format. Must be specified with tag ":timestamp". Could be compared by `eq, ne, gt, lt, gte, lte, in, nin` methods. Validators: `MinTime, MaxTime, MinMaxTime`.
- `mongoid` - MongoDB ObjectID. Must be specified with tag ":mongoid"; the filter name becomes `_id`. Could be compared by `eq, ne, in, nin` methods.
- `intarray`, `stringarray`, `floatarray` - PostgreSQL array columns. Must be specified with the tags ":intarray", ":stringarray", ":floatarray". `eq` matches columns holding exactly the given comma separated values (`ids=1,2` -> `(ids @> ARRAY[?,?]::int[] AND ids <@ ARRAY[?,?]::int[])`), `ne` negates it, `is, not` compare to NULL.
- `bool` - boolean type. Must be specified with tag ":bool". Could be compared by `eq` method.
- `date` - date type. Must be specified with tag ":date". Could be compared by `eq, ne, gt, lt, gte, lte, in, nin, is, not` methods. See "Date usage".

## MongoDB
Besides SQL the parsed query can be used with MongoDB: `q.MongoQueryFilters()` returns the `bson.M` filter, `q.MongoOrder()` the sort and `q.MongoAddFindOptions(&opts)` sets sort, limit and offset on `options.FindOptions`. `NewParseReplaced(values, validations, rqp.Replacer{...})` parses and renames fields in one call.

## Date usage
Filters tagged with `:date` accept an absolute date in the layout `2006-01-02` and relative
values which are resolved while parsing. Every value ends up as an absolute `2006-01-02`
string in `Filter.Value`, so the SQL side never sees a relative value.

```go
    import (
        "fmt"
        "net/url"
    )

    func main() {
        url, _ := url.Parse("http://localhost/?created_at[gte]=-90d")
        q, _ := rqp.NewParse(url.Query(), rqp.Validations{
            "created_at:date": nil,
        })

        fmt.Println(q.SQL("table")) // SELECT * FROM table WHERE created_at >= ?
        fmt.Println(q.Args())       // [2026-05-21]  (90 days before 2026-08-19)
    }
```

### Relative values

| Value | Meaning |
| --- | --- |
| `today`, `yesterday`, `tomorrow` | the respective day |
| `-90d`, `-90day`, `-90days` | 90 days ago |
| `-2w`, `-2week`, `-2weeks` | 2 weeks ago |
| `-6m`, `-6month`, `-6months` | 6 months ago |
| `-1y`, `-1year`, `-1years` | 1 year ago |
| `30d`, `+30d` | 30 days ahead |

Values are case insensitive. A leading `-` points into the past, a leading `+` or no sign at
all points into the future — `+` decodes to a space in URLs and is trimmed, so `+30d` reaches
the parser as `30d`.

Relative values are resolved against the start of the current day in the location of
`rqp.NowFunc`, which defaults to `time.Now` and can be replaced in tests:

```go
    rqp.NowFunc = func() time.Time {
        return time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
    }
```

Month and year arithmetic uses Go's `time.AddDate` normalization: one month before the 31st of
March is the 3rd of March.

`is` and `not` accept `null`, `empty` and `nullorempty` like string filters do.
