package main

import (
	"fmt"
	"go/build"
	"go/importer"
	"go/types"
	"log"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/interp"
	"golang.org/x/tools/go/ssa/ssautil"
)

var (
	mode = ssa.BuilderMode(0)
)

func GetSSA() (*ssa.Program, []*ssa.Package) {

	pattern := "proposal2"

	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
	}

	initial, err := packages.Load(cfg, pattern)
	if err != nil {
		panic(err)
	}
	if len(initial) == 0 {
		panic(fmt.Errorf("no packages"))
	}
	if packages.PrintErrors(initial) > 0 {
		panic(fmt.Errorf("packages contain errors"))
	}

	// Filter out standard library and external packages
	var nonStdPkgs []*packages.Package
	for _, pkg := range initial {
		if !isStdLib(pkg.PkgPath) && !isExternal(pkg.PkgPath) {
			nonStdPkgs = append(nonStdPkgs, pkg)
			fmt.Println(nonStdPkgs)
		}
	}

	fmt.Println(initial[0].GoFiles)

	// Create SSA representation for non-standard library packages
	mode := ssa.BuilderMode(0)
	//mode |= ssa.NaiveForm
	//mode |= ssa.GlobalDebug

	mode |= ssa.InstantiateGenerics

	var interpMode interp.Mode
	interpMode |= interp.EnableTracing

	prog, pkgs := ssautil.Packages(nonStdPkgs, mode)
	//fmt.Println(prog)
	return prog, pkgs

	//pattern := "proposal2"
	//
	//cfg := &packages.Config{
	//	Mode:  packages.LoadAllSyntax,
	//	Tests: true,
	//}
	//
	//initial, err := packages.Load(cfg, pattern)
	////fmt.Println(initial)
	//
	//if err != nil {
	//	panic("error")
	//	//return err
	//}
	//if len(initial) == 0 {
	//	panic(fmt.Errorf("no packages"))
	//}
	//if packages.PrintErrors(initial) > 0 {
	//	panic(fmt.Errorf("packages contain errors"))
	//}
	//
	//mode |= ssa.InstantiateGenerics
	//
	//var interpMode interp.Mode
	//interpMode |= interp.EnableTracing
	//
	//prog, pkgs := ssautil.Packages(initial, mode)
	//fmt.Println("dang", prog.AllPackages())
	//return prog, pkgs
}

func GetSSAFromLoadPackages() {
	// Load, parse, and type-check the initial packages.
	cfg := &packages.Config{Mode: packages.LoadSyntax}
	initial, err := packages.Load(cfg, "proposal2")
	if err != nil {
		log.Fatal(err)
	}

	if packages.PrintErrors(initial) > 0 {
		log.Fatalf("packages contain errors")
	}

	prog, pkgs := ssautil.Packages(initial, ssa.PrintPackages)
	_ = prog

	// Build SSA code for the well-typed initial packages.
	for _, p := range pkgs {
		if p != nil {
			p.Build()
			fmt.Println(p)
		}
	}
}

func isStandardLibrary(path string) bool {
	_, err := importer.Default().(types.ImporterFrom).ImportFrom(path, "", 0)
	return err == nil
}

// Function to check if a package is from the standard library
func isStdLib(pkgPath string) bool {
	for _, srcDir := range build.Default.SrcDirs() {
		if relPath, err := filepath.Rel(srcDir, pkgPath); err == nil {
			if !strings.HasPrefix(relPath, "..") {
				return true
			}
		}
	}
	return false
}

// Function to check if a package is external (not part of the main module)
func isExternal(pkgPath string) bool {
	if strings.HasPrefix(pkgPath, "github.com/") || strings.HasPrefix(pkgPath, "gitlab.com/") {
		return true
	}
	return false
}
