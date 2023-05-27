package stats

import (
	"fmt"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"log"
)

const (
	BarChartByRandomBattlesMethodology = `
### Description

TODO

### Code

TODO

### Methodology

TODO

### Caveats

TODO
`
)

func (server *StatsServer) BarChartByRandomBattles() *charts.Bar {
	// Create the bar chart
	bar := charts.NewBar()

	// Query the players' random battles count
	var results []struct {
		RandomBattles int
		PlayerCount   int
	}
	err := server.DB.Table("players").
		Select("((random_battles - 1) / 500) * 500 AS battles, COUNT(*) AS player_count").
		Where("random_battles > 0").
		Group("battles").
		Order("battles ASC").
		Find(&results).Error
	if err != nil {
		log.Fatal(err)
	}

	// Prepare the data for the bar chart
	xAxisData := make([]string, len(results))
	yAxisData := make([]opts.BarData, len(results))
	for i, result := range results {
		battles := i * 500
		xAxisData[i] = fmt.Sprintf("%d-%d", battles, battles+499)
		yAxisData[i] = opts.BarData{Value: result.PlayerCount, Tooltip: &opts.Tooltip{Show: true, Formatter: fmt.Sprintf("Battles: %d-%d\nPlayers: %d", battles, battles+499, result.PlayerCount)}}
	}

	// Set the options for the bar chart
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Player Count by Random Battles",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type: "log",
		}),
	)

	// Set the data for the bar chart
	bar.SetXAxis(xAxisData).
		AddSeries("Player Count", yAxisData)

	return bar
}
