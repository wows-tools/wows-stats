// Package pester provides additional resiliency over the standard http client methods by
// allowing you to control request rates, retries, and a backoff strategy.
package pester

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	methodDo                  = "Do"
	methodGet                 = "Get"
	methodHead                = "Head"
	methodPost                = "Post"
	methodPostForm            = "PostForm"
	headerKeyContentType      = "Content-Type"
	contentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

type ResponseError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    int    `json:"code"`
	Value   string `json:"value"`
}

type wgResponse struct {
	Status string         `json:"status"`
	Error  *ResponseError `json:"error,omitempty"`
}

var (
	// ErrUnexpectedMethod occurs when an http.Client method is unable to be mapped from a calling method in the pester client
	ErrUnexpectedMethod = errors.New("unexpected client method, must be one of Do, Get, Head, Post, or PostFrom")
	// ErrReadingBody happens when we cannot read the body bytes
	// Deprecated: use ErrReadingRequestBody
	ErrReadingBody = errors.New("error reading body")
	// ErrReadingRequestBody happens when we cannot read the request body bytes
	ErrReadingRequestBody = errors.New("error reading request body")
)

// Client wraps the http client and exposes all the functionality of the http.Client.
// Additionally, Client provides pester specific values for handling resiliency.
type Client struct {
	// wrap it to provide access to http built ins
	hc *http.Client

	transport     http.RoundTripper
	checkRedirect func(req *http.Request, via []*http.Request) error
	timeout       time.Duration
	jar           http.CookieJar
	Ratelimiter   *rate.Limiter

	// pester specific
	MaxRetries     int
	Backoff        BackoffStrategy
	LogHook        LogHook
	ContextLogHook ContextLogHook

	ErrLog         []ErrEntry
	RetryOnHTTP429 bool
}

// ErrEntry is used to provide the LogString() data and is populated
// each time an error happens if KeepLog is set.
// ErrEntry.Retry is deprecated in favor of ErrEntry.Attempt
type ErrEntry struct {
	Time    time.Time
	Method  string
	URL     string
	Verb    string
	Request int
	Retry   int
	Attempt int
	Err     error
}

// result simplifies the channel communication for concurrent request handling
type result struct {
	resp  *http.Response
	err   error
	req   int
	retry int
}

// params represents all the params needed to run http client calls and pester errors
type params struct {
	method   string
	verb     string
	req      *http.Request
	url      string
	bodyType string
	body     io.ReadCloser
	data     url.Values
}

var random *rand.Rand

var DefaultClient = &Client{MaxRetries: 3, Backoff: DefaultBackoff, ErrLog: []ErrEntry{}}

// New constructs a new DefaultClient with sensible default values
func New() *Client {
	return &Client{
		MaxRetries:     DefaultClient.MaxRetries,
		Backoff:        DefaultClient.Backoff,
		ErrLog:         DefaultClient.ErrLog,
		hc:             &http.Client{},
		RetryOnHTTP429: false,
	}
}

func (c *Client) SetCheckRedirect(checkRedirect func(req *http.Request, via []*http.Request) error) {
	c.hc.CheckRedirect = checkRedirect
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.hc.Timeout = timeout
}

func (c *Client) SetJar(jar http.CookieJar) {
	c.hc.Jar = jar
}

func (c *Client) SetTransport(transport http.RoundTripper) {
	c.hc.Transport = transport
}

// LogHook is used to log attempts as they happen. This function is never called,
// however, if KeepLog is set to true.
type LogHook func(e ErrEntry)

// ContextLogHook does the same as LogHook but with passed Context
type ContextLogHook func(ctx context.Context, e ErrEntry)

// BackoffStrategy is used to determine how long a retry request should wait until attempted
type BackoffStrategy func(retry int) time.Duration

// DefaultBackoff always returns 1 second
func DefaultBackoff(_ int) time.Duration {
	return 1 * time.Second
}

// ExponentialBackoff returns ever increasing backoffs by a power of 2
func ExponentialBackoff(i int) time.Duration {
	return time.Duration(1<<uint(i)) * time.Second
}

// ExponentialJitterBackoff returns ever increasing backoffs by a power of 2
// with +/- 0-33% to prevent sychronized reuqests.
func ExponentialJitterBackoff(i int) time.Duration {
	return jitter(int(1 << uint(i)))
}

// LinearBackoff returns increasing durations, each a second longer than the last
func LinearBackoff(i int) time.Duration {
	return time.Duration(i) * time.Second
}

// LinearJitterBackoff returns increasing durations, each a second longer than the last
// with +/- 0-33% to prevent sychronized reuqests.
func LinearJitterBackoff(i int) time.Duration {
	return jitter(i)
}

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// jitter keeps the +/- 0-33% logic in one place
func jitter(i int) time.Duration {
	ms := i * 1000

	maxJitter := ms / 3

	// ms ± rand
	ms += random.Intn(2*maxJitter) - maxJitter

	// a jitter of 0 messes up the time.Tick chan
	if ms <= 0 {
		ms = 1
	}

	return time.Duration(ms) * time.Millisecond
}

func (c *Client) copyBody(src io.ReadCloser) ([]byte, error) {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, ErrReadingRequestBody
	}
	src.Close()

	return b, nil
}

// resetBody resets the Body and GetBody fields of an http.Request to new Readers over
// the originalBody. This is used to refresh http.Requests that may have had their
// bodies closed already.
func resetBody(request *http.Request, originalBody []byte) {
	request.Body = io.NopCloser(bytes.NewBuffer(originalBody))
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(originalBody)), nil
	}
}

// pester provides all the logic of retries, concurrency, backoff, and logging
func (c *Client) pester(p params) (*http.Response, error) {

	// if we have a request body, we need to save it for later
	var (
		err          error
		request      *http.Request
		originalBody []byte
	)

	if p.req != nil && p.req.Body != nil && p.body == nil {
		originalBody, err = c.copyBody(p.req.Body)
	} else if p.body != nil {
		originalBody, err = c.copyBody(p.body)
	}
	if err != nil {
		return nil, err
	}

	// check to make sure that we aren't trying to use an unsupported method
	switch p.method {
	case methodDo:
		request = p.req
	case methodGet, methodHead:
		request, err = http.NewRequest(p.verb, p.url, nil)
	case methodPostForm, methodPost:
		request, err = http.NewRequest(http.MethodPost, p.url, bytes.NewBuffer(originalBody))
	default:
		return nil, ErrUnexpectedMethod
	}

	if err != nil {
		return nil, err
	}

	if len(p.bodyType) > 0 {
		request.Header.Set(headerKeyContentType, p.bodyType)
	}

	AttemptLimit := c.MaxRetries
	if AttemptLimit <= 0 {
		AttemptLimit = 1
	}

	var resp *http.Response
	for i := 1; i <= AttemptLimit; i++ {
		if c.Ratelimiter != nil {
			c.Ratelimiter.Wait(request.Context())
		}
		resp, err = c.hc.Do(request)

		// WG API is weird, in case we exceed the API rate (10 req/s), it returns 200 instead of a 429 error
		// So we need to parse the body, and check the error code in the json payload
		wgResp := wgResponse{}
		if err == nil {
			respBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(respBytes, &wgResp)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewReader(respBytes))
		}
		if wgResp.Status == "error" {
			err = errors.New("WOWS_ERROR: " + wgResp.Status)
		}
		if err == nil && resp.StatusCode < http.StatusInternalServerError && wgResp.Status != "error" {
			return resp, err
		}

		loggingContext := request.Context()
		c.log(
			loggingContext,
			ErrEntry{
				Time:    time.Now(),
				Method:  p.method,
				Verb:    request.Method,
				URL:     request.URL.String(),
				Request: AttemptLimit,
				Retry:   i,
				Attempt: i,
				Err:     err,
			},
		)

		// if we are retrying, we should close this response body to free the fd
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(c.Backoff(i) + 1*time.Millisecond)
	}

	return resp, err
}

// Format the Error to human readable string
func (c *Client) FormatError(e ErrEntry) string {
	if e.Err == nil {
		return fmt.Sprintf("%d %s [%s] %s request-%d retry-%d",
			e.Time.Unix(), e.Method, e.Verb, e.URL, e.Request, e.Retry)

	} else {
		return fmt.Sprintf("%d %s [%s] %s request-%d retry-%d error: %s\n",
			e.Time.Unix(), e.Method, e.Verb, e.URL, e.Request, e.Retry, e.Err)

	}
}

func (c *Client) log(ctx context.Context, e ErrEntry) {
	if c.LogHook != nil {
		c.LogHook(e)
		return
	}
}

// Do provides the same functionality as http.Client.Do
func (c *Client) Do(req *http.Request) (resp *http.Response, err error) {
	return c.pester(params{method: methodDo, req: req, verb: req.Method, url: req.URL.String()})
}

// Get provides the same functionality as http.Client.Get
func (c *Client) Get(url string) (resp *http.Response, err error) {
	return c.pester(params{method: methodGet, url: url, verb: http.MethodGet})
}

// Head provides the same functionality as http.Client.Head
func (c *Client) Head(url string) (resp *http.Response, err error) {
	return c.pester(params{method: methodHead, url: url, verb: http.MethodHead})
}

// Post provides the same functionality as http.Client.Post
func (c *Client) Post(url string, bodyType string, body io.Reader) (resp *http.Response, err error) {
	return c.pester(params{method: methodPost, url: url, bodyType: bodyType, body: ioutil.NopCloser(body), verb: http.MethodPost})
}

// PostForm provides the same functionality as http.Client.PostForm
func (c *Client) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return c.pester(params{method: methodPostForm, url: url, bodyType: contentTypeFormURLEncoded, body: ioutil.NopCloser(strings.NewReader(data.Encode())), verb: http.MethodPost})
}

// set RetryOnHTTP429 for clients,
func (c *Client) SetRetryOnHTTP429(flag bool) {
	c.RetryOnHTTP429 = flag
}

////////////////////////////////////////
// Provide self-constructing variants //
////////////////////////////////////////

// Do provides the same functionality as http.Client.Do and creates its own constructor
func Do(req *http.Request) (resp *http.Response, err error) {
	c := New()
	return c.Do(req)
}

// Get provides the same functionality as http.Client.Get and creates its own constructor
func Get(url string) (resp *http.Response, err error) {
	c := New()
	return c.Get(url)
}

// Head provides the same functionality as http.Client.Head and creates its own constructor
func Head(url string) (resp *http.Response, err error) {
	c := New()
	return c.Head(url)
}

// Post provides the same functionality as http.Client.Post and creates its own constructor
func Post(url string, bodyType string, body io.Reader) (resp *http.Response, err error) {
	c := New()
	return c.Post(url, bodyType, body)
}

// PostForm provides the same functionality as http.Client.PostForm and creates its own constructor
func PostForm(url string, data url.Values) (resp *http.Response, err error) {
	c := New()
	return c.PostForm(url, data)
}
