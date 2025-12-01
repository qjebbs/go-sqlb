package sqlb_test

import (
	"reflect"
	"testing"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

func TestQueryBuilderDistinctElimination(t *testing.T) {
	var (
		users = sqlb.NewTable("users", "u")
		locs  = sqlb.NewTable("locs", "l")
		foo   = sqlb.NewTable("foo", "f")
		bar   = sqlb.NewTable("bar", "b")
	)
	q := sqlb.NewQueryBuilder().
		Distinct().
		With(sqlb.NewTable("xxx", ""), sqlf.F("SELECT 1 AS whatever")). // should be ignored
		With(locs, sqlf.F("SELECT user_id AS id, loc FROM user_locs WHERE country_code = ?", "cn")).
		With(
			users,
			// CTE references another
			sqlf.F("SELECT * FROM ? INNER JOIN ? ON ?=?",
				users.TableAs(), locs.TableAs(),
				users.Column("id"), locs.Column("id"),
			),
		)
	q.Select(foo.Columns("id", "name")...).
		From(users).
		LeftJoinOptional(foo, sqlf.F(
			"?=?",
			foo.Column("user_id"),
			users.Column("id"),
		)).
		LeftJoinOptional(bar, sqlf.F( // not referenced, should be ignored
			"?=?",
			bar.Column("user_id"),
			users.Column("id"),
		))
	gotQuery, gotArgs, err := q.BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		t.Fatal(err)
	}
	wantQuery := "WITH locs AS (SELECT user_id AS id, loc FROM user_locs WHERE country_code = $1), users AS (SELECT * FROM users AS u INNER JOIN locs AS l ON u.id=l.id) SELECT DISTINCT f.id, f.name FROM users AS u LEFT JOIN foo AS f ON f.user_id=u.id"
	wantArgs := []any{"cn"}
	if wantQuery != gotQuery {
		t.Errorf("got:\n%s\nwant:\n%s", gotQuery, wantQuery)
	}
	if !reflect.DeepEqual(wantArgs, gotArgs) {
		t.Errorf("want:\n%v\ngot:\n%v", wantArgs, gotArgs)
	}
}

func TestQueryBuilderGroupbyElimination(t *testing.T) {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
		baz = sqlb.NewTable("baz", "z")
	)
	q := sqlb.NewQueryBuilder().
		With(
			baz,
			sqlf.F("SELECT * FROM baz WHERE type=$1", "user"),
		)
	q.Select(foo.Columns("id", "bar")...).
		From(foo).
		LeftJoinOptional(baz, sqlf.F(
			"?=?",
			foo.Column("baz_id"),
			baz.Column("id"),
		)).
		LeftJoinOptional(bar, sqlf.F( // not referenced, should be ignored
			"?=?",
			bar.Column("baz_id"),
			baz.Column("id"),
		)).
		Where2(foo.Column("id"), "=", 1).
		GroupBy(foo.Column("id"))
	gotQuery, gotArgs, err := q.BuildQuery(sqlf.BindStyleDollar)
	if err != nil {
		t.Fatal(err)
	}
	wantQuery := "SELECT f.id, f.bar FROM foo AS f WHERE f.id=$1 GROUP BY f.id"
	wantArgs := []any{1}
	if wantQuery != gotQuery {
		t.Errorf("got:\n%s\nwant:\n%s", gotQuery, wantQuery)
	}
	if !reflect.DeepEqual(wantArgs, gotArgs) {
		t.Errorf("want:\n%v\ngot:\n%v", wantArgs, gotArgs)
	}
}
