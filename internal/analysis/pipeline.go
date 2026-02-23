package analysis

import (
	"context"

	"sift/internal/core"
)

type Stage interface {
	Process(ctx context.Context, report *core.TestReport, result *core.AnalysisResult)
	Name() string
}

type Pipeline struct {
	stages []Stage
}

func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

func (p *Pipeline) Process(ctx context.Context, report *core.TestReport) *core.AnalysisResult {
	result := &core.AnalysisResult{
		ReportID:    report.ID,
		FailedTests: make([]core.FailedTestInfo, 0),
	}

	for _, stage := range p.stages {
		stage.Process(ctx, report, result)
	}

	return result
}

func (p *Pipeline) AddStage(stage Stage) {
	p.stages = append(p.stages, stage)
}
