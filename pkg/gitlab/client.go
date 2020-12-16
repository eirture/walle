package gitlab

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"walle/pkg/utils"
)

const (
	maxRequestTime = 5 * time.Minute

	defaultMaxRetries      = 8
	defaultMaxInitialDelay = 2 * time.Second
	defaultMaxSleepTime    = 2 * time.Minute
)

type timeClient interface {
	Sleep(time.Duration)
	Until(time.Time) time.Duration
}

type standardTime struct{}

func (s *standardTime) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (s *standardTime) Until(t time.Time) time.Duration {
	return time.Until(t)
}

type MergeRequestClient interface {
	ListMergeRequests(project string, updatedAfter time.Time) ([]MergeRequest, error)
}

type TagClient interface {
	ListTags(project string) ([]Tag, error)
	CreateTag(project string, req TagRequest) error
}

type Config interface {
	GetToken() string
	GetAPIBase() string
}

type Client interface {
	MergeRequestClient
	TagClient
}

type client struct {
	logger *logrus.Entry
	*delegate
}

type delegate struct {
	time         timeClient
	client       httpClient
	maxRetries   int
	initialDelay time.Duration
	maxSleepTime time.Duration
	getAPIBase   func() string
	getToken     func() string
	dry          bool
	fake         bool
}

func (c *client) authHeader() string {
	if c.getToken == nil {
		return ""
	}
	token := c.getToken()
	if len(token) == 0 {
		return ""
	}
	return fmt.Sprintf("Bearer %s", token)
}

func (c *client) request(r *request, ret interface{}) (int, error) {
	statusCode, b, err := c.requestRaw(r)
	if err != nil {
		return statusCode, err
	}
	if ret != nil {
		if err = json.Unmarshal(b, ret); err != nil {
			return statusCode, err
		}
	}
	return statusCode, nil
}

func (c *client) requestRaw(r *request) (int, []byte, error) {
	if c.fake || (c.dry && r.method != http.MethodGet) {
		return r.exitCodes[0], nil, nil
	}
	resp, err := c.requestRetry(r.method, r.path, r.requestBody)
	if err != nil {
		return 0, nil, err
	}
	defer utils.CloseSilently(resp.Body)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil
	}
	var okCode bool
	for _, code := range r.exitCodes {
		if code == resp.StatusCode {
			okCode = true
			break
		}
	}
	if !okCode {
		err = requestError{
			ErrorString: fmt.Sprintf("status code %d not one of %v, body: %s", resp.StatusCode, r.exitCodes, string(b)),
		}
	}
	return resp.StatusCode, b, err
}

func (c *client) requestRetry(method, path string, body interface{}) (*http.Response, error) {
	var resp *http.Response
	var err error
	backoff := c.initialDelay
	for retries := 0; retries < c.maxRetries; retries++ {
		if retries > 0 && resp != nil {
			_ = resp.Body.Close()
		}
		base := c.getAPIBase()
		resp, err = c.doRequest(method, base+path, body)
		if err == nil {
			if resp.StatusCode < 500 {
				break
			} else {
				c.logger.WithField("backoff", backoff.String()).Debug("Retrying 5XX")
				c.time.Sleep(backoff)
				backoff *= 2
			}
		} else if errors.Is(err, &authError{}) {
			c.logger.WithError(err).Error("Stopping retry dur to authError")
			return resp, err
		} else {
			c.logger.WithFields(logrus.Fields{
				"err":      err,
				"backoff":  backoff.String(),
				"endpoint": base,
			}).Debug("Retrying request due to connection problem")
			c.time.Sleep(backoff)
			backoff *= 2
		}
	}
	return resp, err
}

func (c *client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, path, buf)
	if err != nil {
		return nil, err
	}
	if header := c.authHeader(); len(header) > 0 {
		req.Header.Set("Authorization", header)
	}

	req.Close = true
	return c.client.Do(req)
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type request struct {
	method      string
	path        string
	requestBody interface{}
	exitCodes   []int
}

type requestError struct {
	ErrorString string
}

func (r requestError) Error() string {
	return r.ErrorString
}

func (c *client) log(methodName string, args ...interface{}) (logDuration func()) {
	if c.logger == nil {
		return func() {}
	}
	var as []string
	for _, arg := range args {
		as = append(as, fmt.Sprintf("%v", arg))
	}
	start := time.Now()
	c.logger.Debugf("%s(%s)", methodName, strings.Join(as, ", "))
	return func() {
		c.logger.WithField("duration", time.Since(start).String()).Debugf("%s(%s) finished", methodName, strings.Join(as, ", "))
	}
}

func (c *client) ListMergeRequests(project string, updatedAfter time.Time) ([]MergeRequest, error) {
	c.log("GetMergeRequests", project)
	var mrs []MergeRequest
	if c.fake {
		return mrs, nil
	}
	path := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(project))
	values := url.Values{
		"pre_page":      []string{"100"},
		"state":         []string{"merged"},
		"updated_after": []string{updatedAfter.Format(time.RFC3339)},
	}
	err := c.readPaginatedResultsWithValues(
		path,
		values,
		func() interface{} {
			return &[]MergeRequest{}
		},
		func(obj interface{}) {
			mrs = append(mrs, *(obj.(*[]MergeRequest))...)
		},
	)
	if err != nil {
		return nil, err
	}
	return mrs, err
}

func (c *client) ListTags(project string) ([]Tag, error) {
	c.log("ListTags", project)
	var tags []Tag
	if c.fake {
		return tags, nil
	}
	path := fmt.Sprintf("/projects/%s/repository/tags", url.PathEscape(project))
	err := c.readPaginateResults(
		path,
		func() interface{} {
			return &[]Tag{}
		},
		func(obj interface{}) {
			tags = append(tags, *(obj.(*[]Tag))...)
		},
	)
	if err != nil {
		return nil, err
	}
	return tags, err
}

func obj2values(obj interface{}) url.Values {
	v := reflect.ValueOf(obj)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	values := url.Values{}
	vt := v.Type()
	for i := 0; i < vt.NumField(); i++ {
		f := vt.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" {
			tag = strings.ToLower(f.Name)
		}
		tag = strings.Split(tag, ",")[0]
		fv := v.Field(i)
		if fv.Kind() == reflect.Slice {
			n := fv.NumField()
			for j := 0; j < n; j++ {
				values.Add(tag, fv.Field(j).String())
			}
		} else {
			values.Add(tag, fv.String())
		}
	}
	return values
}

func (c *client) CreateTag(project string, req TagRequest) error {
	path := fmt.Sprintf("/projects/%s/repository/tags", url.PathEscape(project))
	values := obj2values(req)

	_, err := c.request(&request{
		method:    http.MethodPost,
		path:      path + "?" + values.Encode(),
		exitCodes: []int{200},
	}, nil)
	return err
}

func (c *client) readPaginateResults(path string, newObj func() interface{}, accumulate func(interface{})) error {
	values := url.Values{
		"per_page": []string{"100"},
	}
	return c.readPaginatedResultsWithValues(path, values, newObj, accumulate)
}

func (c *client) readPaginatedResultsWithValues(path string, values url.Values, newObj func() interface{}, accumulate func(interface{})) error {
	pagedPath := path
	if len(values) > 0 {
		pagedPath += "?" + values.Encode()
	}
	for {
		resp, err := c.requestRetry(http.MethodGet, pagedPath, nil)
		if err != nil {
			return err
		}
		defer utils.CloseSilently(resp.Body)
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("return code not 2XX: %s", resp.Status)
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		obj := newObj()
		if err = json.Unmarshal(b, obj); err != nil {
			return err
		}

		accumulate(obj)

		link := parseLinks(resp.Header.Get("Link"))["next"]
		if link == "" {
			break
		}

		prefix := strings.TrimSuffix(resp.Request.URL.RequestURI(), pagedPath)

		u, err := url.Parse(link)
		if err != nil {
			return fmt.Errorf("failed to parse 'next' link: %v", err)
		}
		pagedPath = strings.TrimPrefix(u.RequestURI(), prefix)
	}
	return nil
}

func NewClient(logger *logrus.Entry, configProvider Config) Client {
	httpClient := &http.Client{Timeout: maxRequestTime}
	c := &client{
		logger: logger,
		delegate: &delegate{
			time:         &standardTime{},
			client:       httpClient,
			getAPIBase:   configProvider.GetAPIBase,
			getToken:     configProvider.GetToken,
			dry:          false,
			maxRetries:   defaultMaxRetries,
			initialDelay: defaultMaxInitialDelay,
			maxSleepTime: defaultMaxSleepTime,
		},
	}

	return c
}
