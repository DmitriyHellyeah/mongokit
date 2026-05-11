package mongokit

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Transaction executes the given function within a MongoDB transaction.
// If fn returns an error, the transaction is rolled back.
func (r *Repository[T]) Transaction(ctx context.Context, fn func(sessCtx context.Context) error, opts ...options.Lister[options.SessionOptions]) error {
	session, err := r.collection.Database().Client().StartSession(opts...)
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
		return nil, fn(sessCtx)
	})

	return err
}
