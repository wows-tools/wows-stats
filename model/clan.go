package model

import (
	"time"
)

type Clan struct {
	ID           int `gorm:"primaryKey"`
	Name         string
	Tag          string `gorm:"index"`
	Language     string `gorm:"index"`
	CreationDate time.Time
	UpdatedDate  time.Time
	Players      []*Player
	PlayerIDs    []int `gorm:"-"`
	ClanLeader   *Player
	PlayerID     int `gorm:"index"`
}
