package mongokit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// --- Test Models ---

type TestUser struct {
	BaseField `bson:",inline"`
	Name      string `bson:"name"`
	Email     string `bson:"email"`
	Status    string `bson:"status"`
}

func (*TestUser) CollectionName() string { return "test_users" }

type TestUserWithIndexes struct {
	BaseField `bson:",inline"`
	Email     string `bson:"email"`
}

func (*TestUserWithIndexes) CollectionName() string { return "test_users_indexed" }

func (*TestUserWithIndexes) Indexes() []mongo.IndexModel {
	return UniqueIndexes("email")
}

// --- Setup / Teardown ---

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	t.Helper()
	ctx := context.Background()
	db, err := Connect(ctx, "mongodb://localhost:27017", "mongokit_test")
	require.NoError(t, err)

	cleanup := func() {
		db.Drop(context.Background())
	}
	return db, cleanup
}

func setupTestRepo(t *testing.T) (*Repository[*TestUser], *mongo.Database, func()) {
	t.Helper()
	db, cleanup := setupTestDB(t)
	ctx := context.Background()

	repo := NewRepository[*TestUser](ctx, db)

	return repo, db, func() {
		repo.Collection().Drop(context.Background())
		cleanup()
	}
}

func seedUsers(t *testing.T, repo *Repository[*TestUser], count int) []*TestUser {
	t.Helper()
	ctx := context.Background()
	var users []*TestUser
	for i := 0; i < count; i++ {
		u, err := repo.InsertOne(ctx, &TestUser{
			Name:   "User " + string(rune('A'+i)),
			Email:  "user" + string(rune('a'+i)) + "@example.com",
			Status: "active",
		})
		require.NoError(t, err)
		users = append(users, u)
	}
	return users
}
