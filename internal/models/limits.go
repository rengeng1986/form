package models

import (
	"context"
	"time"
)

type Limits struct {
	Path      string
	Name      string
	Params    FiledPermit
	Response  FiledPermit
	Condition interface{}
	types     RoleType
}

//PermitMatch PermitMatch
type PermitMatch struct {
	PermitID string
	UserID   string
	AppID    string
}

type LimitsRepo interface {
	CreatePermit(ctx context.Context, roleID string, limits []*Limits) error
	GetPermit(ctx context.Context, roleID, path string) (*Limits, error)
	DeletePermit(ctx context.Context, roleID, path string) error

	CreatePerMatch(ctx context.Context, match *PermitMatch) error
	GetPerMatch(ctx context.Context, appID, userID string) (*PermitMatch, error)
	DeletePerMatch(ctx context.Context, appID string) error

	// Lock 设置分布式锁
	Lock(ctx context.Context, key string, val interface{}, ttl time.Duration) (bool, error)
	// UnLock 解除分布式锁
	UnLock(ctx context.Context, key string) error
	// PerMatchExpire 给某个键设置过期时间
	PerMatchExpire(ctx context.Context, key string, ttl time.Duration) error
	//PermitExpire PermitExpire
	PermitExpire(ctx context.Context, key string, ttl time.Duration) error
}
