package util

import (
	"strings"
)

var (
	//ProgramPath = "/Users/dangdt/Documents/coding/go-hyrts/go-hyrts/example"
	ProgramPath = "/Users/dangdt/teko/footprint"

	TestPrefix = "Test"
	GoExt      = ".go"
	GoTestExt  = "test.go"

	OldDir = ProgramPath
	NewDir = ""

	TracerCovType = "meth-cov"
)

func MergeMap(target, source map[string]string) {
	for key, value := range source {
		target[key] = value
	}
}

func ShortPath(path string) string {
	return strings.Replace(path, ProgramPath, "", -1)
}
