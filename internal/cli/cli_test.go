package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var fixture = map[string]string{
	"go.mod": "module example.com/demo\n\ngo 1.22\n",
	"calc/calc.go": `package calc

// Scale multiplies results.
const Scale = 1

type Op int

func Add(a, b int) int { return (a + b) * Scale }

func Mul(a, b int) int { return a * b * Scale }

func (o Op) Apply(a, b int) int {
	if o == 0 {
		return Add(a, b)
	}
	return Mul(a, b)
}
`,
	"calc/calc_test.go": `package calc

import (
	"os"
	"testing"
)

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("add")
	}
}

func TestMul(t *testing.T) {
	if Mul(2, 3) != 6 {
		t.Fatal("mul")
	}
}

func TestData(t *testing.T) {
	if _, err := os.ReadFile("testdata/x.txt"); err != nil {
		t.Fatal(err)
	}
}
`,
	"calc/testdata/x.txt": "hello\n",
	"app/app.go": `package app

import "example.com/demo/calc"

func Sum(xs ...int) int {
	s := 0
	for _, x := range xs {
		s = calc.Add(s, x)
	}
	return s
}
`,
	"app/app_test.go": `package app

import "testing"

func TestSum(t *testing.T) {
	if Sum(1, 2, 3) != 6 {
		t.Fatal("sum")
	}
}
`,
	"other/other.go":      "package other\n\nfunc Hi() string { return \"hi\" }\n",
	"other/other_test.go": "package other\n\nimport \"testing\"\n\nfunc TestHi(t *testing.T) {\n\tif Hi() != \"hi\" {\n\t\tt.Fatal()\n\t}\n}\n",
	"docs/readme.md":      "# demo\n",
	"lint/lint.go":        "package lint\n",
	"lint/lint_test.go": `package lint

import (
	"os"
	"strings"
	"testing"
)

// TestNoPanic scans source code, like a custom lint rule.
func TestNoPanic(t *testing.T) {
	src, err := os.ReadFile("../app/app.go")
	if err != nil || strings.Contains(string(src), "panic(") {
		t.Fatal("panic in app")
	}
}
`,
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, c := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func selected(t *testing.T, root string) (all bool, ids []string) {
	t.Helper()
	var out, errb bytes.Buffer
	if code := Main([]string{"select", "-C", root, "-json"}, &out, &errb); code != 0 {
		t.Fatalf("select exit %d: %s", code, errb.String())
	}
	var r struct {
		All      bool
		Selected []struct{ ID string }
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	for _, s := range r.Selected {
		ids = append(ids, strings.TrimPrefix(s.ID, "example.com/demo/"))
	}
	sort.Strings(ids)
	return r.All, ids
}

func TestEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a module")
	}
	root := t.TempDir()
	writeFiles(t, root, fixture)
	var out, errb bytes.Buffer
	if code := Main([]string{"collect", "-C", root, "-j", "2"}, &out, &errb); code != 0 {
		t.Fatalf("collect exit %d: %s%s", code, out.String(), errb.String())
	}

	cases := []struct {
		name  string
		files map[string]string
		all   bool
		want  string
	}{
		{name: "no change"},
		{name: "comment and layout only", files: map[string]string{"calc/calc.go": strings.Replace(
			strings.Replace(fixture["calc/calc.go"], "// Scale multiplies results.", "// Scale scales.", 1),
			"func Mul(a, b int) int { return a * b * Scale }", "func Mul(a, b int) int {\n\treturn a * b * Scale // product\n}", 1)}},
		{name: "function body", files: map[string]string{"calc/calc.go": strings.Replace(fixture["calc/calc.go"], "a * b * Scale", "b * a * Scale", 1)},
			want: "calc.TestMul"},
		{name: "transitive callee", files: map[string]string{"calc/calc.go": strings.Replace(fixture["calc/calc.go"], "(a + b) * Scale", "(b + a) * Scale", 1)},
			want: "app.TestSum calc.TestAdd"},
		{name: "source file scanned by a test", files: map[string]string{"app/app.go": strings.Replace(fixture["app/app.go"], "s := 0", "s := 0 // sum", 1)},
			want: "lint.TestNoPanic"},
		{name: "data file read at runtime", files: map[string]string{"calc/testdata/x.txt": "bye\n"},
			want: "calc.TestData"},
		{name: "new file in a directory a test lists", files: map[string]string{"calc/testdata/y.txt": "new\n"}},
		{name: "const change falls back to linking packages", files: map[string]string{"calc/calc.go": strings.Replace(fixture["calc/calc.go"], "Scale = 1", "Scale = 1 + 0", 1)},
			want: "app.TestSum calc.TestAdd calc.TestData calc.TestMul"},
		{name: "new method on existing type", files: map[string]string{"calc/calc.go": fixture["calc/calc.go"] + "\nfunc (o Op) String() string { return \"op\" }\n"},
			want: "app.TestSum calc.TestAdd calc.TestData calc.TestMul"},
		{name: "new plain function", files: map[string]string{"calc/calc.go": fixture["calc/calc.go"] + "\nfunc Sub(a, b int) int { return a - b }\n"}},
		{name: "new type with methods", files: map[string]string{"calc/calc.go": fixture["calc/calc.go"] + "\ntype T struct{}\n\nfunc (T) String() string { return \"t\" }\n"}},
		{name: "test function", files: map[string]string{"calc/calc_test.go": strings.Replace(fixture["calc/calc_test.go"], "Add(1, 2) != 3", "Add(2, 1) != 3", 1)},
			want: "calc.TestAdd"},
		{name: "new test", files: map[string]string{"other/other_test.go": fixture["other/other_test.go"] + "\nfunc TestHi2(t *testing.T) {}\n"},
			want: "other.TestHi2"},
		{name: "test helper", files: map[string]string{"calc/helper_test.go": "package calc\n\nfunc helper() {}\n"}},
		{name: "docs", files: map[string]string{"docs/readme.md": "# changed\n"}},
		{name: "go.mod", files: map[string]string{"go.mod": fixture["go.mod"] + "\n// touched\n"}, all: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			writeFiles(t, root, c.files)
			defer func() {
				for p := range c.files {
					if orig, ok := fixture[p]; ok {
						writeFiles(t, root, map[string]string{p: orig})
					} else {
						os.Remove(filepath.Join(root, p))
					}
				}
			}()
			all, ids := selected(t, root)
			if all != c.all {
				t.Fatalf("all = %v, want %v", all, c.all)
			}
			if c.all {
				return
			}
			if got := strings.Join(ids, " "); got != c.want {
				t.Errorf("selected %q, want %q", got, c.want)
			}
		})
	}
}
