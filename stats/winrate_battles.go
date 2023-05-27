package stats

import (
	"fmt"
	"log"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

const (
	WinRateByBattlesMethodology = `
### Description

Displays the average Win Rate by battle count (steps of 500 battles)

### Code

[stats/winrate_battles.go][https://github.com/wows-tools/wows-stats/blob/main/stats/winrate_battles.go]

### Methodology

* Divide the number of random battles of each player by 500 (Integer Division) 
* Group by this value, and compute the average win rate of each Group
* Plot the chart

We ignore players with less  than 30000 (not enough players with that many battle for the average to be significant)

### Caveats

None identified.
`
)

func (server *StatsServer) WinRateByBattles() *charts.Bar {
	// Query the average win rate by random battles
	var results []struct {
		Battles        int
		AverageWinRate float64
	}

	if err := server.DB.Raw("SELECT random_battles / 500 AS battles, AVG(random_win_rate) AS average_win_rate FROM players WHERE random_battles > 0 AND random_battles < 30000 GROUP BY battles ORDER BY battles ASC").Scan(&results).Error; err != nil {
		log.Fatal(err)
	}

	// Prepare the data for the bar chart
	var categories []string
	var data []opts.BarData
	for _, result := range results {
		battleRange := fmt.Sprintf("%d-%d", result.Battles*500, (result.Battles+1)*500)
		categories = append(categories, battleRange)
		data = append(data, opts.BarData{Value: result.AverageWinRate * 100})
	}

	// Create a bar chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Average Win Rate by Random Battles",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: false,
		}),
	)

	// Set the x-axis and y-axis options
	bar.SetXAxis(categories).AddSeries("Win Rate", data, charts.WithBarChartOpts(
		opts.BarChart{
			Stack: "stack",
		}),
	)

	return bar
}
