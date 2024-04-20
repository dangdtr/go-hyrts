package hybrid_rts

import (
	"fmt"
	"strings"
	"time"

	"github.com/dangdtr/go-hyrts/internal/core/collect"
	"github.com/dangdtr/go-hyrts/internal/core/diff"
	"github.com/dangdtr/go-hyrts/internal/core/util"
)

func Run() map[string]bool {
	startTime := time.Now()

	versionDiff := diff.NewVersionDiff()
	versionDiff.Run()

	tracer := collect.NewCov(versionDiff.GetNewFileMeths())
	tracer.Run()

	included := make(map[string]bool)

	if util.OldDir == "" {
		fmt.Println("[HyRTS] No RTS analysis due to no old collect, but is computing collect info and checksum info for future RTS...")
		return included
	} else {
		if len(tracer.GetTestCovMap()) == 0 {
			return included
		} else {
			for testFile, testDeps := range tracer.GetTestCovMap() {

				isAffect, testFunc := isAffectedByChange(versionDiff, testDeps, util.TracerCovType)
				if /*exists &&*/ isAffect {
					included[testFile+":"+testFunc] = true
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

		if val, exist := versionDiff.GetCFs()[key]; exist && (val == parts[1]) {
			return true, valDeps
		}
		if val, exist := versionDiff.GetAFs()[key]; exist && (val == parts[1]) {
			return true, valDeps
		}
		if val, exist := versionDiff.GetDFs()[key]; exist && (val == parts[1]) {
			return true, valDeps
		}
	}
	return false, ""
}

func isAffectedByChange(versionDiff diff.VersionDiff, depsMap map[string]string, covType string) (bool, string) {
	// Deps(depsMap): /path:GetUserInfo
	// CFs: path -> GetUserInfo
	for key, valDeps := range depsMap {
		parts := strings.Split(key, ":")
		for _, v := range versionDiff.GetCFs() {
			if parts[1] == v {
				return true, valDeps
			}
		}
		for _, v := range versionDiff.GetAFs() {
			if parts[1] == v {
				return true, valDeps
			}
		}

		for _, v := range versionDiff.GetDFs() {
			if parts[1] == v {
				return true, valDeps
			}
		}
	}
	return false, ""
}
