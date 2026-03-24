package repository

// ColumnEntry holds information about a single column on an Eloquent model's table.
type ColumnEntry struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default"`
}

// RelationshipEntry holds information about a relationship method on an Eloquent model.
type RelationshipEntry struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // short Relation class name, e.g. "HasMany"
	Related string `json:"related"` // FQCN of the related model
	Line    int    `json:"line"`    // line number of the method declaration
}

// ScopeEntry holds information about a named scope method on an Eloquent model.
type ScopeEntry struct {
	Name      string `json:"name"`      // scope name without the "scope" prefix, lowercased
	Signature string `json:"signature"` // full method signature string
	Line      int    `json:"line"`      // line number of the method declaration
}

// ModelEntry holds all introspected information for a single Eloquent model.
type ModelEntry struct {
	Class         string              `json:"class"`         // FQCN
	Table         string              `json:"table"`         // database table name
	File          string              `json:"file"`          // absolute path to model file
	Line          int                 `json:"line"`          // line number of the class declaration
	Columns       []ColumnEntry       `json:"columns"`       // table columns (may be empty if DB unavailable)
	Relationships []RelationshipEntry `json:"relationships"` // relationship methods
	Scopes        []ScopeEntry        `json:"scopes"`        // named scope methods
	Casts         map[string]string   `json:"casts"`         // attribute casts from getCasts()
}

// ModelRepository is a type alias for Repository specialized for ModelEntry,
// keyed by short class name (e.g. "User").
type ModelRepository = Repository[ModelEntry]
