package forwardrequest

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"
)

// Config holds the plugin configuration.
type Config struct {
	URL             string   `json:"url,omitempty"`
	ResponseHeaders []string `json:"responseHeaders,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

type forwardRequest struct {
	name            string
	next            http.Handler
	client          http.Client
	url             string
	responseHeaders []string
}

// New creates and returns a plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	return &forwardRequest{
		name:            name,
		next:            next,
		client:          client,
		url:             config.URL,
		responseHeaders: config.ResponseHeaders,
	}, nil
}

func (p *forwardRequest) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fReq, err := newForwardRequest(req, p.url)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	fRes, err := p.client.Do(fReq)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// not 2XX -> return forward response
	if fRes.StatusCode < http.StatusOK || fRes.StatusCode >= http.StatusMultipleChoices {
		p.writeForwardResponse(rw, fRes)
		return
	}

	// 2XX -> next
	overrideHeaders(req.Header, fRes.Header, p.responseHeaders...)
	p.next.ServeHTTP(rw, req)
}

func (p *forwardRequest) writeForwardResponse(rw http.ResponseWriter, fRes *http.Response) {
	body, err := ioutil.ReadAll(fRes.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer fRes.Body.Close()

	copyHeaders(rw.Header(), fRes.Header)
	removeHeaders(rw.Header(), hopHeaders...)

	// Grab the location header, if any.
	redirectURL, err := fRes.Location()

	if err != nil {
		if err != http.ErrNoLocation {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if redirectURL.String() != "" {
		// Set the location in our response if one was sent back.
		rw.Header().Set("Location", redirectURL.String())
	}

	rw.WriteHeader(fRes.StatusCode)
	_, _ = rw.Write(body)
}
