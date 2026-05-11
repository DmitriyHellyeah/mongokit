package mongokit

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// IRepository defines the interface for database operations
type IRepository[T Document] interface {
	Collection() *mongo.Collection
	InsertOne(ctx context.Context, document T, opts ...options.Lister[options.InsertOneOptions]) (T, error)
	InsertMany(ctx context.Context, documents []T, opts ...options.Lister[options.InsertManyOptions]) ([]any, error)
	Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	FindOneRaw(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	FindDecoded(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) ([]T, error)
	FindDecodedWithTotal(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) ([]T, int64, error)
	FindPaginated(ctx context.Context, filter any, page, pageSize int64) ([]T, error)
	FindPaginatedWithTotal(ctx context.Context, filter any, page, pageSize int64) ([]T, int64, error)
	FindCursorPaginated(ctx context.Context, filter any, pagination *CursorPagination, sort any, cursorEncoder func(T) (string, error)) (*CursorPaginationResult[T], error)
	FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) (T, error)
	FindByID(ctx context.Context, id any, opts ...options.Lister[options.FindOneOptions]) (T, error)
	FindOneAndUpdate(ctx context.Context, filter any, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) (T, error)
	FindOneAndUpdateByID(ctx context.Context, id, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) (T, error)
	FindOneAndDelete(ctx context.Context, filter any, opts ...options.Lister[options.FindOneAndDeleteOptions]) (T, error)
	UpdateOne(ctx context.Context, filter any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
	UpdateByID(ctx context.Context, id any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
	UpdateMany(ctx context.Context, filter any, update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) error
	DeleteByID(ctx context.Context, id any, opts ...options.Lister[options.DeleteOneOptions]) error
	DeleteMany(ctx context.Context, filter any, opts ...options.Lister[options.DeleteManyOptions]) (int64, error)
	EstimatedCount(ctx context.Context, opts ...options.Lister[options.EstimatedDocumentCountOptions]) (int64, error)
	CountDocuments(ctx context.Context, filter any, opts ...options.Lister[options.CountOptions]) (int64, error)
	Aggregate(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) ([]bson.M, error)
	AggregateTyped(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) ([]T, error)
	Distinct(ctx context.Context, fieldName string, filter any, opts ...options.Lister[options.DistinctOptions]) ([]any, error)
	Transaction(ctx context.Context, fn func(sessCtx context.Context) error, opts ...options.Lister[options.SessionOptions]) error
	EnsureIndexes(ctx context.Context, indexes []mongo.IndexModel, opts ...options.Lister[options.CreateIndexesOptions]) error
	GetIndexes(ctx context.Context, opts ...options.Lister[options.ListIndexesOptions]) ([]bson.M, error)
	BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...options.Lister[options.BulkWriteOptions]) (*mongo.BulkWriteResult, error)
	Watch(ctx context.Context, pipeline any, opts ...options.Lister[options.ChangeStreamOptions]) (*mongo.ChangeStream, error)
}

// Document represents an interface for common document operations.
type Document interface {
	SetID(id bson.ObjectID)
	BeforeInsert()
	BeforeUpdate()
}

// ICollectionName is implemented by models to declare their collection name.
type ICollectionName interface {
	CollectionName() string
}

// Repository implements IRepository interface for MongoDB
type Repository[T Document] struct {
	collection *mongo.Collection
}

// NewRepository creates a new MongoDB repository.
// Collection name is taken from T.CollectionName() via the ICollectionName interface.
// If T implements IIndex, indexes are ensured automatically.
// T methods must use pointer receivers to avoid nil pointer panics.
//
// Panics if:
//   - T does not implement ICollectionName
//   - CollectionName() returns an empty string
//   - index creation fails
func NewRepository[T Document](ctx context.Context, database *mongo.Database) *Repository[T] {
	var zero T

	namer, ok := any(zero).(ICollectionName)
	if !ok {
		panic("mongokit: model must implement CollectionName() string")
	}

	name := namer.CollectionName()
	if name == "" {
		panic("mongokit: CollectionName() must return a non-empty string")
	}

	repo := &Repository[T]{
		collection: database.Collection(name),
	}

	if indexer, ok := any(zero).(IIndex); ok {
		indexes := indexer.Indexes()
		if len(indexes) > 0 {
			if err := repo.EnsureIndexes(ctx, indexes); err != nil {
				panic("mongokit: failed to ensure indexes for " + name + ": " + err.Error())
			}
		}
	}

	return repo
}

// compile-time check that Repository implements IRepository
var _ IRepository[*BaseField] = (*Repository[*BaseField])(nil)
