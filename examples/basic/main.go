package main

import (
	"context"
	"errors"
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

type DB struct {
	Users *mongokit.Repository[*User]
}

func main() {
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	database := client.Database("example")

	users, err := mongokit.NewRepository[*User](ctx, database)
	if err != nil {
		log.Fatal(err)
	}
	db := DB{Users: users}

	// InsertOne - ID, CreatedAt, UpdatedAt are set automatically by BaseField.BeforeInsert().
	// No need to set them manually.
	inserted, err := db.Users.InsertOne(ctx, &User{
		Name:  "John Doe",
		Email: "john@example.com",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted: %s (ID: %s)\n", inserted.Name, inserted.ID.Hex())
	fmt.Printf("  CreatedAt: %s\n", inserted.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  UpdatedAt: %s\n", inserted.UpdatedAt.Format("2006-01-02 15:04:05"))

	// FindByID
	found, err := db.Users.FindByID(ctx, inserted.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found by ID: %s\n", found.Name)

	// MapNotFoundErr - map mongo.ErrNoDocuments to a domain error
	var ErrUserNotFound = errors.New("user not found")
	fakeID := bson.NewObjectID()
	_, err = db.Users.FindByID(ctx, fakeID)
	err = mongokit.MapNotFoundErr(err, ErrUserNotFound)
	fmt.Printf("Fake ID lookup: %v\n", err) // "user not found"

	// FindOne with filter
	found, err = db.Users.FindOne(ctx, bson.M{"email": "john@example.com"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found by email: %s\n", found.Name)

	// FindRaw (raw cursor) - caller controls decoding
	cursor, err := db.Users.FindRaw(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var u User
		if err := cursor.Decode(&u); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Cursor item: %s\n", u.Name)
	}

	// FindDecoded - decoded into []T automatically
	allUsers, err := db.Users.FindDecoded(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("FindDecoded: %d users\n", len(allUsers))

	// UpdateByID with bson.M - UpdatedAt is NOT updated automatically.
	// To auto-update timestamps, pass a struct instead of bson.M.
	_, err = db.Users.UpdateByID(ctx, inserted.ID, bson.M{"$set": bson.M{"name": "John Updated"}})
	if err != nil {
		log.Fatal(err)
	}

	// FindOneAndUpdateByID - returns the updated document
	updated, err := db.Users.FindOneAndUpdateByID(ctx, inserted.ID, bson.M{"$set": bson.M{"name": "John Final"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated: %s\n", updated.Name)

	// DeleteByID
	err = db.Users.DeleteByID(ctx, inserted.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Deleted")

	// CountDocuments
	count, err := db.Users.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Remaining: %d\n", count)
}
