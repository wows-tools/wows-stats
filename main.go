package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/wows-tools/wows-stats/backend"
	"github.com/wows-tools/wows-stats/model"
	"github.com/wows-tools/wows-stats/stats"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
	"os"
	"time"
)

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func main() {

	key := os.Getenv("WOWS_WOWSAPIKEY")
	server := os.Getenv("WOWS_REALM")
	debug := os.Getenv("WOWS_DEBUG")
	listen := os.Getenv("WOWS_LISTEN")

	var loggerConfig zap.Config
	if debug == "true" {
		loggerConfig = zap.NewDevelopmentConfig()
	} else {
		loggerConfig = zap.NewProductionConfig()
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		fmt.Printf("Error initializing logger: %s\n", err.Error())
		os.Exit(-1)
	}
	defer logger.Sync()
	glogger := zapgorm2.New(logger)
	sugar := logger.Sugar()
	mainLogger := sugar.With("component", "main")

	db, err := gorm.Open(sqlite.Open("wows-stats.db"), &gorm.Config{Logger: glogger})
	if err != nil {
		panic("failed to connect database")
	}

	Schemas := []interface{}{
		&model.Player{},
		&model.PreviousClan{},
		&model.Clan{},
		&model.Version{},
		&model.WowsVersion{},
	}

	// Migrate the schema
	var version model.Version
	version.Version = 0
	version.Name = "version"
	db.First(&version)
	if version.Version < model.DBVersion {
		db.AutoMigrate(Schemas...)
		version.Version = model.DBVersion
		db.Save(&version)
	}

	api := backend.NewBackend(key, server, sugar.With("component", "backend"), db)
	//api.FillShipMapping()

	var count int64
	db.Table("clans").Count(&count)
	if count < 1000 {
		mainLogger.Infof("DB is empty, doing an initial complete scan, please wait (full dump takes roughly a day)")
		err = api.ScrapAll()
		if err != nil {
			mainLogger.Errorf("first scan errored with: %s", err.Error())
		}
	}

	err = api.LoadWowsVersionsFromCSV("./misc/updates.csv")
	if err != nil {
		mainLogger.Errorf("WoWs updates filling failed: %s", err.Error())
	}
	s := gocron.NewScheduler(time.UTC)
	mainLogger.Infof("adding 'updating all player task every 30 days")
	s.Every(30).Days().At("10:30").Do(api.ScrapAll)
	s.StartAsync()

	mainLogger.Infof("Server starting on '%s'.  Press CTRL-C to exit.", listen)
	statsServer := stats.NewStatsServer(listen, sugar.With("component", "stats_server"), db, server)
	statsServer.Server()
}
