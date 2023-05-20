package model

import (
	"gorm.io/gorm"
	"time"
)

type WowsVersion struct {
	gorm.Model
	Version     string    `gorm:"primaryKey"`
	ReleaseDate time.Time `gorm:"index"`
}
