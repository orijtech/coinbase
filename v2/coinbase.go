// Copyright 2017 orijtech. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coinbase

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/orijtech/otils"
)

const (
	baseURL = "https://api.coinbase.com/v2"

	unversionedBaseURL = "https://api.coinbase.com"
)

type Client struct {
	sync.RWMutex

	apiKey    string
	apiSecret string

	rt http.RoundTripper
}

type Credentials struct {
	APIKey    string
	APISecret string
}

var (
	errNilCredentials = errors.New("expecting non-nil credentials")
)

func NewClient(creds *Credentials) (*Client, error) {
	if creds == nil {
		return nil, errNilCredentials
	}
	c := &Client{apiKey: creds.APIKey, apiSecret: creds.APISecret}
	return c, nil
}

func (c *Client) SetCredentials(creds *Credentials) {
	if creds == nil {
		return
	}

	c.Lock()
	defer c.Unlock()

	c.apiKey = creds.APIKey
	c.apiSecret = creds.APISecret
}

const (
	envCoinbaseAPIKey    = "COINBASE_API_KEY"
	envCoinbaseAPISecret = "COINBASE_API_SECRET"

	apiVersion = "2016-05-16"
)

func NewDefaultClient() (*Client, error) {
	var errorsList []string

	apiKey := strings.TrimSpace(os.Getenv(envCoinbaseAPIKey))
	if apiKey == "" {
		errorsList = append(errorsList, fmt.Sprintf("could not find %q in your environment", envCoinbaseAPIKey))
	}
	apiSecret := strings.TrimSpace(os.Getenv(envCoinbaseAPISecret))
	if apiSecret == "" {
		errorsList = append(errorsList, fmt.Sprintf("could not find %q in your environment", envCoinbaseAPISecret))
	}
	return &Client{apiKey: apiKey, apiSecret: apiSecret}, nil
}

func (c *Client) signAndSetHeaders(req *http.Request) {
	// Expecting headers:
	// * CB-ACCESS-KEY
	// * CB-ACCESS-SIGN:
	//    + HMAC(timestamp + method + requestPath + body)
	// * CB-ACCESS-TIMESTAMP: Number of seconds since Unix Epoch of the request
	timestamp := time.Now().Unix()
	req.Header.Set("CB-VERSION", apiVersion)
	req.Header.Set("CB-ACCESS-TIMESTAMP", fmt.Sprintf("%d", timestamp))
	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", c.hmacSignature(req, timestamp))
}

func (c *Client) hmacSignature(req *http.Request, timestampUnix int64) string {
	var body []byte
	if req.Body != nil {
		body, _ = ioutil.ReadAll(req.Body)
		// And we have to reconstruct the body now
		prc, pwc := io.Pipe()
		go func() {
			defer pwc.Close()
			pwc.Write(body)
		}()
		req.Body = prc
	}

	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	sig := fmt.Sprintf("%d%s%s%s", timestampUnix, req.Method, req.URL.Path, body)
	mac.Write([]byte(sig))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (c *Client) SetHTTPRoundTripper(rt http.RoundTripper) {
	c.Lock()
	defer c.Unlock()

	c.rt = rt
}

func (c *Client) httpClient() *http.Client {
	c.RLock()
	rt := c.rt
	c.RUnlock()
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &http.Client{Transport: rt}
}

func (c *Client) doAuthAndReq(req *http.Request) ([]byte, http.Header, error) {
	c.signAndSetHeaders(req)
	return c.doHTTPReq(req)
}

func (c *Client) doHTTPReq(req *http.Request) ([]byte, http.Header, error) {
	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	if otils.StatusOK(res.StatusCode) {
		var slurp []byte
		if res.Body != nil {
			slurp, err = ioutil.ReadAll(res.Body)
		}
		return slurp, res.Header, err
	}

	// Otherwise we've encountered an error
	if res.Body == nil {
		err = errors.New(res.Status)
	} else {
		var slurp []byte
		slurp, err = ioutil.ReadAll(res.Body)
		if err != nil {
			err = errors.New(res.Status)
		} else if len(slurp) > 3 {
			err = errors.New(string(slurp))
		}
	}

	return nil, res.Header, err
}
