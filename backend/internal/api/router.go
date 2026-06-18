package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		// ---- auth (public) ----
		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)

		// ---- public reads ----
		r.Get("/products", h.ListProducts)
		r.Get("/products/{id}", h.GetProduct)
		r.Get("/categories", h.ListCategories)
		r.Get("/categories/{id}", h.GetCategory)
		r.Get("/reviews", h.ListReviews)
		r.Get("/reviews/{id}", h.GetReview)

		// ---- protected (JWT required) ----
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware)

			r.Get("/auth/me", h.Me)

			r.Post("/products", h.CreateProduct)
			r.Put("/products/{id}", h.UpdateProduct)
			r.Delete("/products/{id}", h.DeleteProduct)

			r.Post("/categories", h.CreateCategory)
			r.Put("/categories/{id}", h.UpdateCategory)
			r.Delete("/categories/{id}", h.DeleteCategory)

			r.Get("/customers", h.ListCustomers)
			r.Get("/customers/{id}", h.GetCustomer)
			r.Post("/customers", h.CreateCustomer)
			r.Put("/customers/{id}", h.UpdateCustomer)
			r.Delete("/customers/{id}", h.DeleteCustomer)

			r.Get("/orders", h.ListOrders)
			r.Get("/orders/{id}", h.GetOrder)
			r.Post("/orders", h.CreateOrder)
			r.Put("/orders/{id}", h.UpdateOrder)
			r.Delete("/orders/{id}", h.DeleteOrder)

			r.Post("/reviews", h.CreateReview)
			r.Put("/reviews/{id}", h.UpdateReview)
			r.Delete("/reviews/{id}", h.DeleteReview)
		})
	})

	return r
}
