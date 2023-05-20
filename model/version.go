package model

import (
	"gorm.io/gorm"
)

const (
	DBVersion = 3
)

type Version struct {
	gorm.Model
	Name    string `gorm:"primaryKey"`
	Version int
}
