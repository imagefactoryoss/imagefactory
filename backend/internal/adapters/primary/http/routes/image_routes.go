package routes

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/primary/http/handlers"
	"github.com/srikarm/image-factory/internal/domain/image"
)

// ImageRoutes sets up routes for image operations
func ImageRoutes(r chi.Router, imageService *image.Service, logger *zap.Logger) {
	handler := handlers.NewImageHandler(imageService, logger)

	r.Route("/api/v1/images", func(r chi.Router) {
		// Image CRUD operations
		r.Post("/", handler.CreateImage)       // POST /api/v1/images
		r.Get("/search", handler.SearchImages) // GET /api/v1/images/search
		r.Get("/popular", handler.GetPopularImages) // GET /api/v1/images/popular
		r.Get("/recent", handler.GetRecentImages)   // GET /api/v1/images/recent

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handler.GetImage)    // GET /api/v1/images/{id}
			r.Put("/", handler.UpdateImage) // PUT /api/v1/images/{id}
			r.Delete("/", handler.DeleteImage) // DELETE /api/v1/images/{id}
		})
	})
}