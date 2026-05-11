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

func TestInsertOne(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	input := &TestUser{
		Name:   "Alice",
		Email:  "alice@example.com",
		Status: "active",
	}

	got, err := repo.InsertOne(ctx, input)
	require.NoError(t, err)

	assert.False(t, got.ID.IsZero(), "ID should be set after insert")
	assert.False(t, got.CreatedAt.IsZero(), "CreatedAt should be set after insert")
	assert.False(t, got.UpdatedAt.IsZero(), "UpdatedAt should be set after insert")
	assert.Equal(t, "Alice", got.Name)
	assert.Equal(t, "alice@example.com", got.Email)
}

func TestInsertMany(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("inserts 3 docs and returns 3 IDs", func(t *testing.T) {
		docs := []*TestUser{
			{Name: "A", Email: "a@example.com", Status: "active"},
			{Name: "B", Email: "b@example.com", Status: "active"},
			{Name: "C", Email: "c@example.com", Status: "active"},
		}

		ids, err := repo.InsertMany(ctx, docs)
		require.NoError(t, err)
		assert.Len(t, ids, 3)
	})

	t.Run("empty slice returns ErrEmptySlice", func(t *testing.T) {
		_, err := repo.InsertMany(ctx, []*TestUser{})
		assert.True(t, errors.Is(err, ErrEmptySlice))
	})
}

func TestFindOne(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	t.Run("find by email filter", func(t *testing.T) {
		got, err := repo.FindOne(ctx, bson.M{"email": seeded[0].Email})
		require.NoError(t, err)
		assert.Equal(t, seeded[0].ID, got.ID)
		assert.Equal(t, seeded[0].Email, got.Email)
	})

	t.Run("nil filter returns first doc", func(t *testing.T) {
		got, err := repo.FindOne(ctx, nil)
		require.NoError(t, err)
		assert.False(t, got.ID.IsZero())
	})
}

func TestFindOneRaw(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	result := repo.FindOneRaw(ctx, bson.M{"email": seeded[0].Email})
	require.NotNil(t, result)

	var decoded TestUser
	err := result.Decode(&decoded)
	require.NoError(t, err)
	assert.Equal(t, seeded[0].ID, decoded.ID)
	assert.Equal(t, seeded[0].Name, decoded.Name)
}

func TestFindRaw(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seedUsers(t, repo, 3)

	cursor, err := repo.Find(ctx, bson.M{})
	require.NoError(t, err)
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		count++
	}
	require.NoError(t, cursor.Err())
	assert.Equal(t, 3, count)
}

func TestFindDecoded(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seedUsers(t, repo, 3)

	results, err := repo.FindDecoded(ctx, bson.M{})
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestFindDecodedWithTotal(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seedUsers(t, repo, 3)

	items, total, err := repo.FindDecodedWithTotal(ctx, bson.M{})
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, int64(3), total)
}

func TestFindByID(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	t.Run("find by valid ID", func(t *testing.T) {
		got, err := repo.FindByID(ctx, seeded[0].ID)
		require.NoError(t, err)
		assert.Equal(t, seeded[0].ID, got.ID)
		assert.Equal(t, seeded[0].Name, got.Name)
	})

	t.Run("nil ID returns ErrNilID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, nil)
		assert.True(t, errors.Is(err, ErrNilID))
	})

	t.Run("unknown ID returns mongo.ErrNoDocuments", func(t *testing.T) {
		_, err := repo.FindByID(ctx, bson.NewObjectID())
		assert.True(t, errors.Is(err, mongo.ErrNoDocuments))
	})
}

func TestFindOneAndUpdate(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	t.Run("update with bson.M operator", func(t *testing.T) {
		got, err := repo.FindOneAndUpdate(
			ctx,
			bson.M{"_id": seeded[0].ID},
			bson.M{"$set": bson.M{"name": "Updated"}},
		)
		require.NoError(t, err)
		assert.Equal(t, "Updated", got.Name)
	})

	t.Run("nil update returns ErrNilUpdate", func(t *testing.T) {
		_, err := repo.FindOneAndUpdate(ctx, bson.M{"_id": seeded[0].ID}, nil)
		assert.True(t, errors.Is(err, ErrNilUpdate))
	})
}

func TestFindOneAndUpdateByID(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	t.Run("update by ID", func(t *testing.T) {
		got, err := repo.FindOneAndUpdateByID(
			ctx,
			seeded[0].ID,
			bson.M{"$set": bson.M{"name": "ByID Updated"}},
		)
		require.NoError(t, err)
		assert.Equal(t, "ByID Updated", got.Name)
	})

	t.Run("nil ID returns ErrNilID", func(t *testing.T) {
		_, err := repo.FindOneAndUpdateByID(ctx, nil, bson.M{"$set": bson.M{"name": "x"}})
		assert.True(t, errors.Is(err, ErrNilID))
	})
}

func TestFindOneAndDelete(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	deleted, err := repo.FindOneAndDelete(ctx, bson.M{"_id": seeded[0].ID})
	require.NoError(t, err)
	assert.Equal(t, seeded[0].ID, deleted.ID)

	count, err := repo.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestUpdateOne(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	result, err := repo.UpdateOne(
		ctx,
		bson.M{"_id": seeded[0].ID},
		bson.M{"$set": bson.M{"name": "UpdateOne"}},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.MatchedCount)

	got, err := repo.FindByID(ctx, seeded[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "UpdateOne", got.Name)
}

func TestUpdateByID(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	t.Run("update by valid ID", func(t *testing.T) {
		result, err := repo.UpdateByID(
			ctx,
			seeded[0].ID,
			bson.M{"$set": bson.M{"name": "UpdateByID"}},
		)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.MatchedCount)

		got, err := repo.FindByID(ctx, seeded[0].ID)
		require.NoError(t, err)
		assert.Equal(t, "UpdateByID", got.Name)
	})

	t.Run("nil ID returns ErrNilID", func(t *testing.T) {
		_, err := repo.UpdateByID(ctx, nil, bson.M{"$set": bson.M{"name": "x"}})
		assert.True(t, errors.Is(err, ErrNilID))
	})
}

func TestUpdateMany(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	docs := []*TestUser{
		{Name: "X", Email: "x@example.com", Status: "active"},
		{Name: "Y", Email: "y@example.com", Status: "active"},
		{Name: "Z", Email: "z@example.com", Status: "active"},
	}
	_, err := repo.InsertMany(ctx, docs)
	require.NoError(t, err)

	result, err := repo.UpdateMany(
		ctx,
		bson.M{"status": "active"},
		bson.M{"$set": bson.M{"status": "inactive"}},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.ModifiedCount)
}

func TestDeleteOne(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	t.Run("delete matching doc", func(t *testing.T) {
		err := repo.DeleteOne(ctx, bson.M{"_id": seeded[0].ID})
		require.NoError(t, err)

		count, err := repo.CountDocuments(ctx, bson.M{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("nil filter returns ErrNilFilter", func(t *testing.T) {
		err := repo.DeleteOne(ctx, nil)
		assert.True(t, errors.Is(err, ErrNilFilter))
	})

	t.Run("no match returns mongo.ErrNoDocuments", func(t *testing.T) {
		err := repo.DeleteOne(ctx, bson.M{"_id": bson.NewObjectID()})
		assert.True(t, errors.Is(err, mongo.ErrNoDocuments))
	})
}

func TestDeleteByID(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	seeded := seedUsers(t, repo, 1)

	t.Run("delete by valid ID", func(t *testing.T) {
		err := repo.DeleteByID(ctx, seeded[0].ID)
		require.NoError(t, err)

		count, err := repo.CountDocuments(ctx, bson.M{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("nil ID returns ErrNilID", func(t *testing.T) {
		err := repo.DeleteByID(ctx, nil)
		assert.True(t, errors.Is(err, ErrNilID))
	})
}

func TestDeleteMany(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete all docs with empty filter", func(t *testing.T) {
		seedUsers(t, repo, 3)

		deleted, err := repo.DeleteMany(ctx, bson.M{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), deleted)
	})

	t.Run("nil filter returns ErrNilFilter", func(t *testing.T) {
		_, err := repo.DeleteMany(ctx, nil)
		assert.True(t, errors.Is(err, ErrNilFilter))
	})
}
