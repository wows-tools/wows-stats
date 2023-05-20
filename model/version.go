package model

const (
	DBVersion = 4
)

type Version struct {
	Name    string `gorm:"primaryKey"`
	Version int
}
