package recipes

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// QuoteFunc acquires a quote. It receives the caller's input `In` and
// returns the quote value `Q` or an error.
type QuoteFunc[In any, Q any] func(ctx context.Context, in In) (Q, error)

// ExecuteFunc executes against a quote. It receives the quote `Q` and
// returns the execution result `R`.
type ExecuteFunc[Q any, R any] func(ctx context.Context, q Q) (R, error)

// QuoteThenExecute is the generic quote → execute pattern used by
// payments, tokenops, and Connect. The caller supplies typed
// QuoteFunc / ExecuteFunc; the recipe wires timeouts and error context
// around them. If the optional Decide callback returns false, execution
// is skipped and the zero-valued result is returned alongside the quote.
func QuoteThenExecute[In any, Q any, R any](
	ctx context.Context,
	in In,
	quote QuoteFunc[In, Q],
	execute ExecuteFunc[Q, R],
	decide func(ctx context.Context, q Q) (bool, error),
	quoteTimeout time.Duration,
) (q Q, r R, err error) {
	if quote == nil || execute == nil {
		err = errors.New("recipes: quote and execute funcs are required")
		return
	}
	qctx := ctx
	if quoteTimeout > 0 {
		var cancel context.CancelFunc
		qctx, cancel = context.WithTimeout(ctx, quoteTimeout)
		defer cancel()
	}
	q, err = quote(qctx, in)
	if err != nil {
		err = fmt.Errorf("recipes: quote: %w", err)
		return
	}
	if decide != nil {
		proceed, derr := decide(ctx, q)
		if derr != nil {
			err = fmt.Errorf("recipes: decide: %w", derr)
			return
		}
		if !proceed {
			return
		}
	}
	r, err = execute(ctx, q)
	if err != nil {
		err = fmt.Errorf("recipes: execute: %w", err)
	}
	return
}
