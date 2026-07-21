package anchor

import (
	"regexp"
	"strings"

	"github.com/canhta/cred/internal/claim"
)

// CodeAnchorer is the ladder for source code. It reuses the exact shape the text
// anchorer proved: a definition is a heading, its indentation is the heading
// level, and its section runs to the next definition at the same or shallower
// depth. Tier 1 is the nested symbol path ("Executor > writeOne"); tier 2 is the
// normalized hash of that immediate section.
//
// It reads no syntax tree. Tier 1 does not need a compiler's parse — it needs a
// relocatable identifier that survives line moves and a hash that survives
// reformatting, and the ladder's law (a claim expires only when tiers 1 and 2
// disagree, never on ambiguity) contains what a heuristic scanner cannot know.
// A pure-Go tree-sitter that honours CGO_ENABLED=0 does not exist; every binding
// wraps the C library. A structural scanner is the honest alternative — it
// degrades to tier-4-only on a language it does not recognize rather than
// pretending to a symbol path it cannot produce.
//
// Language detection is a table: each extension maps to a langSpec, an ordered
// list of regex rules. Adding a language is adding a data entry, not code. The
// patterns are RE2-native (no backreferences, no lookaround) and tuned for
// precision — a false-positive anchor re-validates a claim against the wrong
// code, which is worse than no anchor at all, so a line that does not clearly
// open a named definition matches nothing and falls back to tier 4.
type CodeAnchorer struct{}

// Kind reports that this anchorer handles code evidence.
func (CodeAnchorer) Kind() claim.SourceKind { return claim.SourceCode }

// Compute builds the ladder for span within src. Tier 1 is the symbol path at
// the span's first line; tier 2 hashes the enclosing definition's immediate
// section; tier 4 is the raw span hash the caller already has.
func (CodeAnchorer) Compute(src Source, span Span) Anchor {
	a := Anchor{ByteHash: span.ByteHash}

	defs := parseDefs(src.Text, langOf(src.Path))
	if h, ok := enclosing(defs, span.LineStart); ok {
		a.SymbolPath = h.path
		a.NodeHash = hashNormalized(sectionText(src.Text, h.start, h.end))
	}

	a.WindowHash = hashNormalized(sectionText(src.Text, span.LineStart-windowRadius, span.LineEnd+windowRadius))
	return a
}

// Resolve relocates the stored symbol path in the current source and defers the
// decision to classify. A definition renamed or removed is notFound
// (SemanticChange); two definitions sharing a path is ambiguous, which for valid
// code is near-impossible and, when it happens, expires rather than guesses.
func (CodeAnchorer) Resolve(stored Anchor, src Source) Verdict {
	if !stored.Anchored() {
		return classify(stored, located{})
	}

	var matches []heading
	for _, h := range parseDefs(src.Text, langOf(src.Path)) {
		if h.path == stored.SymbolPath {
			matches = append(matches, h)
		}
	}

	switch len(matches) {
	case 0:
		return classify(stored, located{status: notFound})
	case 1:
		h := matches[0]
		return classify(stored, located{
			status:   found,
			nodeHash: hashNormalized(sectionText(src.Text, h.start, h.end)),
		})
	default:
		return classify(stored, located{status: ambiguous})
	}
}

// KindForPath guesses the evidence kind from a file extension: code for a
// recognized source language, document otherwise. Capture uses it to route a
// captured file span to the anchorer that fits it.
func KindForPath(path string) claim.SourceKind {
	if langOf(path) != "" {
		return claim.SourceCode
	}
	return claim.SourceDocument
}

// parseDefs finds every definition in code and returns it with its nested symbol
// path and the extent of the section it immediately owns — the same construction
// parseHeadings performs for Markdown, with indentation standing in for heading
// depth and a per-language matcher standing in for the "#" prefix.
//
// A definition's immediate section ends at the next definition of any depth, so
// a change inside a method does not expire a claim anchored to its enclosing
// class. An unrecognized language yields no definitions, and every span in it
// anchors tier-4-only.
func parseDefs(text, lang string) []heading {
	if lang == "" {
		return nil
	}
	lines := splitLines(text)
	var defs []heading
	var stack []heading

	for i, line := range lines {
		n := i + 1
		name, ok := matchDef(lang, line)
		if !ok {
			continue
		}
		level := indentCols(line)

		if len(defs) > 0 {
			defs[len(defs)-1].end = n - 1
		}
		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			stack = stack[:len(stack)-1]
		}
		path := name
		if len(stack) > 0 {
			path = stack[len(stack)-1].path + " > " + name
		}
		h := heading{level: level, path: path, start: n, end: len(lines)}
		defs = append(defs, h)
		stack = append(stack, h)
	}
	return defs
}

// indentCols is the visual indentation of a line, tabs counted as four columns.
// It is the code analogue of a heading's "#" count: deeper indentation is a
// deeper node. Reformatting changes absolute indentation but not the relative
// ordering of nested definitions, which is all the section extent depends on.
func indentCols(line string) int {
	cols := 0
	for _, r := range line {
		switch r {
		case ' ':
			cols++
		case '\t':
			cols += 4
		default:
			return cols
		}
	}
	return cols
}

// langOf maps a file extension to a registry key, or "" when the extension is
// not one CRED anchors structurally. The key groups extensions that share a
// grammar (all the TypeScript/JavaScript variants under "jsts", C and C++ header
// files under their language).
func langOf(path string) string {
	dot := strings.LastIndexByte(path, '.')
	if dot < 0 {
		return ""
	}
	switch strings.ToLower(path[dot+1:]) {
	case "go":
		return "go"
	case "ts", "tsx", "mts", "cts", "js", "jsx", "mjs", "cjs":
		return "jsts"
	case "py", "pyi":
		return "py"
	case "rs":
		return "rust"
	case "c", "h":
		return "c"
	case "cc", "cpp", "cxx", "hpp", "hh", "hxx":
		return "cpp"
	case "java":
		return "java"
	case "cs":
		return "csharp"
	case "rb":
		return "ruby"
	case "php":
		return "php"
	case "swift":
		return "swift"
	case "kt", "kts":
		return "kotlin"
	case "scala", "sc":
		return "scala"
	case "css", "scss", "less", "sass":
		return "css"
	case "html", "htm":
		return "html"
	default:
		return ""
	}
}

// defRule is one pattern that opens a named definition. The regex has at least
// one capture group; group 1 is the name. With joinReceiver and a second group
// the name is "receiver.member" (a Go method carries its receiver type so two
// types' same-named methods never collide, and a Kotlin extension function
// carries its receiver likewise). needsIndent restricts a rule to indented
// lines, which is how a class method is told apart from a same-shaped call or
// control statement at column zero. reject drops matches whose name is a control
// keyword the pattern is too permissive to exclude on its own.
type defRule struct {
	re           *regexp.Regexp
	joinReceiver bool
	needsIndent  bool
	reject       map[string]bool
}

// langSpec is the ordered rule set for one language. First match wins, so more
// specific rules precede more permissive ones.
type langSpec struct{ rules []defRule }

// matchDef reports whether a line opens a named definition and, if so, the name
// segment for its symbol path. It walks the language's rules in order and takes
// the first that matches, applying the rule's indent gate and reject set.
func matchDef(lang, line string) (string, bool) {
	spec, ok := registry[lang]
	if !ok {
		return "", false
	}
	t := strings.TrimSpace(line)
	if t == "" {
		return "", false
	}
	for _, r := range spec.rules {
		if r.needsIndent && indentCols(line) == 0 {
			continue
		}
		m := r.re.FindStringSubmatch(t)
		if m == nil {
			continue
		}
		name := m[1]
		if r.joinReceiver && len(m) > 2 {
			switch {
			case m[1] != "" && m[2] != "":
				name = m[1] + "." + m[2]
			case m[2] != "":
				name = m[2]
			}
		}
		name = strings.TrimSpace(name)
		if name == "" || (r.reject != nil && r.reject[name]) {
			continue
		}
		return name, true
	}
	return "", false
}

// Shared prefixes for the visibility and modifier keywords a definition line can
// carry before its keyword. Kept as fragments so each language composes only the
// ones its grammar allows.
const (
	rustVis  = `(?:pub(?:\s*\([^)]*\))?\s+)?`
	javaMod  = `(?:(?:public|private|protected|abstract|final|static|sealed|strictfp)\s+)*`
	csMod    = `(?:(?:public|private|protected|internal|abstract|sealed|static|partial|readonly|virtual|override|async|unsafe|new|extern)\s+)*`
	csModOne = `(?:(?:public|private|protected|internal|abstract|sealed|static|partial|readonly|virtual|override|async|unsafe|new|extern)\s+)+`
	swiftMod = `(?:(?:public|private|internal|fileprivate|open|final|static|class|override|mutating|convenience|required|dynamic)\s+)*`
	ktClsMod = `(?:(?:public|private|protected|internal|open|abstract|sealed|data|final|inner|enum|annotation|value)\s+)*`
	ktFunMod = `(?:(?:public|private|protected|internal|open|override|suspend|inline|operator|infix|tailrec|external|final)\s+)*`
	scalaMod = `(?:(?:private|protected|final|sealed|abstract|implicit|lazy|case|override)\s+)*`
)

var (
	goMethod  = regexp.MustCompile(`^func\s+\(\s*\w+\s+\*?([A-Za-z_]\w*)(?:\[[^\]]*\])?\s*\)\s+([A-Za-z_]\w*)`)
	goFunc    = regexp.MustCompile(`^func\s+([A-Za-z_]\w*)`)
	goType    = regexp.MustCompile(`^type\s+([A-Za-z_]\w*)`)
	goVarFunc = regexp.MustCompile(`^var\s+([A-Za-z_]\w*)\s*=\s*func\b`)

	pyDef   = regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_]\w*)`)
	pyClass = regexp.MustCompile(`^class\s+([A-Za-z_]\w*)`)

	jsFunc    = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][\w$]*)`)
	jsClass   = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)`)
	jsDecl    = regexp.MustCompile(`^(?:export\s+)?(?:declare\s+)?(?:interface|enum|namespace|module)\s+([A-Za-z_$][\w$]*)`)
	jsType    = regexp.MustCompile(`^(?:export\s+)?(?:declare\s+)?type\s+([A-Za-z_$][\w$]*)`)
	jsConstFn = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\b.*=\s*(?:async\s+)?(?:function\b|(?:<[^>]*>\s*)?\([^)]*\)\s*(?::[^=>]+)?=>|[A-Za-z_$][\w$]*\s*=>)`)
	jsMethod  = regexp.MustCompile(`^(?:(?:public|private|protected|static|readonly|async|get|set)\s+|\*\s*)*([A-Za-z_$][\w$]*)\s*(?:<[^>]*>)?\s*\([^;]*\)\s*(?::\s*[^;{]+)?\s*\{`)

	rustFn     = regexp.MustCompile(`^` + rustVis + `(?:default\s+)?(?:async\s+)?(?:const\s+)?(?:unsafe\s+)?(?:extern\s+"[^"]*"\s+)?fn\s+([A-Za-z_]\w*)`)
	rustMod    = regexp.MustCompile(`^` + rustVis + `mod\s+([A-Za-z_]\w*)`)
	rustTrait  = regexp.MustCompile(`^` + rustVis + `(?:unsafe\s+)?trait\s+([A-Za-z_]\w*)`)
	rustImpl   = regexp.MustCompile(`^(?:unsafe\s+)?impl(?:\s*<[^>]*>)?\s+(?:.*\bfor\s+)?([A-Za-z_]\w*)`)
	rustStruct = regexp.MustCompile(`^` + rustVis + `struct\s+([A-Za-z_]\w*)`)
	rustEnum   = regexp.MustCompile(`^` + rustVis + `enum\s+([A-Za-z_]\w*)`)
	rustType   = regexp.MustCompile(`^` + rustVis + `type\s+([A-Za-z_]\w*)`)
	rustConst  = regexp.MustCompile(`^` + rustVis + `(?:const|static)\s+(?:mut\s+)?([A-Za-z_]\w*)`)
	rustMacro  = regexp.MustCompile(`^macro_rules!\s+([A-Za-z_]\w*)`)

	cMacro   = regexp.MustCompile(`^#\s*define\s+([A-Za-z_]\w*)`)
	cRecord  = regexp.MustCompile(`^(?:typedef\s+)?(?:struct|union|enum)\s+([A-Za-z_]\w*)`)
	cTypedef = regexp.MustCompile(`^typedef\s+[\w\s*]+?\b([A-Za-z_]\w*)\s*;`)
	cFunc    = regexp.MustCompile(`^(?:[A-Za-z_][\w\s*]*?[\s*])([A-Za-z_]\w*)\s*\([^;{]*\)\s*\{`)

	cppNamespace = regexp.MustCompile(`^namespace\s+([A-Za-z_]\w*)`)
	cppClass     = regexp.MustCompile(`^(?:template\s*<[^>]*>\s*)?(?:(?:public|private|protected)\s+)?class\s+([A-Za-z_]\w*)`)
	cppStruct    = regexp.MustCompile(`^(?:template\s*<[^>]*>\s*)?struct\s+([A-Za-z_]\w*)`)
	cppEnum      = regexp.MustCompile(`^enum(?:\s+(?:class|struct))?\s+([A-Za-z_]\w*)`)
	cppUsing     = regexp.MustCompile(`^using\s+([A-Za-z_]\w*)\s*=`)
	cppFunc      = regexp.MustCompile(`^(?:[A-Za-z_~][\w\s*&:<>,]*?[\s*&>])([~A-Za-z_][\w:]*)\s*\([^;{]*\)\s*(?:const\s*)?(?:noexcept\s*)?(?:->[^{;]*)?\{`)

	javaPkg    = regexp.MustCompile(`^package\s+([\w.]+)`)
	javaClass  = regexp.MustCompile(`^` + javaMod + `class\s+([A-Za-z_]\w*)`)
	javaIface  = regexp.MustCompile(`^` + javaMod + `interface\s+([A-Za-z_]\w*)`)
	javaEnum   = regexp.MustCompile(`^` + javaMod + `enum\s+([A-Za-z_]\w*)`)
	javaRecord = regexp.MustCompile(`^` + javaMod + `record\s+([A-Za-z_]\w*)`)
	javaMethod = regexp.MustCompile(`^(?:@[A-Za-z_]\w*(?:\([^)]*\))?\s+)*(?:(?:public|private|protected|static|final|abstract|synchronized|native|default|strictfp)\s+)+(?:[\w<>\[\].,?\s]+\s+)?([A-Za-z_]\w*)\s*\([^;{]*\)\s*(?:throws[\w\s,.]*)?\{`)

	csNamespace = regexp.MustCompile(`^namespace\s+([\w.]+)`)
	csClass     = regexp.MustCompile(`^` + csMod + `class\s+([A-Za-z_]\w*)`)
	csIface     = regexp.MustCompile(`^` + csMod + `interface\s+([A-Za-z_]\w*)`)
	csStruct    = regexp.MustCompile(`^` + csMod + `struct\s+([A-Za-z_]\w*)`)
	csEnum      = regexp.MustCompile(`^` + csMod + `enum\s+([A-Za-z_]\w*)`)
	csRecord    = regexp.MustCompile(`^` + csMod + `record\s+([A-Za-z_]\w*)`)
	csProp      = regexp.MustCompile(`^` + csModOne + `[\w<>\[\].,?]+\s+([A-Za-z_]\w*)\s*\{\s*(?:get|set|init)\b`)
	csMethod    = regexp.MustCompile(`^` + csModOne + `(?:[\w<>\[\].,?\s]+\s+)?([A-Za-z_]\w*)\s*\([^;{]*\)\s*\{`)

	rbClass  = regexp.MustCompile(`^class\s+([A-Z][\w:]*)`)
	rbModule = regexp.MustCompile(`^module\s+([A-Z][\w:]*)`)
	rbDef    = regexp.MustCompile(`^def\s+([A-Za-z_][\w.]*[?!=]?)`)

	phpNamespace = regexp.MustCompile(`^namespace\s+([\w\\]+)`)
	phpClass     = regexp.MustCompile(`^(?:(?:abstract|final)\s+)?(?:class|interface|trait)\s+([A-Za-z_]\w*)`)
	phpFunc      = regexp.MustCompile(`^(?:(?:public|private|protected|static|abstract|final)\s+)*function\s+&?\s*([A-Za-z_]\w*)`)
	phpConst     = regexp.MustCompile(`^const\s+([A-Za-z_]\w*)`)

	swiftFunc      = regexp.MustCompile(`^` + swiftMod + `func\s+([A-Za-z_]\w*)`)
	swiftType      = regexp.MustCompile(`^` + swiftMod + `(?:class|struct|enum|protocol|extension|actor)\s+([A-Za-z_]\w*)`)
	swiftTypealias = regexp.MustCompile(`^(?:(?:public|private|internal|fileprivate|open)\s+)?typealias\s+([A-Za-z_]\w*)`)

	ktPackage = regexp.MustCompile(`^package\s+([\w.]+)`)
	ktClass   = regexp.MustCompile(`^` + ktClsMod + `(?:class|interface|object)\s+([A-Za-z_]\w*)`)
	ktFun     = regexp.MustCompile(`^` + ktFunMod + `fun\s+(?:<[^>]*>\s*)?(?:([A-Za-z_]\w*)\.)?([A-Za-z_]\w*)`)

	scalaPackage = regexp.MustCompile(`^package\s+(?:object\s+)?([\w.]+)`)
	scalaObject  = regexp.MustCompile(`^` + scalaMod + `object\s+([A-Za-z_]\w*)`)
	scalaClass   = regexp.MustCompile(`^` + scalaMod + `(?:class|trait)\s+([A-Za-z_]\w*)`)
	scalaDef     = regexp.MustCompile(`^` + scalaMod + `def\s+([A-Za-z_]\w*)`)
	scalaVal     = regexp.MustCompile(`^` + scalaMod + `va[lr]\s+([A-Za-z_]\w*)`)

	// A CSS rule opens with a selector or @-rule whose block starts at the line's
	// trailing "{"; that whole prelude is the relocatable name, and the block is
	// the extent. Requiring "{" at the line end skips property declarations and
	// single-line rules, which the extent construction could not bound anyway.
	cssAtRule   = regexp.MustCompile(`^(@[\w-]+[^{]*?)\s*\{$`)
	cssSelector = regexp.MustCompile(`^([^{}@;]+?)\s*\{$`)
)

// jsControl are the keywords a method-shaped line can begin with that are not
// methods. Without this, `if (x) {` inside a class body would anchor as a symbol.
var jsControl = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true,
	"catch": true, "return": true, "function": true, "do": true,
	"else": true, "with": true,
}

// cControl are the keywords a C/C++ function-shaped line can open with. The
// function pattern needs two tokens before the parameter list, which already
// rejects most control statements, but a call like `return foo() {` shape is
// excluded here for safety.
var cControl = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true,
	"return": true, "sizeof": true, "do": true, "else": true,
	"catch": true,
}

// registry maps a language key to its ordered rule set. Adding a language is one
// entry here plus its extensions in langOf. HTML is registered with no rules on
// purpose: line-oriented regex cannot bound an element's extent (its closing tag
// is not a definition and its nesting is not indentation), so HTML degrades to
// tier-4-only rather than emit anchors that would re-validate against the wrong
// markup.
var registry = map[string]langSpec{
	"go": {rules: []defRule{
		{re: goMethod, joinReceiver: true},
		{re: goFunc},
		{re: goType},
		{re: goVarFunc},
	}},
	"jsts": {rules: []defRule{
		{re: jsFunc},
		{re: jsClass},
		{re: jsDecl},
		{re: jsConstFn},
		{re: jsType},
		{re: jsMethod, needsIndent: true, reject: jsControl},
	}},
	"py": {rules: []defRule{
		{re: pyDef},
		{re: pyClass},
	}},
	"rust": {rules: []defRule{
		{re: rustMacro},
		{re: rustMod},
		{re: rustFn},
		{re: rustTrait},
		{re: rustImpl},
		{re: rustStruct},
		{re: rustEnum},
		{re: rustType},
		{re: rustConst},
	}},
	"c": {rules: []defRule{
		{re: cMacro},
		{re: cRecord},
		{re: cTypedef},
		{re: cFunc, reject: cControl},
	}},
	"cpp": {rules: []defRule{
		{re: cMacro},
		{re: cppNamespace},
		{re: cppClass},
		{re: cppStruct},
		{re: cppEnum},
		{re: cppUsing},
		{re: cRecord},
		{re: cppFunc, reject: cControl},
	}},
	"java": {rules: []defRule{
		{re: javaPkg},
		{re: javaClass},
		{re: javaIface},
		{re: javaEnum},
		{re: javaRecord},
		{re: javaMethod},
	}},
	"csharp": {rules: []defRule{
		{re: csNamespace},
		{re: csClass},
		{re: csIface},
		{re: csStruct},
		{re: csEnum},
		{re: csRecord},
		{re: csProp},
		{re: csMethod},
	}},
	"ruby": {rules: []defRule{
		{re: rbClass},
		{re: rbModule},
		{re: rbDef},
	}},
	"php": {rules: []defRule{
		{re: phpNamespace},
		{re: phpClass},
		{re: phpFunc},
		{re: phpConst},
	}},
	"swift": {rules: []defRule{
		{re: swiftFunc},
		{re: swiftType},
		{re: swiftTypealias},
	}},
	"kotlin": {rules: []defRule{
		{re: ktPackage},
		{re: ktClass},
		{re: ktFun, joinReceiver: true},
	}},
	"scala": {rules: []defRule{
		{re: scalaPackage},
		{re: scalaObject},
		{re: scalaClass},
		{re: scalaDef},
		{re: scalaVal},
	}},
	"css": {rules: []defRule{
		{re: cssAtRule},
		{re: cssSelector},
	}},
	"html": {}, // tier-4-only: see registry doc comment
}
