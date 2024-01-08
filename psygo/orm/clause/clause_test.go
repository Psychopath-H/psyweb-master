package clause

import (
	"reflect"
	"testing"
)

func TestClause_Set(t *testing.T) {
	var clause Clause
	clause.Set(INSERT, "user", []string{"Name", "Age"})
	sql := clause.sql[INSERT]
	vars := clause.sqlVars[INSERT]
	t.Log(sql, vars)
	if sql != "INSERT INTO user (Name,Age)" || len(vars) != 0 {
		t.Fatal("failed to get clause")
	}
}

func testSelect(t *testing.T) {
	var clause Clause
	clause.Set(LIMIT, 3)
	clause.Set(SELECT, "user", []string{"*"})
	clause.Set(WHERE, "Name = ?", "Tom")
	clause.Set(ORDERBY, "Age ASC")
	sql, vars := clause.Build(SELECT, WHERE, ORDERBY, LIMIT)
	t.Log(sql, vars)
	if sql != "SELECT * FROM user WHERE Name = ? ORDER BY Age ASC LIMIT ?" {
		t.Fatal("failed to build SQL")
	}
	if !reflect.DeepEqual(vars, []any{"Tom", 3}) {
		t.Fatal("failed to build SQLVars")
	}
}

func testUpdate(t *testing.T) {
	var clause Clause
	clause.Set(UPDATE, "user", map[string]any{"Age": 30})
	clause.Set(WHERE, "Name = ?", "Tom")
	sql, vars := clause.Build(UPDATE, WHERE)
	t.Log(sql, vars)
	if sql != "UPDATE user SET Age = ? WHERE Name = ?" {
		t.Fatal("failed to build SQL")
	}
	if !reflect.DeepEqual(vars, []any{30, "Tom"}) {
		t.Fatal("failed to build SQLVars")
	}
}

func testDelete(t *testing.T) {
	var clause Clause
	clause.Set(DELETE, "user")
	clause.Set(WHERE, "Name = ?", "Tom")

	sql, vars := clause.Build(DELETE, WHERE)
	t.Log(sql, vars)
	if sql != "DELETE FROM user WHERE Name = ?" {
		t.Fatal("failed to build SQL")
	}
	if !reflect.DeepEqual(vars, []any{"Tom"}) {
		t.Fatal("failed to build SQLVars")
	}
}

func TestClause_Build(t *testing.T) {
	t.Run("select", func(t *testing.T) {
		testSelect(t)
	})
	t.Run("update", func(t *testing.T) {
		testUpdate(t)
	})
	t.Run("delete", func(t *testing.T) {
		testDelete(t)
	})
}
