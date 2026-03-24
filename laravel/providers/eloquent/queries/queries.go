package queries

import (
	"github.com/laravel-ls/laravel-ls/parser"
	"github.com/laravel-ls/laravel-ls/treesitter"
	"github.com/laravel-ls/laravel-ls/treesitter/language"
)

const (
	// QueryCaptureModel is the capture name for the model class name identifier.
	QueryCaptureModel = "eloquent.model"
	// QueryCaptureColumn is the capture name for a column name string argument.
	QueryCaptureColumn = "eloquent.column"
	// QueryCaptureRelationship is the capture name for a relationship name string argument.
	QueryCaptureRelationship = "eloquent.relationship"
	// QueryCaptureScope is the capture name for a scope method name identifier.
	QueryCaptureScope = "eloquent.scope"
)

func queryEloquentCalls(file *parser.File, lang language.Identifier) treesitter.CaptureSlice {
	r, err := file.FindTags(lang,
		QueryCaptureModel,
		QueryCaptureColumn,
		QueryCaptureRelationship,
		QueryCaptureScope,
	)
	if err != nil {
		return treesitter.CaptureSlice{}
	}
	return r
}

// EloquentCalls returns all Eloquent-related captures from a file.
func EloquentCalls(file *parser.File) treesitter.CaptureSlice {
	return append(
		queryEloquentCalls(file, language.PHP),
		queryEloquentCalls(file, language.PHPOnly)...,
	)
}
