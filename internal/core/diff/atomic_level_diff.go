package diff

func (v *versionDiff) atomicLevelDiff(fileName string) {
	oldMethMap := v.oldFileMeths[fileName]
	newMethMap := v.newFileMeths[fileName]

	for method, checksum := range oldMethMap {
		if _, containsKey := newMethMap[method]; !containsKey {
			v.DFs[fileName] = method
		} else if newMethMap[method] != checksum {
			v.CFs[fileName] = method
		}
		delete(newMethMap, method)
	}
	for method := range newMethMap {
		v.AFs[fileName] = method
	}

}
