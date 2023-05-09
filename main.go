package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/wows-tools/wows-stats/backend"
	"github.com/wows-tools/wows-stats/model"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	}

	// Migrate the schema
	db.AutoMigrate(Schemas...)

	api := backend.NewBackend(key, server, sugar.With("component", "backend"), db)
	//api.FillShipMapping()

	var count int64
	db.Table("clans").Count(&count)
	if count < 1000 {
		mainLogger.Infof("DB is empty, doing an initial complete scan, please wait (can take a few hours)")
		//err = api.ScrapAllPlayers()
		err = api.UpdateDetailsAllPlayers()
		if err != nil {
			mainLogger.Errorf("first scan errored with: %s", err.Error())
		}
	}
	s := gocron.NewScheduler(time.UTC)
	mainLogger.Infof("adding 'updating all clans' task every 7 days")
	s.Every(30).Days().At("10:30").Do(api.ScrapAllPlayers)
	s.StartAsync()

	var wg sync.WaitGroup

	mainLogger.Infof("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	wg.Wait()
}
