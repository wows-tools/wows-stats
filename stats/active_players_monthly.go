package stats

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"sort"
	"time"
)

const (
	ActivePlayersMonthlyMethodology = `
### Description

Estimation of the number of active players each month.

### Code

[active_players_monthly.go](https://github.com/wows-tools/wows-stats/blob/main/stats/active_players_monthly.go)

### Methodology

Take each months an account was created (trick to get all the individual months in the lifespan of the game).
For each month, count the number of players who have created their account before the end of the month and have their last battle after the last day of the month.

### Caveats

This graph is problematic, counting a player "active" only based on _account creation date_ and last battle date is too much of an approximation.

It over-estimates the number of active players, making for a misleading graph.

This chart **will be removed** and replaced with actual historical data in the future.
`
)

func (server *StatsServer) ActivePlayersMonthly() *charts.Line {
	// Query the database to get the number of active players each month
	playerCounts, months, _ := server.getMonthlyPlayerCounts()

	// Create a line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Number of Active Players Each Month",
		}),
	)

	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Estimate Active Players Each Month",
			Subtitle: "Based on last battle and account creation dates (warning: assumes player is active between these dates)",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
			AxisPointer: &opts.AxisPointer{
				Type: "line",
			},
		}),
	)

	players := make([]opts.LineData, 0)
	for _, count := range playerCounts[:len(playerCounts)-2] {
		players = append(players, opts.LineData{Value: count})
	}

	// Add data to the line chart
	line.SetXAxis(months[:len(months)-2]).
		AddSeries("Active Players", players)

	return line
}

func (server *StatsServer) getMonthlyPlayerCounts() ([]int, []string, error) {
	var monthlyPlayerCounts []int
	var monthlyPlayerMonths []string

	// Get the distinct months for last battle date and account creation date from the database
	var creationMonths []string

	// Query distinct months for account creation date
	if err := server.DB.Raw("SELECT DISTINCT strftime('%Y-%m', account_creation_date) as creation_month FROM players ORDER BY creation_month").Pluck("creation_month", &creationMonths).Error; err != nil {
		return nil, nil, err
	}
	sort.Strings(creationMonths)
	// Iterate over the distinct months and calculate the player counts
	for _, month := range creationMonths[1:] {
		format := "2006-01"
		monthStart, _ := time.Parse(format, month)
		monthEnd := monthStart.AddDate(0, 1, 0)
		var count int64

		// Count the players with last battle date falling within the month
		if err := server.DB.Raw("SELECT COUNT(*) FROM players WHERE last_battle_date > ? AND account_creation_date < ?", monthStart, monthEnd, monthStart).Count(&count).Error; err != nil {
			return nil, nil, err
		}

		monthlyPlayerCounts = append(monthlyPlayerCounts, int(count))
		monthlyPlayerMonths = append(monthlyPlayerMonths, month)
	}

	return monthlyPlayerCounts, monthlyPlayerMonths, nil
}
