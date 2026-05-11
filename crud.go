package mongokit

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// InsertOne inserts a new document. Calls BeforeInsert to set ID and timestamps, then returns the mutated document.
func (r *Repository[T]) InsertOne(ctx context.Context, document T, opts ...options.Lister[options.InsertOneOptions]) (T, error) {
	var zero T

	document.BeforeInsert()

	_, err := r.collection.InsertOne(ctx, document, opts...)
	if err != nil {
		return zero, err
	}

	return document, nil
}

// InsertMany inserts multiple documents and returns their IDs.
// Calls BeforeInsert on each document. Returns ErrEmptySlice if slice is empty.
func (r *Repository[T]) InsertMany(ctx context.Context, documents []T, opts ...options.Lister[options.InsertManyOptions]) ([]any, error) {
	if len(documents) == 0 {
		return nil, ErrEmptySlice
	}

	interfaces := make([]any, len(documents))
	for i, doc := range documents {
		doc.BeforeInsert()
		interfaces[i] = doc
	}

	result, err := r.collection.InsertMany(ctx, interfaces, opts...)
	if err != nil {
		return nil, err
	}

	return result.InsertedIDs, nil
}

// FindOneRaw returns a raw *mongo.SingleResult. The caller decodes it.
func (r *Repository[T]) FindOneRaw(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	if filter == nil {
		filter = bson.M{}
	}
	return r.collection.FindOne(ctx, filter, opts...)
}

// FindOne retrieves a single document matching the filter. Nil filter matches all.
func (r *Repository[T]) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) (T, error) {
	var result T
	if filter == nil {
		filter = bson.M{}
	}
	return result, r.collection.FindOne(ctx, filter, opts...).Decode(&result)
}

// FindRaw returns a raw cursor over documents matching the filter.
// The caller is responsible for closing the cursor.
func (r *Repository[T]) FindRaw(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	if filter == nil {
		filter = bson.M{}
	}
	return r.collection.Find(ctx, filter, opts...)
}

// FindDecoded returns all documents matching the filter, decoded into []T.
func (r *Repository[T]) FindDecoded(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) ([]T, error) {
	cursor, err := r.FindRaw(ctx, filter, opts...)
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

// FindDecodedWithTotal returns all documents matching the filter with the total count.
func (r *Repository[T]) FindDecodedWithTotal(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) ([]T, int64, error) {
	total, err := r.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	results, err := r.FindDecoded(ctx, filter, opts...)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// FindByID finds a document by its ID. Returns ErrNilID if id is nil.
func (r *Repository[T]) FindByID(ctx context.Context, id any, opts ...options.Lister[options.FindOneOptions]) (T, error) {
	if id == nil {
		var zero T
		return zero, ErrNilID
	}
	return r.FindOne(ctx, bson.M{"_id": id}, opts...)
}

// FindOneAndUpdate finds a document and updates it, returning the document after update.
// Accepts bson.M operators or structs (auto-wrapped in $set).
func (r *Repository[T]) FindOneAndUpdate(ctx context.Context, filter, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) (T, error) {
	var result T

	if filter == nil {
		return result, ErrNilFilter
	}

	update, err := prepareUpdate(update)
	if err != nil {
		return result, err
	}

	opts = append(opts, options.FindOneAndUpdate().SetReturnDocument(options.After))
	return result, r.collection.FindOneAndUpdate(ctx, filter, update, opts...).Decode(&result)
}

// FindOneAndUpdateByID finds a document by ID and updates it. Returns ErrNilID if id is nil.
func (r *Repository[T]) FindOneAndUpdateByID(ctx context.Context, id, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) (T, error) {
	if id == nil {
		var zero T
		return zero, ErrNilID
	}
	return r.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts...)
}

// FindOneAndDelete finds a document and deletes it, returning the deleted document.
func (r *Repository[T]) FindOneAndDelete(ctx context.Context, filter any, opts ...options.Lister[options.FindOneAndDeleteOptions]) (T, error) {
	var result T
	if filter == nil {
		filter = bson.M{}
	}
	return result, r.collection.FindOneAndDelete(ctx, filter, opts...).Decode(&result)
}

// UpdateOne updates a single document matching the filter.
// Accepts bson.M operators or structs (auto-wrapped in $set).
func (r *Repository[T]) UpdateOne(ctx context.Context, filter, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	update, err := prepareUpdate(update)
	if err != nil {
		return nil, err
	}
	return r.collection.UpdateOne(ctx, filter, update, opts...)
}

// UpdateByID updates a document by its ID. Returns ErrNilID if id is nil.
func (r *Repository[T]) UpdateByID(ctx context.Context, id any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	if id == nil {
		return nil, ErrNilID
	}
	return r.UpdateOne(ctx, bson.M{"_id": id}, update, opts...)
}

// UpdateMany updates all documents matching the filter.
// Accepts bson.M operators or structs (auto-wrapped in $set).
func (r *Repository[T]) UpdateMany(ctx context.Context, filter any, update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error) {
	update, err := prepareUpdate(update)
	if err != nil {
		return nil, err
	}
	return r.collection.UpdateMany(ctx, filter, update, opts...)
}

// DeleteOne removes a single document matching the filter.
// Returns ErrNilFilter if filter is nil. Returns mongo.ErrNoDocuments if nothing matched.
func (r *Repository[T]) DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) error {
	if filter == nil {
		return ErrNilFilter
	}
	result, err := r.collection.DeleteOne(ctx, filter, opts...)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// DeleteByID removes a document by its ID. Returns ErrNilID if id is nil.
func (r *Repository[T]) DeleteByID(ctx context.Context, id any, opts ...options.Lister[options.DeleteOneOptions]) error {
	if id == nil {
		return ErrNilID
	}
	return r.DeleteOne(ctx, bson.M{"_id": id}, opts...)
}

// DeleteMany removes all documents matching the filter.
// Returns ErrNilFilter if filter is nil. Use bson.M{} to delete all documents explicitly.
func (r *Repository[T]) DeleteMany(ctx context.Context, filter any, opts ...options.Lister[options.DeleteManyOptions]) (int64, error) {
	if filter == nil {
		return 0, ErrNilFilter
	}
	result, err := r.collection.DeleteMany(ctx, filter, opts...)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}
