package hybrid_rts

import (
	"fmt"
	"strings"
	"time"

	"github.com/dangdtr/go-hyrts/internal/core/coverage"
	"github.com/dangdtr/go-hyrts/internal/core/diff"
	"github.com/dangdtr/go-hyrts/internal/core/util"
)

func Run() map[string]bool {
	versionDiff := diff.NewVersionDiff()
	versionDiff.Run()

	tracer := coverage.NewCov(versionDiff.GetNewFileMeths())
	tracer.Run()

	startTime := time.Now()

	included := make(map[string]bool)

	if util.OldDir == "" {
		fmt.Println("[HyRTS] No RTS analysis due to no old coverage, but is computing coverage info and checksum info for future RTS...")
		return included
	} else {
		if len(tracer.GetTestCovMap()) == 0 {
			return included
		} else {
			for testFile, testDeps := range tracer.GetTestCovMap() {
				//testPath := filepath.Join(Properties_OLD_DIR, test)

				for fileDep, _ := range testDeps {
					parts := strings.Split(fileDep, ":")

					exists := versionDiff.GetChangedFiles()[parts[0]]
					//fmt.Println(exists)

					isAffect, testFunc := isAffected(versionDiff, tracer.GetTestCovMap()[testFile], util.TracerCovType)
					//fmt.Println(exists, isAffect)
					if exists && isAffect {

						included[testFile+":"+testFunc] = true

					}
				}

			}
		}

	}

	endTime := time.Now()
	fmt.Printf("[HyRTS] RTS included %d of %d test file using %dms\n", len(included), len(tracer.GetTestCovMap()), endTime.Sub(startTime).Milliseconds())

	fmt.Println()
	//fmt.Println(included)
	return included
}

func isAffected(versionDiff diff.VersionDiff, depsMap map[string]string, covType string) (bool, string) {
	// Deps(depsMap): /path:GetUserInfo
	// CFs: path -> GetUserInfo
	for key, valDeps := range depsMap {
		parts := strings.Split(key, ":")

		if val, exist := versionDiff.GetCFs()[parts[0]]; exist && (val == parts[1]) {
			return true, valDeps
		}
		if val, exist := versionDiff.GetAFs()[parts[0]]; exist && (val == parts[1]) {
			return true, valDeps
		}
		if val, exist := versionDiff.GetDFs()[parts[0]]; exist && (val == parts[1]) {
			return true, valDeps
		}
	}
	return false, ""
}
