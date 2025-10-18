package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"strconv"
)

type RPCClient interface {
	Call(ctx context.Context, method string, params ...any) (*RPCResponse, error)
	CallRaw(ctx context.Context, request *RPCRequest) (*RPCResponse, error)
	CallFor(ctx context.Context, out any, method string, params ...any) error
	CallBatch(ctx context.Context, requests RPCRequests) (RPCResponses, error)
	CallBatchRaw(ctx context.Context, requests RPCRequests) (RPCResponses, error)
}

type RPCRequest struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
	ID     int    `json:"id"`
}

func NewRequest(method string, params ...any) *RPCRequest {
	return &RPCRequest{Method: method, Params: Params(params...)}
}

func NewRequestWithID(id int, method string, params ...any) *RPCRequest {
	return &RPCRequest{ID: id, Method: method, Params: Params(params...)}
}

type RPCResponse struct {
	Result any       `json:"result,omitempty"`
	Error  *RPCError `json:"error,omitempty"`
	ID     int       `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return strconv.Itoa(e.Code) + ": " + e.Message
}

type HTTPError struct {
	Code int
	err  error
}

func (e *HTTPError) Error() string { return e.err.Error() }

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type rpcClient struct {
	endpoint           string
	httpClient         HTTPClient
	customHeaders      map[string]string
	allowUnknownFields bool
	defaultRequestID   int
}

type RPCClientOpts struct {
	HTTPClient         HTTPClient
	CustomHeaders      map[string]string
	AllowUnknownFields bool
	DefaultRequestID   int
}

type RPCResponses []*RPCResponse
type RPCRequests []*RPCRequest

func (res RPCResponses) AsMap() map[int]*RPCResponse {
	m := make(map[int]*RPCResponse, len(res))
	for _, r := range res {
		m[r.ID] = r
	}
	return m
}

func (res RPCResponses) GetByID(id int) *RPCResponse {
	for _, r := range res {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (res RPCResponses) HasError() bool {
	for _, r := range res {
		if r.Error != nil {
			return true
		}
	}
	return false
}

func NewClient(endpoint string) RPCClient {
	return NewClientWithOpts(endpoint, nil)
}

func NewClientWithOpts(endpoint string, opts *RPCClientOpts) RPCClient {
	c := &rpcClient{
		endpoint:      endpoint,
		httpClient:    &http.Client{},
		customHeaders: make(map[string]string),
	}
	if opts == nil {
		return c
	}
	if opts.HTTPClient != nil {
		c.httpClient = opts.HTTPClient
	}
	if opts.CustomHeaders != nil {
		maps.Copy(c.customHeaders, opts.CustomHeaders)
	}
	c.allowUnknownFields = opts.AllowUnknownFields
	c.defaultRequestID = opts.DefaultRequestID
	return c
}

// Modified: Call now returns RPC errors as Go error.
func (c *rpcClient) Call(ctx context.Context, method string, params ...any) (*RPCResponse, error) {
	req := &RPCRequest{
		ID:     c.defaultRequestID,
		Method: method,
		Params: Params(params...),
	}
	resp, err := c.doCall(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.Error != nil {
		return resp, resp.Error
	}
	return resp, nil
}

func (c *rpcClient) CallRaw(ctx context.Context, req *RPCRequest) (*RPCResponse, error) {
	return c.doCall(ctx, req)
}

func (c *rpcClient) CallFor(ctx context.Context, out any, method string, params ...any) error {
	resp, err := c.Call(ctx, method, params...)
	if err != nil {
		return err
	}
	// resp.Error will already have been returned by Call
	return resp.GetObject(out)
}

func (c *rpcClient) CallBatch(ctx context.Context, requests RPCRequests) (RPCResponses, error) {
	if len(requests) == 0 {
		return nil, errors.New("empty request list")
	}
	for i, req := range requests {
		req.ID = i
	}
	return c.doBatchCall(ctx, requests)
}

func (c *rpcClient) CallBatchRaw(ctx context.Context, requests RPCRequests) (RPCResponses, error) {
	if len(requests) == 0 {
		return nil, errors.New("empty request list")
	}
	return c.doBatchCall(ctx, requests)
}

func (c *rpcClient) newRequest(ctx context.Context, req any) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	for k, v := range c.customHeaders {
		if k == "Host" {
			httpReq.Host = v
		} else {
			httpReq.Header.Set(k, v)
		}
	}
	return httpReq, nil
}

func (c *rpcClient) doCall(ctx context.Context, req *RPCRequest) (*RPCResponse, error) {
	httpReq, err := c.newRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rpc call %v() on %v: %w", req.Method, c.endpoint, err)
	}
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("rpc call %v() on %v: %w", req.Method, httpReq.URL.Redacted(), err)
	}
	defer httpResp.Body.Close()

	var resp *RPCResponse
	dec := json.NewDecoder(httpResp.Body)
	if !c.allowUnknownFields {
		dec.DisallowUnknownFields()
	}
	dec.UseNumber()
	err = dec.Decode(&resp)
	if err != nil {
		return nil, fmt.Errorf("rpc call %v() decode error: %w", req.Method, err)
	}
	if httpResp.StatusCode >= 400 {
		return resp, &HTTPError{Code: httpResp.StatusCode, err: fmt.Errorf("rpc error status %v", httpResp.StatusCode)}
	}
	return resp, nil
}

func (c *rpcClient) doBatchCall(ctx context.Context, reqs []*RPCRequest) ([]*RPCResponse, error) {
	httpReq, err := c.newRequest(ctx, reqs)
	if err != nil {
		return nil, err
	}
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	var resps RPCResponses
	dec := json.NewDecoder(httpResp.Body)
	if !c.allowUnknownFields {
		dec.DisallowUnknownFields()
	}
	dec.UseNumber()
	if err := dec.Decode(&resps); err != nil {
		return nil, fmt.Errorf("decode batch: %w", err)
	}
	if httpResp.StatusCode >= 400 {
		return resps, &HTTPError{Code: httpResp.StatusCode, err: fmt.Errorf("rpc batch error %v", httpResp.StatusCode)}
	}
	return resps, nil
}

func Params(params ...any) any {
	if len(params) == 0 {
		return nil
	}
	if len(params) == 1 && params[0] != nil {
		t := reflect.TypeOf(params[0])
		for t != nil && t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t == nil {
			return params
		}
		switch t.Kind() {
		case reflect.Struct, reflect.Array, reflect.Slice, reflect.Interface, reflect.Map:
			return params[0]
		default:
			return params
		}
	}
	return params
}

func (r *RPCResponse) GetInt() (int64, error) {
	val, ok := r.Result.(json.Number)
	if !ok {
		return 0, fmt.Errorf("invalid int: %v", r.Result)
	}
	return val.Int64()
}

func (r *RPCResponse) GetFloat() (float64, error) {
	val, ok := r.Result.(json.Number)
	if !ok {
		return 0, fmt.Errorf("invalid float: %v", r.Result)
	}
	return val.Float64()
}

func (r *RPCResponse) GetBool() (bool, error) {
	val, ok := r.Result.(bool)
	if !ok {
		return false, fmt.Errorf("invalid bool: %v", r.Result)
	}
	return val, nil
}

func (r *RPCResponse) GetString() (string, error) {
	val, ok := r.Result.(string)
	if !ok {
		return "", fmt.Errorf("invalid string: %v", r.Result)
	}
	return val, nil
}

func (r *RPCResponse) GetObject(to any) error {
	js, err := json.Marshal(r.Result)
	if err != nil {
		return err
	}
	return json.Unmarshal(js, to)
}
