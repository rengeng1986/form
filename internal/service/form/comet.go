package form

import (
	"context"
	"fmt"

	"github.com/quanxiang-cloud/form/internal/service/consensus"
	client2 "github.com/quanxiang-cloud/form/pkg/misc/client"
)

type comet struct {
	formClient *client2.FormAPI
}

func newForm() (consensus.Guidance, error) {
	formApi, err := client2.NewFormAPI()
	if err != nil {
		return nil, err
	}
	return &comet{
		formClient: formApi,
	}, nil
}

func (c *comet) Do(ctx context.Context, bus *consensus.Bus) (*consensus.Response, error) {
	// TODO
	base := Base{
		AppID:   bus.AppID,
		TableID: bus.TableID,
		UserID:  bus.UserID,
	}
	switch bus.Foundation.Method {
	case "get":
		req := &GetReq{
			Base:  base,
			Query: bus.Query,
		}
		req.Base = base
		req.Query = bus.Query
		return c.callGet(ctx, req)

	case "search":
		req := &SearchReq{
			Sort:  bus.List.Sort,
			Page:  bus.List.Page,
			Size:  bus.List.Size,
			Query: bus.Query,
			Base:  base,
		}
		return c.callSearch(ctx, req)
	case "create":
		req := &CreateReq{
			Entity: bus.CreatedOrUpdate.Entity,
			Base:   base,
		}
		return c.callCreate(ctx, req)
	case "update":
		req := &UpdateReq{
			Entity: bus.CreatedOrUpdate.Entity,
			Query:  bus.Query,
			Base:   base,
		}
		return c.callUpdate(ctx, req)
	case "delete":
		req := &DeleteReq{
			Query: bus.Query,
			Base:  base,
		}
		return c.callDelete(ctx, req)
	}
	return nil, nil
}

func (c *comet) callSearch(ctx context.Context, req *SearchReq) (*consensus.Response, error) {
	dsl := make(map[string]interface{})
	if req.Aggs != nil {
		dsl["aggs"] = req.Aggs
	}
	if req.Query != nil {
		dsl["query"] = req.Query
	}

	if len(dsl) == 0 {
		dsl = nil
	}
	formReq := &client2.FormReq{
		DslQuery: dsl,
	}
	formReq.Size = req.Size
	formReq.Page = req.Page
	formReq.Sort = req.Sort
	formReq.TableID = getTableID(req.AppID, req.TableID)

	searchResp, err := c.formClient.Search(ctx, formReq)
	if err != nil {
		return nil, err
	}
	response := new(consensus.Response)
	response.ListResp.Total = searchResp.Total
	response.ListResp.Entities = searchResp.Entities
	return &consensus.Response{}, nil
}

func (c *comet) callCreate(ctx context.Context, req *CreateReq) (*consensus.Response, error) {
	formReq := &client2.FormReq{
		Entity:  req.Entity,
		TableID: getTableID(req.AppID, req.TableID),
	}
	insert, err := c.formClient.Insert(ctx, formReq)
	if err != nil {
		return nil, err
	}
	resp := new(consensus.Response)
	resp.CreatedOrUpdateResp.Count = insert.SuccessCount
	resp.CreatedOrUpdateResp.Data = req.Entity
	return resp, nil
}

func (c *comet) callUpdate(ctx context.Context, req *UpdateReq) (*consensus.Response, error) {
	req.Entity = consensus.DefaultField(req.Entity,
		consensus.WithUpdated(req.UserID, req.UserName))
	dsl := make(map[string]interface{})
	if req.Query != nil {
		dsl["query"] = req.Query
	}
	if len(dsl) == 0 {
		dsl = nil
	}

	formReq := &client2.FormReq{
		Entity:   req.Entity,
		TableID:  getTableID(req.AppID, req.TableID),
		DslQuery: dsl,
	}
	update, err := c.formClient.Update(ctx, formReq)
	if err != nil {
		return nil, err
	}
	resp := &consensus.Response{}
	resp.CreatedOrUpdateResp.Count = update.SuccessCount
	resp.CreatedOrUpdateResp.Data = req.Entity
	return resp, nil
}

func getTableID(appID, tableID string) string {
	if len(appID) == 36 {
		return fmt.Sprintf("%s%s%s", "A", appID, tableID)
	}
	return fmt.Sprintf("%s%s%s", "a", appID, tableID)
}

func (c *comet) callGet(ctx context.Context, req *GetReq) (*consensus.Response, error) {
	dsl := make(map[string]interface{})
	if req.Query != nil {
		dsl["query"] = req.Query
	}
	if len(dsl) == 0 {
		dsl = nil
	}

	formReq := &client2.FormReq{
		DslQuery: dsl,
		TableID:  getTableID(req.AppID, req.TableID),
	}
	gets, err := c.formClient.Get(ctx, formReq)
	if err != nil {
		return nil, err
	}
	resp := &consensus.Response{}
	resp.GetResp.Entity = gets.Entity
	return resp, nil
}

func (c *comet) callDelete(ctx context.Context, req *DeleteReq) (*consensus.Response, error) {
	dsl := make(map[string]interface{})
	if req.Query != nil {
		dsl["query"] = req.Query
	}
	if len(dsl) == 0 {
		dsl = nil
	}
	formReq := &client2.FormReq{
		DslQuery: dsl,
		TableID:  getTableID(req.AppID, req.TableID),
	}
	deletes, err := c.formClient.Delete(ctx, formReq)
	if err != nil {
		return nil, err
	}
	resp := &consensus.Response{}
	resp.DeleteResp.Count = deletes.SuccessCount
	return resp, nil
}
