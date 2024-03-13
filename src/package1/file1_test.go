// package1_test.go
package package1

import (
	"testing"
)

func TestGetUserInfo(t *testing.T) {
	result := GetUserInfo("Dang", 20)

	expectedResult := "Name: Dang, Age: 20"
	if result != expectedResult {
		t.Errorf("GetUserInfo result is incorrect, got: %s, want: %s", result, expectedResult)
	}
}

func TestJoinStrings(t *testing.T) {
	words := []string{"Hello", "World"}
	result := JoinStrings(words)

	expectedResult := "Hello, World"
	if result != expectedResult {
		t.Errorf("JoinStrings result is incorrect, got: %s, want: %s", result, expectedResult)
	}
}
