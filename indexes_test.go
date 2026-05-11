package mongokit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// indexHasField reports whether any index in the list contains the given field in its key.
func indexHasField(indexes []bson.M, field string) bool {
	for _, idx := range indexes {
		if keys, ok := idx["key"].(bson.D); ok {
			for _, elem := range keys {
				if elem.Key == field {
					return true
				}
			}
		}
	}
	return false
}

func TestEnsureIndexes(t *testing.T) {
	t.Run("creates unique index and verifies via GetIndexes", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		indexes := UniqueIndexes("email")

		err := repo.EnsureIndexes(ctx, indexes)
		require.NoError(t, err)

		listed, err := repo.GetIndexes(ctx)
		require.NoError(t, err)

		// MongoDB always includes the default _id index, so at least 2 indexes present.
		assert.GreaterOrEqual(t, len(listed), 2)
		assert.True(t, indexHasField(listed, "email"), "expected email index to be listed")
	})
}

func TestEnsureIndexes_Idempotent(t *testing.T) {
	t.Run("creating same index twice does not error", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		indexes := UniqueIndexes("email")

		err := repo.EnsureIndexes(ctx, indexes)
		require.NoError(t, err)

		err = repo.EnsureIndexes(ctx, indexes)
		assert.NoError(t, err)
	})
}

func TestGetIndexes(t *testing.T) {
	t.Run("lists all indexes including the default _id index", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()

		err := repo.EnsureIndexes(ctx, UniqueIndexes("email"))
		require.NoError(t, err)

		indexes, err := repo.GetIndexes(ctx)
		require.NoError(t, err)

		// At minimum: _id index + email index.
		assert.GreaterOrEqual(t, len(indexes), 2)

		names := make([]string, 0, len(indexes))
		for _, idx := range indexes {
			if name, ok := idx["name"].(string); ok {
				names = append(names, name)
			}
		}
		assert.Contains(t, names, "_id_")
	})
}

func TestNewRepository_AutoIndexes(t *testing.T) {
	t.Run("indexes defined via IIndex are ensured automatically on NewRepository", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer func() {
			db.Collection((&TestUserWithIndexes{}).CollectionName()).Drop(context.Background())
			cleanup()
		}()

		ctx := context.Background()

		repo, err := NewRepository[*TestUserWithIndexes](ctx, db)
		require.NoError(t, err)
		require.NotNil(t, repo)

		indexes, err := repo.GetIndexes(ctx)
		require.NoError(t, err)

		// Expect at least _id index + email unique index.
		assert.GreaterOrEqual(t, len(indexes), 2)
		assert.True(t, indexHasField(indexes, "email"), "expected auto-created email index to be present")
	})
}
