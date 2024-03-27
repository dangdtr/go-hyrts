package diff

import (
	"github.com/davecgh/go-spew/spew"
)

type versionDiff struct {
	oldFiles map[string]string
	newFiles map[string]string

	oldFileMeths map[string]map[string]string
	newFileMeths map[string]map[string]string

	deletedFiles map[string]bool
	addedFiles   map[string]bool
	changedFiles map[string]bool
}

type VersionDiff interface {
	Run()
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
	}
}

func (v *versionDiff) Run() {
	v.deserializeOldContents()
	v.parseAndSerializeNewContents()
	v.diff()
	spew.Dump(v)
}

func (v *versionDiff) diff() {
	newFilesShadow := make(map[string]string)
	for key, value := range v.newFiles {
		newFilesShadow[key] = value
	}

	for key := range v.oldFiles {
		if _, containsKey := newFilesShadow[key]; !containsKey {
			v.deletedFiles[key] = true

		} else if containsKey && newFilesShadow[key] != v.oldFiles[key] {
			v.changedFiles[key] = true
			v.atomicLevelDiff() //todo
		}
		delete(newFilesShadow, key)
	}

	for key := range newFilesShadow {
		v.addedFiles[key] = true
	}
}
