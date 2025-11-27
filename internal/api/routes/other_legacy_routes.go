package routes

import (
	"net/http"
	"strings"

	"httpserver/server/handlers"
)

// DuplicateDetectionHandlers содержит handlers для обнаружения дубликатов
type DuplicateDetectionHandlers struct {
	Handler *handlers.DuplicateDetectionHandler
	HandleStartDetection    http.HandlerFunc
	HandleGetStatus         http.HandlerFunc
	HandleDuplicateDetection        http.HandlerFunc
	HandleDuplicateDetectionStatus  http.HandlerFunc
}

// PatternDetectionHandlers содержит handlers для тестирования паттернов
type PatternDetectionHandlers struct {
	Handler *handlers.PatternDetectionHandler
	HandleDetectPatterns    http.HandlerFunc
	HandleSuggestPatterns   http.HandlerFunc
	HandleTestBatch         http.HandlerFunc
	HandlePatternDetect     http.HandlerFunc
	HandlePatternSuggest    http.HandlerFunc
	HandlePatternTestBatch  http.HandlerFunc
}

// NormalizationBenchmarkHandlers содержит handlers для бенчмарков нормализации
type NormalizationBenchmarkHandlers struct {
	Handler *handlers.NormalizationBenchmarkHandler
	HandleUploadBenchmark  http.HandlerFunc
	HandleListBenchmarks   http.HandlerFunc
	HandleGetBenchmark     http.HandlerFunc
	HandleNormalizationBenchmarkUpload http.HandlerFunc
	HandleNormalizationBenchmarkList   http.HandlerFunc
	HandleNormalizationBenchmarkGet    http.HandlerFunc
}

// ReclassificationHandlers содержит handlers для переклассификации
type ReclassificationHandlers struct {
	Handler *handlers.ReclassificationHandler
	HandleStart     http.HandlerFunc
	HandleEvents    http.HandlerFunc
	HandleStatus    http.HandlerFunc
	HandleStop      http.HandlerFunc
	HandleReclassificationStart    http.HandlerFunc
	HandleReclassificationEvents   http.HandlerFunc
	HandleReclassificationStatus   http.HandlerFunc
	HandleReclassificationStop     http.HandlerFunc
}

// WorkerHandlers содержит handlers для управления воркерами
type WorkerHandlers struct {
	Handler *handlers.WorkerHandler
	WorkerTraceHandler *handlers.WorkerTraceHandler
	HandleGetWorkerConfig              http.HandlerFunc
	HandleUpdateWorkerConfig            http.HandlerFunc
	HandleGetAvailableProviders         http.HandlerFunc
	HandleCheckArliaiConnection         http.HandlerFunc
	HandleCheckOpenRouterConnection     http.HandlerFunc
	HandleCheckHuggingFaceConnection    http.HandlerFunc
	HandleGetModels                     http.HandlerFunc
	HandleOrchestratorStrategy          http.HandlerFunc
	HandleOrchestratorStats             http.HandlerFunc
	HandleWorkerTraceStream             http.HandlerFunc
}

// BenchmarkHandlers содержит handlers для эталонов
type BenchmarkHandlers struct {
	Handler *handlers.BenchmarkHandler
	HandleImportManufacturers http.HandlerFunc
}

// GISPHandlers содержит handlers для GISP
type GISPHandlers struct {
	Handler *handlers.GISPHandler
	HandleImportNomenclatures    http.HandlerFunc
	HandleGetNomenclatures       http.HandlerFunc
	HandleGetNomenclatureDetail  http.HandlerFunc
	HandleGetReferenceBooks      http.HandlerFunc
	HandleSearchReferenceBook    http.HandlerFunc
	HandleGetStatistics           http.HandlerFunc
	HandleImportGISPNomenclatures http.HandlerFunc
	HandleGetGISPNomenclatures    http.HandlerFunc
	HandleGetGISPNomenclatureDetail http.HandlerFunc
	HandleGetGISPReferenceBooks   http.HandlerFunc
	HandleSearchGISPReferenceBook  http.HandlerFunc
	HandleGetGISPStatistics       http.HandlerFunc
}

// DashboardHandlers содержит handlers для дашборда
type DashboardHandlers struct {
	Handler *handlers.DashboardHandler
	HandleGetStats                    http.HandlerFunc
	HandleDashboardOverview           http.HandlerFunc
	HandleGetNormalizationStatus      http.HandlerFunc
	HandleGetQualityMetrics           http.HandlerFunc
	HandleGetDashboardStats           http.HandlerFunc
	HandleGetDashboardNormalizationStatus http.HandlerFunc
}

// Processing1CHandlers содержит handlers для обработки 1С
type Processing1CHandlers struct {
	Handler *handlers.Processing1CHandler
	HandleGenerateProcessingXML http.HandlerFunc
	Handle1CProcessingXML       http.HandlerFunc
}

// ReportHandlers содержит handlers для отчетов
type ReportHandlers struct {
	Handler *handlers.ReportHandler
	HandleGenerateNormalizationReport http.HandlerFunc
	HandleGenerateDataQualityReport   http.HandlerFunc
}

// RegisterDuplicateDetectionRoutes регистрирует маршруты для обнаружения дубликатов
func RegisterDuplicateDetectionRoutes(mux *http.ServeMux, h *DuplicateDetectionHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/duplicates/detect", h.Handler.HandleStartDetection)
		mux.HandleFunc("/api/duplicates/detect/", h.Handler.HandleGetStatus)
		return
	}
	if h.HandleDuplicateDetection != nil {
		mux.HandleFunc("/api/duplicates/detect", h.HandleDuplicateDetection)
	}
	if h.HandleDuplicateDetectionStatus != nil {
		mux.HandleFunc("/api/duplicates/detect/", h.HandleDuplicateDetectionStatus)
	}
}

// RegisterPatternDetectionRoutes регистрирует маршруты для тестирования паттернов
func RegisterPatternDetectionRoutes(mux *http.ServeMux, h *PatternDetectionHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/patterns/detect", h.Handler.HandleDetectPatterns)
		mux.HandleFunc("/api/patterns/suggest", h.Handler.HandleSuggestPatterns)
		mux.HandleFunc("/api/patterns/test-batch", h.Handler.HandleTestBatch)
		return
	}
	if h.HandlePatternDetect != nil {
		mux.HandleFunc("/api/patterns/detect", h.HandlePatternDetect)
	}
	if h.HandlePatternSuggest != nil {
		mux.HandleFunc("/api/patterns/suggest", h.HandlePatternSuggest)
	}
	if h.HandlePatternTestBatch != nil {
		mux.HandleFunc("/api/patterns/test-batch", h.HandlePatternTestBatch)
	}
}

// RegisterNormalizationBenchmarkRoutes регистрирует маршруты для бенчмарков нормализации
func RegisterNormalizationBenchmarkRoutes(mux *http.ServeMux, h *NormalizationBenchmarkHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/normalization/benchmark/upload", h.Handler.HandleUploadBenchmark)
		mux.HandleFunc("/api/normalization/benchmark/list", h.Handler.HandleListBenchmarks)
		mux.HandleFunc("/api/normalization/benchmark/", h.Handler.HandleGetBenchmark)
		return
	}
	if h.HandleNormalizationBenchmarkUpload != nil {
		mux.HandleFunc("/api/normalization/benchmark/upload", h.HandleNormalizationBenchmarkUpload)
	}
	if h.HandleNormalizationBenchmarkList != nil {
		mux.HandleFunc("/api/normalization/benchmark/list", h.HandleNormalizationBenchmarkList)
	}
	if h.HandleNormalizationBenchmarkGet != nil {
		mux.HandleFunc("/api/normalization/benchmark/", h.HandleNormalizationBenchmarkGet)
	}
}

// RegisterReclassificationRoutes регистрирует маршруты для переклассификации
func RegisterReclassificationRoutes(mux *http.ServeMux, h *ReclassificationHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/reclassification/start", h.Handler.HandleStart)
		mux.HandleFunc("/api/reclassification/events", h.Handler.HandleEvents)
		mux.HandleFunc("/api/reclassification/status", h.Handler.HandleStatus)
		mux.HandleFunc("/api/reclassification/stop", h.Handler.HandleStop)
		return
	}
	if h.HandleReclassificationStart != nil {
		mux.HandleFunc("/api/reclassification/start", h.HandleReclassificationStart)
	}
	if h.HandleReclassificationEvents != nil {
		mux.HandleFunc("/api/reclassification/events", h.HandleReclassificationEvents)
	}
	if h.HandleReclassificationStatus != nil {
		mux.HandleFunc("/api/reclassification/status", h.HandleReclassificationStatus)
	}
	if h.HandleReclassificationStop != nil {
		mux.HandleFunc("/api/reclassification/stop", h.HandleReclassificationStop)
	}
}

// RegisterWorkerRoutes регистрирует маршруты для управления воркерами
func RegisterWorkerRoutes(mux *http.ServeMux, h *WorkerHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/workers/config", h.Handler.HandleGetWorkerConfig)
		mux.HandleFunc("/api/workers/config/update", h.Handler.HandleUpdateWorkerConfig)
		mux.HandleFunc("/api/workers/providers", h.Handler.HandleGetAvailableProviders)
		mux.HandleFunc("/api/workers/arliai/status", h.Handler.HandleCheckArliaiConnection)
		mux.HandleFunc("/api/workers/openrouter/status", h.Handler.HandleCheckOpenRouterConnection)
		mux.HandleFunc("/api/workers/huggingface/status", h.Handler.HandleCheckHuggingFaceConnection)
		mux.HandleFunc("/api/workers/models", h.Handler.HandleGetModels)
		mux.HandleFunc("/api/workers/orchestrator/strategy", h.Handler.HandleOrchestratorStrategy)
		mux.HandleFunc("/api/workers/orchestrator/stats", h.Handler.HandleOrchestratorStats)
	} else {
		if h.HandleGetWorkerConfig != nil {
			mux.HandleFunc("/api/workers/config", h.HandleGetWorkerConfig)
		}
		if h.HandleUpdateWorkerConfig != nil {
			mux.HandleFunc("/api/workers/config/update", h.HandleUpdateWorkerConfig)
		}
		if h.HandleGetAvailableProviders != nil {
			mux.HandleFunc("/api/workers/providers", h.HandleGetAvailableProviders)
		}
		if h.HandleCheckArliaiConnection != nil {
			mux.HandleFunc("/api/workers/arliai/status", h.HandleCheckArliaiConnection)
		}
		if h.HandleCheckOpenRouterConnection != nil {
			mux.HandleFunc("/api/workers/openrouter/status", h.HandleCheckOpenRouterConnection)
		}
		if h.HandleCheckHuggingFaceConnection != nil {
			mux.HandleFunc("/api/workers/huggingface/status", h.HandleCheckHuggingFaceConnection)
		}
		if h.HandleGetModels != nil {
			mux.HandleFunc("/api/workers/models", h.HandleGetModels)
		}
		if h.HandleOrchestratorStrategy != nil {
			mux.HandleFunc("/api/workers/orchestrator/strategy", h.HandleOrchestratorStrategy)
		}
		if h.HandleOrchestratorStats != nil {
			mux.HandleFunc("/api/workers/orchestrator/stats", h.HandleOrchestratorStats)
		}
	}
	
	// Worker trace handler
	if h.WorkerTraceHandler != nil {
		mux.HandleFunc("/api/internal/worker-trace/stream", h.WorkerTraceHandler.HandleWorkerTraceStream)
	} else if h.HandleWorkerTraceStream != nil {
		mux.HandleFunc("/api/internal/worker-trace/stream", h.HandleWorkerTraceStream)
	}
}

// RegisterBenchmarkRoutes регистрирует маршруты для эталонов
func RegisterBenchmarkRoutes(mux *http.ServeMux, h *BenchmarkHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/benchmarks/import-manufacturers", h.Handler.HandleImportManufacturers)
		// Новые эндпоинты для управления эталонами
		mux.HandleFunc("/api/benchmarks/from-upload", h.Handler.CreateFromUpload)
		mux.HandleFunc("/api/benchmarks/search", h.Handler.Search)
		mux.HandleFunc("/api/benchmarks", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				h.Handler.List(w, r)
			} else if r.Method == http.MethodPost {
				h.Handler.Create(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})
		mux.HandleFunc("/api/benchmarks/", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
			if path == "" {
				http.Error(w, "Benchmark ID is required", http.StatusBadRequest)
				return
			}
			switch r.Method {
			case http.MethodGet:
				h.Handler.GetByID(w, r)
			case http.MethodPut:
				h.Handler.Update(w, r)
			case http.MethodDelete:
				h.Handler.Delete(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})
		return
	}
	if h.HandleImportManufacturers != nil {
		mux.HandleFunc("/api/benchmarks/import-manufacturers", h.HandleImportManufacturers)
	}
}

// RegisterGISPRoutes регистрирует маршруты для GISP
func RegisterGISPRoutes(mux *http.ServeMux, h *GISPHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/gisp/nomenclatures/import", h.Handler.HandleImportNomenclatures)
		mux.HandleFunc("/api/gisp/nomenclatures", h.Handler.HandleGetNomenclatures)
		mux.HandleFunc("/api/gisp/nomenclatures/", h.Handler.HandleGetNomenclatureDetail)
		mux.HandleFunc("/api/gisp/reference-books", h.Handler.HandleGetReferenceBooks)
		mux.HandleFunc("/api/gisp/reference-books/search", h.Handler.HandleSearchReferenceBook)
		mux.HandleFunc("/api/gisp/statistics", h.Handler.HandleGetStatistics)
		return
	}
	if h.HandleImportGISPNomenclatures != nil {
		mux.HandleFunc("/api/gisp/nomenclatures/import", h.HandleImportGISPNomenclatures)
	}
	if h.HandleGetGISPNomenclatures != nil {
		mux.HandleFunc("/api/gisp/nomenclatures", h.HandleGetGISPNomenclatures)
	}
	if h.HandleGetGISPNomenclatureDetail != nil {
		mux.HandleFunc("/api/gisp/nomenclatures/", h.HandleGetGISPNomenclatureDetail)
	}
	if h.HandleGetGISPReferenceBooks != nil {
		mux.HandleFunc("/api/gisp/reference-books", h.HandleGetGISPReferenceBooks)
	}
	if h.HandleSearchGISPReferenceBook != nil {
		mux.HandleFunc("/api/gisp/reference-books/search", h.HandleSearchGISPReferenceBook)
	}
	if h.HandleGetGISPStatistics != nil {
		mux.HandleFunc("/api/gisp/statistics", h.HandleGetGISPStatistics)
	}
}

// RegisterDashboardRoutes регистрирует маршруты для дашборда
func RegisterDashboardRoutes(mux *http.ServeMux, h *DashboardHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/dashboard/stats", h.Handler.HandleGetStats)
		mux.HandleFunc("/api/dashboard/overview", h.Handler.HandleDashboardOverview)
		mux.HandleFunc("/api/dashboard/normalization-status", h.Handler.HandleGetNormalizationStatus)
		mux.HandleFunc("/api/quality/metrics", h.Handler.HandleGetQualityMetrics)
		return
	}
	if h.HandleGetDashboardStats != nil {
		mux.HandleFunc("/api/dashboard/stats", h.HandleGetDashboardStats)
	}
	if h.HandleGetDashboardNormalizationStatus != nil {
		mux.HandleFunc("/api/dashboard/normalization-status", h.HandleGetDashboardNormalizationStatus)
	}
	if h.HandleGetQualityMetrics != nil {
		mux.HandleFunc("/api/quality/metrics", h.HandleGetQualityMetrics)
	}
}

// RegisterProcessing1CRoutes регистрирует маршруты для обработки 1С
func RegisterProcessing1CRoutes(mux *http.ServeMux, h *Processing1CHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/1c/processing/xml", h.Handler.HandleGenerateProcessingXML)
		return
	}
	if h.Handle1CProcessingXML != nil {
		mux.HandleFunc("/api/1c/processing/xml", h.Handle1CProcessingXML)
	}
}

// RegisterReportRoutes регистрирует маршруты для отчетов
func RegisterReportRoutes(mux *http.ServeMux, h *ReportHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/reports/generate-normalization-report", h.Handler.HandleGenerateNormalizationReport)
		mux.HandleFunc("/api/reports/generate-data-quality-report", h.Handler.HandleGenerateDataQualityReport)
		return
	}
	if h.HandleGenerateNormalizationReport != nil {
		mux.HandleFunc("/api/reports/generate-normalization-report", h.HandleGenerateNormalizationReport)
	}
	if h.HandleGenerateDataQualityReport != nil {
		mux.HandleFunc("/api/reports/generate-data-quality-report", h.HandleGenerateDataQualityReport)
	}
}

