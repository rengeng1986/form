package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	error2 "github.com/quanxiang-cloud/cabin/error"
	"github.com/quanxiang-cloud/cabin/logger"
	"github.com/quanxiang-cloud/form/internal/auth/filters"
	"github.com/quanxiang-cloud/form/internal/service/consensus"
	"github.com/quanxiang-cloud/form/pkg/misc/code"
	"github.com/quanxiang-cloud/form/pkg/misc/config"
)

type formAuth struct {
	auth   *auth
	permit *consensus.Permit
}

func NewFormAuth(conf *config.Config) (Auth, error) {
	auth, err := newAuth(conf)

	return &formAuth{
		auth: auth,
	}, err
}

func (f *formAuth) Auth(ctx context.Context, req *ReqParam) (bool, error) {
	resp, err := f.auth.Auth(ctx, req)
	if err != nil {
		return false, err
	}

	if resp == nil {
		return false, nil
	}

	// access judgment
	if !filters.Pre(req.Entity, resp.Permit.Params) {
		return false, error2.New(code.ErrNotPermit)
	}

	f.permit = resp.Permit
	return true, nil
}

func (f *formAuth) Filter(resp *http.Response, method string) error {
	respDate, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	conResp := &consensus.Response{}

	err = json.Unmarshal(respDate, conResp)
	if err != nil {
		return err
	}

	var entity interface{}
	switch method {
	case "get":
		entity = conResp.GetResp.Entity
	case "search":
		entity = conResp.ListResp.Entities
	}
	filters.Post(entity, f.permit.Response)

	data, err := json.Marshal(entity)
	if err != nil {
		logger.Logger.Errorf("entity json marshal failed: %s", err.Error())
		return err
	}

	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.ContentLength = int64(len(data))
	return nil
}
