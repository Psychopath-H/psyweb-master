package clause

import (
	"fmt"
	"strings"
)

type generator func(values ...any) (string, []any)

var generators map[Type]generator

func init() {
	generators = make(map[Type]generator)
	generators[INSERT] = _insert
	generators[VALUES] = _values
	generators[SELECT] = _select
	generators[LIMIT] = _limit
	generators[WHERE] = _where
	generators[ORDERBY] = _orderBy
	generators[UPDATE] = _update
	generators[DELETE] = _delete
	generators[COUNT] = _count
}

func genBindVars(num int) string {
	var vars []string
	for i := 0; i < num; i++ {
		vars = append(vars, "?")
	}
	return strings.Join(vars, ", ")
}

func _insert(values ...any) (string, []any) { // _insert("user", []string{"Name", "Age"})
	// INSERT INTO $tableName ($fields)
	tableName := values[0]                            // -> "user"
	fields := strings.Join(values[1].([]string), ",") // -> []string{"Name,Age"}
	// "INSERT INTO user (Name,Age)"
	return fmt.Sprintf("INSERT INTO %s (%v)", tableName, fields), []any{} // INSERT INTO user (Name,Age)
}

func _values(values ...any) (string, []any) { // _values([]any{{"Tom", 18}, {"Sam", 25}})
	// VALUES ($v1), ($v2), ...
	var bindStr string
	var sql strings.Builder
	var vars []any
	sql.WriteString("VALUES ")
	for i, value := range values { //value -> {"Tom", 18}
		v := value.([]any)
		if bindStr == "" {
			bindStr = genBindVars(len(v)) // bindStr -> "?,?"
		}
		sql.WriteString(fmt.Sprintf("(%v)", bindStr)) //sql -> "VALUES (?,?)"
		if i+1 != len(values) {
			sql.WriteString(", ") // sql -> "VALUES (?,?), (?,?)"
		}
		vars = append(vars, v...) // vars -> []any{"Tom", 18, "Sam", 25}
	}
	return sql.String(), vars // VALUES (?,?), (?,?)"  vars -> []any{"Tom", 18, "Sam", 25}

}

func _select(values ...any) (string, []any) { //_select("user", []string{"Name", "Age"})
	// SELECT $fields FROM $tableName
	tableName := values[0]
	fields := strings.Join(values[1].([]string), ",")
	return fmt.Sprintf("SELECT %v FROM %s", fields, tableName), []any{} // SELECT Name,Age FROM user
}

func _limit(values ...any) (string, []any) { // _limit(1)
	// LIMIT $num
	return "LIMIT ?", values // LIMIT ?; values -> 1
}

func _where(values ...any) (string, []any) { // _where("Age = ?", 25")
	// WHERE $desc
	desc, vars := values[0], values[1:]
	return fmt.Sprintf("WHERE %s", desc), vars // where Age = ?; vars -> [25]
}

func _orderBy(values ...any) (string, []any) { // _orderBy("Age ASC")
	return fmt.Sprintf("ORDER BY %s", values[0]), []any{} // ORDER BY Age ASC
}

func _update(values ...any) (string, []any) { //_update("user", map[string]any{"Age": 30})
	tableName := values[0]          // tableName -> "user"
	m := values[1].(map[string]any) // m -> map[string]any{"Age": 30}
	var keys []string
	var vars []any
	for k, v := range m {
		keys = append(keys, k+" = ?")
		vars = append(vars, v)
	}
	// keys -> []string{"Age = ?"}
	// vars -> []any{30}
	return fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(keys, ", ")), vars // UPDATE user SET Age = ?; vars -> [30]
}

func _delete(values ...any) (string, []any) { // _delete(s.RefTable().Name)
	return fmt.Sprintf("DELETE FROM %s", values[0]), []any{} // DELETE FROM user
}

func _count(values ...any) (string, []any) { // _count(s.RefTable().Name)
	return _select(values[0], []string{"count(*)"}) // select count(*)
}
