package routes

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/primary/rest"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
)

// RepositoryAuthRoutes sets up routes for repository authentication operations
func RepositoryAuthRoutes(r chi.Router, service repositoryauth.ServiceInterface, projectService *project.Service, logger *zap.Logger) {
	handler := rest.NewRepositoryAuthHandler(service, projectService, logger)

	r.Route("/api/v1/projects/{projectId}/repository-auth", func(r chi.Router) {
		r.Post("/", handler.CreateRepositoryAuth)
		r.Get("/", handler.GetRepositoryAuths)
		r.Get("/available", handler.ListAvailableRepositoryAuths)
		r.Post("/clone", handler.CloneRepositoryAuth)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handler.GetRepositoryAuth)
			r.Put("/", handler.UpdateRepositoryAuth)
			r.Delete("/", handler.DeleteRepositoryAuth)
			r.Post("/test-connection", handler.TestRepositoryAuth)
		})
	})
}
