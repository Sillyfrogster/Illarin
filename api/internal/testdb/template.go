package testdb

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const templatePrefix = "illarin_test_tpl_"

// server is one Postgres server the tests clone databases on, from a template holding the current schema
type server struct {
	address  *url.URL
	admin    *pgxpool.Pool
	template string
}

var (
	serversMu sync.Mutex
	servers   = map[string]*server{}
)

// serverAt returns the server for the address, building its template the first time any test asks
func serverAt(ctx context.Context, address string) (*server, error) {
	serversMu.Lock()
	defer serversMu.Unlock()
	if found, ok := servers[address]; ok {
		return found, nil
	}

	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme == "" {
		return nil, errors.New("TEST_DATABASE_URL must be a postgres:// URL")
	}
	admin, err := pgxpool.New(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("connect to the test server: %w", err)
	}
	found := &server{address: parsed, admin: admin}
	if err := found.prepareTemplate(ctx); err != nil {
		admin.Close()
		return nil, err
	}
	servers[address] = found
	return found, nil
}

// databaseURL points the server address at one database and turns off commit durability, which tests never need
func (s *server) databaseURL(name string) string {
	address := *s.address
	address.Path = "/" + name
	query := address.Query()
	query.Set("synchronous_commit", "off")
	address.RawQuery = query.Encode()
	return address.String()
}

func (s *server) create(ctx context.Context, name string) error {
	_, err := s.admin.Exec(ctx, "create database "+pgx.Identifier{name}.Sanitize()+
		" template "+pgx.Identifier{s.template}.Sanitize())
	return err
}

func (s *server) drop(ctx context.Context, name string) error {
	_, err := s.admin.Exec(ctx, "drop database if exists "+pgx.Identifier{name}.Sanitize())
	return err
}

// prepareTemplate names the template after a hash of the migrations and builds it once per schema, across every test process
func (s *server) prepareTemplate(ctx context.Context) error {
	dir, digest, err := readMigrations()
	if err != nil {
		return err
	}
	s.template = templatePrefix + hex.EncodeToString(digest[:6])
	if ready, err := s.templateReady(ctx); err != nil || ready {
		return err
	}

	lock, err := s.admin.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("connect to the test server: %w", err)
	}
	defer lock.Release()
	key := int64(binary.BigEndian.Uint64(digest[:8]))
	if _, err := lock.Exec(ctx, "select pg_advisory_lock($1)", key); err != nil {
		return fmt.Errorf("wait for the template build: %w", err)
	}
	defer lock.Exec(context.Background(), "select pg_advisory_unlock($1)", key)

	if ready, err := s.templateReady(ctx); err != nil || ready {
		return err
	}
	return s.buildTemplate(ctx, dir)
}

func (s *server) templateReady(ctx context.Context) (bool, error) {
	var ready bool
	err := s.admin.QueryRow(ctx,
		"select coalesce((select datistemplate from pg_database where datname = $1), false)",
		s.template).Scan(&ready)
	if err != nil {
		return false, fmt.Errorf("look for the template: %w", err)
	}
	return ready, nil
}

func (s *server) buildTemplate(ctx context.Context, dir string) error {
	identifier := pgx.Identifier{s.template}.Sanitize()
	if _, err := s.admin.Exec(ctx, "drop database if exists "+identifier); err != nil {
		return fmt.Errorf("clear a half-built template: %w", err)
	}
	if _, err := s.admin.Exec(ctx, "create database "+identifier+" template template0"); err != nil {
		return fmt.Errorf("create the template: %w", err)
	}
	if err := migrate(ctx, s.databaseURL(s.template), dir); err != nil {
		return err
	}
	if _, err := s.admin.Exec(ctx, "alter database "+identifier+" is_template true"); err != nil {
		return fmt.Errorf("mark the template: %w", err)
	}
	return nil
}

// readMigrations hashes every migration file, so a changed migration gets a fresh template and invalidates Go's test cache
func readMigrations() (string, [sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return "", digest, errors.New("the migrations directory cannot be located")
	}
	dir := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "migrations"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", digest, fmt.Errorf("list the migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "", digest, fmt.Errorf("read migration %s: %w", name, err)
		}
		fmt.Fprintf(hash, "%s\x00%d\x00", name, len(body))
		hash.Write(body)
	}
	copy(digest[:], hash.Sum(nil))
	return dir, digest, nil
}

func migrate(ctx context.Context, address, dir string) error {
	db, err := sql.Open("pgx", address)
	if err != nil {
		return fmt.Errorf("open the template: %w", err)
	}
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(dir))
	if err != nil {
		return fmt.Errorf("read the migrations: %w", err)
	}
	defer provider.Close()
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate the template: %w", err)
	}
	return nil
}
