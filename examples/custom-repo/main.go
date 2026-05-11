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

type Task struct {
	mongokit.BaseField `bson:",inline"`
	Title              string        `bson:"title"`
	UserID             bson.ObjectID `bson:"userId"`
}

func (*Task) CollectionName() string { return "tasks" }

// --- Custom Repository ---

type UserRepo struct {
	*mongokit.Repository[*User]
}

func NewUserRepo(ctx context.Context, db *mongo.Database) *UserRepo {
	return &UserRepo{mongokit.NewRepository[*User](ctx, db)}
}

func (r *UserRepo) FindActiveByEmail(ctx context.Context, email string) (*User, error) {
	return r.FindOne(ctx, bson.M{"email": email, "status": "active"})
}

func (r *UserRepo) FindByStatus(ctx context.Context, status string) ([]*User, error) {
	return r.FindDecoded(ctx, bson.M{"status": status})
}

func (r *UserRepo) Deactivate(ctx context.Context, id bson.ObjectID) error {
	_, err := r.UpdateByID(ctx, id, bson.M{"$set": bson.M{"status": "inactive"}})
	return err
}

// --- Database ---

type DB struct {
	Users *UserRepo
	Tasks *mongokit.Repository[*Task]
}

func Initialize(ctx context.Context, db *mongo.Database) *DB {
	return &DB{
		Users: NewUserRepo(ctx, db),
		Tasks: mongokit.NewRepository[*Task](ctx, db),
	}
}

// --- Main ---

func main() {
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := Initialize(ctx, client.Database("example"))

	// Custom method
	user, err := db.Users.FindActiveByEmail(ctx, "john@example.com")
	if err != nil {
		fmt.Printf("FindActiveByEmail: %v\n", err)
	} else {
		fmt.Printf("Found: %s\n", user.Name)
	}

	// Insert via inherited mongokit method
	user, err = db.Users.InsertOne(ctx, &User{
		Name:   "John Doe",
		Email:  "john@example.com",
		Status: "active",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted: %s (ID: %s)\n", user.Name, user.ID.Hex())

	// Custom method again
	active, err := db.Users.FindByStatus(ctx, "active")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Active users: %d\n", len(active))

	// Custom method - deactivate
	if err := db.Users.Deactivate(ctx, user.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Println("User deactivated")

	// Tasks - standard mongokit repository, no custom methods
	task, err := db.Tasks.InsertOne(ctx, &Task{
		Title:  "Write tests",
		UserID: user.ID,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Task created: %s\n", task.Title)

	// Cleanup
	db.Users.DeleteMany(ctx, bson.M{})
	db.Tasks.DeleteMany(ctx, bson.M{})
}
