package model

const (
	DBVersion = 5
)

type Version struct {
	Name    string `gorm:"primaryKey"`
	Version int
}
