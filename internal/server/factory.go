package server

import (
	"sift/internal/analysis"
	"sift/internal/analysis/stages"
	"sift/internal/core"
	"sift/internal/database"
	"sift/internal/mcp"
	"sift/internal/parser"
)

type Factory struct {
	dbProvider database.Provider
}

func NewFactory(dbProvider database.Provider) *Factory {
	return &Factory{dbProvider: dbProvider}
}

func (f *Factory) CreateParser() *parser.JUnitParser {
	return parser.NewJUnitParser()
}

func (f *Factory) CreateAnalysisPipeline(repo core.ReportRepository) *analysis.Pipeline {
	return analysis.NewPipeline(
		stages.NewExtractFailuresStage(),
		stages.NewRootCauseExtractStage(),
		stages.NewFingerprintStage(repo),
		stages.NewEnrichHistoryStage(repo),
		stages.NewCascadeDetectStage(),
		stages.NewSummarizeStage(repo),
	)
}

func (f *Factory) CreateReportRepository() *database.ReportRepository {
	return database.NewReportRepository(f.dbProvider.GetDB())
}

func (f *Factory) CreateAnalysisService() *analysis.Service {
	repository := f.CreateReportRepository()
	pipeline := f.CreateAnalysisPipeline(repository)
	return analysis.NewService(pipeline, repository)
}

func (f *Factory) CreateMCPServer() *mcp.Server {
	junitParser := f.CreateParser()
	analysisService := f.CreateAnalysisService()
	reportRepo := f.CreateReportRepository()

	handler := mcp.NewHandler(false)

	tools := mcp.NewTools(junitParser, analysisService, reportRepo, false)
	tools.RegisterAll(handler)

	cfg := mcp.ServerConfig{
		Transport: mcp.TransportStdio,
		ReadOnly:  false,
	}

	return mcp.NewServer(handler, cfg)
}
