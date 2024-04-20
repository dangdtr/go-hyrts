package util

import (
	"strings"
)

var (
	//ProgramPath = "/Users/dangdt/Documents/coding/go-hyrts/go-hyrts/example"
	ProgramPath = "/Users/dangdt/Documents/KLTN_draft/rs/fiber-main"

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

var StandardLibraries = map[string]bool{
	"fmt":                       true,
	"math":                      true,
	"strings":                   true,
	"bufio":                     true,
	"bytes":                     true,
	"io":                        true,
	"os":                        true,
	"strconv":                   true,
	"time":                      true,
	"encoding":                  true,
	"encoding/binary":           true,
	"encoding/csv":              true,
	"encoding/json":             true,
	"encoding/xml":              true,
	"net":                       true,
	"net/http":                  true,
	"net/url":                   true,
	"path":                      true,
	"regexp":                    true,
	"sort":                      true,
	"sync":                      true,
	"sync/atomic":               true,
	"text/scanner":              true,
	"text/tabwriter":            true,
	"text/template":             true,
	"text/template/parse":       true,
	"unicode":                   true,
	"unicode/utf8":              true,
	"archive/tar":               true,
	"archive/zip":               true,
	"compress/bzip2":            true,
	"compress/flate":            true,
	"compress/gzip":             true,
	"compress/lzw":              true,
	"compress/zlib":             true,
	"database/sql":              true,
	"database/sql/driver":       true,
	"encoding/gob":              true,
	"encoding/asn1":             true,
	"encoding/hex":              true,
	"encoding/pem":              true,
	"encoding/base64":           true,
	"flag":                      true,
	"go/ast":                    true,
	"go/build":                  true,
	"go/constant":               true,
	"go/doc":                    true,
	"go/format":                 true,
	"go/importer":               true,
	"go/internal/gcimporter":    true,
	"go/internal/gccgoimporter": true,
	"go/internal/srcimporter":   true,
	"go/parser":                 true,
	"go/printer":                true,
	"go/scanner":                true,
	"go/token":                  true,
	"hash":                      true,
	"hash/adler32":              true,
	"hash/crc32":                true,
	"hash/crc64":                true,
	"hash/fnv":                  true,
	"html":                      true,
	"html/template":             true,
	"image":                     true,
	"image/color":               true,
	"image/draw":                true,
	"image/gif":                 true,
	"image/jpeg":                true,
	"image/png":                 true,
	"index/suffixarray":         true,
	"io/ioutil":                 true,
	"log":                       true,
	"log/syslog":                true,
	"math/big":                  true,
	"math/cmplx":                true,
	"math/rand":                 true,
	"mime":                      true,
	"mime/multipart":            true,
	"mime/quotedprintable":      true,
	"net/mail":                  true,
	"net/rpc":                   true,
	"net/rpc/jsonrpc":           true,
	"net/smtp":                  true,
	"net/textproto":             true,
	"net/http/cgi":              true,
	"net/http/cookiejar":        true,
	"net/http/fcgi":             true,
	"net/http/httptest":         true,
	"net/http/httputil":         true,
	"net/http/pprof":            true,
	"net/http/httptrace":        true,
	"net/http/httpproxy":        true,
	"plugin":                    true,
	"reflect":                   true,
	"regexp/syntax":             true,
	"runtime":                   true,
	"runtime/cgo":               true,
	"runtime/debug":             true,
	"runtime/pprof":             true,
	"runtime/race":              true,
	"runtime/trace":             true,
	"syscall":                   true,
	"testing":                   true,
	"testing/iotest":            true,
	"testing/quick":             true,
	"unicode/utf16":             true,
	"unsafe":                    true,
}
