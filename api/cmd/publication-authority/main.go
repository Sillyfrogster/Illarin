package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	handle := flag.String("handle", "", "the handle of the account that holds publication authority")
	flag.Parse()
	if err := run(context.Background(), *handle); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, handle string) error {
	if handle == "" {
		return errors.New("set -handle to the account that holds publication authority")
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

	accountID, err := publication.NewService(pool, nil).AssignAuthority(ctx, handle)
	if errors.Is(err, publication.ErrAccountNotFound) {
		return fmt.Errorf("no account uses the handle %q", handle)
	}
	if err != nil {
		return err
	}
	fmt.Printf("account %s holds publication authority\n", accountID)
	return nil
}
