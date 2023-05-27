package stats

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/wows-tools/wows-stats/model"
	"log"
)

const (
	DivVsSoloWRMethodology = `
### Description

Plot a random sample (5000 entries) of players by "solo+div win rate" x "div win rate".

Each blue dot is a player, with the following coordinates:

* **x**: Random Win Rate
* **y**: Division Win Rate

The green line represent "win rate" = "div win rate".

If a player's dot is above this line, the player has a better division win rate.

If a player's dot is bellow this line, the player win rate than division win rate.


### Code

[/stats/matrix_div_wr.go](https://github.com/wows-tools/wows-stats/blob/main/stats/matrix_div_wr.go)

### Methodology

* Select a random sample of players with at least 100 random battles and 100 division random battles.
* Plot them on a scatter chart

### Caveats

None identified.
`
)

func (server *StatsServer) DivVsSoloWR() *charts.Scatter {
	// Create the scatter plot matrix
	scatterMatrix := charts.NewScatter()

	// Query a random sample of 5000 players from the database
	var players []model.Player
	err := server.DB.
		Where("random_battles >= 100").
		Where("random_div_battles >= 100").
		Order("RANDOM()").
		Limit(5000).
		Find(&players).Error
	if err != nil {
		log.Fatal(err)
	}

	// Prepare the data for the scatter plot matrix
	data := make([]opts.ScatterData, len(players))
	for i, player := range players {
		data[i] = opts.ScatterData{Value: []interface{}{player.RandomWinRate, player.RandomDivWinRate}, SymbolSize: 7}
	}

	// Set the options for the scatter plot matrix
	scatterMatrix.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Win Rate vs Div Win Rate",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Random Win Rate",
			SplitLine: &opts.SplitLine{
				Show: true,
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Division Win Rate",
			SplitLine: &opts.SplitLine{
				Show: true,
			}}),
	)
	xy := make([]opts.ScatterData, 0)
	for i := 0; i < 1000; i++ {
		value := float64(i) / 1000
		xy = append(xy, opts.ScatterData{Value: []interface{}{value, value}, SymbolSize: 2})
	}

	// Set the series data for the scatter plot matrix
	scatterMatrix.AddSeries("player", data)
	scatterMatrix.AddSeries("wr = div wr", xy)

	return scatterMatrix
}
