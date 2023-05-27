package stats

import (
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
	page := NewPage()

	// Create a FlexLayout component
	page.SetLayout(PageFlexLayout)
	page.Initialization.PageTitle = "WoWs Stats (" + server.Realm + ")"

	page.AddRow(NewMarkdown("# Statitics for the **" + server.Realm + "** server"))

	page.AddRow(NewMarkdown("---\n## Server Activity Statistics\n---"))
	gainloss := server.PlayerGainLossBar(0)
	page.AddRow(gainloss[0], NewMarkdown(PlayerGainLossBarMethodology))

	page.AddRow(gainloss[1], NewMarkdown(PlayerGainLossBarNetMethodology))

	gainloss200 := server.PlayerGainLossBar(200)
	page.AddRow(gainloss200[0], NewMarkdown(PlayerGainLossBarNetMethodology))
	page.AddRow(gainloss200[1], NewMarkdown(PlayerGainLossBarNetWithBattlesMethodoloy))

	gainloss2000 := server.PlayerGainLossBar(2000)
	page.AddRow(gainloss2000[0], NewMarkdown(PlayerGainLossBarNetWithBattlesMethodoloy))
	page.AddRow(gainloss2000[1], NewMarkdown(PlayerGainLossBarNetMethodology2000))

	activePlayersLast3Months := server.ActivePlayersPie()
	page.AddRow(activePlayersLast3Months, NewMarkdown(ActivePlayersPieMethodology))

	startStopHeatmap := server.PlayerStartStopChart()
	page.AddRow(startStopHeatmap, NewMarkdown(PlayerStartStopChartMethodology))

	monthlyBattles := server.MonthlyBattleEstimation()
	page.AddRow(monthlyBattles, NewMarkdown(MonthlyBattleEstimationMethodology))


	page.AddRow(NewMarkdown("---\n## Player Base Statistics\n---"))

	pieWinRateDist := server.DistributionByWinRate(1)
	page.AddRow(pieWinRateDist, NewMarkdown(DistributionByWinRateMethodology))

	pieWinRateDist200 := server.DistributionByWinRate(200)
	page.AddRow(pieWinRateDist200, NewMarkdown(DistributionByWinRateMethodology))

	pieWinRateDist2000 := server.DistributionByWinRate(2000)
	page.AddRow(pieWinRateDist2000, NewMarkdown(PlayerGainLossBarMethodology))

	barWRBattles := server.WinRateByBattles()
	page.AddRow(barWRBattles, NewMarkdown(WinRateByBattlesMethodology))

	pieHiddenProfile := server.PieHiddenProfiles()
	page.AddRow(pieHiddenProfile, NewMarkdown(PieHiddenProfilesMethodology))

	pieInClan := server.PieClanProfiles()
	page.AddRow(pieInClan, NewMarkdown(PieClanProfilesMethodology))

	matrix := server.DivVsSoloWR()
	page.AddRow(matrix, NewMarkdown(DivVsSoloWRMethodology))

	barBattles := server.BarChartByRandomBattles()
	page.AddRow(barBattles, NewMarkdown(BarChartByRandomBattlesMethodology))

	page.AddRow(NewMarkdown("---\n## Problematic Charts\n---"))
	activePlayersMonthly := server.ActivePlayersMonthly()
	page.AddRow(activePlayersMonthly, NewMarkdown(ActivePlayersMonthlyMethodology))

	page.AddRow(NewMarkdown("## Other Considerations"))
	page.AddRow(NewMarkdown(OtherNotes))

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
