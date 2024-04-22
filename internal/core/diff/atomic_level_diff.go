package diff

import "strings"

func (v *versionDiff) atomicLevelDiff(fileName string) {
	oldMethMap := v.oldFileMeths[fileName]
	newMethMap := make(map[string]string)
	pkgName := fileName[:strings.LastIndex(fileName, "/")]
	for fn, cs := range v.newFileMeths[fileName] {
		newMethMap[fn] = cs
	}
	for method, checksum := range oldMethMap {
		if _, containsKey := newMethMap[method]; !containsKey {
			v.DMs[pkgName+":"+method] = method
		} else if newMethMap[method] != checksum {
			v.CMs[pkgName+":"+method] = method
		}
		delete(newMethMap, method)
	}
	for method := range newMethMap {
		v.AMs[pkgName+":"+method] = method
	}
}
