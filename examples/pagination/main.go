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

type User struct {
	mongokit.BaseField `bson:",inline"`
	Name               string `bson:"name"`
	Email              string `bson:"email"`
}

func (*User) CollectionName() string { return "users" }

// encodeCursor converts a document to a cursor string (using ID hex).
func encodeCursor(u *User) (string, error) {
	return u.ID.Hex(), nil
}

func main() {
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	database := client.Database("example")

	users := mongokit.NewRepository[*User](ctx, database)

	// Seed some data
	for i := 0; i < 50; i++ {
		_, err := users.InsertOne(ctx, &User{
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	// --- Offset pagination ---
	fmt.Println("=== Offset Pagination ===")

	results, total, err := users.FindPaginatedWithTotal(ctx, bson.M{}, 1, 10)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page 1: %d items (total: %d)\n", len(results), total)

	// --- Cursor pagination ---
	fmt.Println("\n=== Cursor Pagination ===")

	// First page
	page, err := users.FindCursorPaginated(ctx,
		bson.M{},
		&mongokit.CursorPagination{Limit: 10, Direction: mongokit.CursorNext},
		bson.D{{Key: "_id", Value: 1}},
		encodeCursor,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page 1: %d items, hasNext=%v, hasPrev=%v\n",
		len(page.Items), page.HasNext, page.HasPrev)

	// Second page (forward) — caller builds filter with cursor
	nextID, _ := bson.ObjectIDFromHex(page.NextCursor)
	page, err = users.FindCursorPaginated(ctx,
		bson.M{"_id": bson.M{"$gt": nextID}},
		&mongokit.CursorPagination{Cursor: page.NextCursor, Limit: 10, Direction: mongokit.CursorNext},
		bson.D{{Key: "_id", Value: 1}},
		encodeCursor,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page 2: %d items, hasNext=%v, hasPrev=%v\n",
		len(page.Items), page.HasNext, page.HasPrev)

	// Go back (previous page) — reverse sort + $lt
	prevID, _ := bson.ObjectIDFromHex(page.PrevCursor)
	page, err = users.FindCursorPaginated(ctx,
		bson.M{"_id": bson.M{"$lt": prevID}},
		&mongokit.CursorPagination{Cursor: page.PrevCursor, Limit: 10, Direction: mongokit.CursorPrev},
		bson.D{{Key: "_id", Value: -1}},
		encodeCursor,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Back to page 1: %d items, hasNext=%v, hasPrev=%v\n",
		len(page.Items), page.HasNext, page.HasPrev)

	// Cleanup
	users.DeleteMany(ctx, bson.M{})
}
