package traefik_plugin_forward_request_body

import (
	"bytes"
	"context"
	"io"
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

func (p *forwardRequest) writeForwardResponse(rw http.ResponseWriter, fRes *http.Response) {
	body, err := ioutil.ReadAll(fRes.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer fRes.Body.Close()

	CopyHeaders(rw.Header(), fRes.Header)
	RemoveHeaders(rw.Header(), hopHeaders...)

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
	//check first request body
	data, err := io.ReadAll(req.Body)
	req.Body.Close()
	log.Printf("Got body (%d bytes): %s", len(data), string(data))

	forwardReq, err := http.NewRequest(req.Method, p.url, bytes.NewBuffer(data))
	forwardReq.Header = req.Header
	if err != nil {
		log.Printf("Error request " + err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	forwardResponse, forwardErr := p.client.Do(forwardReq)
	if forwardErr != nil {
		log.Printf("Error response " + forwardErr.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// not 2XX -> return forward response
	if forwardResponse.StatusCode < http.StatusOK || forwardResponse.StatusCode >= http.StatusMultipleChoices {
		p.writeForwardResponse(rw, forwardResponse)
		return
	}

	req.RequestURI = req.URL.RequestURI()
	req.Header = forwardResponse.Header.Clone()

	p.next.ServeHTTP(rw, req)
}

func RemoveHeaders(headers http.Header, names ...string) {
	for _, h := range names {
		headers.Del(h)
	}
}

func CopyHeaders(dst http.Header, src http.Header) {
	for k, vv := range src {
		dst[k] = append(dst[k], vv...)
	}
}