package orm

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/url"
	"orm/session"
	"reflect"
	"testing"
)

func OpenDB(t *testing.T) *Engine {
	t.Helper()
	dataSourceName := fmt.Sprintf("root:root@tcp(localhost:3306)/psygo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	engine, err := NewEngine("mysql", dataSourceName)
	if err != nil {
		t.Fatal("failed to connect", err)
	}
	return engine
}

func TestNewEngine(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
}

type user struct {
	Name string `psyorm:"PRIMARY KEY"`
	Age  int
}

func transactionRollback(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
	s := engine.NewSession()
	_ = s.Model(&user{}).DropTable()
	_ = s.Model(&user{}).CreateTable()
	_ = s.Begin()
	{
		_, _ = s.Insert(&user{"Tom", 18})
		_, _ = s.Insert(&user{"Leo", 24})
	}
	_ = s.Rollback()
	if num, _ := s.Count(); num != 0 {
		t.Fatal("failed to rollback")
	}
}

func transactionCommit(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
	s := engine.NewSession()
	_ = s.Model(&user{}).DropTable()
	_, err := engine.Transaction(func(s *session.Session) (result any, err error) {
		_ = s.Model(&user{}).CreateTable()
		_, err = s.Insert(&user{"Tom", 18})
		return
	})
	u := &user{}
	_ = s.First(u)
	if err != nil || u.Name != "Tom" {
		t.Fatal("failed to commit")
	}
}

func TestEngine_Transaction(t *testing.T) {
	t.Run("rollback", func(t *testing.T) {
		transactionRollback(t)
	})
	t.Run("commit", func(t *testing.T) {
		transactionCommit(t)
	})
}

func TestEngine_Migrate(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
	s := engine.NewSession()
	_, _ = s.Raw("DROP TABLE IF EXISTS user;").Exec()
	_, _ = s.Raw("CREATE TABLE user(Name VARCHAR(255) PRIMARY KEY, Id integer);").Exec()
	_, _ = s.Raw("INSERT INTO user(`Name`) values (?), (?)", "Tom", "Sam").Exec()
	engine.Migrate(&user{})

	rows, _ := s.Raw("SELECT * FROM user").QueryRows()
	columns, _ := rows.Columns()
	if !reflect.DeepEqual(columns, []string{"Name", "Age"}) {
		t.Fatal("Failed to migrate table User, got columns", columns)
	}
}
