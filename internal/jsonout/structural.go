package jsonout

import "github.com/openedi/ediforge/internal/model"

func Structural(doc *model.Document) any {
	return doc
}

func Annotated(doc *model.Document) any {
	return doc
}
