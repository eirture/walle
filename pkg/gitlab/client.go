package gitlab

import (
	"bytes"
	"encoding/base64"
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

	datetimeFormat = time.RFC3339
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
	GetMergeRequest(project string, iid int) (*MergeRequest, error)
	CreateMergeRequest(project string, req MergeRequestRequest) (*MergeRequest, error)
	AcceptMR(project string, mrid int) (*MergeRequest, error)
	ListMergeRequests(project string, updatedAfter time.Time) ([]MergeRequest, error)
}

type TagClient interface {
	GetTag(project, tagName string) (Tag, error)
	ListTags(project string) ([]Tag, error)
	CreateTag(project string, req TagRequest) error
	UpsertRelease(project string, tag, desc string) error
}

type RepoClient interface {
	GetFile(project, filepath, ref string) (string, error)
	UpdateFile(project, filepath string, req RepoFileRequest) error
	NewBranch(project, branchName, ref string) error
	ListCommits(project, ref string, since, until *time.Time) ([]*Commit, error)
}

type ProjectClient interface {
	GetProject(project string) (Project, error)
}

type Config interface {
	GetToken() string
	GetAPIBase() string
}

type Client interface {
	MergeRequestClient
	TagClient
	RepoClient
	ProjectClient
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
	if c.dry && r.method != http.MethodGet {
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
	headers := make(map[string]string)
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(b)
		headers["Content-Type"] = "application/json"
	}

	req, err := http.NewRequest(method, path, buf)
	if err != nil {
		return nil, err
	}

	if header := c.authHeader(); len(header) > 0 {
		req.Header.Set("Authorization", header)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
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

func (c *client) ListMergeRequests(project string, mergedAfter time.Time) ([]MergeRequest, error) {
	c.log("GetMergeRequests", project)
	var mrs []MergeRequest

	path := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(project))
	values := url.Values{
		"pre_page":      []string{"100"},
		"state":         []string{"merged"},
		"updated_after": []string{mergedAfter.Format(datetimeFormat)},
	}
	err := c.readPaginatedResultsWithValues(
		path,
		values,
		func() interface{} {
			return &[]MergeRequest{}
		},
		func(obj interface{}) {
			partialMrs := *(obj.(*[]MergeRequest))
			for _, mr := range partialMrs {
				if mergedAfter.After(mr.MergedAt) {
					continue
				}
				mrs = append(mrs, mr)
			}
		},
	)
	if err != nil {
		return nil, err
	}
	return mrs, err
}

func (c *client) GetMergeRequest(project string, iid int) (*MergeRequest, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%d", url.PathEscape(project), iid)

	mr := &MergeRequest{}
	_, err := c.request(&request{
		method:    http.MethodGet,
		path:      path,
		exitCodes: []int{200},
	}, mr)
	if err != nil {
		return nil, err
	}
	return mr, nil
}

func (c *client) CreateMergeRequest(project string, req MergeRequestRequest) (*MergeRequest, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests", url.PathEscape(project))
	mr := MergeRequest{}
	_, err := c.request(&request{
		method:      http.MethodPost,
		path:        path,
		requestBody: &req,
		exitCodes:   []int{201},
	}, &mr)
	return &mr, err
}

func (c *client) AcceptMR(project string, mrid int) (*MergeRequest, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%d/merge",
		url.PathEscape(project),
		mrid,
	)
	mr := MergeRequest{}
	_, err := c.request(&request{
		method:    http.MethodPut,
		path:      path,
		exitCodes: []int{200},
	}, &mr)
	return &mr, err
}

func (c *client) ListTags(project string) ([]Tag, error) {
	c.log("ListTags", project)
	var tags []Tag

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
		exitCodes: []int{200, 201},
	}, nil)
	return err
}

func (c *client) GetTag(project, tag string) (Tag, error) {
	path := fmt.Sprintf(
		"/projects/%s/repository/tags/%s",
		url.PathEscape(project),
		url.PathEscape(tag),
	)

	t := Tag{}

	_, err := c.request(&request{
		method:    http.MethodGet,
		path:      path,
		exitCodes: []int{200},
	}, &t)
	return t, err
}

func (c *client) UpsertRelease(project string, tag, desc string) error {
	method := http.MethodPost // Create
	if t, err := c.GetTag(project, tag); err == nil && t.Release != nil {
		method = http.MethodPut
	}

	path := fmt.Sprintf(
		"/projects/%s/repository/tags/%s/release",
		url.PathEscape(project),
		url.PathEscape(tag),
	)

	values := url.Values{
		"description": []string{desc},
	}

	_, err := c.request(&request{
		method:    method,
		path:      path + "?" + values.Encode(),
		exitCodes: []int{200, 201},
	}, nil)
	return err
}

func (c *client) GetFile(project, filepath, ref string) (string, error) {
	path := fmt.Sprintf(
		"/projects/%s/repository/files/%s?ref=%s",
		url.PathEscape(project),
		url.PathEscape(filepath),
		ref,
	)
	file := struct {
		Content string `json:"content"`
	}{}
	_, err := c.request(&request{
		method:      http.MethodGet,
		path:        path,
		requestBody: nil,
		exitCodes:   []int{200},
	}, &file)
	if err != nil {
		return "", err
	}
	content, err := base64.StdEncoding.DecodeString(file.Content)
	return string(content), err

}

func (c *client) UpdateFile(project, filepath string, req RepoFileRequest) error {
	path := fmt.Sprintf(
		"/projects/%s/repository/files/%s",
		url.PathEscape(project),
		url.PathEscape(filepath),
	)
	_, err := c.request(&request{
		method:      http.MethodPut,
		path:        path,
		requestBody: req,
		exitCodes:   []int{200},
	}, nil)
	return err
}

func (c *client) NewBranch(project, branchName, ref string) error {
	path := fmt.Sprintf("/projects/%s/repository/branches", url.PathEscape(project))
	params := url.Values{
		"branch": []string{branchName},
		"ref":    []string{ref},
	}
	_, err := c.request(&request{
		method:    http.MethodPost,
		path:      path + "?" + params.Encode(),
		exitCodes: []int{201},
	}, nil)
	return err
}

func (c *client) ListCommits(project, ref string, since, until *time.Time) ([]*Commit, error) {
	path := fmt.Sprintf("/projects/%s/repository/commits", url.PathEscape(project))
	values := url.Values{}
	if ref != "" {
		values.Set("ref_name", ref)
	}
	if since != nil {
		values.Set("since", since.Format(datetimeFormat))
	}
	if until != nil {
		values.Set("until", until.Format(datetimeFormat))
	}

	var results []*Commit
	err := c.readPaginatedResultsWithValues(
		path,
		values,
		func() interface{} {
			return &[]*Commit{}
		},
		func(obj interface{}) {
			results = append(results, *(obj.(*[]*Commit))...)
		},
	)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (c *client) GetProject(project string) (pro Project, err error) {
	path := fmt.Sprintf("/projects/%s", url.PathEscape(project))
	_, err = c.request(&request{
		method:    http.MethodGet,
		path:      path,
		exitCodes: []int{200},
	}, &pro)
	return
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
