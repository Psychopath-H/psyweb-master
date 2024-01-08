package session

import (
	"testing"
)

type user struct {
	Name string `psyorm:"PRIMARY KEY"`
	Age  int
}

func TestSession_CreateTable(t *testing.T) {
	s := NewSession().Model(&user{})
	_ = s.DropTable()
	_ = s.CreateTable()
	if !s.HasTable() {
		t.Fatal("Failed to create table user")
	}
}

func TestSession_Model(t *testing.T) {
	s := NewSession().Model(&user{})
	table := s.RefTable()
	s.Model(&Session{})
	if table.Name != "user" || s.RefTable().Name != "Session" {
		t.Fatal("Failed to change model")
	}
}
