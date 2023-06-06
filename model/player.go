package model

import (
	"time"
)

type Player struct {
	ID                  int        `gorm:"primaryKey"`
	Nick                *string    `gorm:"index"`
	AccountCreationDate *time.Time `gorm:"index"`
	LastBattleDate      *time.Time `gorm:"index"`
	LastLogoutDate      *time.Time `gorm:"index"`
	NumberT10           *int       `gorm:"index"`
	RandomBattles       *int       `gorm:"index"`
	RandomWinRate       *float64   `gorm:"index"`
	RandomDivBattles    *int       `gorm:"index"`
	RandomDivWinRate    *float64   `gorm:"index"`
	RankedBattles       *int       `gorm:"index"`
	RankedWinRate       *float64   `gorm:"index"`
	CoopBattles         *int       `gorm:"index"`
	CoopWinRate         *float64   `gorm:"index"`
	OperBattles         *int       `gorm:"index"`
	OperWinRate         *float64   `gorm:"index"`
	HiddenProfile       *bool      `gorm:"index"`
	ClanID              *int       `gorm:"index"`
	ClanJoinDate        *time.Time
	Clan                *Clan
	PreviousClans       []PreviousClan
}
