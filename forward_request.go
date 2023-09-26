package traefik_plugin_forward_request_body

import (
	//"bytes"
	"context"
	"encoding/json"
	"errors"
	//"io/ioutil"
	//"log"
	"net/http"
	"net/url"
	"strconv"
	//"io"
	"strings"
	"time"
)

// Config holds the plugin configuration.
type Config struct {
	URL string `json:"url,omitempty"`
}

type token struct {
	AccessToken 	   string `json:"access_token"`
	ExpiresIn   	   int    `json:"expires_in"`
	RefreshExpiresIn   int    `json:"refresh_expires_in"`
	RefreshToken 	   string `json:"refresh_token"`
	TokenType 	   	   string `json:"token_type"`
	NotBeforePolicy    int    `json:"not-before-policy"`
	SessionState 	   string `json:"session_state"`
	Scope 	   		   string `json:"scope"`
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
func New(_ context.Context, next http.Handler, config *Config, _ string) (http.Handler, error) {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	return &forwardRequest{
		next:   next,
		client: client,
		url:    config.URL,
	}, nil
}

func (p *forwardRequest) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	data, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		errorResponse(rw,"Error " + err.Error(), http.StatusInternalServerError)
		return
	}

	forwardReq, err := http.NewRequest(http.MethodPost, p.url,strings.NewReader(data.Encode()))
	forwardReq.Header = req.Header
	forwardReq.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	if err != nil {
		errorResponse(rw,"Bad Request " + err.Error(), http.StatusBadRequest)
		return
	}

	forwardResponse, err := p.client.Do(forwardReq)
	if err != nil {
		errorResponse(rw,"Bad Request " + err.Error(), http.StatusBadRequest)
		return
	}
	defer forwardResponse.Body.Close()

	// not 2XX -> return forward response
	if forwardResponse.StatusCode < http.StatusOK || forwardResponse.StatusCode >= http.StatusMultipleChoices {
		errorResponse(rw, "Bad Request " + err.Error(), http.StatusInternalServerError)
		return
	} else {
		p.writeForwardResponse(rw, forwardResponse)
	}

	p.next.ServeHTTP(rw, req)
}

func (p *forwardRequest) writeForwardResponse(rw http.ResponseWriter, fRes *http.Response) {
	//body, err := ioutil.ReadAll(fRes.Body)
	//if err != nil {
	//	rw.WriteHeader(http.StatusInternalServerError)
	//	return
	//}
	//defer fRes.Body.Close()

	//add access_token to header if exist
	var t token
	var unmarshalErr *json.UnmarshalTypeError

	decoder := json.NewDecoder(fRes.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&t)

	if err != nil {
		if errors.As(err, &unmarshalErr) {
			errorResponse(rw,"Bad Request. Wrong Type provided for field " + unmarshalErr.Field,
				http.StatusInternalServerError)
			return
		} else {
			errorResponse(rw,"Bad Request " + err.Error(), http.StatusInternalServerError)
			return
		}
	}
	defer fRes.Body.Close()

	copyHeaders(rw.Header(), fRes.Header)
	removeHeaders(rw.Header(), hopHeaders...)

	rw.Header().Set("Authorization", t.TokenType + " " + t.AccessToken)

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
	//_, _ = rw.Write(body)
}

func errorResponse(rw http.ResponseWriter, message string, httpStatusCode int) {
	resp := make(map[string]string)
	rw.WriteHeader(httpStatusCode)
	resp["error"] = message
	jsonResp, _ := json.Marshal(resp)
	rw.Write(jsonResp)
}