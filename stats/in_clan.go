package stats

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func (server *StatsServer) PieClanProfiles() *charts.Pie {
	// Get the number of players in clans
	inClanCount, _ := server.getPlayerCountByClan(true)

	// Get the number of players not in clans
	notInClanCount, _ := server.getPlayerCountByClan(false)

	// Create a pie chart
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Player Clan Profiles",
		}),
	)

	// Add data to the pie chart
	pie.AddSeries("Player Clan Profiles", []opts.PieData{
		{Value: inClanCount, Name: "In Clans"},
		{Value: notInClanCount, Name: "Not in Clans"},
	})
	pie.SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show:      true,
			Formatter: "{b} : {d}% ({c})",
		}),
	)

	return pie
}

func (server *StatsServer) getPlayerCountByClan(inClan bool) (int, error) {
	var count int64
	var oper string
	if inClan {
		oper = "clan_id > 0"
	} else {
		oper = "clan_id = 0 OR clan_id IS NULL"
	}
	err := server.DB.Table("players").Where(oper).Count(&count).Error
	return int(count), err
}
