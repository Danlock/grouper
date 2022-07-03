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

// JustErrors illustrates the use of a Group in place of a sync.WaitGroup to
// simplify goroutine counting and error handling. This example is derived from
// the sync.WaitGroup example at https://golang.org/pkg/sync/#example_WaitGroup.
func ExampleDynamicGroup_justErrors() {
	g, _ := grouper.NewDynamic[*http.Response](context.Background())
	var urls = []string{
		"http://www.golang.org/",
		"http://www.google.com/",
		"http://www.somestupidname.com/",
	}
	for _, url := range urls {
		// Launch a goroutine to fetch the URL.
		url := url // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func(context.Context) (*http.Response, error) {
			// Fetch the URL.
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
			}
			return resp, err
		})
	}
	// Wait for all HTTP fetches to complete.
	if _, err := g.Wait(); err == nil {
		fmt.Println("Successfully fetched all URLs.")
	}
}

// Parallel illustrates the use of a Group for synchronizing a simple parallel
// task: the "Google Search 2.0" function from
// https://talks.golang.org/2012/concurrency.slide#46, augmented with a Context
// and error-handling.
func ExampleDynamicGroup_parallel() {
	Google := func(ctx context.Context, query string) ([]Result, error) {
		g, ctx := grouper.NewDynamic[Result](ctx)

		searches := []Search{Web, Image, Video}
		for _, search := range searches {
			search := search // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func(ctx context.Context) (Result, error) { return search(ctx, query) })
		}
		if results, err := g.Wait(); err != nil {
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

	// Unordered Output:
	// web result for "golang"
	// image result for "golang"
	// video result for "golang"
}

func TestDynamicGroup(t *testing.T) {
	err1 := errors.New("grouper_test: 1")
	err2 := errors.New("grouper_test: 2")

	cases := []struct {
		errs []error
	}{
		{errs: []error{}},
		{errs: []error{nil}},
		{errs: []error{err1}},
		{errs: []error{err1, nil}},
		{errs: []error{err1, nil, err2}},
	}

	for _, tc := range cases {
		g, _ := grouper.NewDynamic[int](context.Background())
		println("testing with case ", tc.errs)
		var firstErr error
		for i, err := range tc.errs {
			err := err
			g.Go(func(context.Context) (int, error) { return i, err })

			if firstErr == nil && err != nil {
				firstErr = err
			}

			if _, gErr := g.Wait(); gErr != firstErr {
				t.Errorf("after %T.Go(func() error { return err }) for err in %+v want %v",
					g, tc.errs, firstErr)
			}
		}
	}
}

func TestDynamicWithContext(t *testing.T) {
	errDoom := errors.New("group_test: doomed")

	cases := []struct {
		errs []error
		want error
	}{
		{want: nil},
		{errs: []error{nil}, want: nil},
		{errs: []error{errDoom}, want: errDoom},
		{errs: []error{errDoom, nil}, want: errDoom},
	}

	for _, tc := range cases {
		g, ctx := grouper.NewDynamic[int](context.Background())

		for _, err := range tc.errs {
			err := err
			g.Go(func(context.Context) (int, error) { return 0, err })
		}

		if _, err := g.Wait(); err != tc.want {
			t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
				"g.Wait() = %v; want %v",
				g, tc.errs, err, tc.want)
		}

		canceled := false
		select {
		case <-ctx.Done():
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
