package session

import (
	"errors"
	"orm/clause"
	"reflect"
)

// Insert one or more records in database
// u1 := &user{Name: "Tom", Age: 18}
// u2 := &user{Name: "Sam", Age: 25}
func (s *Session) Insert(values ...any) (int64, error) { // s.Insert(u1, u2, ...)
	recordValues := make([]any, 0)
	for _, value := range values { //value -> &user{Name:"Tom", Age:18} , &user{Name:"Sam", Age:25}
		s.CallMethod(BeforeInsert, value)
		table := s.Model(value).RefTable()
		//table.Name -> "user"; table.FieldsNames -> []string{"Name","Age"}
		s.clause.Set(clause.INSERT, table.Name, table.FieldNames) // -> INSERT INTO user (Name,Age)
		//table.RecordValues(&user{Name:"Tom", Age:18}) -> []any{{"Tom", 18}}
		recordValues = append(recordValues, table.RecordValues(value))
	}
	// recordValues -> []any{{"Tom", 18}, {"Sam", 25}}
	s.clause.Set(clause.VALUES, recordValues...) // -> "VALUES (?,?), (?,?)"
	sql, vars := s.clause.Build(clause.INSERT, clause.VALUES)
	//sql -> "INSERT INTO user (Name,Age) VALUES (?,?), (?,?)"
	//vars -> []any{"Tom", 18, "Sam", 25}
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterInsert, nil)
	return result.RowsAffected()
}

// Find gets all eligible records
func (s *Session) Find(values any) error { // s.Fine(&[]user)
	s.CallMethod(BeforeQuery, nil)
	destSlice := reflect.Indirect(reflect.ValueOf(values))                // destSlice -> []user
	destType := destSlice.Type().Elem()                                   // destSilce.Type() -> reflect.Slice; .Type()操作得到类型是反射切片，再.Elem()得到切片中元素的类型
	table := s.Model(reflect.New(destType).Elem().Interface()).RefTable() //reflect.New()将反射类型(reflect.Type)转化为反射值(reflect.Value)的指针，再对其取元素得到reflect.Value，再.interface()得到golang里的数据类型

	s.clause.Set(clause.SELECT, table.Name, table.FieldNames)
	sql, vars := s.clause.Build(clause.SELECT, clause.WHERE, clause.ORDERBY, clause.LIMIT)
	rows, err := s.Raw(sql, vars...).QueryRows()
	if err != nil {
		return err
	}

	for rows.Next() {
		dest := reflect.New(destType).Elem() //reflect.New(destType).Elem() 将反射类型转化为反射值，再对其取元素得到reflect.Value
		var values []any
		for _, name := range table.FieldNames {
			values = append(values, dest.FieldByName(name).Addr().Interface()) //将各个字段的地址抽取出来并转化为golang里的类型
		}
		if err := rows.Scan(values...); err != nil {
			return err
		}
		s.CallMethod(AfterQuery, dest.Addr().Interface())
		destSlice.Set(reflect.Append(destSlice, dest))
	}
	return rows.Close()
}

// First gets the 1st row
func (s *Session) First(value any) error { // s.First(&user{})
	dest := reflect.Indirect(reflect.ValueOf(value))
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem() //为什么这里和上面的Find方法的反射操作不一样呢，因为这里的框架使用者传的未必是切片，所以这里要改造成切片类型才能用Find方法
	if err := s.Limit(1).Find(destSlice.Addr().Interface()); err != nil {
		return err
	}
	if destSlice.Len() == 0 {
		return errors.New("NOT FOUND")
	}
	dest.Set(destSlice.Index(0))
	return nil
}

// Limit adds limit condition to clause
func (s *Session) Limit(num int) *Session { // s.Limit(1)
	s.clause.Set(clause.LIMIT, num)
	return s
}

// Where adds limit condition to clause
func (s *Session) Where(desc string, args ...any) *Session { // s.Where("Name = ?", 25)
	var vars []any
	s.clause.Set(clause.WHERE, append(append(vars, desc), args...)...)
	return s
}

// OrderBy adds order by condition to clause
func (s *Session) OrderBy(desc string) *Session { //s.OrderBy("Age DESC")
	s.clause.Set(clause.ORDERBY, desc)
	return s
}

// Update records with where clause
// support map[string]any
// also support kv list: "Name", "Tom", "Age", 18, ....
func (s *Session) Update(kv ...any) (int64, error) { //s.Update("Age", 30)
	s.CallMethod(BeforeUpdate, nil)
	m, ok := kv[0].(map[string]any)
	if !ok {
		m = make(map[string]any)
		for i := 0; i < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
	}
	s.clause.Set(clause.UPDATE, s.RefTable().Name, m)
	sql, vars := s.clause.Build(clause.UPDATE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterUpdate, nil)
	return result.RowsAffected()
}

// Delete records with where clause
func (s *Session) Delete() (int64, error) { // s.Where("Name = ?", "Tom").Delete()
	s.CallMethod(BeforeDelete, nil)
	s.clause.Set(clause.DELETE, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.DELETE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterDelete, nil)
	return result.RowsAffected()
}

// Count records with where clause
func (s *Session) Count() (int64, error) { // s.Count()
	s.clause.Set(clause.COUNT, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.COUNT, clause.WHERE)
	row := s.Raw(sql, vars...).QueryRow()
	var tmp int64
	if err := row.Scan(&tmp); err != nil {
		return 0, err
	}
	return tmp, nil
}
