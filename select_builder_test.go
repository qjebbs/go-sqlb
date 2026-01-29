package sqlb_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

func TestSelectBuilderDistinctElimination(t *testing.T) {
	var (
		users = sqlb.NewTable("users", "u")
		locs  = sqlb.NewTable("locs", "l")
		foo   = sqlb.NewTable("foo", "f")
		bar   = sqlb.NewTable("bar", "b")
	)
	q := sqlb.NewSelectBuilder().
		EnableElimination().
		With(sqlb.NewTable("xxx", ""), sqlf.F("SELECT 1 AS whatever")). // should be ignored
		With(locs, sqlf.F("SELECT user_id AS id, loc FROM user_locs WHERE country_code = ?", "cn")).
		With(
			users,
			// CTE references another
			sqlf.F("SELECT * FROM ? INNER JOIN ? ON ?=?",
				users.TableAs(), locs.TableAs(),
				users.Column("id"), locs.Column("id"),
			),
		).
		Distinct().Select(foo.Columns("id", "name")...).
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
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	gotQuery, gotArgs, err := q.Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantQuery := `WITH "locs" AS (SELECT user_id AS id, loc FROM user_locs WHERE country_code = $1), "users" AS (SELECT * FROM "users" AS "u" INNER JOIN "locs" AS "l" ON "u"."id"="l"."id") SELECT DISTINCT "f"."id", "f"."name" FROM "users" AS "u" LEFT JOIN "foo" AS "f" ON "f"."user_id"="u"."id"`
	wantArgs := []any{"cn"}
	if wantQuery != gotQuery {
		t.Errorf("got:\n%s\nwant:\n%s", gotQuery, wantQuery)
	}
	if !reflect.DeepEqual(wantArgs, gotArgs) {
		t.Errorf("want:\n%v\ngot:\n%v", wantArgs, gotArgs)
	}
}

func TestSelectBuilderGroupbyElimination(t *testing.T) {
	var (
		foo = sqlb.NewTable("foo", "f")
		bar = sqlb.NewTable("bar", "b")
		baz = sqlb.NewTable("baz", "z")
	)
	q := sqlb.NewSelectBuilder().
		EnableElimination().
		With(
			baz,
			sqlf.F("SELECT * FROM baz WHERE type=$1", "user"),
		).
		Select(foo.Columns("id", "bar")...).
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
		WhereEquals(foo.Column("id"), 1).
		GroupBy(foo.Column("id"))
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	gotQuery, gotArgs, err := q.Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantQuery := `SELECT "f"."id", "f"."bar" FROM "foo" AS "f" WHERE "f"."id" = $1 GROUP BY "f"."id"`
	wantArgs := []any{1}
	if wantQuery != gotQuery {
		t.Errorf("got:\n%s\nwant:\n%s", gotQuery, wantQuery)
	}
	if !reflect.DeepEqual(wantArgs, gotArgs) {
		t.Errorf("want:\n%v\ngot:\n%v", wantArgs, gotArgs)
	}
}

func TestSelectBuilderComplexDeps(t *testing.T) {
	var (
		baseTable  = sqlb.NewTable("base_table", "b")
		baseTable2 = sqlb.NewTable("base_table2", "b2")
		baseTable3 = sqlb.NewTable("base_table3", "b3")
		foo        = sqlb.NewTable("foo", "f")
		bar        = sqlb.NewTable("bar", "r")
		baz        = sqlb.NewTable("baz", "z")
	)
	q := sqlb.NewSelectBuilder().
		EnableElimination().
		With(
			foo,
			sqlf.F(
				"SELECT * FROM ? WHERE ? = 1",
				baseTable2.TableAs(),
				baseTable2.Column("id"),
			),
		).
		With(
			bar,
			sqlf.F(
				"SELECT * FROM ? WHERE ? = ?",
				foo.TableAs(),
				foo.Column("active"),
				true,
			),
		).
		With(
			baz,
			sqlf.F("SELECT 1"), // not referenced, should be eliminated
		).
		Distinct().Select(baseTable.AllColumns()).
		From(baseTable).
		LeftJoinOptional(baseTable3, sqlf.F( // required by outer table of subquery
			"? = ?",
			baseTable3.Column("b_id"),
			baseTable.Column("id"),
		)).
		LeftJoinOptional(baz, sqlf.F( // not referenced, should be eliminated
			"? = ?",
			baz.Column("b_id"),
			baseTable.Column("id"),
		)).
		Where(sqlf.F(
			"EXISTS (?)",
			sqlb.NewSelectBuilder().
				Select(sqlf.F("1")).
				From(bar).
				Where(sqlf.F(
					"? = ? AND ? LIKE ?",
					bar.Column("base_id"),
					baseTable3.Column("id"), // reference outer table
					bar.Column("name"),
					"%something%",
				)),
		))
	ctx := sqlb.NewContext(context.Background(), dialect.PostgreSQL{})
	query, args, err := q.Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	query, _ = util.Interpolate(query, args, ctx.BaseDialect())
	wantQuery := `WITH "foo" AS (SELECT * FROM "base_table2" AS "b2" WHERE "b2"."id" = 1), "bar" AS (SELECT * FROM "foo" AS "f" WHERE "f"."active" = TRUE) SELECT DISTINCT "b".* FROM "base_table" AS "b" LEFT JOIN "base_table3" AS "b3" ON "b3"."b_id" = "b"."id" WHERE EXISTS (SELECT 1 FROM "bar" AS "r" WHERE "r"."base_id" = "b3"."id" AND "r"."name" LIKE '%something%')`
	if query != wantQuery {
		t.Errorf("got:\n%s\nwant:\n%s", query, wantQuery)
	}
}
