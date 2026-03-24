package eloquent

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/laravel-ls/laravel-ls/file"
	"github.com/laravel-ls/laravel-ls/laravel/providers/eloquent/queries"
	"github.com/laravel-ls/laravel-ls/project"
	"github.com/laravel-ls/laravel-ls/provider"
	"github.com/laravel-ls/laravel-ls/treesitter"
	"github.com/laravel-ls/laravel-ls/utils/repository"
	"github.com/laravel-ls/protocol"
	log "github.com/sirupsen/logrus"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// Provider implements LSP features for Eloquent model calls:
// column, relationship, and scope completion, hover, definition, and diagnostics.
type Provider struct {
	rootPath string
	project  *project.Project

	mu          sync.Mutex
	modelCache  repository.ModelRepository
	cacheLoaded bool
	modelGen    uint64
}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Register(manager *provider.Manager) {
	manager.Register(file.TypePHP, p)
	manager.Register(file.TypeTinker, p)
}

func (p *Provider) Init(ctx provider.InitContext) {
	p.rootPath = ctx.RootPath
	p.project = ctx.Project
}

// OnFileSaved invalidates the model cache when a file under app/Models/ or app/ is saved.
func (p *Provider) OnFileSaved(filename string) <-chan struct{} {
	modelsDir := path.Join(p.rootPath, "app", "Models") + "/"
	appDir := path.Join(p.rootPath, "app") + "/"

	isModelFile := (strings.HasPrefix(filename, modelsDir) || strings.HasPrefix(filename, appDir)) &&
		strings.HasSuffix(filename, ".php")

	if !isModelFile {
		return nil
	}

	if p.project == nil {
		return nil
	}

	p.mu.Lock()
	p.modelCache = nil
	p.cacheLoaded = false
	p.modelGen++
	gen := p.modelGen
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		repo, err := p.project.Models()
		if err != nil {
			log.WithError(err).Debug("eloquent: OnFileSaved pre-warm failed")
			return
		}
		p.mu.Lock()
		if p.modelGen == gen {
			log.WithField("models", len(repo)).WithField("gen", gen).Debug("eloquent: pre-warm complete, storing in cache")
			p.modelCache = repo
			p.cacheLoaded = true
		}
		p.mu.Unlock()
	}()
	return done
}

// models returns the model repository, cached in memory.
func (p *Provider) models() (repository.ModelRepository, error) {
	p.mu.Lock()
	cache, loaded, gen := p.modelCache, p.cacheLoaded, p.modelGen
	p.mu.Unlock()

	if loaded {
		log.WithField("models", len(cache)).Debug("eloquent: models() cache hit")
		return cache, nil
	}

	log.Debug("eloquent: models() cache miss — calling PHP")

	if p.project == nil {
		return nil, fmt.Errorf("eloquent: provider not initialized")
	}

	repo, err := p.project.Models()
	if err != nil {
		log.WithError(err).Warn("eloquent: models() PHP call failed")
		return nil, err
	}

	p.mu.Lock()
	if p.modelGen == gen {
		log.WithField("models", len(repo)).WithField("gen", gen).Debug("eloquent: PHP call complete, storing in cache")
		p.modelCache = repo
		p.cacheLoaded = true
	}
	p.mu.Unlock()

	return repo, nil
}

// chainRoot walks up from a node through member_call_expression parents to find
// the root scoped_call_expression that started the chain (i.e. ClassName::method(...)).
// For a direct scoped_call_expression it returns it immediately.
// Returns nil if no scoped_call_expression ancestor is found.
func chainRoot(node *ts.Node) *ts.Node {
	n := node
	for n != nil {
		switch n.Kind() {
		case "scoped_call_expression":
			return n
		case "member_call_expression":
			// keep walking up — the object of this call is further up the chain
		default:
			// If we leave the call-chain AST nodes without finding a root, give up.
			if n.Kind() != "argument" && n.Kind() != "arguments" &&
				n.Kind() != "expression_statement" && n.Kind() != "assignment_expression" &&
				n.Kind() != "return_statement" {
				return nil
			}
		}
		n = n.Parent()
	}
	return nil
}

// findModelCapture finds the @eloquent.model capture that belongs to the root
// scoped_call_expression of the call chain containing the given node.
// Works for both direct scoped calls and chained member calls.
func findModelCapture(node *ts.Node, captures treesitter.CaptureSlice) *ts.Node {
	// Walk up to the immediate containing call expression (scoped or member).
	n := node
	for n != nil {
		k := n.Kind()
		if k == "scoped_call_expression" || k == "member_call_expression" {
			break
		}
		n = n.Parent()
	}
	if n == nil {
		return nil
	}

	// Find the chain root (the scoped_call_expression at the start of the chain).
	root := chainRoot(n)
	if root == nil {
		return nil
	}
	rootRange := root.Range()

	for _, c := range captures.Name(queries.QueryCaptureModel) {
		// The model capture's parent is the scope: node of the scoped_call_expression,
		// whose parent is the scoped_call_expression itself.
		parent := c.Node.Parent()
		if parent == nil {
			continue
		}
		if parent.Range() == rootRange {
			return &c.Node
		}
	}
	return nil
}

// modelName extracts the short model class name from a model capture node.
// If the node is a relative_scope (self/static), it resolves to the model
// whose file matches currentFile (i.e. the model being edited), returning ""
// if no match is found in the repo.
func modelName(node *ts.Node, src []byte, currentFile string, repo repository.ModelRepository) string {
	text := node.Utf8Text(src)
	if text == "self" || text == "static" || text == "parent" {
		// Resolve relative scope: find a model whose file matches currentFile
		for name, entry := range repo {
			if entry.File == currentFile {
				return name
			}
		}
		return ""
	}
	return text
}

// ResolveCompletion handles completions for column, relationship, and scope captures.
func (p *Provider) ResolveCompletion(ctx provider.CompletionContext) {
	all := queries.EloquentCalls(ctx.File)
	ctx.Logger.WithFields(log.Fields{
		"pos":         ctx.Position,
		"total_caps":  len(all),
		"column_caps": len(all.Name(queries.QueryCaptureColumn)),
		"rel_caps":    len(all.Name(queries.QueryCaptureRelationship)),
		"scope_caps":  len(all.Name(queries.QueryCaptureScope)),
	}).Debug("eloquent: ResolveCompletion")

	// Check for column completion
	if node := all.Name(queries.QueryCaptureColumn).At(ctx.Position); node != nil {
		ctx.Logger.Debug("eloquent: ResolveCompletion matched column capture")
		p.resolveColumnCompletion(ctx, all, node)
		return
	}
	// Check for relationship completion
	if node := all.Name(queries.QueryCaptureRelationship).At(ctx.Position); node != nil {
		ctx.Logger.Debug("eloquent: ResolveCompletion matched relationship capture")
		p.resolveRelationshipCompletion(ctx, all, node)
		return
	}
	// Check for scope completion
	if node := all.Name(queries.QueryCaptureScope).At(ctx.Position); node != nil {
		ctx.Logger.Debug("eloquent: ResolveCompletion matched scope capture")
		p.resolveScopeCompletion(ctx, all, node)
		return
	}
	ctx.Logger.Debug("eloquent: ResolveCompletion no matching capture at position")
}

func (p *Provider) resolveColumnCompletion(ctx provider.CompletionContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		ctx.Logger.Debug("eloquent: resolveColumnCompletion could not find model capture")
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	ctx.Logger.WithField("model", name).Debug("eloquent: resolveColumnCompletion resolved model")
	entry, ok := repo.Get(name)
	if !ok {
		ctx.Logger.WithField("model", name).Debug("eloquent: resolveColumnCompletion model not found in repo")
		return
	}
	typed := typedText(node, ctx.File.Src)
	ctx.Logger.WithFields(log.Fields{"model": name, "typed": typed, "columns": len(entry.Columns)}).Debug("eloquent: resolveColumnCompletion publishing completions")
	for _, col := range entry.Columns {
		if strings.HasPrefix(col.Name, typed) {
			ctx.Publish(columnCompletionItem(col))
		}
	}
}

func (p *Provider) resolveRelationshipCompletion(ctx provider.CompletionContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		ctx.Logger.Debug("eloquent: resolveRelationshipCompletion could not find model capture")
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	ctx.Logger.WithField("model", name).Debug("eloquent: resolveRelationshipCompletion resolved model")
	entry, ok := repo.Get(name)
	if !ok {
		ctx.Logger.WithField("model", name).Debug("eloquent: resolveRelationshipCompletion model not found in repo")
		return
	}
	typed := typedText(node, ctx.File.Src)
	ctx.Logger.WithFields(log.Fields{"model": name, "typed": typed, "relationships": len(entry.Relationships)}).Debug("eloquent: resolveRelationshipCompletion publishing completions")
	for _, rel := range entry.Relationships {
		if strings.HasPrefix(rel.Name, typed) {
			ctx.Publish(relationshipCompletionItem(rel))
		}
	}
}

func (p *Provider) resolveScopeCompletion(ctx provider.CompletionContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		ctx.Logger.Debug("eloquent: resolveScopeCompletion could not find model capture")
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	ctx.Logger.WithField("model", name).Debug("eloquent: resolveScopeCompletion resolved model")
	entry, ok := repo.Get(name)
	if !ok {
		ctx.Logger.WithField("model", name).Debug("eloquent: resolveScopeCompletion model not found in repo")
		return
	}
	typed := node.Utf8Text(ctx.File.Src)
	ctx.Logger.WithFields(log.Fields{"model": name, "typed": typed, "scopes": len(entry.Scopes)}).Debug("eloquent: resolveScopeCompletion publishing completions")
	for _, scope := range entry.Scopes {
		if strings.HasPrefix(scope.Name, typed) {
			ctx.Publish(scopeCompletionItem(scope))
		}
	}
}

// Hover handles hover requests for column, relationship, and scope captures.
func (p *Provider) Hover(ctx provider.HoverContext) {
	all := queries.EloquentCalls(ctx.File)
	ctx.Logger.WithFields(log.Fields{
		"pos":         ctx.Position,
		"total_caps":  len(all),
		"column_caps": len(all.Name(queries.QueryCaptureColumn)),
		"rel_caps":    len(all.Name(queries.QueryCaptureRelationship)),
		"scope_caps":  len(all.Name(queries.QueryCaptureScope)),
	}).Debug("eloquent: Hover")

	if node := all.Name(queries.QueryCaptureColumn).At(ctx.Position); node != nil {
		ctx.Logger.Debug("eloquent: Hover matched column capture")
		p.hoverColumn(ctx, all, node)
		return
	}
	if node := all.Name(queries.QueryCaptureRelationship).At(ctx.Position); node != nil {
		ctx.Logger.Debug("eloquent: Hover matched relationship capture")
		p.hoverRelationship(ctx, all, node)
		return
	}
	if node := all.Name(queries.QueryCaptureScope).At(ctx.Position); node != nil {
		ctx.Logger.Debug("eloquent: Hover matched scope capture")
		p.hoverScope(ctx, all, node)
		return
	}
	ctx.Logger.Debug("eloquent: Hover no matching capture at position")
}

func (p *Provider) hoverColumn(ctx provider.HoverContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	entry, ok := repo.Get(name)
	if !ok {
		return
	}
	colName := stringContent(node, ctx.File.Src)
	for _, col := range entry.Columns {
		if col.Name == colName {
			ctx.Publish(provider.Hover{Content: columnHoverContent(col, entry)})
			return
		}
	}
}

func (p *Provider) hoverRelationship(ctx provider.HoverContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	entry, ok := repo.Get(name)
	if !ok {
		return
	}
	relName := stringContent(node, ctx.File.Src)
	for _, rel := range entry.Relationships {
		if rel.Name == relName {
			ctx.Publish(provider.Hover{Content: relationshipHoverContent(rel)})
			return
		}
	}
}

func (p *Provider) hoverScope(ctx provider.HoverContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	entry, ok := repo.Get(name)
	if !ok {
		return
	}
	scopeName := node.Utf8Text(ctx.File.Src)
	for _, scope := range entry.Scopes {
		if scope.Name == scopeName {
			ctx.Publish(provider.Hover{Content: scopeHoverContent(scope)})
			return
		}
	}
}

// ResolveDefinition handles go-to-definition for column, relationship, and scope captures.
func (p *Provider) ResolveDefinition(ctx provider.DefinitionContext) {
	all := queries.EloquentCalls(ctx.File)

	if node := all.Name(queries.QueryCaptureColumn).At(ctx.Position); node != nil {
		p.defineColumn(ctx, all, node)
		return
	}
	if node := all.Name(queries.QueryCaptureRelationship).At(ctx.Position); node != nil {
		p.defineRelationship(ctx, all, node)
		return
	}
	if node := all.Name(queries.QueryCaptureScope).At(ctx.Position); node != nil {
		p.defineScope(ctx, all, node)
		return
	}
}

func (p *Provider) defineColumn(ctx provider.DefinitionContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	entry, ok := repo.Get(name)
	if !ok {
		return
	}
	ctx.Publish(locationFromEntry(entry, entry.Line))
}

func (p *Provider) defineRelationship(ctx provider.DefinitionContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	entry, ok := repo.Get(name)
	if !ok {
		return
	}
	relName := stringContent(node, ctx.File.Src)
	for _, rel := range entry.Relationships {
		if rel.Name == relName {
			ctx.Publish(locationFromEntry(entry, rel.Line))
			return
		}
	}
}

func (p *Provider) defineScope(ctx provider.DefinitionContext, all treesitter.CaptureSlice, node *ts.Node) {
	modelNode := findModelCapture(node, all)
	if modelNode == nil {
		return
	}
	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}
	name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
	entry, ok := repo.Get(name)
	if !ok {
		return
	}
	scopeName := node.Utf8Text(ctx.File.Src)
	for _, scope := range entry.Scopes {
		if scope.Name == scopeName {
			ctx.Publish(locationFromEntry(entry, scope.Line))
			return
		}
	}
}

// Diagnostic reports warnings for unknown column, relationship, and scope references.
func (p *Provider) Diagnostic(ctx provider.DiagnosticContext) {
	all := queries.EloquentCalls(ctx.File)
	ctx.Logger.WithFields(log.Fields{
		"total_caps":  len(all),
		"column_caps": len(all.Name(queries.QueryCaptureColumn)),
		"rel_caps":    len(all.Name(queries.QueryCaptureRelationship)),
		"scope_caps":  len(all.Name(queries.QueryCaptureScope)),
	}).Debug("eloquent: Diagnostic")
	if len(all) == 0 {
		return
	}

	repo, err := p.models()
	if err != nil {
		ctx.Logger.WithError(err).Warn("eloquent: failed to get model repository")
		return
	}

	for _, capture := range all.Name(queries.QueryCaptureColumn) {
		modelNode := findModelCapture(&capture.Node, all)
		if modelNode == nil {
			continue
		}
		name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
		entry, ok := repo.Get(name)
		if !ok {
			// Model not in repo — suppress (user story 17)
			continue
		}
		// If no columns were introspected (e.g. DB unreachable), suppress all
		// column diagnostics to avoid false positives.
		if len(entry.Columns) == 0 {
			continue
		}
		colName := stringContent(&capture.Node, ctx.File.Src)
		if colName == "" {
			continue
		}
		if !columnExists(entry, colName) {
			ctx.Publish(provider.Diagnostic{
				Range:    capture.Node.Range(),
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  fmt.Sprintf("Column [%s] not found on model [%s]", colName, name),
			})
		}
	}

	for _, capture := range all.Name(queries.QueryCaptureRelationship) {
		modelNode := findModelCapture(&capture.Node, all)
		if modelNode == nil {
			continue
		}
		name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
		entry, ok := repo.Get(name)
		if !ok {
			continue
		}
		relName := stringContent(&capture.Node, ctx.File.Src)
		if relName == "" {
			continue
		}
		if !relationshipExists(entry, relName) {
			ctx.Publish(provider.Diagnostic{
				Range:    capture.Node.Range(),
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  fmt.Sprintf("Relationship [%s] not found on model [%s]", relName, name),
			})
		}
	}

	for _, capture := range all.Name(queries.QueryCaptureScope) {
		modelNode := findModelCapture(&capture.Node, all)
		if modelNode == nil {
			continue
		}
		name := modelName(modelNode, ctx.File.Src, ctx.Filename, repo)
		entry, ok := repo.Get(name)
		if !ok {
			continue
		}
		scopeName := capture.Node.Utf8Text(ctx.File.Src)
		if scopeName == "" {
			continue
		}
		if !scopeExists(entry, scopeName) {
			ctx.Publish(provider.Diagnostic{
				Range:    capture.Node.Range(),
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  fmt.Sprintf("Scope [%s] not found on model [%s]", scopeName, name),
			})
		}
	}
}

// ---- formatting helpers ----

func columnCompletionItem(col repository.ColumnEntry) protocol.CompletionItem {
	detail := col.Type
	if col.Nullable {
		detail += ", nullable"
	}
	return protocol.CompletionItem{
		Label:  col.Name,
		Kind:   protocol.CompletionItemKindField,
		Detail: detail,
	}
}

func relationshipCompletionItem(rel repository.RelationshipEntry) protocol.CompletionItem {
	detail := rel.Type
	if rel.Related != "" {
		// Extract short related model name from FQCN
		parts := strings.Split(rel.Related, "\\")
		detail = fmt.Sprintf("%s → %s", rel.Type, parts[len(parts)-1])
	}
	return protocol.CompletionItem{
		Label:  rel.Name,
		Kind:   protocol.CompletionItemKindReference,
		Detail: detail,
	}
}

func scopeCompletionItem(scope repository.ScopeEntry) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label:  scope.Name,
		Kind:   protocol.CompletionItemKindMethod,
		Detail: "scope",
	}
}

func columnHoverContent(col repository.ColumnEntry, entry repository.ModelEntry) string {
	parts := []string{fmt.Sprintf("**%s** `%s`", col.Name, col.Type)}
	if col.Nullable {
		parts = append(parts, "nullable")
	}
	if col.Default != "" {
		parts = append(parts, fmt.Sprintf("default: `%s`", col.Default))
	}
	if castType, ok := entry.Casts[col.Name]; ok {
		parts = append(parts, fmt.Sprintf("cast: `%s`", castType))
	}
	return strings.Join(parts, "\n\n")
}

func relationshipHoverContent(rel repository.RelationshipEntry) string {
	if rel.Related != "" {
		parts := strings.Split(rel.Related, "\\")
		return fmt.Sprintf("**%s** `%s → %s`", rel.Name, rel.Type, parts[len(parts)-1])
	}
	return fmt.Sprintf("**%s** `%s`", rel.Name, rel.Type)
}

func scopeHoverContent(scope repository.ScopeEntry) string {
	return fmt.Sprintf("**%s**\n\n`%s`", scope.Name, scope.Signature)
}

func locationFromEntry(entry repository.ModelEntry, line int) protocol.Location {
	// line from PHP is 1-indexed; LSP lines are 0-indexed
	lspLine := uint32(0)
	if line > 0 {
		lspLine = uint32(line - 1)
	}
	return protocol.Location{
		URI: protocol.DocumentURI("file://" + entry.File),
		Range: protocol.Range{
			Start: protocol.Position{Line: lspLine},
			End:   protocol.Position{Line: lspLine},
		},
	}
}

// ---- existence checks ----

func columnExists(entry repository.ModelEntry, name string) bool {
	for _, col := range entry.Columns {
		if col.Name == name {
			return true
		}
	}
	return false
}

func relationshipExists(entry repository.ModelEntry, name string) bool {
	for _, rel := range entry.Relationships {
		if rel.Name == name {
			return true
		}
	}
	return false
}

func scopeExists(entry repository.ModelEntry, name string) bool {
	for _, scope := range entry.Scopes {
		if scope.Name == name {
			return true
		}
	}
	return false
}

// ---- string extraction ----

// stringContent extracts text from string/encapsed_string capture nodes.
// For @eloquent.column and @eloquent.relationship these are string literal content nodes.
func stringContent(node *ts.Node, src []byte) string {
	kind := node.Kind()
	if kind == "string_content" {
		return node.Utf8Text(src)
	}
	// Handle empty string nodes (string with no string_content child)
	if kind == "string" || kind == "encapsed_string" {
		if node.NamedChildCount() > 0 {
			child := node.NamedChild(0)
			if child.Kind() == "string_content" {
				return child.Utf8Text(src)
			}
		}
		return ""
	}
	return node.Utf8Text(src)
}

// typedText gets the text already typed inside a string argument node.
func typedText(node *ts.Node, src []byte) string {
	return stringContent(node, src)
}
