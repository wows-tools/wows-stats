package model

import (
	"gorm.io/gorm"
	"time"
)

type PreviousClan struct {
	gorm.Model
	JoinDate  time.Time `gorm:"index"`
	LeaveDate time.Time `gorm:"index"`
	ClanID    int       `gorm:"index"`
	Clan      *Clan
	PlayerID  int `gorm:"index"`
	Player    *Player
}
