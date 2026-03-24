package queries_test

import (
	"testing"

	"github.com/laravel-ls/laravel-ls/file"
	"github.com/laravel-ls/laravel-ls/laravel/providers/eloquent/queries"
	"github.com/laravel-ls/laravel-ls/parser"
	"github.com/laravel-ls/laravel-ls/treesitter"
	"github.com/stretchr/testify/require"
)

func mustParsePHP(t *testing.T, src string) *parser.File {
	t.Helper()
	f, err := parser.Parse([]byte(src), file.TypePHP)
	require.NoError(t, err)
	return f
}

// captureNames returns the capture names from a CaptureSlice.
func captureNames(captures treesitter.CaptureSlice) []string {
	names := make([]string, len(captures))
	for i, c := range captures {
		names[i] = c.Name
	}
	return names
}

// captureTexts returns the text of each node for captures with the given name.
func captureTexts(captures treesitter.CaptureSlice, name string, src []byte) []string {
	var texts []string
	for _, c := range captures.Name(name) {
		texts = append(texts, c.Node.Utf8Text(src))
	}
	return texts
}

func TestEloquentCalls_ColumnCaptures(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		expectModels []string
		// expectColumns contains the raw node text (including quotes)
		expectColumns []string
	}{
		{
			name:          "where call",
			src:           `<?php User::where('email', 'test@example.com');`,
			expectModels:  []string{"User"},
			expectColumns: []string{"'email'"},
		},
		{
			name:          "whereNot call",
			src:           `<?php User::whereNot('name', null);`,
			expectModels:  []string{"User"},
			expectColumns: []string{"'name'"},
		},
		{
			name:          "orWhere call",
			src:           `<?php User::orWhere('id', 1);`,
			expectModels:  []string{"User"},
			expectColumns: []string{"'id'"},
		},
		{
			name:          "orderBy call",
			src:           `<?php User::orderBy('created_at');`,
			expectModels:  []string{"User"},
			expectColumns: []string{"'created_at'"},
		},
		{
			name:          "select call",
			src:           `<?php User::select('email');`,
			expectModels:  []string{"User"},
			expectColumns: []string{"'email'"},
		},
		{
			name:          "pluck call",
			src:           `<?php User::pluck('name');`,
			expectModels:  []string{"User"},
			expectColumns: []string{"'name'"},
		},
		{
			name:          "sum call",
			src:           `<?php Order::sum('total');`,
			expectModels:  []string{"Order"},
			expectColumns: []string{"'total'"},
		},
		{
			name:          "avg call",
			src:           `<?php Order::avg('amount');`,
			expectModels:  []string{"Order"},
			expectColumns: []string{"'amount'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParsePHP(t, tt.src)
			captures := queries.EloquentCalls(f)

			require.ElementsMatch(t, tt.expectModels, captureTexts(captures, queries.QueryCaptureModel, f.Src))
			require.ElementsMatch(t, tt.expectColumns, captureTexts(captures, queries.QueryCaptureColumn, f.Src))
		})
	}
}

func TestEloquentCalls_RelationshipCaptures(t *testing.T) {
	tests := []struct {
		name               string
		src                string
		expectModels       []string
		expectRelationship []string
	}{
		{
			name:               "with call",
			src:                `<?php User::with('posts');`,
			expectModels:       []string{"User"},
			expectRelationship: []string{"'posts'"},
		},
		{
			name:               "has call",
			src:                `<?php User::has('profile');`,
			expectModels:       []string{"User"},
			expectRelationship: []string{"'profile'"},
		},
		{
			name:               "whereHas call",
			src:                `<?php User::whereHas('posts');`,
			expectModels:       []string{"User"},
			expectRelationship: []string{"'posts'"},
		},
		{
			name:               "whereDoesntHave call",
			src:                `<?php User::whereDoesntHave('comments');`,
			expectModels:       []string{"User"},
			expectRelationship: []string{"'comments'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParsePHP(t, tt.src)
			captures := queries.EloquentCalls(f)

			require.ElementsMatch(t, tt.expectModels, captureTexts(captures, queries.QueryCaptureModel, f.Src))
			require.ElementsMatch(t, tt.expectRelationship, captureTexts(captures, queries.QueryCaptureRelationship, f.Src))
		})
	}
}

func TestEloquentCalls_ScopeCaptures(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		expectModels []string
		expectScopes []string
	}{
		{
			name:         "scope call",
			src:          `<?php User::active();`,
			expectModels: []string{"User"},
			expectScopes: []string{"active"},
		},
		{
			name:         "another scope call",
			src:          `<?php User::verified();`,
			expectModels: []string{"User"},
			expectScopes: []string{"verified"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParsePHP(t, tt.src)
			captures := queries.EloquentCalls(f)

			require.ElementsMatch(t, tt.expectModels, captureTexts(captures, queries.QueryCaptureModel, f.Src))
			require.ElementsMatch(t, tt.expectScopes, captureTexts(captures, queries.QueryCaptureScope, f.Src))
		})
	}
}

func TestEloquentCalls_ChainedCaptures(t *testing.T) {
	t.Run("chained where on column", func(t *testing.T) {
		src := `<?php ManagementGoal::where('status', 1)->where('archived', false)->get();`
		f := mustParsePHP(t, src)
		captures := queries.EloquentCalls(f)
		// model captured from the scoped_call_expression root only
		require.ElementsMatch(t, []string{"ManagementGoal"}, captureTexts(captures, queries.QueryCaptureModel, f.Src))
		// both column strings should be captured
		require.ElementsMatch(t, []string{"'status'", "'archived'"}, captureTexts(captures, queries.QueryCaptureColumn, f.Src))
	})

	t.Run("chained whereIn and with", func(t *testing.T) {
		src := `<?php ManagementGoal::whereIn('id', $ids)->with(['posts'])->get();`
		f := mustParsePHP(t, src)
		captures := queries.EloquentCalls(f)
		require.ElementsMatch(t, []string{"ManagementGoal"}, captureTexts(captures, queries.QueryCaptureModel, f.Src))
		require.ElementsMatch(t, []string{"'id'"}, captureTexts(captures, queries.QueryCaptureColumn, f.Src))
		// 'posts' is inside an array, not a direct string arg — not captured
		require.Empty(t, captureTexts(captures, queries.QueryCaptureRelationship, f.Src))
	})

	t.Run("chained whereNull", func(t *testing.T) {
		src := `<?php ManagementGoal::whereIn('id', $ids)->whereNull('deleted_at')->get();`
		f := mustParsePHP(t, src)
		captures := queries.EloquentCalls(f)
		require.ElementsMatch(t, []string{"ManagementGoal"}, captureTexts(captures, queries.QueryCaptureModel, f.Src))
		require.ElementsMatch(t, []string{"'id'", "'deleted_at'"}, captureTexts(captures, queries.QueryCaptureColumn, f.Src))
	})

	t.Run("chained with string relationship", func(t *testing.T) {
		src := `<?php User::where('active', 1)->with('posts')->get();`
		f := mustParsePHP(t, src)
		captures := queries.EloquentCalls(f)
		require.ElementsMatch(t, []string{"User"}, captureTexts(captures, queries.QueryCaptureModel, f.Src))
		require.ElementsMatch(t, []string{"'active'"}, captureTexts(captures, queries.QueryCaptureColumn, f.Src))
		require.ElementsMatch(t, []string{"'posts'"}, captureTexts(captures, queries.QueryCaptureRelationship, f.Src))
	})
}

func TestEloquentCalls_NoCaptures(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "no eloquent calls",
			src:  `<?php echo "hello world";`,
		},
		{
			// Instance method calls on a variable (e.g. $user->where(...)) are now
			// captured at the query layer — the provider layer discards them because
			// no scoped_call_expression root (ClassName::...) can be found in the chain.
			name: "non-static call is captured at query layer (discarded by provider)",
			src:  `<?php $user->where('email', 'test');`,
		},
		{
			name: "known builder methods are not captured as scopes",
			src:  `<?php User::find(1);`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParsePHP(t, tt.src)
			captures := queries.EloquentCalls(f)

			columnCaptures := captures.Name(queries.QueryCaptureColumn)
			relCaptures := captures.Name(queries.QueryCaptureRelationship)
			scopeCaptures := captures.Name(queries.QueryCaptureScope)

			switch tt.name {
			case "known builder methods are not captured as scopes":
				// find() should not produce column, relationship, or scope captures
				require.Empty(t, columnCaptures, "expected no column captures for find()")
				require.Empty(t, relCaptures, "expected no relationship captures for find()")
			case "non-static call is captured at query layer (discarded by provider)":
				// $var->where('col') emits a column capture at the query layer;
				// the provider layer discards it because there is no model class root.
				require.NotEmpty(t, columnCaptures, "expected column capture from member call")
				require.Empty(t, relCaptures, "expected no relationship captures")
				require.Empty(t, scopeCaptures, "expected no scope captures")
			default:
				require.Empty(t, columnCaptures, "expected no column captures")
				require.Empty(t, relCaptures, "expected no relationship captures")
				require.Empty(t, scopeCaptures, "expected no scope captures")
			}
		})
	}
}
