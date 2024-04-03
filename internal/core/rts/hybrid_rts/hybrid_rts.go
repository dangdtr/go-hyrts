package hybrid_rts

import (
	"fmt"
	"time"

	"github.com/dangdtr/go-hyrts/internal/core/coverage"
	"github.com/dangdtr/go-hyrts/internal/core/diff"
	"github.com/dangdtr/go-hyrts/internal/core/util"
)

func Run() map[string]bool {
	versionDiff := diff.NewVersionDiff()
	versionDiff.Run()

	tracer := coverage.NewCov()
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
			for testFile := range tracer.GetTestCovMap() {
				//testPath := filepath.Join(Properties_OLD_DIR, test)

				//depsMap := tracer.GetTestCovMap()[testFile]

				exists := versionDiff.GetChangedFiles()[testFile]
				isAffect := isAffected(versionDiff, tracer.GetTestCovMap()[testFile], util.TracerCovType)
				fmt.Println(exists, isAffect)
				if _, exists := versionDiff.GetChangedFiles()[testFile]; exists && isAffected(versionDiff, tracer.GetTestCovMap()[testFile], util.TracerCovType) {

					//keyRun := fmt.Sprintf("%s:%s", testFile, testName)
					included[testFile] = true
					//if util.NewDir != util.OldDir {
					//	oldVPath := getTestCovFilePath(util.OldDir, test)
					//	newVPath := getTestCovFilePath(util.NewDir, test)
					//	oldV, err := os.Open(oldVPath)
					//	if err != nil {
					//		fmt.Println(err)
					//		continue
					//	}
					//	defer oldV.Close()
					//
					//	newV, err := os.Create(newVPath)
					//	if err != nil {
					//		fmt.Println(err)
					//		continue
					//	}
					//	defer newV.Close()
					//
					//	_, err = io.Copy(newV, oldV)
					//	if err != nil {
					//		fmt.Println(err)
					//		continue
					//	}
					//	fmt.Println(oldVPath)
					//}
				}
			}
		}

	}

	endTime := time.Now()
	fmt.Printf("[HyRTS] RTS included %d of %d test file using %dms\n", len(tracer.GetTestCovMap())-len(included), len(tracer.GetTestCovMap()), endTime.Sub(startTime).Milliseconds())

	return included
}

func isAffected(versionDiff diff.VersionDiff, depsMap coverage.Deps, covType string) bool {
	// Deps(depsMap): /path:GetUserInfo
	// CFs: path -> GetUserInfo
	for key := range depsMap {

		if _, exist := versionDiff.GetCFs()[key]; exist && versionDiff.GetCFs()[key] == key {
			return true
		}
		if _, exist := versionDiff.GetAFs()[key]; exist && versionDiff.GetAFs()[key] == key {
			return true
		}
		if _, exist := versionDiff.GetDFs()[key]; exist && versionDiff.GetDFs()[key] == key {
			return true
		}
	}
	return false
}
