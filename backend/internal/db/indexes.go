package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureIndexes creates the indexes that back the most common access patterns.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	unique := options.Index().SetUnique(true)

	_, err := s.Categories.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: unique},
	})
	if err != nil {
		return err
	}

	_, err = s.Products.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: unique},
		{Keys: bson.D{{Key: "category.id", Value: 1}}},
		{Keys: bson.D{{Key: "petType", Value: 1}}},
		{Keys: bson.D{{Key: "tags", Value: 1}}},
		{Keys: bson.D{{Key: "ratingSummary.avg", Value: -1}}},
		{Keys: bson.D{{Key: "name", Value: "text"}, {Key: "description", Value: "text"}}},
	})
	if err != nil {
		return err
	}

	_, err = s.Customers.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: unique},
	})
	if err != nil {
		return err
	}

	_, err = s.Orders.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "orderNumber", Value: 1}}, Options: unique},
		{Keys: bson.D{{Key: "customer.id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "placedAt", Value: -1}}},
	})
	if err != nil {
		return err
	}

	_, err = s.Reviews.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "product.id", Value: 1}}},
		{Keys: bson.D{{Key: "customer.id", Value: 1}}},
		{Keys: bson.D{{Key: "rating", Value: -1}}},
	})
	return err
}
