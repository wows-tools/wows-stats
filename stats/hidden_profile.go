package stats

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func (server *StatsServer) PieHiddenProfiles() *charts.Pie {
	// Get the number of players with hidden profiles
	hiddenProfileCount, _ := server.getPlayerCountByProfile(true)

	// Get the number of players without hidden profiles
	publicProfileCount, _ := server.getPlayerCountByProfile(false)

	// Create a pie chart
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Player Profiles",
		}),
	)

	// Add data to the pie chart
	pie.AddSeries("Player Profiles", []opts.PieData{
		{Value: hiddenProfileCount, Name: "Hidden Profiles"},
		{Value: publicProfileCount, Name: "Public Profiles"},
	})
	pie.SetSeriesOptions(
		charts.WithLabelOpts(
			opts.Label{
				Show:      true,
				Formatter: "{b} : {d}%",
			},
		),
	)
	return pie
}

func (server *StatsServer) getPlayerCountByProfile(hidden bool) (int, error) {
	var count int64
	err := server.DB.Table("players").Where("hidden_profile = ?", hidden).Count(&count).Error
	return int(count), err
}
