// Package jscode reads the calls and imports in JavaScript or TypeScript source without running it.
package jscode

// maxCallee is how many names of a member chain a call keeps, counting back from the method.
const maxCallee = 4

// File is what Read found in one source file.
type File struct {
	Calls []Call
	// Imports names the modules the file imports, as it spells them, leaving out those it imports only for their types.
	Imports []string
}

// Call is one call to a method, with the names it was reached through, such as spindle.registerTool.
type Call struct {
	Callee []string
	source []byte
	open   int
}

// Value is the source of one expression, read only as far as it is written out literally.
type Value struct {
	source     []byte
	start, end int
}

// Read finds the calls in source to any of the named methods, and the modules it imports by literal name.
func Read(source []byte, methods ...string) File {
	wanted := make(map[string]bool, len(methods))
	for _, method := range methods {
		wanted[method] = true
	}
	var file File
	var recent trail
	typesOnly := -1
	lex := lexer{source: source}
	for {
		next, ok := lex.next()
		if !ok {
			return file
		}
		if next.is(source, "(") {
			if callee := recent.callee(source); len(callee) > 0 && wanted[callee[len(callee)-1]] {
				file.Calls = append(file.Calls, Call{Callee: callee, source: source, open: next.start})
			}
		}
		if module, ok := recent.imported(source, next); ok && next.start != typesOnly {
			file.Imports = append(file.Imports, module)
		}
		if module, ok := recent.typeImport(source, next); ok {
			typesOnly = module
		}
		recent.push(next)
	}
}

// Arg returns the argument at index, or nothing when the call has no such argument.
func (c Call) Arg(index int) Value {
	if c.source == nil {
		return Value{}
	}
	args, _, ok := list(c.source, c.open)
	if !ok || index >= len(args) {
		return Value{}
	}
	return args[index]
}

// Literal returns the text of a string written out in full.
func (v Value) Literal() (string, bool) {
	if v.source == nil {
		return "", false
	}
	lex := lexer{source: v.source, at: v.start}
	only, ok := lex.next()
	if !ok || only.kind != text || only.end != v.end {
		return "", false
	}
	return unquote(v.source[only.start:only.end])
}

// Property returns the value an object literal gives key, or nothing when the value is no object literal.
func (v Value) Property(key string) Value {
	var found Value
	for _, entry := range v.entries("{") {
		lex := lexer{source: v.source, at: entry.start}
		named, _ := lex.next()
		colon, _ := lex.next()
		value, ok := lex.next()
		if !ok || value.start >= entry.end || !colon.is(v.source, ":") || !namesKey(v.source, named, key) {
			continue
		}
		found = Value{source: v.source, start: value.start, end: entry.end}
	}
	return found
}

// Items returns what an array literal holds, or nothing when the value is no array literal.
func (v Value) Items() []Value { return v.entries("[") }

// Member returns the names of a chain such as event_types.CHAT_CHANGED, or nothing when the value is anything else.
func (v Value) Member() []string {
	if v.source == nil {
		return nil
	}
	lex := lexer{source: v.source, at: v.start}
	var names []string
	for {
		part, ok := lex.next()
		if !ok || part.kind != name || part.end > v.end {
			return nil
		}
		names = append(names, part.spelled(v.source))
		if part.end == v.end {
			return names
		}
		if dot, ok := lex.next(); !ok || !dot.isDot(v.source) || dot.end >= v.end {
			return nil
		}
	}
}

// Call returns the call a value makes, such as SlashCommand.fromProps({ ... }), or nothing when it is anything else.
func (v Value) Call() Call {
	if v.source == nil {
		return Call{}
	}
	lex := lexer{source: v.source, at: v.start}
	var recent trail
	for {
		next, ok := lex.next()
		if !ok || next.end > v.end {
			return Call{}
		}
		if next.is(v.source, "(") {
			callee := recent.callee(v.source)
			_, closed, ok := list(v.source, next.start)
			if len(callee) == 0 || recent.count != 2*len(callee)-1 || !ok || closed != v.end {
				return Call{}
			}
			return Call{Callee: callee, source: v.source, open: next.start}
		}
		recent.push(next)
	}
}

// entries splits a bracketed literal into what sits between its commas, when the value is exactly that literal.
func (v Value) entries(opener string) []Value {
	if v.source == nil {
		return nil
	}
	lex := lexer{source: v.source, at: v.start}
	first, ok := lex.next()
	if !ok || !first.is(v.source, opener) {
		return nil
	}
	entries, closed, ok := list(v.source, first.start)
	if !ok || (closed != v.end && !castsTo(v.source, closed, v.end)) {
		return nil
	}
	return entries
}

// castsTo says whether what follows a literal up to end is a TypeScript type the literal is cast to or checked against.
func castsTo(source []byte, from, end int) bool {
	lex := lexer{source: source, at: from}
	keyword, ok := lex.next()
	if !ok || keyword.kind != name || keyword.end >= end {
		return false
	}
	spelled := keyword.spelled(source)
	return spelled == "as" || spelled == "satisfies"
}

func namesKey(source []byte, key token, want string) bool {
	switch key.kind {
	case name:
		return key.spelled(source) == want
	case text:
		spelled, ok := unquote(source[key.start:key.end])
		return ok && spelled == want
	default:
		return false
	}
}

// list reads the comma-separated parts of the brackets opened at open, and where the brackets close.
func list(source []byte, open int) ([]Value, int, bool) {
	lex := lexer{source: source, at: open + 1}
	var parts []Value
	part := Value{source: source, start: -1}
	flush := func() {
		if part.start >= 0 {
			parts = append(parts, part)
		}
		part = Value{source: source, start: -1}
	}
	depth := 0
	for {
		next, ok := lex.next()
		if !ok {
			return nil, 0, false
		}
		switch {
		case next.kind == punct && isCloser(source[next.start]):
			if depth == 0 {
				flush()
				return parts, next.end, true
			}
			depth--
		case next.kind == punct && isOpener(source[next.start]):
			depth++
		case depth == 0 && next.is(source, ","):
			flush()
			continue
		}
		if part.start < 0 {
			part.start = next.start
		}
		part.end = next.end
	}
}

func isOpener(c byte) bool { return c == '(' || c == '[' || c == '{' }

func isCloser(c byte) bool { return c == ')' || c == ']' || c == '}' }

// trail keeps the last few tokens read, enough to see the chain of names a call is made through.
type trail struct {
	tokens [2*maxCallee + 2]token
	count  int
}

func (t *trail) push(next token) {
	t.tokens[t.count%len(t.tokens)] = next
	t.count++
}

// back returns the token n places before the latest.
func (t *trail) back(n int) (token, bool) {
	if n >= t.count || n >= len(t.tokens) {
		return token{}, false
	}
	return t.tokens[(t.count-1-n)%len(t.tokens)], true
}

// follows says whether the token n places back is the punctuation spelled.
func (t *trail) follows(source []byte, n int, spelled string) bool {
	before, ok := t.back(n)
	return ok && before.is(source, spelled)
}

// callee reads the member chain that ends just before an opening parenthesis, leaving out a function being declared.
func (t *trail) callee(source []byte) []string {
	at := 0
	if t.follows(source, 0, "?.") {
		at = 1
	}
	method, ok := t.back(at)
	if !ok || method.kind != name {
		return nil
	}
	names := []string{method.spelled(source)}
	for len(names) < maxCallee {
		dot, dotOK := t.back(at + 1)
		owner, ownerOK := t.back(at + 2)
		if !dotOK || !ownerOK || !dot.isDot(source) || owner.kind != name {
			break
		}
		names = append(names, owner.spelled(source))
		at += 2
	}
	if before, ok := t.back(at + 1); ok && before.kind == name && before.spelled(source) == "function" {
		return nil
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return names
}

// imported reads the module named when next completes an import, whether static, dynamic or required.
func (t *trail) imported(source []byte, next token) (string, bool) {
	switch {
	case next.kind == text:
		keyword, ok := t.back(0)
		if !ok || keyword.kind != name {
			return "", false
		}
		spelled := keyword.spelled(source)
		if spelled == "from" || (spelled == "import" && !t.follows(source, 1, ".")) {
			return unquote(source[next.start:next.end])
		}
	case next.is(source, ")"):
		module, _ := t.back(0)
		keyword, _ := t.back(2)
		if module.kind != text || !t.follows(source, 1, "(") || keyword.kind != name || t.follows(source, 3, ".") {
			return "", false
		}
		if spelled := keyword.spelled(source); spelled == "import" || spelled == "require" {
			return unquote(source[module.start:module.end])
		}
	}
	return "", false
}

// maxTypeClause is how many tokens an import of types may hold before the module it names.
const maxTypeClause = 64

// typeImport finds where the module is named when next is the type keyword of an import or export of types alone.
func (t *trail) typeImport(source []byte, next token) (int, bool) {
	keyword, ok := t.back(0)
	if !ok || next.kind != name || next.spelled(source) != "type" || keyword.kind != name || t.follows(source, 1, ".") {
		return 0, false
	}
	if spelled := keyword.spelled(source); spelled != "import" && spelled != "export" {
		return 0, false
	}
	lex := lexer{source: source, at: next.end}
	var clause []token
	for len(clause) < maxTypeClause {
		ahead, ok := lex.next()
		if !ok {
			return 0, false
		}
		if ahead.kind == text {
			return ahead.start, typeClause(source, clause)
		}
		clause = append(clause, ahead)
	}
	return 0, false
}

// typeClause says whether the tokens between a type keyword and its module name only types.
func typeClause(source []byte, clause []token) bool {
	last := len(clause) - 1
	if last < 1 || clause[last].kind != name || clause[last].spelled(source) != "from" {
		return false
	}
	body := clause[:last]
	switch {
	case len(body) == 1:
		return body[0].kind == name || body[0].is(source, "*")
	case len(body) == 3 && body[0].is(source, "*"):
		return body[1].kind == name && body[1].spelled(source) == "as" && body[2].kind == name
	case body[0].is(source, "{") && body[len(body)-1].is(source, "}"):
		for _, inside := range body[1 : len(body)-1] {
			if inside.kind != name && !inside.is(source, ",") {
				return false
			}
		}
		return true
	}
	return false
}
