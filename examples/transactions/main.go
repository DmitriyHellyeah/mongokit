package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/DmitriyHellyeah/mongokit"
)

type Account struct {
	mongokit.BaseField `bson:",inline"`
	Owner              string  `bson:"owner"`
	Balance            float64 `bson:"balance"`
}

func (*Account) CollectionName() string { return "accounts" }

func main() {
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	database := client.Database("example")

	accounts := mongokit.NewRepository[*Account](ctx, database)

	// Create two accounts
	alice, err := accounts.InsertOne(ctx, &Account{Owner: "Alice", Balance: 1000})
	if err != nil {
		log.Fatal(err)
	}

	bob, err := accounts.InsertOne(ctx, &Account{Owner: "Bob", Balance: 500})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Before: Alice=%.0f, Bob=%.0f\n", alice.Balance, bob.Balance)

	// Transfer $200 from Alice to Bob inside a transaction.
	// If any operation fails, both are rolled back.
	err = accounts.Transaction(ctx, func(sessCtx context.Context) error {
		_, err := accounts.UpdateByID(sessCtx, alice.ID, map[string]any{
			"$inc": map[string]any{"balance": -200},
		})
		if err != nil {
			return fmt.Errorf("debit alice: %w", err)
		}

		_, err = accounts.UpdateByID(sessCtx, bob.ID, map[string]any{
			"$inc": map[string]any{"balance": 200},
		})
		if err != nil {
			return fmt.Errorf("credit bob: %w", err)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Verify
	alice, _ = accounts.FindByID(ctx, alice.ID)
	bob, _ = accounts.FindByID(ctx, bob.ID)
	fmt.Printf("After:  Alice=%.0f, Bob=%.0f\n", alice.Balance, bob.Balance)
}
