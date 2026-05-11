package mongokit

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collection returns the underlying *mongo.Collection.
func (r *Repository[T]) Collection() *mongo.Collection {
	return r.collection
}

// EstimatedCount returns the estimated number of documents in the collection.
func (r *Repository[T]) EstimatedCount(ctx context.Context, opts ...options.Lister[options.EstimatedDocumentCountOptions]) (int64, error) {
	return r.collection.EstimatedDocumentCount(ctx, opts...)
}

// CountDocuments returns the exact number of documents matching the filter. Nil filter counts all.
func (r *Repository[T]) CountDocuments(ctx context.Context, filter any, opts ...options.Lister[options.CountOptions]) (int64, error) {
	if filter == nil {
		filter = bson.M{}
	}
	return r.collection.CountDocuments(ctx, filter, opts...)
}

// Aggregate performs an aggregation pipeline and returns raw bson.M results.
// Returns ErrNilPipeline if pipeline is nil.
func (r *Repository[T]) Aggregate(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) ([]bson.M, error) {
	if pipeline == nil {
		return nil, ErrNilPipeline
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline, opts...)
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

// AggregateTyped performs an aggregation pipeline and returns typed results.
// Returns ErrNilPipeline if pipeline is nil.
func (r *Repository[T]) AggregateTyped(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) ([]T, error) {
	if pipeline == nil {
		return nil, ErrNilPipeline
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []T
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// Distinct returns the distinct values for a specified field. Nil filter matches all.
func (r *Repository[T]) Distinct(ctx context.Context, fieldName string, filter any, opts ...options.Lister[options.DistinctOptions]) ([]any, error) {
	if filter == nil {
		filter = bson.M{}
	}
	var arr []any
	err := r.collection.Distinct(ctx, fieldName, filter, opts...).Decode(&arr)
	if err != nil {
		return nil, err
	}
	return arr, nil
}

// BulkWrite performs multiple write operations in bulk.
// Returns ErrEmptySlice if models is empty.
func (r *Repository[T]) BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...options.Lister[options.BulkWriteOptions]) (*mongo.BulkWriteResult, error) {
	if len(models) == 0 {
		return nil, ErrEmptySlice
	}
	return r.collection.BulkWrite(ctx, models, opts...)
}

// Watch creates a change stream for the collection.
// Nil pipeline watches all changes.
func (r *Repository[T]) Watch(ctx context.Context, pipeline any, opts ...options.Lister[options.ChangeStreamOptions]) (*mongo.ChangeStream, error) {
	return r.collection.Watch(ctx, pipeline, opts...)
}
