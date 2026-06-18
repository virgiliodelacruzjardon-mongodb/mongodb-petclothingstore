package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"petstore/internal/models"
)

const (
	taxRate      = 0.08
	flatShipping = 6.99
)

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, limit := paginate(r)
	filter := bson.M{}
	if s := r.URL.Query().Get("status"); s != "" {
		filter["status"] = s
	}
	if cid := r.URL.Query().Get("customerId"); cid != "" {
		if id, err := primitive.ObjectIDFromHex(cid); err == nil {
			filter["customer.id"] = id
		}
	}
	total, _ := h.S.Orders.CountDocuments(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "placedAt", Value: -1}}).
		SetSkip((page - 1) * limit).SetLimit(limit)
	cur, err := h.S.Orders.Find(ctx, filter, opts)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []models.Order
	if err := cur.All(ctx, &out); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Data: out, Page: page, Limit: limit, Total: total})
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var o models.Order
	if err := h.S.Orders.FindOne(r.Context(), bson.M{"_id": id}).Decode(&o); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// computePricing applies the computed pattern: line totals + order totals are
// calculated once on write and stored on the document.
func computePricing(items []models.OrderItem, discount float64) ([]models.OrderItem, models.Pricing) {
	var subtotal float64
	for i := range items {
		items[i].LineTotal = round2(items[i].Price * float64(items[i].Qty))
		subtotal += items[i].LineTotal
	}
	tax := round2(subtotal * taxRate)
	shipping := flatShipping
	if subtotal > 75 {
		shipping = 0 // free shipping threshold
	}
	total := round2(subtotal + tax + shipping - discount)
	return items, models.Pricing{
		Subtotal: round2(subtotal),
		Tax:      tax,
		Shipping: shipping,
		Discount: round2(discount),
		Total:    total,
	}
}

// errInsufficientStock signals that a variant did not have enough inventory.
// It is an application error (not a transient transaction error), so
// WithTransaction will NOT retry it — the whole transaction is aborted and
// every stock decrement already applied is rolled back automatically.
type insufficientStockError struct{ sku string }

func (e insufficientStockError) Error() string {
	return fmt.Sprintf("insufficient stock for sku %s", e.sku)
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var o models.Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(o.Items) == 0 {
		writeErr(w, http.StatusBadRequest, "order must have at least one item")
		return
	}

	now := time.Now().UTC()
	o.ID = primitive.NewObjectID()
	o.Items, o.Pricing = computePricing(o.Items, o.Pricing.Discount)
	if o.Status == "" {
		o.Status = "pending"
	}
	if o.OrderNumber == "" {
		o.OrderNumber = fmt.Sprintf("PET-%d", now.UnixNano()/1e6)
	}
	if o.PlacedAt.IsZero() {
		o.PlacedAt = now
	}
	o.CreatedAt = now
	o.UpdatedAt = now

	// --- multi-document transaction -------------------------------------
	// Atomically: decrement each variant's stock (only if enough is on hand)
	// AND insert the order. If any item lacks stock, the whole thing aborts.
	session, err := h.S.Client.StartSession()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not start session: "+err.Error())
		return
	}
	defer session.EndSession(ctx)

	txnOpts := options.Transaction().
		SetReadConcern(readconcern.Snapshot()).
		SetWriteConcern(writeconcern.Majority())

	callback := func(sc mongo.SessionContext) (interface{}, error) {
		for _, it := range o.Items {
			// Conditional update: match the product + the specific variant only
			// when its stock is >= the requested quantity. The arrayFilter makes
			// the $inc target exactly that variant.
			res, uerr := h.S.Products.UpdateOne(
				sc,
				bson.M{
					"_id": it.ProductID,
					"variants": bson.M{"$elemMatch": bson.M{
						"sku":   it.SKU,
						"stock": bson.M{"$gte": it.Qty},
					}},
				},
				bson.M{"$inc": bson.M{
					"variants.$[v].stock": -it.Qty,
					"totalStock":          -it.Qty,
				}},
				options.Update().SetArrayFilters(options.ArrayFilters{
					Filters: []interface{}{
						bson.M{"v.sku": it.SKU, "v.stock": bson.M{"$gte": it.Qty}},
					},
				}),
			)
			if uerr != nil {
				return nil, uerr
			}
			if res.ModifiedCount == 0 {
				return nil, insufficientStockError{sku: it.SKU}
			}
		}
		if _, ierr := h.S.Orders.InsertOne(sc, o); ierr != nil {
			return nil, ierr
		}
		return nil, nil
	}

	if _, err = session.WithTransaction(ctx, callback, txnOpts); err != nil {
		if _, ok := err.(insufficientStockError); ok {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "transaction failed: "+err.Error())
		return
	}

	// computed pattern: refresh denormalized customer stats (post-commit)
	if !o.Customer.ID.IsZero() {
		_ = h.recomputeCustomerStats(ctx, o.Customer.ID)
	}
	writeJSON(w, http.StatusCreated, o)
}

func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body models.Order
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.Items, body.Pricing = computePricing(body.Items, body.Pricing.Discount)
	set := bson.M{
		"items":           body.Items,
		"pricing":         body.Pricing,
		"status":          body.Status,
		"shippingAddress": body.ShippingAddress,
		"updatedAt":       time.Now().UTC(),
	}
	_, err = h.S.Orders.UpdateByID(r.Context(), id, bson.M{"$set": set})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var o models.Order
	_ = h.S.Orders.FindOne(r.Context(), bson.M{"_id": id}).Decode(&o)
	if !o.Customer.ID.IsZero() {
		_ = h.recomputeCustomerStats(r.Context(), o.Customer.ID)
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var o models.Order
	_ = h.S.Orders.FindOne(r.Context(), bson.M{"_id": id}).Decode(&o)
	if _, err := h.S.Orders.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !o.Customer.ID.IsZero() {
		_ = h.recomputeCustomerStats(r.Context(), o.Customer.ID)
	}
	w.WriteHeader(http.StatusNoContent)
}
