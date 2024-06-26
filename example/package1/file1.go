package package1

import (
	"strings"

	"github.com/dangdtr/go-hyrts/example/package2"
)

func GetUserInfo(name string, age int) string {
	user := package2.User{
		Name: name,
		Age:  20,
	}
	return package2.UserInfo(user)
}

// JoinStrings joins strings
func JoinStrings(words []string) string {
	return strings.Join(words, ",")
}
