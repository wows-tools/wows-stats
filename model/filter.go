package model

type Filter struct {
	DiscordChannelID    string   `gorm:"primaryKey"`
	TrackedClans        []Clan   `gorm:"many2many:filter_tracked_clan;"`
	TrackedPlayers      []Player `gorm:"many2many:filter_tracked_player;"`
	MinPlayerWR         float64
	DaysSinceLastBattle int
	MinNumT10           int
	MinNumBattles       int
	DiscordGuildID      string
}
