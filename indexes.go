package mongokit

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// IIndex is optionally implemented by models to declare their indexes.
// If implemented, indexes are ensured automatically in NewRepository.
type IIndex interface {
	Indexes() []mongo.IndexModel
}

// EnsureIndexes creates indexes for the collection if they don't exist.
func (r *Repository[T]) EnsureIndexes(ctx context.Context, indexes []mongo.IndexModel, opts ...options.Lister[options.CreateIndexesOptions]) error {
	_, err := r.collection.Indexes().CreateMany(ctx, indexes, opts...)
	return err
}

// GetIndexes returns all indexes for the collection.
func (r *Repository[T]) GetIndexes(ctx context.Context, opts ...options.Lister[options.ListIndexesOptions]) ([]bson.M, error) {
	cursor, err := r.collection.Indexes().List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
