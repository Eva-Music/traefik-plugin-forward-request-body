package forwardrequest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		desc string
		cfg  *Config
	}{
		{
			desc: "should return no error",
			cfg: func() *Config {
				c := CreateConfig()
				c.URL = "https://example.com/" // nolint:goconst
				return c
			}(),
		},
	}

	for _, test := range tests {
		test := test // pin

		t.Run(test.desc, func(t *testing.T) {
			h, err := New(context.Background(), nil, test.cfg, "forwardrequest")

			if err != nil {
				t.Errorf("Received unexpected error:\n%+v", err)
			}
			if h == nil {
				t.Errorf("Expected value not to be nil.")
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	tests := []struct {
		desc          string
		cfg           *Config
		expNextCall   bool
		expStatusCode int
	}{
		{
			desc: "should return ok status",
			cfg: func() *Config {
				c := CreateConfig()
				c.URL = "https://example.com/"
				return c
			}(),
			expNextCall:   true,
			expStatusCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		test := test // pin

		t.Run(test.desc, func(t *testing.T) {
			nextCall := false
			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				nextCall = true
			})

			h, err := New(context.Background(), next, test.cfg, "forwardrequest")
			if err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()

			url := "https://example.com/"
			req := httptest.NewRequest(http.MethodPost, url, strings.NewReader("example"))

			h.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			if nextCall != test.expNextCall {
				t.Errorf("next handler should not be called")
			}
			if res.StatusCode != test.expStatusCode {
				t.Errorf("got status code %d, want %d", rec.Code, test.expStatusCode)
			}
		})
	}
}
