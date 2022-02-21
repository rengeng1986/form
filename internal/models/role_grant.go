package models

import "gorm.io/gorm"

type RoleGrant struct {
	ID        string
	RoleID    string
	Owner     string
	OwnerName string
	Types     int
}

type RoleGrantQuery struct {
	ID string
}

// RoleRantRepo RoleRantRepo
type RoleRantRepo interface {
	BatchCreate(db *gorm.DB, roleGrant ...*RoleGrant) error
	Get(db *gorm.DB, id string) (*RoleGrant, error)
	Find(db *gorm.DB, query *RoleGrantQuery) ([]*RoleGrant, error)
	Update(db *gorm.DB, id string, roleGrant *RoleGrant) error
	Delete(db *gorm.DB, roleGrant *RoleGrant) error
}
