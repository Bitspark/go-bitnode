package factories

import (
	"bytes"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"io"
	"net/http"
	"strings"
)

// Web factory.

type WebFactory struct {
}

var _ bitnode.Factory = &WebFactory{}

func NewWebFactory() *WebFactory {
	return &WebFactory{}
}

func (f *WebFactory) Name() string {
	return "web"
}

func (f *WebFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &WebImpl{}, nil
	}
	nImpl, ok := impl.(*WebImpl)
	if !ok {
		return nil, fmt.Errorf("not a web implementation")
	}
	return nImpl, nil
}

// Web implementation.

type WebImpl struct {
	System string `json:"system" yaml:"system"`
	creds  bitnode.Credentials
}

var _ bitnode.Implementation = &WebImpl{}

func (m *WebImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	if m.System == "HTTPClient" {
		c := &webImpl{m: m, client: &http.Client{}}

		sys.AddExtension("web", c)

		// Hubs

		getHub := sys.GetHub("getText")
		getHub.Handle(bitnode.NewNativeFunction(func(user bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
			req, err := http.NewRequest("GET", vals[0].(string), nil)
			if err != nil {
				return nil, err
			}
			for _, e := range vals[1].([]bitnode.HubItem) {
				em := e.(map[string]bitnode.HubItem)
				req.Header.Set(em["key"].(string), em["value"].(string))
			}
			resp, err := c.client.Do(req)
			if err != nil {
				return nil, err
			}
			bodyBts, _ := io.ReadAll(resp.Body)
			statusCode := resp.StatusCode
			respHeaders := []map[string]bitnode.HubItem{}
			for k, v := range resp.Header {
				respHeaders = append(respHeaders, map[string]bitnode.HubItem{
					"key":   k,
					"value": strings.Join(v, ","),
				})
			}
			return []bitnode.HubItem{string(bodyBts), statusCode, respHeaders}, nil
		}))

		postHub := sys.GetHub("postText")
		postHub.Handle(bitnode.NewNativeFunction(func(user bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
			reqBody := &bytes.Buffer{}
			reqBody.WriteString(vals[1].(string))
			req, err := http.NewRequest("POST", vals[0].(string), reqBody)
			if err != nil {
				return nil, err
			}
			for _, e := range vals[2].([]bitnode.HubItem) {
				em := e.(map[string]bitnode.HubItem)
				req.Header.Set(em["key"].(string), em["value"].(string))
			}
			resp, err := c.client.Do(req)
			if err != nil {
				return nil, err
			}
			bodyBts, _ := io.ReadAll(resp.Body)
			statusCode := resp.StatusCode
			respHeaders := []map[string]bitnode.HubItem{}
			for k, v := range resp.Header {
				respHeaders = append(respHeaders, map[string]bitnode.HubItem{
					"key":   k,
					"value": strings.Join(v, ","),
				})
			}
			return []bitnode.HubItem{string(bodyBts), statusCode, respHeaders}, nil
		}))

		deleteText := sys.GetHub("deleteText")
		deleteText.Handle(bitnode.NewNativeFunction(func(user bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
			req, err := http.NewRequest("DELETE", vals[0].(string), nil)
			if err != nil {
				return nil, err
			}
			for _, e := range vals[1].([]bitnode.HubItem) {
				em := e.(map[string]bitnode.HubItem)
				req.Header.Set(em["key"].(string), em["value"].(string))
			}
			resp, err := c.client.Do(req)
			if err != nil {
				return nil, err
			}
			bodyBts, _ := io.ReadAll(resp.Body)
			statusCode := resp.StatusCode
			respHeaders := []map[string]bitnode.HubItem{}
			for k, v := range resp.Header {
				respHeaders = append(respHeaders, map[string]bitnode.HubItem{
					"key":   k,
					"value": strings.Join(v, ","),
				})
			}
			return []bitnode.HubItem{string(bodyBts), statusCode, respHeaders}, nil
		}))

		putText := sys.GetHub("putText")
		putText.Handle(bitnode.NewNativeFunction(func(user bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
			reqBody := &bytes.Buffer{}
			reqBody.WriteString(vals[1].(string))
			req, err := http.NewRequest("PUT", vals[0].(string), reqBody)
			if err != nil {
				return nil, err
			}
			for _, e := range vals[2].([]bitnode.HubItem) {
				em := e.(map[string]bitnode.HubItem)
				req.Header.Set(em["key"].(string), em["value"].(string))
			}
			resp, err := c.client.Do(req)
			if err != nil {
				return nil, err
			}
			bodyBts, _ := io.ReadAll(resp.Body)
			statusCode := resp.StatusCode
			respHeaders := []map[string]bitnode.HubItem{}
			for k, v := range resp.Header {
				respHeaders = append(respHeaders, map[string]bitnode.HubItem{
					"key":   k,
					"value": strings.Join(v, ","),
				})
			}
			return []bitnode.HubItem{string(bodyBts), statusCode, respHeaders}, nil
		}))

		// Status and message

		sys.LogInfo("Web client running")
		sys.SetStatus(bitnode.SystemStatusRunning)

		return nil
	}
	return nil
}

func (m *WebImpl) Extend(node *bitnode.NativeNode, ext bitnode.Implementation) (bitnode.Implementation, error) {
	panic("implement me")
}

func (m *WebImpl) ToInterface() (any, error) {
	return nil, nil
}

func (m *WebImpl) FromInterface(i any) error {
	ti := i.(map[string]any)
	m.System = ti["system"].(string)
	return nil
}

func (m *WebImpl) Validate() error {
	panic("implement me")
}

type webImpl struct {
	m      *WebImpl
	client *http.Client
}

var _ bitnode.SystemExtension = &webImpl{}

func (h *webImpl) Implementation() bitnode.Implementation {
	return h.m
}
