package translator

import (
	"context"
	"io"

	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/translate"
)

type Options struct {
	Standard       string
	Mode           string
	SchemaPath     string
	SchemaID       string
	IncludeRaw     bool
	IncludeOffsets bool
	AllowPartial   bool
}

type Result = translate.TranslateResult

type Client struct {
	service *translate.Service
}

func New() *Client {
	return &Client{service: translate.NewService()}
}

func (c *Client) Translate(ctx context.Context, reader io.Reader, opts Options) (*Result, error) {
	return c.service.Translate(ctx, translate.Input{Reader: reader}, translate.TranslateOptions{
		Standard:       model.Standard(opts.Standard),
		Mode:           model.Mode(opts.Mode),
		SchemaPath:     opts.SchemaPath,
		SchemaID:       opts.SchemaID,
		IncludeRaw:     opts.IncludeRaw,
		IncludeOffsets: opts.IncludeOffsets,
		AllowPartial:   opts.AllowPartial,
	})
}
