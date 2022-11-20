package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func TestCreateUser(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	userRequest := &User{Email: "xyz@invalid.com", UserName: "Aayush012", Name: UserName{"go", "developer"}, Active: true}
	user, err := client.CreateUser(context.Background(), userRequest, false)
	if err != nil {
		t.Errorf("%s", err)
	}
	if user == nil {
		t.Errorf("%+v", user)
	}
	Config["userId"] = fmt.Sprintf("%d", user.ID)
	fmt.Printf("%+v", user)
}

// Test ListUsers
func TestListUsers(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	users, err := client.ListUsers(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
	if users == nil {
		t.Errorf("%s, %+v", err, users)
	}
	for _, user := range users {
		fmt.Printf("%+v", user)
	}

}

// Test Create folder
func TestGetUser(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	user, err := client.GetUser(context.Background(), 1)
	if err != nil {
		t.Errorf("%s", err)
	}
	if user == nil {
		t.Errorf("%+v", user)
	}
	if user.ID != 1 {
		t.Errorf("user id is not correct %+v", user)
	}

	fmt.Printf("%+v", user)

}

func TestUserinfo(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	user, err := client.Userinfo(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
	if user == nil {
		t.Errorf("%+v", user)
	}

	fmt.Printf("%+v", user)
}

func TestDeleteUser(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	userId, err := strconv.Atoi(Config["userId"])
	if err != nil {
		t.Errorf("%s", err)
	}
	err = client.DeleteUser(context.Background(), userId)
	if err != nil {
		t.Errorf("%s", err)
	}

}
