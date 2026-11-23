// Package runner turns a selection into one `go test` invocation.
package runner

import (
	"regexp"
	"sort"
	"strings"

	"github.com/dangdtr/go-hyrts/internal/selector"
)

// Args returns the go test arguments for r, or nil if nothing is selected.
// Every selected package is run in one invocation so the go command keeps
// its own package parallelism and caching; a shared -run pattern may run a
// same-named test in another selected package, which is harmless.
func Args(r *selector.Result, goFlags []string) []string {
	args := append([]string{"test"}, goFlags...)
	if r.All {
		return append(args, "./...")
	}
	byPkg := r.ByPackage()
	if len(byPkg) == 0 {
		return nil
	}
	pkgs := make([]string, 0, len(byPkg))
	names := map[string]bool{}
	for p, ns := range byPkg {
		pkgs = append(pkgs, p)
		for _, n := range ns {
			names[n] = true
		}
	}
	sort.Strings(pkgs)
	list := make([]string, 0, len(names))
	for n := range names {
		list = append(list, regexp.QuoteMeta(n))
	}
	sort.Strings(list)
	args = append(args, "-run", "^("+strings.Join(list, "|")+")$")
	return append(args, pkgs...)
}
