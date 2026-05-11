package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/DmitriyHellyeah/mongokit"
)

// --- Models ---

type User struct {
	mongokit.BaseField `bson:",inline"`
	Name               string `bson:"name"`
	Email              string `bson:"email"`
	Status             string `bson:"status"`
}

func (*User) CollectionName() string { return "users" }

// Indexes implements mongokit.IIndex - applied automatically on NewRepository.
func (*User) Indexes() []mongo.IndexModel {
	return mongokit.BuildIndexes(
		mongokit.UniqueIndexes("email"),
		mongokit.NonUniqueIndexes("status", "-createdAt"),
	)
}

type Session struct {
	mongokit.BaseField `bson:",inline"`
	UserID             string `bson:"userId"`
	ExpiresAt          int64  `bson:"expiresAt"`
}

func (*Session) CollectionName() string { return "sessions" }

// Indexes implements mongokit.IIndex - helpers + manual TTL index.
func (*Session) Indexes() []mongo.IndexModel {
	return mongokit.BuildIndexes(
		mongokit.NonUniqueIndexes("userId"),
		[]mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "expiresAt", Value: 1}},
				Options: options.Index().SetExpireAfterSeconds(0),
			},
		},
	)
}

type Product struct {
	mongokit.BaseField `bson:",inline"`
	SKU                string  `bson:"sku"`
	Price              float64 `bson:"price"`
}

func (*Product) CollectionName() string { return "products" }

// Product has no Indexes() - no indexes created automatically.
// Can still add them explicitly via EnsureIndexes.

// --- Main ---

func main() {
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	database := client.Database("example")

	// User and Session have Indexes() - indexes created automatically
	users := mongokit.NewRepository[*User](ctx, database)
	fmt.Println("User indexes created automatically")

	_ = mongokit.NewRepository[*Session](ctx, database)
	fmt.Println("Session indexes created automatically (helpers + TTL)")

	// Product has no Indexes() - add explicitly if needed
	products := mongokit.NewRepository[*Product](ctx, database)
	productIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "sku", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "price", Value: -1}},
		},
	}
	if err := products.EnsureIndexes(ctx, productIndexes); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Product indexes created explicitly")

	// List all user indexes
	userIndexes, err := users.GetIndexes(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nUser indexes (%d):\n", len(userIndexes))
	for _, idx := range userIndexes {
		fmt.Printf("  %v\n", idx)
	}
}
