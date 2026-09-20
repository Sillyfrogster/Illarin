// Command make-admin makes an existing account an admin
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	handle := flag.String("handle", "", "the handle of the account to make an admin")
	flag.Parse()
	if err := run(context.Background(), *handle); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, handle string) error {
	if handle == "" {
		return errors.New("set -handle to the account to make an admin")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	made, err := pool.Exec(ctx, `update users set role = 'admin' where username = $1`, handle)
	if err != nil {
		return fmt.Errorf("make admin: %w", err)
	}
	if made.RowsAffected() == 0 {
		return fmt.Errorf("no account uses the handle %q", handle)
	}
	fmt.Printf("%s is an admin\n", handle)
	return nil
}
