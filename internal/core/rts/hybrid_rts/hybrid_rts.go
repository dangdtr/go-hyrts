package hybrid_rts

import (
	"github.com/dangdtr/go-hyrts/internal/core/diff"
)

func Run() {
	versionDiff := diff.NewVersionDiff()
	versionDiff.Run()
}
