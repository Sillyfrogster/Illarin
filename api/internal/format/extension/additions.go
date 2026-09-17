package extension

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/jscode"
	"github.com/google/uuid"
)

const (
	// maxCodeBytes is how much code is read for what an extension adds, as much as the largest archive may hold.
	maxCodeBytes = MaxArchiveBytes
	// readingTime is how long the code is read before the page makes do with what was found.
	readingTime = 5 * time.Second
	// maxAdditions is how many additions one page lists.
	maxAdditions = 1000
	// maxAdditionRunes is the longest name listed, since a longer one is no name a reader would type.
	maxAdditionRunes = 120
)

// group is a kind of thing an extension adds to its app, in the order a page lists them.
type group int

const (
	commands group = iota
	macros
	tools
	surfaces
	hooks
)

var groupTitles = [...]string{
	commands: "Commands", macros: "Macros", tools: "Tools", surfaces: "UI surfaces", hooks: "Generation hooks",
}

// addition is one thing an extension's code registers with its app.
type addition struct {
	group group
	name  string
}

// rule reads what one method of an app's extension interface registers, from a call made on the object it belongs to.
type rule struct {
	method string
	on     func(callee []string) bool
	read   func(call jscode.Call) []addition
}

var codeEndings = []string{".js", ".mjs", ".cjs", ".jsx", ".ts", ".mts", ".cts", ".tsx"}

// readAdditions reads what an extension's code adds to its app, from the files the app loads and the files they import.
func readAdditions(
	ctx context.Context, file format.Inspection, entries []string, rules []rule, limit time.Duration,
) (*block.Element, error) {
	reading, stop := context.WithTimeout(ctx, limit)
	defer stop()
	archive, err := file.OpenZIPFiles(reading)
	if err != nil {
		return nil, format.InternalFailure(fmt.Errorf("open the archive to read its code: %w", err))
	}
	code := codeFiles(file)
	methods := make([]string, 0, len(rules))
	for _, rule := range rules {
		methods = append(methods, rule.method)
	}
	var found additionList
	queue := slices.DeleteFunc(slices.Clone(entries), func(entry string) bool { return !code[entry] })
	queued := make(map[string]bool, len(queue))
	for _, entry := range queue {
		queued[entry] = true
	}
	budget := maxCodeBytes
	for len(queue) > 0 && budget > 0 {
		name := queue[0]
		queue = queue[1:]
		source, err := readCode(archive, file.ArchiveBase+name, budget)
		if reading.Err() != nil {
			break
		}
		if errors.Is(err, format.ErrRangeRead) {
			return nil, format.InternalFailure(fmt.Errorf("read %s: %w", name, err))
		}
		if err != nil {
			continue
		}
		budget -= len(source)
		read := jscode.Read(source, methods...)
		for _, call := range read.Calls {
			found.add(applied(rules, call)...)
		}
		for _, module := range read.Imports {
			if next, ok := resolveImport(name, module, code); ok && !queued[next] {
				queued[next] = true
				queue = append(queue, next)
			}
		}
	}
	return found.element(), nil
}

func applied(rules []rule, call jscode.Call) []addition {
	var found []addition
	method := call.Callee[len(call.Callee)-1]
	for _, rule := range rules {
		if rule.method == method && rule.on(call.Callee) {
			found = append(found, rule.read(call)...)
		}
	}
	return found
}

func readCode(archive format.ZIPFiles, name string, budget int) ([]byte, error) {
	opened, err := archive.Open(name)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	return io.ReadAll(io.LimitReader(opened, int64(budget)))
}

// codeFiles lists the JavaScript and TypeScript files in the archive, by their names inside its folder.
func codeFiles(file format.Inspection) map[string]bool {
	code := make(map[string]bool)
	for _, entry := range file.ZIPEntries {
		name, inside := strings.CutPrefix(entry.Name, file.ArchiveBase)
		declaration := strings.HasSuffix(name, ".d.ts") || strings.HasSuffix(name, ".d.mts") || strings.HasSuffix(name, ".d.cts")
		if inside && !entry.Directory && !declaration && slices.Contains(codeEndings, path.Ext(name)) {
			code[name] = true
		}
	}
	return code
}

// resolveImport finds the file a relative import loads, trying the endings a bundler or TypeScript would.
func resolveImport(from, module string, code map[string]bool) (string, bool) {
	module, _, _ = strings.Cut(module, "?")
	module, _, _ = strings.Cut(module, "#")
	if !strings.HasPrefix(module, "./") && !strings.HasPrefix(module, "../") {
		return "", false
	}
	target := path.Join(path.Dir(from), module)
	if !insideArchive(target) {
		return "", false
	}
	candidates := []string{target}
	if ending := path.Ext(target); ending == ".js" || ending == ".jsx" || ending == ".mjs" || ending == ".cjs" {
		stem := strings.TrimSuffix(target, ending)
		candidates = append(candidates, stem+".ts", stem+".tsx", stem+".mts", stem+".cts")
	}
	for _, ending := range codeEndings {
		candidates = append(candidates, target+ending, target+"/index"+ending)
	}
	for _, candidate := range candidates {
		if code[candidate] {
			return candidate, true
		}
	}
	return "", false
}

// additionList keeps each addition once, every group in the order its additions were found.
type additionList struct {
	byGroup [len(groupTitles)][]string
	seen    map[addition]bool
	count   int
}

func (l *additionList) add(found ...addition) {
	for _, one := range found {
		if l.count >= maxAdditions || l.seen[one] {
			continue
		}
		if l.seen == nil {
			l.seen = make(map[addition]bool)
		}
		l.seen[one] = true
		l.byGroup[one.group] = append(l.byGroup[one.group], one.name)
		l.count++
	}
}

func (l *additionList) element() *block.Element {
	if l.count == 0 {
		return nil
	}
	fields := make([]block.FieldItem, 0, l.count)
	for group, names := range l.byGroup {
		for _, name := range names {
			fields = append(fields, block.FieldItem{ID: block.NewItemID(), Name: groupTitles[group], Value: name})
		}
	}
	return &block.Element{
		ID: uuid.New(), Type: block.TypeFieldList, Role: block.RoleExtensionAdditions,
		Content: block.FieldList{Fields: fields},
	}
}

// named reads the name an object argument gives what the call registers.
func named(group group, key string, spell func(string) string) func(jscode.Call) []addition {
	return func(call jscode.Call) []addition { return one(group, call.Arg(0).Property(key), spell) }
}

// first reads the name a call registers from its first argument.
func first(group group, spell func(string) string) func(jscode.Call) []addition {
	return func(call jscode.Call) []addition { return one(group, call.Arg(0), spell) }
}

func one(group group, value jscode.Value, spell func(string) string) []addition {
	name, ok := literalName(value)
	if !ok {
		return nil
	}
	return []addition{{group: group, name: spell(name)}}
}

// hook lists a hook the extension attaches to generation, which has no name of its own.
func hook(label string) func(jscode.Call) []addition {
	return func(jscode.Call) []addition { return []addition{{group: hooks, name: label}} }
}

// surface lists a part of the app's interface the extension adds, which has no title of its own.
func surface(label string) func(jscode.Call) []addition {
	return func(jscode.Call) []addition { return []addition{{group: surfaces, name: label}} }
}

// titledSurface lists a part of the app's interface, with its title where the code writes the title out.
func titledSurface(label, key string) func(jscode.Call) []addition {
	return func(call jscode.Call) []addition {
		if title, ok := literalName(call.Arg(0).Property(key)); ok {
			return []addition{{group: surfaces, name: label + ": " + title}}
		}
		return []addition{{group: surfaces, name: label}}
	}
}

// literalName reads a name written out in full, leaving out one no reader could type.
func literalName(value jscode.Value) (string, bool) {
	name, ok := value.Literal()
	name = strings.TrimSpace(name)
	if !ok || name == "" || utf8.RuneCountInString(name) > maxAdditionRunes || strings.ContainsFunc(name, unicode.IsControl) {
		return "", false
	}
	return name, true
}

func asWritten(name string) string { return name }

func slash(name string) string { return "/" + strings.TrimPrefix(name, "/") }

func macro(name string) string {
	return "{{" + strings.TrimSuffix(strings.TrimPrefix(name, "{{"), "}}") + "}}"
}

// owner accepts a call made on the object with this name.
func owner(object string) func([]string) bool {
	return func(callee []string) bool { return len(callee) >= 2 && callee[len(callee)-2] == object }
}

func onAnything([]string) bool { return true }
