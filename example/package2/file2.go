package package2

import (
	"fmt"
)

type User struct {
	Name string
	Age  int
}

type UserInterface interface {
	UserInfo(user User) string
}

// UserInfo says Hello
func UserInfo(user User) string {
	return fmt.Sprintf("Name: %s, Age: %d", user.Name, user.Age)
}
