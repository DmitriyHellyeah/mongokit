package mongokit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestConnect(t *testing.T) {
	t.Run("connects to localhost and returns non-nil database", func(t *testing.T) {
		ctx := context.Background()

		db, err := Connect(ctx, "mongodb://localhost:27017", "mongokit_connect_test")
		require.NoError(t, err)
		require.NotNil(t, db)

		assert.Equal(t, "mongokit_connect_test", db.Name())

		db.Drop(ctx) //nolint:errcheck
	})
}

func TestConnect_InvalidURI(t *testing.T) {
	t.Run("invalid URI returns error", func(t *testing.T) {
		ctx := context.Background()

		_, err := Connect(ctx, "not-a-valid-uri://???", "mongokit_test")
		assert.Error(t, err)
	})
}

func TestConnectWithOptions(t *testing.T) {
	t.Run("connects using custom client options", func(t *testing.T) {
		ctx := context.Background()

		opts := options.Client().ApplyURI("mongodb://localhost:27017")

		db, err := ConnectWithOptions(ctx, "mongokit_options_test", opts)
		require.NoError(t, err)
		require.NotNil(t, db)

		assert.Equal(t, "mongokit_options_test", db.Name())

		db.Drop(ctx) //nolint:errcheck
	})
}
