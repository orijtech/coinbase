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
	mu sync.RWMutex

	apiKey    string
	apiSecret string

	passphrase string

	rt http.RoundTripper
}

type Credentials struct {
	APIKey     string
	APISecret  string
	Passphrase string
}

var (
	errNilCredentials = errors.New("expecting non-nil credentials")
)

func NewClient(creds *Credentials) (*Client, error) {
	if creds == nil {
		return nil, errNilCredentials
	}
	c := &Client{apiKey: creds.APIKey, apiSecret: creds.APISecret, passphrase: creds.Passphrase}
	return c, nil
}

func (c *Client) SetCredentials(creds *Credentials) {
	if creds == nil {
		return
	}

	c.mu.Lock()
	c.apiKey = creds.APIKey
	c.apiSecret = creds.APISecret
	c.passphrase = creds.Passphrase
	c.mu.Unlock()
}

const (
	envCoinbaseAPIKey     = "COINBASE_API_KEY"
	envCoinbaseAPISecret  = "COINBASE_API_SECRET"
	envCoinbasePassphrase = "COINBASE_API_PASSPHRASE"

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
	if len(errorsList) > 0 {
		return nil, errors.New(strings.Join(errorsList, "\n"))
	}

	// Passphrase is an optional field that's only used when
	// purchasing, canceling and viewing private content.
	passphrase := strings.TrimSpace(os.Getenv(envCoinbasePassphrase))

	return &Client{apiKey: apiKey, apiSecret: apiSecret, passphrase: passphrase}, nil
}

const (
	hdrTimestampKey  = "CB-ACCESS-TIMESTAMP"
	hdrAPIKeyKey     = "CB-ACCESS-KEY"
	hdrSignatureKey  = "CB-ACCESS-SIGN"
	hdrPassphraseKey = "CB-ACCESS-PASSPHRASE"
	hdrVersionKey    = "CB-VERSION"
)

func (c *Client) SetPassphrase(passphrase string) {
	c.mu.Lock()
	c.passphrase = passphrase
	c.mu.Unlock()
}

func (c *Client) signAndSetHeaders(req *http.Request) {
	// Expecting headers:
	// * CB-ACCESS-KEY
	// * CB-ACCESS-SIGN:
	//    + HMAC(timestamp + method + requestPath + body)
	// * CB-ACCESS-TIMESTAMP: Number of seconds since Unix Epoch of the request
	timestamp := time.Now().Unix()
	req.Header.Set(hdrVersionKey, apiVersion)
	req.Header.Set(hdrTimestampKey, fmt.Sprintf("%d", timestamp))
	if c.passphrase != "" {
		req.Header.Set(hdrPassphraseKey, c.passphrase)
	}
	req.Header.Set(hdrAPIKeyKey, c.apiKey)
	req.Header.Set(hdrSignatureKey, c.hmacSignature(req, timestamp))
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
	urlPath := req.URL.Path
	if q := req.URL.Query(); len(q) > 0 {
		urlPath += "?" + q.Encode()
	}
	sig := fmt.Sprintf("%d%s%s%s", timestampUnix, req.Method, urlPath, body)
	mac.Write([]byte(sig))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (c *Client) SetHTTPRoundTripper(rt http.RoundTripper) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rt = rt
}

func (c *Client) httpClient() *http.Client {
	c.mu.RLock()
	rt := c.rt
	c.mu.RUnlock()
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
