package repository_test

import (
	"testing"

	"github.com/laravel-ls/laravel-ls/utils/repository"
	"github.com/stretchr/testify/require"
)

func makeModelRepo() repository.ModelRepository {
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
			Casts: map[string]string{"email_verified_at": "datetime"},
		},
		"Post": {
			Class: `App\Models\Post`,
			Table: "posts",
			File:  "/app/app/Models/Post.php",
			Line:  5,
			Columns: []repository.ColumnEntry{
				{Name: "id", Type: "bigint", Nullable: false},
				{Name: "title", Type: "varchar", Nullable: false},
			},
			Relationships: []repository.RelationshipEntry{},
			Scopes:        []repository.ScopeEntry{},
			Casts:         map[string]string{},
		},
	}
}

func TestModelRepository_Get(t *testing.T) {
	repo := makeModelRepo()

	entry, ok := repo.Get("User")
	require.True(t, ok)
	require.Equal(t, `App\Models\User`, entry.Class)
	require.Equal(t, "users", entry.Table)

	_, ok = repo.Get("NonExistent")
	require.False(t, ok)
}

func TestModelRepository_Exists(t *testing.T) {
	repo := makeModelRepo()

	require.True(t, repo.Exists("User"))
	require.True(t, repo.Exists("Post"))
	require.False(t, repo.Exists("Order"))
}

func TestModelRepository_Find(t *testing.T) {
	repo := makeModelRepo()

	// Find with "U" prefix should return User
	results := repo.Find("U")
	require.Len(t, results, 1)
	_, ok := results["User"]
	require.True(t, ok)

	// Find with empty prefix should return all
	all := repo.Find("")
	require.Len(t, all, 2)

	// Find with non-matching prefix
	none := repo.Find("XYZ")
	require.Empty(t, none)
}

func TestModelRepository_Clear(t *testing.T) {
	repo := makeModelRepo()
	require.Len(t, repo, 2)

	repo.Clear()
	require.Empty(t, repo)
}

func TestModelEntry_Fields(t *testing.T) {
	repo := makeModelRepo()
	entry, ok := repo.Get("User")
	require.True(t, ok)

	// Columns
	require.Len(t, entry.Columns, 3)
	require.Equal(t, "email", entry.Columns[1].Name)
	require.Equal(t, "varchar", entry.Columns[1].Type)
	require.False(t, entry.Columns[1].Nullable)
	require.True(t, entry.Columns[2].Nullable)

	// Relationships
	require.Len(t, entry.Relationships, 2)
	require.Equal(t, "posts", entry.Relationships[0].Name)
	require.Equal(t, "HasMany", entry.Relationships[0].Type)
	require.Equal(t, `App\Models\Post`, entry.Relationships[0].Related)
	require.Equal(t, 30, entry.Relationships[0].Line)

	// Scopes
	require.Len(t, entry.Scopes, 2)
	require.Equal(t, "active", entry.Scopes[0].Name)
	require.Equal(t, "active()", entry.Scopes[0].Signature)
	require.Equal(t, 50, entry.Scopes[0].Line)

	// Casts
	require.Equal(t, "datetime", entry.Casts["email_verified_at"])
}
