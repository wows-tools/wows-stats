package stats

import (
	"github.com/go-echarts/go-echarts/v2/components"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
)

type StatsServer struct {
	Logger *zap.SugaredLogger
	DB     *gorm.DB
	Listen string
	Realm  string
}

func (server *StatsServer) All(w http.ResponseWriter, r *http.Request) {
	// Set up the Go-Echarts components
	page := components.NewPage()

	// Create a FlexLayout component
	page.SetLayout(components.PageFlexLayout)

	gainloss := server.PlayerGainLossBar(0, 0)
	page.AddCharts(gainloss[0])
	page.AddCharts(gainloss[1])

	gainloss200 := server.PlayerGainLossBar(200, 0)
	page.AddCharts(gainloss200[0])
	page.AddCharts(gainloss200[1])

	gainloss2000 := server.PlayerGainLossBar(2000, 0.55)
	page.AddCharts(gainloss2000[0])
	page.AddCharts(gainloss2000[1])

	activePlayersLast3Months := server.ActivePlayersPie()
	page.AddCharts(activePlayersLast3Months)

	activePlayersMonthly := server.ActivePlayersMonthly()
	page.AddCharts(activePlayersMonthly)

	pieHiddenProfile := server.PieHiddenProfiles()
	page.AddCharts(pieHiddenProfile)

	pieInClan := server.PieClanProfiles()
	page.AddCharts(pieInClan)

	pieWinRateDist := server.DistributionByWinRate(100)
	page.AddCharts(pieWinRateDist)

	pieWinRateDist1000 := server.DistributionByWinRate(1000)
	page.AddCharts(pieWinRateDist1000)

	barWRBattles := server.WinRateByBattles()
	page.AddCharts(barWRBattles)

	page.Render(w)
	return
}

func NewStatsServer(listen string, logger *zap.SugaredLogger, db *gorm.DB, realm string) *StatsServer {
	return &StatsServer{
		Logger: logger,
		DB:     db,
		Listen: listen,
		Realm:  realm,
	}
}

func (server *StatsServer) Server() {
	http.HandleFunc("/", server.All)
	http.ListenAndServe(server.Listen, nil)
}
