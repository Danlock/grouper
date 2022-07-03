package grouper_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/danlock/grouper"
)

var (
	Web   = fakeSearch("web")
	Image = fakeSearch("image")
	Video = fakeSearch("video")
)

type Result string
type Search func(ctx context.Context, query string) (Result, error)

func fakeSearch(kind string) Search {
	return func(_ context.Context, query string) (Result, error) {
		return Result(fmt.Sprintf("%s result for %q", kind, query)), nil
	}
}

// JustErrors illustrates the use of a Group in place of a sync.WaitGroup to
// simplify goroutine counting and error handling. This example is derived from
// the sync.WaitGroup example at https://golang.org/pkg/sync/#example_WaitGroup.
func ExampleGroup_justErrors() {
	var urls = []string{
		"http://www.golang.org/",
		"http://www.google.com/",
		"http://www.somestupidname.com/",
	}
	funcs := make([]func(context.Context) (*http.Response, error), len(urls))
	for i, url := range urls {
		// Launch a goroutine to fetch the URL.
		url := url // https://golang.org/doc/faq#closures_and_goroutines
		funcs[i] = func(context.Context) (*http.Response, error) {
			// Fetch the URL.
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
			}
			return resp, err
		}
	}
	g := grouper.New(funcs...)
	// Wait for all HTTP fetches to complete.
	if _, err := g.Wait(context.Background()); err == nil {
		fmt.Println("Successfully fetched all URLs.")
	}
}

// Parallel illustrates the use of a Group for synchronizing a simple parallel
// task: the "Google Search 2.0" function from
// https://talks.golang.org/2012/concurrency.slide#46, augmented with a Context
// and error-handling.
func ExampleGroup_parallel() {
	Google := func(ctx context.Context, query string) ([]Result, error) {
		searches := []Search{Web, Image, Video}
		funcs := make([]func(context.Context) (Result, error), len(searches))
		for i, search := range searches {
			search := search // https://golang.org/doc/faq#closures_and_goroutines
			funcs[i] = func(context.Context) (Result, error) {
				return search(ctx, query)
			}
		}
		g := grouper.New(funcs...)
		if results, err := g.Wait(ctx); err != nil {
			return nil, err
		} else {
			return results, nil
		}
	}

	results, err := Google(context.Background(), "golang")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, result := range results {
		fmt.Println(result)
	}

	// Output:
	// web result for "golang"
	// image result for "golang"
	// video result for "golang"
}

func TestZeroGroup(t *testing.T) {
	err1 := errors.New("grouper_test: 1")

	cases := []struct {
		errs []error
	}{
		// {errs: []error{}},
		{errs: []error{nil}},
		{errs: []error{err1}},
		{errs: []error{err1, nil}},
		{errs: []error{err1, nil, err1, nil}},
	}

	for _, tc := range cases {
		funcs := make([]func(context.Context) (error, error), len(tc.errs))
		var firstErr error
		for i, err := range tc.errs {
			err := err
			funcs[i] = func(context.Context) (error, error) { return err, err }

			if firstErr == nil && err != nil {
				firstErr = err
			}
		}
		g := grouper.New(funcs...)
		if _, gErr := g.Wait(context.Background()); gErr != firstErr {
			t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
				"g.Wait() = %v; want %v",
				g, tc.errs, gErr, firstErr)
		}

	}
}

func TestWithContext(t *testing.T) {
	errDoom := errors.New("group_test: doomed")

	cases := []struct {
		errs []error
		want error
	}{
		{errs: []error{nil}, want: nil},
		{errs: []error{errDoom}, want: errDoom},
		{errs: []error{errDoom, nil}, want: errDoom},
	}

	for _, tc := range cases {
		funcs := make([]func(context.Context) (context.Context, error), len(tc.errs))
		for i, err := range tc.errs {
			err := err
			funcs[i] = func(ctx context.Context) (context.Context, error) { return ctx, err }
		}
		g := grouper.New(funcs...)
		ctxs, err := g.Wait(context.Background())
		if err != tc.want {
			t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
				"g.Wait() = %v; want %v",
				g, tc.errs, err, tc.want)
		}

		canceled := false
		select {
		case <-ctxs[0].Done():
			canceled = true
		default:
		}
		if !canceled {
			t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
				"ctx.Done() was not closed",
				g, tc.errs)
		}
	}
}

func TestValueGroup(t *testing.T) {
	cases := []struct {
		results []uint
		err     error
	}{
		{results: []uint{0, 2, 4}},
		{results: []uint{6, 2, 4}, err: errors.New("ugh")},
		{results: []uint{0, 8, 4}},
		{results: []uint{0, 2, 9}, err: errors.New("ugh")},
	}

	for _, tc := range cases {
		funcs := make([]func(context.Context) (uint, error), len(tc.results))

		for i, res := range tc.results {
			res := res
			funcs[i] = func(context.Context) (uint, error) { return res, tc.err }
		}
		g := grouper.New(funcs...)
		if _, gErr := g.Wait(context.Background()); gErr != tc.err {
			t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
				"g.Wait() = %v; want %v",
				g, tc.err, gErr, tc.err)
		}

	}
}
