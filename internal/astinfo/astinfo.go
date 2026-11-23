// Package astinfo extracts per-declaration checksums from Go source. Comments
// and formatting do not affect a checksum; any change to code does.
package astinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Func describes one function or method declaration.
type Func struct {
	Hash       string
	Start, End int    // line span, used to map coverage blocks
	Recv       string // receiver type name for methods, "" otherwise
	Init       bool
	Test       bool // Test*, Example* or Fuzz* entry point in a _test.go file
}

// Decl is a non-function declaration: a type, var or const group, or an
// import that runs init code for its side effects (blank or dot import).
type Decl struct {
	Hash string
	// Effect is true when adding the declaration can change behaviour by
	// itself: a var with an initializer, or a side-effect import.
	Effect bool
	Types  []string // type names declared, used to tell new types apart
}

// File is the declaration summary of one Go file.
type File struct {
	Package string
	Funcs   map[string]Func // key: "F", "T.M" or "init#N"
	Decls   map[string]Decl // key: "type T", "var a,b", "const X", "import _ path"
}

// Parse summarises src. name is used only in error messages and to decide
// whether test entry points can be declared in it.
func Parse(name string, src []byte) (*File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution) // no comments
	if err != nil {
		return nil, err
	}
	isTestFile := strings.HasSuffix(name, "_test.go")
	out := &File{Package: f.Name.Name, Funcs: map[string]Func{}, Decls: map[string]Decl{}}
	inits := 0
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			key, recv := d.Name.Name, ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv = recvName(d.Recv.List[0].Type)
				key = recv + "." + d.Name.Name
			}
			fn := Func{
				Hash:  hash(fset, src, d),
				Start: fset.Position(d.Pos()).Line,
				End:   fset.Position(d.End()).Line,
				Recv:  recv,
			}
			if recv == "" && d.Name.Name == "init" {
				key = fmt.Sprintf("init#%d", inits)
				inits++
				fn.Init = true
			}
			if isTestFile && recv == "" && IsTestName(d.Name.Name) {
				fn.Test = true
			}
			out.Funcs[key] = fn
		case *ast.GenDecl:
			switch d.Tok {
			case token.IMPORT:
				for _, s := range d.Specs {
					s := s.(*ast.ImportSpec)
					if s.Name != nil && (s.Name.Name == "_" || s.Name.Name == ".") {
						out.Decls["import "+s.Name.Name+" "+s.Path.Value] = Decl{Hash: "1", Effect: true}
					}
				}
			default:
				var names, types []string
				effect := false
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						names = append(names, s.Name.Name)
						types = append(types, s.Name.Name)
					case *ast.ValueSpec:
						for _, n := range s.Names {
							names = append(names, n.Name)
						}
						if d.Tok == token.VAR && len(s.Values) > 0 {
							effect = true
						}
					}
				}
				out.Decls[d.Tok.String()+" "+strings.Join(names, ",")] = Decl{Hash: hash(fset, src, d), Effect: effect, Types: types}
			}
		}
	}
	return out, nil
}

// IsTestName reports whether a top-level function name is a test entry point
// that `go test` runs by default (benchmarks are not).
func IsTestName(name string) bool {
	if name == "TestMain" {
		return false
	}
	for _, prefix := range []string{"Test", "Fuzz"} {
		if strings.HasPrefix(name, prefix) {
			if len(name) == len(prefix) {
				return true
			}
			r, _ := utf8.DecodeRuneInString(name[len(prefix):])
			return !unicode.IsLower(r)
		}
	}
	return strings.HasPrefix(name, "Example")
}

// SortedFuncs returns function keys ordered by start line.
func (f *File) SortedFuncs() []string {
	keys := make([]string, 0, len(f.Funcs))
	for k := range f.Funcs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return f.Funcs[keys[i]].Start < f.Funcs[keys[j]].Start })
	return keys
}

func recvName(e ast.Expr) string {
	for {
		switch t := e.(type) {
		case *ast.StarExpr:
			e = t.X
		case *ast.ParenExpr:
			e = t.X
		case *ast.IndexExpr:
			e = t.X
		case *ast.IndexListExpr:
			e = t.X
		case *ast.Ident:
			return t.Name
		default:
			return fmt.Sprintf("%T", e)
		}
	}
}

func hash(fset *token.FileSet, src []byte, n ast.Node) string {
	// Hash the token stream of the declaration: comments and layout do not
	// count, every token including string literal contents does.
	from, to := fset.Position(n.Pos()).Offset, fset.Position(n.End()).Offset
	body := src[from:to]
	var s scanner.Scanner
	fs := token.NewFileSet()
	s.Init(fs.AddFile("", -1, len(body)), body, nil, 0)
	h := sha256.New()
	// A semicolon right before a closing brace or paren is optional (one
	// line vs several lines), so it is dropped; all others are kept.
	pendingSemi := false
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON {
			pendingSemi = true
			continue
		}
		if pendingSemi && tok != token.RBRACE && tok != token.RPAREN {
			fmt.Fprintf(h, "%d\x00", token.SEMICOLON)
		}
		pendingSemi = false
		fmt.Fprintf(h, "%d %s\x00", tok, lit)
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}
