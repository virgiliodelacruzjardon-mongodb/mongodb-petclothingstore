package api

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---------------------------------------------------------------------------
// Computed pattern helpers.
// These recompute the pre-aggregated values that are denormalized onto parent
// documents, so that reads never have to aggregate on the fly.
// ---------------------------------------------------------------------------

// recomputeProductRating recalculates ratingSummary for a single product from
// the reviews collection and writes it back onto the product document.
func (h *Handler) recomputeProductRating(ctx context.Context, productID primitive.ObjectID) error {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"product.id": productID}},
		bson.M{"$group": bson.M{
			"_id":   "$rating",
			"count": bson.M{"$sum": 1},
		}},
	}
	cur, err := h.S.Reviews.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	dist := map[string]int64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}
	var total, weighted int64
	for cur.Next(ctx) {
		var row struct {
			ID    int   `bson:"_id"`
			Count int64 `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return err
		}
		if row.ID >= 1 && row.ID <= 5 {
			dist[itoa(row.ID)] = row.Count
		}
		total += row.Count
		weighted += int64(row.ID) * row.Count
	}

	var avg float64
	if total > 0 {
		avg = round1(float64(weighted) / float64(total))
	}

	_, err = h.S.Products.UpdateByID(ctx, productID, bson.M{"$set": bson.M{
		"ratingSummary.avg":          avg,
		"ratingSummary.count":        total,
		"ratingSummary.distribution": dist,
	}})
	return err
}

// recomputeCategoryCount recalculates productCount for a category.
func (h *Handler) recomputeCategoryCount(ctx context.Context, categoryID primitive.ObjectID) error {
	count, err := h.S.Products.CountDocuments(ctx, bson.M{"category.id": categoryID})
	if err != nil {
		return err
	}
	_, err = h.S.Categories.UpdateByID(ctx, categoryID, bson.M{"$set": bson.M{"productCount": count}})
	return err
}

// recomputeCustomerStats recalculates orderCount and totalSpent for a customer.
func (h *Handler) recomputeCustomerStats(ctx context.Context, customerID primitive.ObjectID) error {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"customer.id": customerID, "status": bson.M{"$ne": "cancelled"}}},
		bson.M{"$group": bson.M{
			"_id":        nil,
			"orderCount": bson.M{"$sum": 1},
			"totalSpent": bson.M{"$sum": "$pricing.total"},
		}},
	}
	cur, err := h.S.Orders.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	var stats struct {
		OrderCount int64   `bson:"orderCount"`
		TotalSpent float64 `bson:"totalSpent"`
	}
	if cur.Next(ctx) {
		_ = cur.Decode(&stats)
	}
	_, err = h.S.Customers.UpdateByID(ctx, customerID, bson.M{"$set": bson.M{
		"stats.orderCount": stats.OrderCount,
		"stats.totalSpent": round2(stats.TotalSpent),
	}})
	return err
}

func itoa(i int) string {
	return string(rune('0' + i))
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }
func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
