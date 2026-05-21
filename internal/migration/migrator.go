package migration

import (
	"go/ast"
	"go/token"
)

type FileContext struct {
	Path          string
	Source        []byte
	FileSet       *token.FileSet
	File          *ast.File
	IncludeReview bool
}

type Migrator interface {
	Name() string
	Run(*FileContext) ([]Edit, []Finding)
}
