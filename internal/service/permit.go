package service

import (
	"context"
	"time"

	"git.internal.yunify.com/qxp/misc/logger"
	id2 "github.com/quanxiang-cloud/cabin/id"
	redis2 "github.com/quanxiang-cloud/cabin/tailormade/db/redis"
	time2 "github.com/quanxiang-cloud/cabin/time"
	"github.com/quanxiang-cloud/form/internal/models"
	"github.com/quanxiang-cloud/form/internal/models/mysql"
	"github.com/quanxiang-cloud/form/internal/models/redis"
	config2 "github.com/quanxiang-cloud/form/pkg/misc/config"
	"gorm.io/gorm"
)

const (
	lockPermission = "lockPermission"
	lockPerMatch   = "lockPerMatch"
	lockTimeout    = time.Duration(30) * time.Second // 30秒
	timeSleep      = time.Millisecond * 500          // 0.5 秒
)

type Permit interface {
	CreateRole(ctx context.Context, req *CreateRoleReq) (*CreateRoleResp, error)

	UpdateRole(ctx context.Context, req *UpdateRoleReq) (*UpdateRoleResp, error)

	DeleteRole(ctx context.Context, req *DeleteRoleReq) (*DeleteRoleResp, error) // 这个删除需要关心的东西比较多

	GetRole(ctx context.Context, req *GetRoleReq) (*GetRoleResp, error)

	FindRole(ctx context.Context, req *FindRoleReq) (*FindRoleResp, error)

	AssignRoleGrant(ctx context.Context, req *AssignRoleGrantReq) (*AssignRoleGrantResp, error)

	FindGrantRole(ctx context.Context, req *FindGrantRoleReq) (*FindGrantRoleResp, error)

	CreatePermit(ctx context.Context, req *CreatePerReq) (*CreatePerResp, error)

	UpdatePermit(ctx context.Context, req *UpdatePerReq) (*UpdatePerResp, error)

	DeletePermit(ctx context.Context, req *DeletePerReq) (*DeletePerResp, error)

	GetPermit(ctx context.Context, req *GetPermitReq) (*GetPermitResp, error)

	FindPermit(ctx context.Context, req *FindPermitReq) (*FindPermitResp, error)

	SaveUserPerMatch(ctx context.Context, req *SaveUserPerMatchReq) (*SaveUserPerMatchResp, error)
}

type permit struct {
	db            *gorm.DB
	roleRepo      models.RoleRepo
	roleGrantRepo models.RoleRantRepo
	permitRepo    models.PermitRepo
	limitRepo     models.LimitsRepo
}

type FindPermitReq struct {
	RoleID string `json:"roleID"`
}

type FindPermitResp struct {
	List []*Permits `json:"list"`
}

type Permits struct {
	ID        string             `json:"id"`
	RoleID    string             `json:"roleID"`
	Path      string             `json:"path"`
	Params    models.FiledPermit `json:"params"`
	Response  models.FiledPermit `json:"response"`
	Condition models.Condition   `json:"condition"`

}

func (p *permit) FindPermit(ctx context.Context, req *FindPermitReq) (*FindPermitResp, error) {
	permits, err := p.permitRepo.Find(p.db, &models.PermitQuery{
		RoleID: req.RoleID,
	})
	if err != nil {
		return nil, err
	}
	resp := &FindPermitResp{
		List: make([]*Permits, len(permits)),
	}
	for index, value := range permits {
		resp.List[index] = &Permits{
			ID:        value.ID,
			RoleID:    value.RoleID,
			Path:      value.Path,
			Params:    value.Params,
			Condition: value.Condition,
		}
	}
	return resp, nil
}

type FindGrantRoleReq struct {
	Owners []string `json:"owners"`
	AppID  string   `json:"appID"`
	RoleID string   `json:"roleID"`
}

type FindGrantRoleResp struct {
	List []*GrantRoles `json:"list"`
}

type GrantRoles struct {
	RoleID    string `json:"roleID"`
	Owner     string `json:"id"`
	OwnerName string `json:"name"`
	Types     int    `json:"type"`
}

func (p *permit) FindGrantRole(ctx context.Context, req *FindGrantRoleReq) (*FindGrantRoleResp, error) {
	//
	grantRole, err := p.roleGrantRepo.Find(p.db, &models.RoleGrantQuery{
		Owners: req.Owners,
		AppID:  req.AppID,
		RoleID: req.RoleID,
	})
	if err != nil {
		return nil, err
	}
	resp := &FindGrantRoleResp{
		List: make([]*GrantRoles, 0, len(grantRole)),
	}

	for _, value := range grantRole {
		resp.List = append(resp.List, &GrantRoles{
			RoleID:    value.RoleID,
			Owner:     value.Owner,
			OwnerName: value.OwnerName,
			Types:     value.Types,
		})
	}
	return resp, nil
}

type SaveUserPerMatchReq struct {
	PermitID string
	UserID   string
	AppID    string
}

type SaveUserPerMatchResp struct{}

func (p *permit) SaveUserPerMatch(ctx context.Context, req *SaveUserPerMatchReq) (*SaveUserPerMatchResp, error) {
	match := &models.PermitMatch{
		UserID: req.UserID,
		AppID:  req.AppID,
		RoleID: req.PermitID,
	}
	err := p.limitRepo.CreatePerMatch(ctx, match)
	if err != nil {
		return nil, err
	}
	return &SaveUserPerMatchResp{}, nil
}

func NewPermit(conf *config2.Config) (Permit, error) {
	db, err := CreateMysqlConn(conf)
	if err != nil {
		return nil, err
	}
	redisClient, err := redis2.NewClient(conf.Redis)
	if err != nil {
		return nil, err
	}

	return &permit{
		db:            db,
		roleRepo:      mysql.NewRoleRepo(),
		roleGrantRepo: mysql.NewRoleGrantRepo(),
		permitRepo:    mysql.NewPermitRepo(),
		limitRepo:     redis.NewLimitRepo(redisClient),
	}, nil
}

type CreateRoleReq struct {
	UserID      string          `json:"user_id"`
	UserName    string          `json:"user_name"`
	AppID       string          `json:"app_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Types       models.RoleType `json:"types"`
}

type CreateRoleResp struct {
	ID string `json:"id"`
}

func (p *permit) CreateRole(ctx context.Context, req *CreateRoleReq) (*CreateRoleResp, error) {
	roles := &models.Role{
		ID:          id2.HexUUID(true),
		AppID:       req.AppID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time2.NowUnix(),
		CreatorName: req.UserName,
		CreatorID:   req.UserID,
	}
	if req.Types == 0 {
		roles.Types = models.CreateType
	}
	roles.Types = req.Types
	err := p.roleRepo.BatchCreate(p.db, roles)
	if err != nil {
		return nil, err
	}
	return &CreateRoleResp{
		ID: roles.ID,
	}, nil
}

type UpdateRoleReq struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateRoleResp struct{}

func (p *permit) UpdateRole(ctx context.Context, req *UpdateRoleReq) (*UpdateRoleResp, error) {
	err := p.roleRepo.Update(p.db, req.ID, &models.Role{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return &UpdateRoleResp{}, nil
}

type GetRoleReq struct {
	ID string `json:"id"`
}
type GetRoleResp struct {
	Types       models.RoleType `json:"type"`
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
}

func (p *permit) GetRole(ctx context.Context, req *GetRoleReq) (*GetRoleResp, error) {
	permits, err := p.roleRepo.Get(p.db, req.ID)
	if err != nil {
		return nil, err
	}
	return &GetRoleResp{
		ID:          permits.ID,
		Types:       permits.Types,
		Name:        permits.Name,
		Description: permits.Description,
	}, nil
}

type FindRoleReq struct {
	AppID string `json:"appID"`
}

type FindRoleResp struct {
	List []*roleVo `json:"list"`
}

type roleVo struct {
	Types models.RoleType `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
}

func (p *permit) FindRole(ctx context.Context, req *FindRoleReq) (*FindRoleResp, error) {
	list, err := p.roleRepo.Find(p.db, &models.RoleQuery{
		AppID: req.AppID,
	})
	if err != nil {
		return nil, err
	}
	resp := &FindRoleResp{
		List: make([]*roleVo, len(list)),
	}
	for index, value := range list {
		resp.List[index] = &roleVo{
			ID:    value.ID,
			Name:  value.Name,
			Types: value.Types,
		}
	}
	return resp, nil
}

type AssignRoleGrantReq struct {
	Add     []*Owners `json:"add"`
	RoleID  string    `json:"roleID"`
	AppID   string    `json:"appID"`
	Removes []string  `json:"removes"`
}
type Owners struct {
	Owner     string `json:"id"`
	OwnerName string `json:"name"`
	Types     int    `json:"type"`
}

type AssignRoleGrantResp struct{}

func (p *permit) AssignRoleGrant(ctx context.Context, req *AssignRoleGrantReq) (*AssignRoleGrantResp, error) {
	roleGrants := make([]*models.RoleGrant, len(req.Add))
	for index, value := range req.Add {
		roleGrants[index] = &models.RoleGrant{
			ID:        id2.HexUUID(true),
			RoleID:    req.RoleID,
			Owner:     value.Owner,
			OwnerName: value.OwnerName,
			Types:     value.Types,
			AppID:     req.AppID,
			CreatedAt: time2.NowUnix() + int64(index),
		}
	}
	err := p.roleGrantRepo.BatchCreate(p.db, roleGrants...)
	if err != nil {
		return nil, err
	}

	if len(req.Removes) == 0 {
		return &AssignRoleGrantResp{}, nil
	}
	err = p.roleGrantRepo.Delete(p.db, &models.RoleGrantQuery{
		RoleID: req.RoleID,
		Owners: req.Removes,
	})
	if err != nil {
		return nil, err
	}
	return &AssignRoleGrantResp{}, nil
}

type CreatePerReq struct {
	Path      string             `json:"path"`
	Params    models.FiledPermit `json:"params"`
	Response  models.FiledPermit `json:"response"`
	RoleID    string             `json:"roleID"`
	UserID    string             `json:"userID"`
	UserName  string             `json:"userName"`
	Condition models.Condition   `json:"condition"`
}

type CreatePerResp struct{}

func (p *permit) CreatePermit(ctx context.Context, req *CreatePerReq) (*CreatePerResp, error) {
	permits := &models.Permit{
		ID:          id2.HexUUID(true),
		Path:        req.Path,
		Params:      req.Params,
		Response:    req.Response,
		RoleID:      req.RoleID,
		CreatorID:   req.UserID,
		CreatorName: req.UserName,
		CreatedAt:   time2.NowUnix(),
		Condition:   req.Condition,
	}
	err := p.permitRepo.BatchCreate(p.db, permits)
	if err != nil {
		return nil, err
	}
	p.modifyRedis(ctx, permits)
	return &CreatePerResp{}, nil
}

type UpdatePerReq struct {
	ID        string             `json:"id"`
	Params    models.FiledPermit `json:"params"`
	Response  models.FiledPermit `json:"response"`
	Condition models.Condition   `json:"condition"`
}

type UpdatePerResp struct{}

func (p *permit) UpdatePermit(ctx context.Context, req *UpdatePerReq) (*UpdatePerResp, error) {
	err := p.permitRepo.Update(p.db, req.ID, &models.Permit{
		Params:   req.Params,
		Response: req.Response,
	})
	if err != nil {
		return nil, err
	}
	// add redis cache
	return &UpdatePerResp{}, nil
}

func (p *permit) modifyRedis(ctx context.Context, permits *models.Permit) {
	if p.limitRepo.ExistsKey(ctx, permits.RoleID) {
		return
	}
	// add redis cache
	err := p.limitRepo.CreatePermit(ctx, permits.RoleID, &models.Limits{
		Path:      permits.Path,
		Params:    permits.Params,
		Response:  permits.Response,
		Condition: permits.Condition,
	})
	if err != nil {
		logger.Logger.Errorw("add permit cache ", permits.RoleID, err.Error())
	}
}

type DeletePerReq struct {
	roleID string `json:"roleID"`
	Path   string `json:"path"`
}

type DeletePerResp struct{}

func (p *permit) DeletePermit(ctx context.Context, req *DeletePerReq) (*DeletePerResp, error) {
	err := p.permitRepo.Delete(p.db, &models.PermitQuery{
		RoleID: req.roleID,
		Path:   req.Path,
	})
	if err != nil {
		return nil, err
	}
	return &DeletePerResp{}, nil
}


//type GetPerInCacheReq struct {
//	UserID string
//	DepID  string
//	AppID  string
//	Path   string
//}
//
//type GetPerInCacheResp struct {
//	Params    models.FiledPermit
//	Response  models.FiledPermit
//	Condition *models.Condition
//	Types     models.RoleType
//}
//
//func (p *permit) GetPerInCache(ctx context.Context, req *GetPerInCacheReq) (*GetPerInCacheResp, error) {
//	// 获取api
//
//	match, err := p.getCacheMatch(ctx, req.AppID, req.DepID, req.UserID)
//	if err != nil {
//		return nil, err
//	}
//	if match == nil {
//		return nil, error2.New(code.ErrNotPermit)
//	}
//	if match.Types == models.InitType {
//		return &GetPerInCacheResp{
//			Types: match.Types,
//		}, nil
//	}
//	permits, err := p.getCachePermit(ctx, match.RoleID, req.Path)
//	if err != nil {
//		return nil, err
//	}
//	return &GetPerInCacheResp{
//		Params:    permits.Params,
//		Response:  permits.Response,
//		Condition: permits.Condition,
//		Types:     match.Types,
//	}, err
//}
//
//// 那就是在管理端维护，缓存。
//func (p *permit) getCacheMatch(ctx context.Context, appID, depID, userID string) (*models.PermitMatch, error) {
//	for i := 0; i < 5; i++ {
//		perMatch, err := p.limitRepo.GetPerMatch(ctx, userID, appID)
//		if err != nil {
//			logger.Logger.Errorw(userID, header.GetRequestIDKV(ctx).Fuzzy(), err.Error())
//			return nil, err
//		}
//		if perMatch != nil {
//			return perMatch, nil
//		}
//		lock, err := p.limitRepo.Lock(ctx, lockPerMatch, 1, lockTimeout) // 抢占分布式锁
//		if err != nil {
//			logger.Logger.Errorw(err.Error(), userID, logger.STDRequestID(ctx))
//			return nil, err
//		}
//		if !lock {
//			<-time.After(timeSleep)
//			continue
//		}
//		break
//	}
//	defer p.limitRepo.UnLock(ctx, lockPerMatch) // 删除锁
//	// 找到匹配的权限
//	roles, err := p.roleGrantRepo.Find(p.db, &models.RoleGrantQuery{
//		Owners: []string{depID, userID},
//	})
//	if err != nil {
//		return nil, err
//	}
//	if len(roles) == 0 {
//		return nil, error2.New(code.ErrNotPermit)
//	}
//
//	role, err := p.roleRepo.Get(p.db, roles[0].RoleID)
//	if err != nil {
//		return nil, err
//	}
//
//	perMatch := &models.PermitMatch{
//		UserID: userID,
//		AppID:  appID,
//		RoleID: roles[0].RoleID,
//		Types:  role.Types,
//	}
//	err = p.limitRepo.CreatePerMatch(ctx, perMatch)
//	if err != nil {
//		// 打印错误日志
//	}
//	return perMatch, nil
//}
//
//func (p *permit) getCachePermit(ctx context.Context, roleID, path string) (*models.Limits, error) {
//	for i := 0; i < 5; i++ {
//		exist := p.limitRepo.ExistsKey(ctx, roleID)
//		if exist { // 存在
//			// 判断 path
//			getPermit, err := p.limitRepo.GetPermit(ctx, roleID, path)
//			if err != nil {
//				return nil, err
//			}
//			if getPermit.Path == "" {
//				return nil, error2.New(code.ErrNotPermit)
//			}
//			return getPermit, nil
//		}
//		lock, err := p.limitRepo.Lock(ctx, lockPermission, 1, lockTimeout) // 抢占分布式锁
//		if err != nil {
//			logger.Logger.Errorw(err.Error(), roleID, logger.STDRequestID(ctx))
//			return nil, err
//		}
//		if !lock {
//			<-time.After(timeSleep)
//			continue
//		}
//		break
//	}
//	defer p.limitRepo.UnLock(ctx, lockPermission) // 删除锁
//	permits, err := p.permitRepo.Find(p.db, &models.PermitQuery{
//		RoleID: roleID,
//	})
//	if err != nil {
//		return nil, err
//	}
//	limits := make([]*models.Limits, len(permits))
//	var getPermit *models.Limits
//	for index, value := range permits {
//		per := &models.Limits{
//			Path:      value.Path,
//			Condition: value.Condition,
//			Params:    value.Params,
//			Response:  value.Response,
//		}
//		if value.Path == path {
//			getPermit = per
//		}
//		limits[index] = per
//	}
//	err = p.limitRepo.CreatePermit(ctx, roleID, limits...)
//	if err != nil {
//		logger.Logger.Errorw("create permit err", roleID, err.Error())
//	}
//	if getPermit == nil {
//		return nil, error2.New(code.ErrNotPermit)
//	}
//	return getPermit, nil
//}

type GetPermitReq struct {
	RoleID string `json:"roleID"`
	Path   string `json:"path"`
}

type GetPermitResp struct {
	ID        string             `json:"id"`
	RoleID    string             `json:"roleID"`
	Path      string             `json:"path,omitempty"`
	Params    models.FiledPermit `json:"params,omitempty"`
	Response  models.FiledPermit `json:"response,omitempty"`
	Condition models.Condition   `json:"condition,omitempty"`
}

func (p *permit) GetPermit(ctx context.Context, req *GetPermitReq) (*GetPermitResp, error) {
	permits, err := p.permitRepo.Get(p.db, req.RoleID, req.Path)
	if err != nil {
		return nil, err
	}
	return &GetPermitResp{
		ID:        permits.ID,
		RoleID:    permits.RoleID,
		Path:      permits.Path,
		Params:    permits.Params,
		Response:  permits.Response,
		Condition: permits.Condition,
	}, nil
}
