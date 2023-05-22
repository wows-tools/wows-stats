package stats

import (
	"github.com/go-echarts/go-echarts/v2/components"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
)

type StatsServer struct {
	Logger *zap.SugaredLogger
	DB     *gorm.DB
	Output string
	Realm  string
}

func (server *StatsServer) GenerateReport() {
	// Set up the Go-Echarts components
	page := components.NewPage()

	// Create a FlexLayout component
	page.SetLayout(components.PageFlexLayout)
	page.Initialization.PageTitle = "WoWs Stats || server " + server.Realm

	gainloss := server.PlayerGainLossBar(0, 0)
	page.AddCharts(gainloss[0])
	page.AddCharts(gainloss[1])

	gainloss200 := server.PlayerGainLossBar(200, 0)
	page.AddCharts(gainloss200[0])
	page.AddCharts(gainloss200[1])

	gainloss2000 := server.PlayerGainLossBar(2000, 0)
	page.AddCharts(gainloss2000[0])
	page.AddCharts(gainloss2000[1])

	gainloss2000wr55 := server.PlayerGainLossBar(2000, 0.55)
	page.AddCharts(gainloss2000wr55[0])
	page.AddCharts(gainloss2000wr55[1])

	activePlayersLast3Months := server.ActivePlayersPie()
	page.AddCharts(activePlayersLast3Months)

	activePlayersMonthly := server.ActivePlayersMonthly()
	page.AddCharts(activePlayersMonthly)

	//monthlyBattles := server.MonthlyBattleEstimation()
	//page.AddCharts(monthlyBattles)

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

	matrix := server.ScatterPlotMatrix()
	page.AddCharts(matrix)

	barBattles := server.BarChartByRandomBattles()
	page.AddCharts(barBattles)

	f, err := os.OpenFile(server.Output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		server.Logger.Errorf("error opening the file %s", err.Error())
		return
	}
	defer f.Close()
	page.Render(f)
	return
}

func NewStatsServer(output string, logger *zap.SugaredLogger, db *gorm.DB, realm string) *StatsServer {
	return &StatsServer{
		Logger: logger,
		DB:     db,
		Output: output,
		Realm:  realm,
	}
}
