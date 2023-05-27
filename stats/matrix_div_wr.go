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

TODO

### Code

TODO

### Methodology

TODO

### Caveats

TODO
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
