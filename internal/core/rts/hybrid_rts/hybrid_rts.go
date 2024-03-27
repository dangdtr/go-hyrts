package hybrid_rts

import (
	"github.com/dangdtr/go-hyrts/internal/core/coverage"
	"github.com/dangdtr/go-hyrts/internal/core/diff"
)

func Run() {
	versionDiff := diff.NewVersionDiff()
	versionDiff.Run()

	tracer := coverage.NewCov()
	tracer.Run()

}
