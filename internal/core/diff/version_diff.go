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

	AMs map[string]string
	CMs map[string]string
	DMs map[string]string
}

type VersionDiff interface {
	Run()
	GetAddedFiles() map[string]bool
	GetChangedFiles() map[string]bool
	GetDeletedFiles() map[string]bool
	GetAMs() map[string]string
	GetCMs() map[string]string
	GetDMs() map[string]string
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
		AMs:          make(map[string]string),
		CMs:          make(map[string]string),
		DMs:          make(map[string]string),
	}
}

func (v *versionDiff) GetAddedFiles() map[string]bool {
	return v.addedFiles
}
func (v *versionDiff) GetChangedFiles() map[string]bool {
	return v.changedFiles
}
func (v *versionDiff) GetDeletedFiles() map[string]bool {
	return v.deletedFiles
}

func (v *versionDiff) GetCMs() map[string]string {
	return v.CMs
}

func (v *versionDiff) GetDMs() map[string]string {
	return v.DMs
}

func (v *versionDiff) GetAMs() map[string]string {
	return v.AMs
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
