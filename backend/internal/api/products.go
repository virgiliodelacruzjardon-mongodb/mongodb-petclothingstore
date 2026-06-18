package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"petstore/internal/models"
)

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, limit := paginate(r)

	filter := bson.M{}
	if q := r.URL.Query().Get("q"); q != "" {
		filter["$or"] = bson.A{
			bson.M{"name": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"tags": bson.M{"$regex": q, "$options": "i"}},
		}
	}
	if pet := r.URL.Query().Get("petType"); pet != "" {
		filter["petType"] = pet
	}
	if cat := r.URL.Query().Get("categoryId"); cat != "" {
		if cid, err := primitive.ObjectIDFromHex(cat); err == nil {
			filter["category.id"] = cid
		}
	}

	total, _ := h.S.Products.CountDocuments(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip((page - 1) * limit).SetLimit(limit)
	cur, err := h.S.Products.Find(ctx, filter, opts)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []models.Product
	if err := cur.All(ctx, &out); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Data: out, Page: page, Limit: limit, Total: total})
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p models.Product
	if err := h.S.Products.FindOne(r.Context(), bson.M{"_id": id}).Decode(&p); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	now := time.Now().UTC()
	p.ID = primitive.NilObjectID
	if p.Slug == "" {
		p.Slug = slugify(p.Name)
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	// computed pattern: derive totalStock from embedded variants
	p.TotalStock = 0
	for _, v := range p.Variants {
		p.TotalStock += v.Stock
	}
	p.RatingSummary = models.RatingSummary{Distribution: map[string]int64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}}
	p.CreatedAt = now
	p.UpdatedAt = now

	res, err := h.S.Products.InsertOne(r.Context(), p)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	// keep category.productCount in sync (computed pattern)
	if !p.Category.ID.IsZero() {
		_ = h.recomputeCategoryCount(r.Context(), p.Category.ID)
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body models.Product
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	stock := 0
	for _, v := range body.Variants {
		stock += v.Stock
	}
	set := bson.M{
		"name":        body.Name,
		"slug":        body.Slug,
		"description": body.Description,
		"brand":       body.Brand,
		"petType":     body.PetType,
		"category":    body.Category,
		"basePrice":   body.BasePrice,
		"currency":    body.Currency,
		"variants":    body.Variants,
		"images":      body.Images,
		"tags":        body.Tags,
		"active":      body.Active,
		"totalStock":  stock,
		"updatedAt":   time.Now().UTC(),
	}
	if body.Slug == "" {
		set["slug"] = slugify(body.Name)
	}
	_, err = h.S.Products.UpdateByID(r.Context(), id, bson.M{"$set": set})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !body.Category.ID.IsZero() {
		_ = h.recomputeCategoryCount(r.Context(), body.Category.ID)
	}
	h.GetProduct(w, r)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p models.Product
	_ = h.S.Products.FindOne(r.Context(), bson.M{"_id": id}).Decode(&p)
	if _, err := h.S.Products.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !p.Category.ID.IsZero() {
		_ = h.recomputeCategoryCount(r.Context(), p.Category.ID)
	}
	w.WriteHeader(http.StatusNoContent)
}
