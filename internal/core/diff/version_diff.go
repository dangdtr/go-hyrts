package diff

import (
	"sync"
)

type versionDiff struct {
	oldFiles map[string]string
	newFiles map[string]string

	oldFileMeths map[string]map[string]string
	newFileMeths map[string]map[string]string

	deletedFiles map[string]bool
	addedFiles   map[string]bool
	changedFiles map[string]bool

	AFs map[string]string
	CFs map[string]string
	DFs map[string]string
}

type VersionDiff interface {
	Run()
	GetChangedFiles() map[string]bool
	GetAFs() map[string]string
	GetCFs() map[string]string
	GetDFs() map[string]string
	GetNewFileMeths() map[string]map[string]string
}

func NewVersionDiff() VersionDiff {
	return &versionDiff{
		oldFiles:     make(map[string]string),
		newFiles:     make(map[string]string),
		oldFileMeths: make(map[string]map[string]string),
		newFileMeths: make(map[string]map[string]string),
		deletedFiles: make(map[string]bool),
		addedFiles:   make(map[string]bool),
		changedFiles: make(map[string]bool),
		AFs:          make(map[string]string),
		CFs:          make(map[string]string),
		DFs:          make(map[string]string),
	}
}

func (v *versionDiff) GetChangedFiles() map[string]bool {
	return v.changedFiles
}

func (v *versionDiff) GetCFs() map[string]string {
	return v.CFs
}

func (v *versionDiff) GetDFs() map[string]string {
	return v.DFs
}

func (v *versionDiff) GetAFs() map[string]string {
	return v.AFs
}

func (v *versionDiff) GetNewFileMeths() map[string]map[string]string {
	return v.newFileMeths
}

func (v *versionDiff) Run() {
	v.deserializeOldContents()
	v.parseAndSerializeNewContents()
	v.diff()
	//v.cleanContents()
}

func (v *versionDiff) diff() {

	var mu sync.Mutex

	for key, content := range v.oldFiles {
		mu.Lock()

		if _, containsKey := v.newFiles[key]; !containsKey {
			v.deletedFiles[key] = true

		} else if v.newFiles[key] != content {
			v.changedFiles[key] = true
			v.atomicLevelDiff(key) //todo
		}
		delete(v.newFiles, key)

		mu.Unlock()
	}

	for key := range v.newFiles {
		mu.Lock()
		v.addedFiles[key] = true
		mu.Unlock()

	}
}

func (v *versionDiff) cleanContents() {
	v.newFiles = nil
	v.oldFiles = nil
	v.oldFileMeths = nil
	v.newFileMeths = nil
}
