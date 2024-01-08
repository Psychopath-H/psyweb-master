package service

import (
	"fmt"
	_ "github.com/Psychopath-H/psyweb-master/psygo/orm"
	_ "github.com/go-sql-driver/mysql"
	"net/url"
	"orm"
	"orm/log"
	"orm/session"
	"reflect"
	"testing"
)

type user struct {
	Name string `psyorm:"PRIMARY KEY"`
	Age  int
}

var (
	user1 = &user{"Tom", 18}
	user2 = &user{"Sam", 25}
	user3 = &user{"Jack", 25}
)

func OpenDB(t *testing.T) *orm.Engine {
	t.Helper()
	dataSourceName := fmt.Sprintf("root:root@tcp(localhost:3306)/psygo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	engine, err := orm.NewEngine("mysql", dataSourceName)
	if err != nil {
		t.Fatal("failed to connect", err)
	}
	return engine
}

func TestNewEngine(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
}

func TestSession_CreateTable(t *testing.T) {
	engine := OpenDB(t)
	s := engine.NewSession().Model(&user{})
	_ = s.DropTable()
	_ = s.CreateTable()
	if !s.HasTable() {
		t.Fatal("Failed to create table user")
	}
}

func TestSession_Model(t *testing.T) {
	engine := OpenDB(t)
	s := engine.NewSession().Model(&user{})
	table := s.RefTable()
	s.Model(&session.Session{})
	if table.Name != "user" || s.RefTable().Name != "Session" {
		t.Fatal("Failed to change model")
	}
}

func testRecordInit(t *testing.T) *session.Session {
	t.Helper()
	engine := OpenDB(t)
	s := engine.NewSession().Model(&user{})
	err1 := s.DropTable()
	err2 := s.CreateTable()
	_, err3 := s.Insert(user1, user2)
	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("failed init test records")
	}
	return s
}

func TestSession_Insert(t *testing.T) {
	s := testRecordInit(t)
	affected, err := s.Insert(user3)
	if err != nil || affected != 1 {
		t.Fatal("failed to create record")
	}
}

func TestSession_Find(t *testing.T) {
	s := testRecordInit(t)
	var users []user
	if err := s.Find(&users); err != nil || len(users) != 2 {
		t.Fatal("failed to query all")
	}
	fmt.Println(users)
}

func TestSession_First(t *testing.T) {
	s := testRecordInit(t)
	u := &user{}
	err := s.Where("Age = ?", 18).First(u)
	if err != nil || u.Name != "Tom" || u.Age != 18 {
		t.Fatal("failed to query first")
	}
}

func TestSession_Limit(t *testing.T) {
	s := testRecordInit(t)
	var users []user
	err := s.Limit(1).Find(&users)
	if err != nil || len(users) != 1 {
		t.Fatal("failed to query with limit condition")
	}
}

func TestSession_Where(t *testing.T) {
	s := testRecordInit(t)
	var users []user
	_, err1 := s.Insert(user3)
	err2 := s.Where("Age = ?", 25).Find(&users)

	if err1 != nil || err2 != nil || len(users) != 2 {
		t.Fatal("failed to query with where condition")
	}
}

func TestSession_OrderBy(t *testing.T) {
	s := testRecordInit(t)
	u := &user{}
	err := s.OrderBy("Age DESC").First(u)

	if err != nil || u.Age != 25 {
		t.Fatal("failed to query with order by condition")
	}
}

func TestSession_Update(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Update("Age", 30)
	u := &user{}
	_ = s.OrderBy("Age DESC").First(u)

	if affected != 1 || u.Age != 30 {
		t.Fatal("failed to update")
	}
}

func TestSession_DeleteAndCount(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Delete()
	count, _ := s.Count()

	if affected != 1 || count != 1 {
		t.Fatal("failed to delete or count")
	}
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

type Account struct {
	ID       int `psyorm:"PRIMARY KEY"`
	Password string
}

func (account *Account) BeforeInsert(s *session.Session) error {
	log.Info("before inert", account)
	account.ID += 1000
	return nil
}

func (account *Account) AfterQuery(s *session.Session) error {
	log.Info("after query", account)
	account.Password = "******"
	return nil
}

func TestSession_CallMethod(t *testing.T) {
	engine := OpenDB(t)
	defer engine.Close()
	s := engine.NewSession().Model(&Account{})
	_ = s.DropTable()
	_ = s.CreateTable()
	_, _ = s.Insert(&Account{1, "123456"}, &Account{2, "qwerty"})

	u := &Account{}

	err := s.First(u)
	if err != nil || u.ID != 1001 || u.Password != "******" {
		t.Fatal("Failed to call hooks after query, got", u)
	}
}
