package package2

import (
	"testing"

	"github.com/dangdtr/go-hyrts/example/package1"
)

func TestFile2(t *testing.T) {
	result := package1.GetUserInfo("Dang", 20)

	expectedResult := "Name: Dang, Age: 20"
	if result != expectedResult {
		t.Errorf("GetUserInfo result is incorrect, got: %s, want: %s", result, expectedResult)
	}
}
