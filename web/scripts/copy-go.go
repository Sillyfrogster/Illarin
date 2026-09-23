package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type copyText struct {
	Text string `json:"text"`
	File string `json:"file"`
	Line int    `json:"line"`
}

func main() {
	if err := extract(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func extract(root string) error {
	positions := token.NewFileSet()
	output := json.NewEncoder(os.Stdout)
	return filepath.WalkDir(filepath.Join(root, "api/internal"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "db", "apitest", "testdb", "testdata", "config":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(positions, path, nil, 0)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		var writeErr error
		var visit func(ast.Node) bool
		visit = func(node ast.Node) bool {
			if node == nil || writeErr != nil {
				return false
			}
			switch node := node.(type) {
			case *ast.CallExpr:
				if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
					if selector.Sel.Name == "Header" || selector.Sel.Name == "GetHeader" {
						return false
					}
					if name, ok := selector.X.(*ast.Ident); ok && (name.Name == "log" || name.Name == "slog") {
						return false
					}
				}
			case *ast.ImportSpec:
				return false
			case *ast.Field:
				return false
			case *ast.KeyValueExpr:
				ast.Inspect(node.Value, visit)
				return false
			case *ast.BinaryExpr:
				if node.Op == token.ADD && hasText(node) {
					writeErr = output.Encode(copyText{joinedText(node, positions), relative, positions.Position(node.Pos()).Line})
					return false
				}
				if node.Op == token.EQL || node.Op == token.NEQ {
					return false
				}
			case *ast.BasicLit:
				if node.Kind == token.STRING {
					value, _ := strconv.Unquote(node.Value)
					writeErr = output.Encode(copyText{value, relative, positions.Position(node.Pos()).Line})
				}
			}
			return true
		}
		ast.Inspect(file, visit)
		return writeErr
	})
}

func hasText(node ast.Expr) bool {
	switch node := node.(type) {
	case *ast.BasicLit:
		return node.Kind == token.STRING
	case *ast.BinaryExpr:
		return node.Op == token.ADD && (hasText(node.X) || hasText(node.Y))
	}
	return false
}

func joinedText(node ast.Expr, positions *token.FileSet) string {
	if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		value, _ := strconv.Unquote(literal.Value)
		return value
	}
	if binary, ok := node.(*ast.BinaryExpr); ok && binary.Op == token.ADD {
		return joinedText(binary.X, positions) + joinedText(binary.Y, positions)
	}
	var expression bytes.Buffer
	_ = printer.Fprint(&expression, positions, node)
	return "${" + expression.String() + "}"
}
