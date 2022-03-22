package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"git.internal.yunify.com/qxp/misc/logger"
	"github.com/quanxiang-cloud/form/internal/permit"
	"github.com/quanxiang-cloud/form/internal/service/consensus"
	"github.com/quanxiang-cloud/form/pkg/misc/config"
)

const _query = "query"

type Proxy struct {
	next      permit.Form
	url       *url.URL
	transport http.RoundTripper
}

func NewProxy(conf *config.Config) (*Proxy, error) {
	url, err := url.Parse(conf.Endpoint.Form)
	if err != nil {
		return nil, err
	}

	return &Proxy{
		url: url,
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   conf.Transport.Timeout,
				KeepAlive: conf.Transport.KeepAlive,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          conf.Transport.MaxIdleConns,
			IdleConnTimeout:       conf.Transport.IdleConnTimeout,
			TLSHandshakeTimeout:   conf.Transport.TLSHandshakeTimeout,
			ExpectContinueTimeout: conf.Transport.ExpectContinueTimeout,
		},
	}, nil
}

func (p *Proxy) Guard(ctx context.Context, req *permit.GuardReq) (*permit.GuardResp, error) {
	proxy := httputil.NewSingleHostReverseProxy(p.url)
	proxy.Transport = p.transport
	proxy.ModifyResponse = func(resp *http.Response) error {
		return p.filter(resp, req)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Logger.Errorf("Got error while modifying response: %v \n", err)
		return
	}

	r := req.Request
	r.Host = p.url.Host
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		data, err := json.Marshal(req.Body)
		if err != nil {
			logger.Logger.Errorf("entity json marshal failed: %s", err.Error())
			return nil, err
		}

		r.Body = io.NopCloser(bytes.NewReader(data))
		r.ContentLength = int64(len(data))
	} else {
		value, err := json.Marshal(req.Get.Query)
		if err != nil {
			return nil, err
		}

		r.URL.Query().Add(_query, string(value))
	}

	proxy.ServeHTTP(req.Writer, r)

	return nil, nil
}

const (
	contentType         = "Content-Type"
	mimeApplicationJSON = "application/json"
)

func (p *Proxy) filter(resp *http.Response, req *permit.GuardReq) error {
	ctype := resp.Header.Get(contentType)
	if !strings.HasPrefix(ctype, mimeApplicationJSON) {
		return fmt.Errorf("response data content-type is not %s", mimeApplicationJSON)
	}

	respDate, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	conResp := &consensus.Response{}
	err = json.Unmarshal(respDate, conResp)
	if err != nil {
		return err
	}
	//
	//var entity interface{}
	//switch req.Param.Action {
	//case "get":
	//	entity = conResp.GetResp.Entity
	//case "search":
	//	entity = conResp.ListResp.Entities
	//}
	//filter.Post(entity, req.Permit.Response)
	//
	//data, err := json.Marshal(entity)
	//if err != nil {
	//	logger.Logger.Errorf("entity json marshal failed: %s", err.Error())
	//	return err
	//}
	//
	//resp.Body = io.NopCloser(bytes.NewReader(data))
	//resp.ContentLength = int64(len(data))
	resp.Body = io.NopCloser(bytes.NewReader(respDate))
	resp.ContentLength = int64(len(respDate))
	return nil
}
