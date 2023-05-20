package backend

import (
	"encoding/csv"
	"github.com/wows-tools/wows-stats/model"
	"gorm.io/gorm/clause"
	"os"
	"time"
)

func (backend *Backend) LoadWowsVersionsFromCSV(path string) error {
	backend.Logger.Debugf("Start loading wows versions (CSV)")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	if err != nil {
		return err
	}
	for _, line := range data {
		version := line[0]
		format := "2 January 2006"
		releaseDate, err := time.Parse(format, line[1])
		if err != nil {
			return err
		}

		err = backend.UpsertWowsVersion(version, releaseDate)
		if err != nil {
			return err
		}

	}

	backend.Logger.Debugf("Finish loading wows versions (CSV)")
	return nil
}

func (backend *Backend) UpsertWowsVersion(version string, releaseDate time.Time) error {
	backend.Logger.Debugf("Upserting version %s", version)
	wowsVersion := model.WowsVersion{
		Version:     version,
		ReleaseDate: releaseDate,
	}
	backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&wowsVersion)
	return nil
}
