package main

import (
	"flag"
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/wows-tools/wows-stats/backend"
	"github.com/wows-tools/wows-stats/model"
	"github.com/wows-tools/wows-stats/stats"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
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

	var (
		apiKey         string
		server         string
		debug          bool
		output         string
		skipScraping   bool
		skipGeneration bool
	)

	flag.StringVar(&apiKey, "apikey", "", "Wargaming.net API key")
	flag.StringVar(&server, "server", "", "World of Warships server")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.StringVar(&output, "output", "", "Output file path (required)")
	flag.BoolVar(&skipScraping, "skip-scraping", false, "Skip scraping data")
	flag.BoolVar(&skipGeneration, "skip-generation", false, "Skip report generation")
	flag.Parse()

	if apiKey == "" || server == "" || output == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	var loggerConfig zap.Config

	if debug {
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
	glogger.SlowThreshold = time.Millisecond * 100000
	sugar := logger.Sugar()
	mainLogger := sugar.With("component", "main")

	dbName := server + "-stats.db"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{Logger: glogger})
	if err != nil {
		panic("failed to connect database")
	}
	sqldb, err := db.DB()
	if err != nil {
		panic("failed to set max connections")
	}
	sqldb.SetMaxOpenConns(2)

	Schemas := []interface{}{
		&model.Player{},
		&model.PreviousClan{},
		&model.Clan{},
		&model.Scan{},
		&model.Version{},
		&model.WowsVersion{},
	}

	// Migrate the schema
	var version model.Version
	version.Version = 0
	version.Name = "version"
	db.FirstOrInit(&version, version)
	if version.Version < model.DBVersion {
		db.AutoMigrate(Schemas...)
		db.Model(&version).Update("version", model.DBVersion)
	}

	api := backend.NewBackend(apiKey, server, sugar.With("component", "backend"), db)
	// api.PrefixBreak = 100
	// api.ClanBreak = 10

	if !skipScraping {
		err = api.ScrapAll()
		if err != nil {
			mainLogger.Errorf("first scan errored with: %s", err.Error())
		}
	}

	if !skipGeneration {
		statsServer := stats.NewStatsServer(output, sugar.With("component", "stats_server"), db, server)
		statsServer.GenerateReport()
	}
}
