package util

var (
	ProgramPath = "/Users/dangdt/Documents/coding/go-hyrts/go-hyrts/example"

	TestPrefix = "Test"
	GoExt      = ".go"
)

func MergeMap(target, source map[string]string) {
	for key, value := range source {
		target[key] = value
	}
}
