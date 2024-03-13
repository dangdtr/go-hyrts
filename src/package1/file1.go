package package1

import (
	"strings"

	"proposal2/src/package2"
)

func GetUserInfo(name string, age int) string {
	user := package2.User{
		Name: name,
		Age:  12,
	}
	return package2.UserInfo(user)
}

// JoinStrings joins strings
func JoinStrings(words []string) string {
	return strings.Join(words, ",")
}
