package mongokit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestCountDocuments(t *testing.T) {
	t.Run("returns count after seeding", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		seedUsers(t, repo, 3)

		count, err := repo.CountDocuments(context.Background(), bson.M{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("nil filter counts all documents", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		seedUsers(t, repo, 3)

		count, err := repo.CountDocuments(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("empty collection returns zero", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		count, err := repo.CountDocuments(context.Background(), bson.M{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestEstimatedCount(t *testing.T) {
	t.Run("returns non-negative count after seeding", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		seedUsers(t, repo, 3)

		count, err := repo.EstimatedCount(context.Background())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0))
	})
}

func TestAggregate(t *testing.T) {
	t.Run("returns bson.M results matching pipeline", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		seedUsers(t, repo, 3)

		pipeline := bson.A{
			bson.M{"$match": bson.M{"status": "active"}},
		}

		results, err := repo.Aggregate(context.Background(), pipeline)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		for _, r := range results {
			assert.Equal(t, "active", r["status"])
		}
	})

	t.Run("nil pipeline returns ErrNilPipeline", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		_, err := repo.Aggregate(context.Background(), nil)
		assert.True(t, errors.Is(err, ErrNilPipeline))
	})
}

func TestAggregateTyped(t *testing.T) {
	t.Run("returns typed results matching pipeline", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		seedUsers(t, repo, 3)

		pipeline := bson.A{
			bson.M{"$match": bson.M{"status": "active"}},
		}

		results, err := repo.AggregateTyped(context.Background(), pipeline)
		require.NoError(t, err)
		assert.Len(t, results, 3)
		for _, u := range results {
			assert.Equal(t, "active", u.Status)
			assert.False(t, u.ID.IsZero())
		}
	})

	t.Run("nil pipeline returns ErrNilPipeline", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		_, err := repo.AggregateTyped(context.Background(), nil)
		assert.True(t, errors.Is(err, ErrNilPipeline))
	})
}

func TestDistinct(t *testing.T) {
	t.Run("returns unique values for field", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		statuses := []string{"active", "inactive", "pending"}
		for i, s := range statuses {
			_, err := repo.InsertOne(ctx, &TestUser{
				Name:   "User " + string(rune('A'+i)),
				Email:  "user" + string(rune('a'+i)) + "@example.com",
				Status: s,
			})
			require.NoError(t, err)
		}

		values, err := repo.Distinct(ctx, "status", bson.M{})
		require.NoError(t, err)
		assert.Len(t, values, 3)
	})

	t.Run("nil filter matches all documents", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		seedUsers(t, repo, 2)

		values, err := repo.Distinct(context.Background(), "status", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(values), 1)
	})
}

func TestBulkWrite(t *testing.T) {
	t.Run("inserts documents via bulk write", func(t *testing.T) {
		repo, _, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()

		u1 := &TestUser{Name: "Bulk A", Email: "bulka@example.com", Status: "active"}
		u1.BeforeInsert()
		u2 := &TestUser{Name: "Bulk B", Email: "bulkb@example.com", Status: "active"}
		u2.BeforeInsert()

		models := []mongo.WriteModel{
			mongo.NewInsertOneModel().SetDocument(u1),
			mongo.NewInsertOneModel().SetDocument(u2),
		}

		result, err := repo.BulkWrite(ctx, models)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.InsertedCount)

		count, err := repo.CountDocuments(ctx, bson.M{})
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}
