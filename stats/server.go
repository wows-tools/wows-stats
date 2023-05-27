package stats

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"strings"
)

type StatsServer struct {
	Logger *zap.SugaredLogger
	DB     *gorm.DB
	Output string
	Realm  string
}

func (server *StatsServer) GenerateReport() {
	// Set up the Go-Echarts components
	page := NewPage()

	// Create a FlexLayout component
	page.SetLayout(PageFlexLayout)
	page.Initialization.PageTitle = "WoWs Stats (" + strings.ToUpper(server.Realm) + ")"

	page.AddRow(NewMarkdown("# Statitics for the **" + strings.ToUpper(server.Realm) + "** server"))

	page.AddRow(NewMarkdown("___\n## Server Activity Statistics"))

	page.AddRow(NewMarkdown("---"))

	activePlayersLast3Months := server.ActivePlayersPie()
	page.AddRow(activePlayersLast3Months, NewMarkdown(ActivePlayersPieMethodology))

	page.AddRow(NewMarkdown("---"))
	startStopHeatmap := server.PlayerStartStopChart()
	page.AddRow(startStopHeatmap, NewMarkdown(PlayerStartStopChartMethodology))

	gainloss := server.PlayerGainLossBar(0)

	page.AddRow(NewMarkdown("---"))
	page.AddRow(gainloss[0], NewMarkdown(PlayerGainLossBarMethodology))

	page.AddRow(NewMarkdown("---"))
	page.AddRow(gainloss[1], NewMarkdown(PlayerGainLossBarNetMethodology))

	page.AddRow(NewMarkdown("---"))
	monthlyBattles := server.MonthlyBattleEstimation()
	page.AddRow(monthlyBattles, NewMarkdown(MonthlyBattleEstimationMethodology))

	page.AddRow(NewMarkdown("___\n## Player Base Statistics"))

	page.AddRow(NewMarkdown("---"))
	pieWinRateDist := server.DistributionByWinRate(1, 0.01)
	page.AddRow(pieWinRateDist, NewMarkdown(DistributionByWinRateMethodology))

	page.AddRow(NewMarkdown("---"))
	pieWinRateDist200 := server.DistributionByWinRate(200, 0.005)
	page.AddRow(pieWinRateDist200, NewMarkdown(SeePrevious))

	page.AddRow(NewMarkdown("---"))
	pieWinRateDist2000 := server.DistributionByWinRate(2000, 0.001)
	page.AddRow(pieWinRateDist2000, NewMarkdown(SeePrevious))

	page.AddRow(NewMarkdown("---"))
	barWRBattles := server.WinRateByBattles()
	page.AddRow(barWRBattles, NewMarkdown(WinRateByBattlesMethodology))

	page.AddRow(NewMarkdown("---"))
	pieHiddenProfile := server.PieHiddenProfiles()
	page.AddRow(pieHiddenProfile, NewMarkdown(PieHiddenProfilesMethodology))

	page.AddRow(NewMarkdown("---"))
	pieInClan := server.PieClanProfiles()
	page.AddRow(pieInClan, NewMarkdown(PieClanProfilesMethodology))

	page.AddRow(NewMarkdown("---"))
	matrix := server.DivVsSoloWR()
	page.AddRow(matrix, NewMarkdown(DivVsSoloWRMethodology))

	page.AddRow(NewMarkdown("---"))
	barBattles := server.BarChartByRandomBattles()
	page.AddRow(barBattles, NewMarkdown(BarChartByRandomBattlesMethodology))

	page.AddRow(NewMarkdown("___\n## Problematic Charts\n___"))
	activePlayersMonthly := server.ActivePlayersMonthly()
	page.AddRow(activePlayersMonthly, NewMarkdown(ActivePlayersMonthlyMethodology))

	gainloss200 := server.PlayerGainLossBar(200)

	page.AddRow(NewMarkdown("---"))
	page.AddRow(gainloss200[0], NewMarkdown(PlayerGainLossBarNetMethodology))

	page.AddRow(NewMarkdown("---"))
	page.AddRow(gainloss200[1], NewMarkdown(PlayerGainLossBarNetWithBattlesMethodoloy))

	gainloss2000 := server.PlayerGainLossBar(2000)

	page.AddRow(NewMarkdown("---"))
	page.AddRow(gainloss2000[0], NewMarkdown(PlayerGainLossBarNetWithBattlesMethodoloy))

	page.AddRow(NewMarkdown("---"))
	page.AddRow(gainloss2000[1], NewMarkdown(PlayerGainLossBarNetMethodology2000))

	page.AddRow(NewMarkdown("___\n## Other Considerations\n___"))
	page.AddRow(NewMarkdown(OtherNotes))
	page.AddRow(NewMarkdown("___"))

	f, err := os.OpenFile(server.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		server.Logger.Errorf("error opening the file %s", err.Error())
		return
	}
	defer f.Close()
	err = page.Render(f)
	if err != nil {
		server.Logger.Errorf("error rendering the report %s", err.Error())
		return
	}

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
