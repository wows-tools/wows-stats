package model

import (
	"gorm.io/gorm"
	"time"
)

type Player struct {
	gorm.Model
	ID                  int       `gorm:"primaryKey"`
	Nick                string    `gorm:"index"`
	AccountCreationDate time.Time `gorm:"index"`
	LastBattleDate      time.Time `gorm:"index"`
	LastLogoutDate      time.Time `gorm:"index"`
	NumberT10           int       `gorm:"index"`
	Battles             int       `gorm:"index"`
	WinRate             float64   `gorm:"index"`
	HiddenProfile       bool      `gorm:"index"`
	ClanID              int       `gorm:"index"`
	ClanJoinDate        time.Time
	Clan                *Clan
	Tracked             bool `gorm:"index"`
	PreviousClans       []PreviousClan
	Filters             []Filter `gorm:"many2many:filter_tracked_player;"`
}
