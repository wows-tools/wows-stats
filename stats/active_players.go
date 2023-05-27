package stats

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"time"
)

const (
	ActivePlayersPieMethodology = `
### Description

Ratio between players who have played in the last 3 months or not.

### Code

[stats/active_players.go](https://github.com/wows-tools/wows-stats/blob/main/stats/active_players.go)

### Methodology

* count the number of players with 'last battle date' >= 'current_date() - 3 months'
* count the number of players with 'last battle date' < 'current_date() - 3 months'

Display a pie chart with the 2 values and their respective ratio.

### Caveats

None identified.
`
)

func (server *StatsServer) ActivePlayersPie() *charts.Pie {
	// Get the number of active players during the last 3 months
	activePlayersCount, _ := server.getActivePlayersCount(3)

	// Get the number of inactive players during the last 3 months
	inactivePlayersCount, _ := server.getInactivePlayersCount(3)

	// Create a pie chart
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Active Players in Last 3 Months",
		}),
	)

	// Add data to the pie chart
	pie.AddSeries("Player Status", []opts.PieData{
		{Value: float64(activePlayersCount), Name: "Active Players",
			Label: &opts.Label{
				Show:      true,
				Formatter: "{b} : {d}% ({c})",
			},
		},
		{Value: float64(inactivePlayersCount), Name: "Inactive Players",
			Label: &opts.Label{
				Show:      true,
				Formatter: "{b} : {d}% ({c})",
			},
		},
	})

	return pie
}

func (server *StatsServer) getActivePlayersCount(months int) (int, error) {
	// Calculate the start date for the last 3 months
	// FIXME use the last scan date instead of the current date
	// Not a critical issue if we generate the graph right after the scan
	startDate := time.Now().AddDate(0, -months, 0)

	// Get the count of active players during the specified period
	var count int64
	query := server.DB.Table("players").Where("last_battle_date >= ?", startDate)
	err := query.Count(&count).Error
	return int(count), err
}

func (server *StatsServer) getInactivePlayersCount(months int) (int, error) {
	// Calculate the start date for the last 3 months
	// FIXME use the last scan date instead of the current date
	// Not a critical issue if we generate the graph right after the scan
	startDate := time.Now().AddDate(0, -months, 0)

	// Get the count of active players during the specified period
	var count int64
	query := server.DB.Table("players").Where("last_battle_date < ? AND hidden_profile = 0", startDate)
	err := query.Count(&count).Error
	return int(count), err
}
