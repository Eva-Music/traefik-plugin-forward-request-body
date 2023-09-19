package traefik_plugin_forward_request_body

import (
	"strconv"
	"bytes"
	"context"
	"encoding/json"
	//"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Config holds the plugin configuration.
type Config struct {
	URL string `json:"url,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

type forwardRequest struct {
	name   string
	next   http.Handler
	client http.Client
	url    string
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
		name:   name,
		next:   next,
		client: client,
		url:    config.URL,
	}, nil
}

func (p *forwardRequest) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	jsonPayload, err := json.Marshal(req.Body)
	forwardReq, err := http.NewRequest(req.Method, p.url, bytes.NewBuffer(jsonPayload))
	forwardReq.Header.Set("Content-Type", req.Header.Values("Content-Type")[0])
	forwardReq.Header.Set("Accept","*/*")
	forwardReq.ContentLength = int64(len(jsonPayload))

	//proxyRequest.Header = req.Header

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Print(forwardReq.PostForm["grant_type"])

	forwardResponse, err := p.client.Do(forwardReq)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer forwardResponse.Body.Close()

	// not 2XX -> return forward response
	if forwardResponse.StatusCode < http.StatusOK || forwardResponse.StatusCode >= http.StatusMultipleChoices {
		p.writeForwardResponse(rw, forwardResponse)
		return
	}

	// 2XX -> next
	//overrideHeaders(req.Header, fRes.Header, req.Header.)
	req.Header = forwardResponse.Header
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
