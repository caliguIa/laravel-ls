package eloquent

import (
	"testing"

	"github.com/laravel-ls/laravel-ls/file"
	"github.com/laravel-ls/laravel-ls/parser"
	"github.com/laravel-ls/laravel-ls/provider"
	"github.com/laravel-ls/laravel-ls/utils/repository"
	"github.com/laravel-ls/protocol"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// ---- test helpers ----

func mustParsePHP(t *testing.T, src string) *parser.File {
	t.Helper()
	f, err := parser.Parse([]byte(src), file.TypePHP)
	require.NoError(t, err)
	return f
}

func providerWithModels(models repository.ModelRepository) *Provider {
	p := &Provider{}
	p.modelCache = models
	p.cacheLoaded = true
	return p
}

func sampleRepo() repository.ModelRepository {
	return repository.ModelRepository{
		"User": {
			Class: `App\Models\User`,
			Table: "users",
			File:  "/app/app/Models/User.php",
			Line:  10,
			Columns: []repository.ColumnEntry{
				{Name: "id", Type: "bigint", Nullable: false, Default: ""},
				{Name: "email", Type: "varchar", Nullable: false, Default: ""},
				{Name: "name", Type: "varchar", Nullable: true, Default: ""},
			},
			Relationships: []repository.RelationshipEntry{
				{Name: "posts", Type: "HasMany", Related: `App\Models\Post`, Line: 30},
				{Name: "profile", Type: "HasOne", Related: `App\Models\Profile`, Line: 40},
			},
			Scopes: []repository.ScopeEntry{
				{Name: "active", Signature: "active()", Line: 50},
				{Name: "verified", Signature: "verified(bool $flag = true)", Line: 60},
			},
			Casts: map[string]string{"email": "string"},
		},
	}
}

func point(row, col uint) ts.Point {
	return ts.Point{Row: row, Column: col}
}

func logger(t *testing.T) *log.Entry {
	return log.WithField("test", t.Name())
}

// ---- ResolveCompletion ----

func TestResolveCompletionColumn(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("all columns returned on empty prefix", func(t *testing.T) {
		// User::where('|', ...) — cursor inside the empty string
		src := `<?php User::where('', 'v');`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 19),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.Label
		}
		require.ElementsMatch(t, []string{"id", "email", "name"}, labels)
	})

	t.Run("prefix-filtered columns returned", func(t *testing.T) {
		src := `<?php User::where('em', 'v');`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 20),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.Label
		}
		require.ElementsMatch(t, []string{"email"}, labels)
	})

	t.Run("no completions for unknown model", func(t *testing.T) {
		src := `<?php NonExistent::where('', 'v');`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 26),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		require.Empty(t, items)
	})

	t.Run("column detail shows type and nullability", func(t *testing.T) {
		src := `<?php User::where('name', 'v');`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 20),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		require.Len(t, items, 1)
		require.Equal(t, "name", items[0].Label)
		require.Contains(t, items[0].Detail, "varchar")
		require.Contains(t, items[0].Detail, "nullable")
	})
}

func TestResolveCompletionRelationship(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("all relationships returned on empty prefix", func(t *testing.T) {
		src := `<?php User::with('');`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 18),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.Label
		}
		require.ElementsMatch(t, []string{"posts", "profile"}, labels)
	})

	t.Run("prefix-filtered relationships", func(t *testing.T) {
		src := `<?php User::with('po');`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 19),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.Label
		}
		require.ElementsMatch(t, []string{"posts"}, labels)
	})

	t.Run("relationship detail shows type and related model", func(t *testing.T) {
		src := `<?php User::with('posts');`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 19),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		require.Len(t, items, 1)
		require.Equal(t, "posts", items[0].Label)
		require.Contains(t, items[0].Detail, "HasMany")
		require.Contains(t, items[0].Detail, "Post")
	})
}

func TestResolveCompletionScope(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("scope completion contains scope name", func(t *testing.T) {
		src := `<?php User::active();`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 12),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		require.NotEmpty(t, items)
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.Label
		}
		require.Contains(t, labels, "active")
	})

	t.Run("scope detail shows 'scope'", func(t *testing.T) {
		src := `<?php User::active();`
		f := mustParsePHP(t, src)
		var items []protocol.CompletionItem
		p.ResolveCompletion(provider.CompletionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 12),
			Publish:     func(item protocol.CompletionItem) { items = append(items, item) },
		})
		for _, it := range items {
			if it.Label == "active" {
				require.Equal(t, "scope", it.Detail)
				return
			}
		}
		t.Fatal("active scope completion not found")
	})
}

// ---- Hover ----

func TestHoverColumn(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("hover on existing column shows type and nullability", func(t *testing.T) {
		src := `<?php User::where('email', 'v');`
		f := mustParsePHP(t, src)
		var hovers []provider.Hover
		p.Hover(provider.HoverContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 20),
			Publish:     func(h provider.Hover) { hovers = append(hovers, h) },
		})
		require.Len(t, hovers, 1)
		require.Contains(t, hovers[0].Content, "email")
		require.Contains(t, hovers[0].Content, "varchar")
	})

	t.Run("nullable column shows nullable in hover", func(t *testing.T) {
		src := `<?php User::where('name', 'v');`
		f := mustParsePHP(t, src)
		var hovers []provider.Hover
		p.Hover(provider.HoverContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 20),
			Publish:     func(h provider.Hover) { hovers = append(hovers, h) },
		})
		require.Len(t, hovers, 1)
		require.Contains(t, hovers[0].Content, "nullable")
	})

	t.Run("no hover for unknown column", func(t *testing.T) {
		src := `<?php User::where('nonexistent', 'v');`
		f := mustParsePHP(t, src)
		var hovers []provider.Hover
		p.Hover(provider.HoverContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 20),
			Publish:     func(h provider.Hover) { hovers = append(hovers, h) },
		})
		require.Empty(t, hovers)
	})

	t.Run("no hover for unknown model", func(t *testing.T) {
		src := `<?php Unknown::where('email', 'v');`
		f := mustParsePHP(t, src)
		var hovers []provider.Hover
		p.Hover(provider.HoverContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 22),
			Publish:     func(h provider.Hover) { hovers = append(hovers, h) },
		})
		require.Empty(t, hovers)
	})
}

func TestHoverRelationship(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("hover on existing relationship shows type and model", func(t *testing.T) {
		src := `<?php User::with('posts');`
		f := mustParsePHP(t, src)
		var hovers []provider.Hover
		p.Hover(provider.HoverContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 18),
			Publish:     func(h provider.Hover) { hovers = append(hovers, h) },
		})
		require.Len(t, hovers, 1)
		require.Contains(t, hovers[0].Content, "posts")
		require.Contains(t, hovers[0].Content, "HasMany")
	})

	t.Run("no hover for unknown relationship", func(t *testing.T) {
		src := `<?php User::with('unknown_rel');`
		f := mustParsePHP(t, src)
		var hovers []provider.Hover
		p.Hover(provider.HoverContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 18),
			Publish:     func(h provider.Hover) { hovers = append(hovers, h) },
		})
		require.Empty(t, hovers)
	})
}

func TestHoverScope(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("hover on existing scope shows signature", func(t *testing.T) {
		src := `<?php User::active();`
		f := mustParsePHP(t, src)
		var hovers []provider.Hover
		p.Hover(provider.HoverContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 12),
			Publish:     func(h provider.Hover) { hovers = append(hovers, h) },
		})
		require.Len(t, hovers, 1)
		require.Contains(t, hovers[0].Content, "active")
		require.Contains(t, hovers[0].Content, "active()")
	})
}

// ---- ResolveDefinition ----

func TestResolveDefinitionColumn(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("definition on column jumps to model class line", func(t *testing.T) {
		src := `<?php User::where('email', 'v');`
		f := mustParsePHP(t, src)
		var locations []protocol.Location
		p.ResolveDefinition(provider.DefinitionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 20),
			Publish:     func(loc protocol.Location) { locations = append(locations, loc) },
		})
		require.Len(t, locations, 1)
		require.Contains(t, string(locations[0].URI), "User.php")
		// Line should be class line (10 - 1 = 9 for LSP 0-indexed)
		require.Equal(t, uint32(9), locations[0].Range.Start.Line)
	})

	t.Run("no definition for unknown model", func(t *testing.T) {
		src := `<?php Unknown::where('email', 'v');`
		f := mustParsePHP(t, src)
		var locations []protocol.Location
		p.ResolveDefinition(provider.DefinitionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 22),
			Publish:     func(loc protocol.Location) { locations = append(locations, loc) },
		})
		require.Empty(t, locations)
	})
}

func TestResolveDefinitionRelationship(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("definition on relationship jumps to method line", func(t *testing.T) {
		src := `<?php User::with('posts');`
		f := mustParsePHP(t, src)
		var locations []protocol.Location
		p.ResolveDefinition(provider.DefinitionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 18),
			Publish:     func(loc protocol.Location) { locations = append(locations, loc) },
		})
		require.Len(t, locations, 1)
		require.Contains(t, string(locations[0].URI), "User.php")
		// posts relationship is at line 30; LSP is 0-indexed so 29
		require.Equal(t, uint32(29), locations[0].Range.Start.Line)
	})

	t.Run("no definition for unknown relationship", func(t *testing.T) {
		src := `<?php User::with('unknown_rel');`
		f := mustParsePHP(t, src)
		var locations []protocol.Location
		p.ResolveDefinition(provider.DefinitionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 18),
			Publish:     func(loc protocol.Location) { locations = append(locations, loc) },
		})
		require.Empty(t, locations)
	})
}

func TestResolveDefinitionScope(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("definition on scope jumps to scope method line", func(t *testing.T) {
		src := `<?php User::active();`
		f := mustParsePHP(t, src)
		var locations []protocol.Location
		p.ResolveDefinition(provider.DefinitionContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Position:    point(0, 12),
			Publish:     func(loc protocol.Location) { locations = append(locations, loc) },
		})
		require.Len(t, locations, 1)
		// active scope is at line 50; LSP is 0-indexed so 49
		require.Equal(t, uint32(49), locations[0].Range.Start.Line)
	})
}

// ---- Diagnostic ----

func TestDiagnosticColumn(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("no diagnostic for known column", func(t *testing.T) {
		src := `<?php User::where('email', 'v');`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Empty(t, diags)
	})

	t.Run("warning diagnostic for unknown column", func(t *testing.T) {
		src := `<?php User::where('emal', 'v');`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Len(t, diags, 1)
		require.Contains(t, diags[0].Message, "emal")
		require.Equal(t, protocol.DiagnosticSeverityWarning, diags[0].Severity)
	})

	t.Run("no diagnostic when model not in repo", func(t *testing.T) {
		src := `<?php NonExistent::where('emal', 'v');`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Empty(t, diags)
	})

	t.Run("no diagnostic for empty string argument", func(t *testing.T) {
		src := `<?php User::where('', 'v');`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Empty(t, diags)
	})
}

func TestDiagnosticRelationship(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("no diagnostic for known relationship", func(t *testing.T) {
		src := `<?php User::with('posts');`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Empty(t, diags)
	})

	t.Run("warning diagnostic for unknown relationship", func(t *testing.T) {
		src := `<?php User::with('post');`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Len(t, diags, 1)
		require.Contains(t, diags[0].Message, "post")
		require.Equal(t, protocol.DiagnosticSeverityWarning, diags[0].Severity)
	})
}

func TestDiagnosticScope(t *testing.T) {
	p := providerWithModels(sampleRepo())

	t.Run("no diagnostic for known scope", func(t *testing.T) {
		src := `<?php User::active();`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Empty(t, diags)
	})

	t.Run("warning diagnostic for unknown scope", func(t *testing.T) {
		src := `<?php User::actve();`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Len(t, diags, 1)
		require.Contains(t, diags[0].Message, "actve")
		require.Equal(t, protocol.DiagnosticSeverityWarning, diags[0].Severity)
	})

	t.Run("no diagnostic when model not in repo", func(t *testing.T) {
		src := `<?php Unknown::actve();`
		f := mustParsePHP(t, src)
		var diags []provider.Diagnostic
		p.Diagnostic(provider.DiagnosticContext{
			BaseContext: provider.BaseContext{Logger: logger(t), File: f},
			Publish:     func(d provider.Diagnostic) { diags = append(diags, d) },
		})
		require.Empty(t, diags)
	})
}

// ---- OnFileSaved ----

func TestOnFileSaved(t *testing.T) {
	t.Run("file outside app/ returns nil", func(t *testing.T) {
		p := &Provider{rootPath: "/app"}
		ch := p.OnFileSaved("/app/routes/web.php")
		require.Nil(t, ch)
	})

	t.Run("non-php file returns nil", func(t *testing.T) {
		p := &Provider{rootPath: "/app"}
		ch := p.OnFileSaved("/app/app/Models/User.txt")
		require.Nil(t, ch)
	})

	t.Run("nil project returns nil for model file", func(t *testing.T) {
		p := &Provider{rootPath: "/app", project: nil}
		ch := p.OnFileSaved("/app/app/Models/User.php")
		require.Nil(t, ch)
	})

	t.Run("app file outside app/Models also invalidates", func(t *testing.T) {
		p := &Provider{rootPath: "/app", project: nil}
		ch := p.OnFileSaved("/app/app/User.php")
		require.Nil(t, ch)
	})
}
