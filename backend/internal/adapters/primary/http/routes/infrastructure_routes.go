package routes

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/primary/http/handlers"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/infrastructure/k8s"
)

// InfrastructureRoutes sets up routes for infrastructure operations
func InfrastructureRoutes(r chi.Router, buildService *build.Service, infraService *infrastructure.Service, selector *k8s.InfrastructureSelector, logger *zap.Logger) {
	handler := handlers.NewInfrastructureHandler(buildService, infraService, selector, logger)

	r.Route("/api/v1", func(r chi.Router) {
		// Infrastructure recommendation for builds
		r.Post("/builds/infrastructure-recommendation", handler.GetInfrastructureRecommendation)

		// Admin infrastructure usage metrics
		r.Route("/admin/infrastructure", func(r chi.Router) {
			r.Get("/usage", handler.GetInfrastructureUsage)
		})
	})
}
