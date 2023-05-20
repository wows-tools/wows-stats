package model

import (
	"time"
)

type PreviousClan struct {
	JoinDate  time.Time `gorm:"index"`
	LeaveDate time.Time `gorm:"index"`
	ClanID    int       `gorm:"index"`
	Clan      *Clan
	PlayerID  int `gorm:"index"`
	Player    *Player
}
