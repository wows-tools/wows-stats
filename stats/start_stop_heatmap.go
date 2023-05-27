package stats

import (
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"log"
	"sort"
)

const (
	PlayerStartStopChartMethodology = `
### Description

This chart represent on x,y,z axis:

* **x**: Month of Account creation
* **y**: Month of Last Battle
* **x**: Number of players who have created their account month **x** and played their last battle month **y**

This chart uses a [logarithmic scale](https://en.wikipedia.org/wiki/Logarithmic_scale). The high number of players playing just for one or two months makes it necessary.

### Code

[/start_stop_heatmap.go](https://github.com/wows-tools/wows-stats/blob/main/stats/start_stop_heatmap.go)

### Methodology

Here is how this graph is computed

* Select all players
* trim account creation date to "Year-Month" to get "start_month"
* trim last battle date to "Year-Month" to get "stop_month"
* group players by (start_month, stop_month)
* for each (start_month, stop_month) pair, compute the number of players
* display the result in a 3D Bar graph (using log scale)

### Caveats

Last Battle date must not be misinterpreted for "player left game" date, specially for the most recent months.
`
)

func (server *StatsServer) PlayerStartStopChart() *charts.Bar3D {
	// Query the players' start and stop months
	var results []struct {
		StartMonth  string
		StopMonth   string
		PlayerCount int
	}

	// Execute the query to get player start and stop months along with player count
	if err := server.DB.
		Raw("SELECT strftime('%Y-%m', account_creation_date) AS start_month, strftime('%Y-%m', last_battle_date) AS stop_month, COUNT(*) AS player_count FROM players WHERE last_battle_date > '2000-01-01 00:00:00+00:00' AND hidden_profile = 0  GROUP BY start_month, stop_month ORDER BY start_month ASC, stop_month ASC").
		Scan(&results).Error; err != nil {
		log.Fatal(err)
	}
	// Create a heatmap chart
	// Prepare data for the heatmap
	var Labels []string
	var data []opts.Chart3DData
	playerCounts := make(map[string]map[string]int)

	for _, result := range results {
		startMonth := result.StartMonth
		stopMonth := result.StopMonth
		playerCount := result.PlayerCount

		// Add start month and stop month to respective label lists if they don't exist
		if _, ok := playerCounts[startMonth]; !ok {
			playerCounts[startMonth] = make(map[string]int)
			Labels = append(Labels, startMonth)
		}
		playerCounts[startMonth][stopMonth] = playerCount
	}

	// Sort the label lists
	sort.Strings(Labels)
	max := 0
	lenData := len(Labels) - 2
	//lenData := 10
	for i, xlabel := range Labels[:lenData] {
		for j, ylabel := range Labels[:lenData] {
			// Add data point to heatmap
			playerCount := 0
			if _, ok := playerCounts[xlabel]; ok {
				if _, ok := playerCounts[xlabel][ylabel]; ok {
					playerCount = playerCounts[xlabel][ylabel]
				}
			}

			if playerCount != 0 {
				data = append(data, opts.Chart3DData{Value: []interface{}{i, j, playerCount, xlabel, ylabel}})
			}
			if max < playerCount {
				max = playerCount
			}
		}
	}

	var ToolTipFormatter = `
function (params) {
      return (
        params.value[2] + ' started playing "' +params.value[3] + '" and stopped "' + params.value[4] + '"'
      );
}
`

	//bar3DRangeColor := []string{
	//	"#313695", "#4575b4", "#74add1", "#abd9e9", "#e0f3f8",
	//	"#fee090", "#fdae61", "#f46d43", "#d73027", "#a50026",
	//}
	//bar3DRangeColor := generateColorRange()

	//heatmap := charts.NewBar3D()
	//heatmap.SetGlobalOptions(
	//	charts.WithTitleOpts(opts.Title{
	//		Title: "Player Start & Stop Months Heatmap",
	//	}),
	//	charts.WithVisualMapOpts(opts.VisualMap{
	//		Calculable: true,
	//		Max:        float32(max),
	//		Range:      []float32{0, float32(max)},
	//		InRange:    &opts.VisualMapInRange{Color: bar3DRangeColor},
	//	}),
	//	charts.WithXAxis3DOpts(opts.XAxis3D{Data: Labels[:lenData]}),
	//	charts.WithYAxis3DOpts(opts.YAxis3D{Data: Labels[:lenData]}),
	//	//charts.WithZAxis3DOpts(opts.ZAxis3D{Type: "log"}),
	//	//charts.WithVisualMapOpts(opts.VisualMap{
	//	//	Show:       true,
	//	//	Calculable: true,
	//	//}),
	//)

	//// Set the label lists to the X-axis and Y-axis options
	////heatmap.SetXAxis(Labels)
	////heatmap.SetYAxis(Labels)
	//// Add the data to the heatmap
	//heatmap.AddSeries("Player Start-Stop", data)
	//return heatmap

	bar3d := charts.NewBar3D()
	bar3d.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Players by 'Join * Last Battle' Months",
		}),
		charts.WithGrid3DOpts(opts.Grid3D{
			BoxWidth: 200,
			BoxDepth: 200,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Formatter: opts.FuncOpts(ToolTipFormatter),
		}),

		charts.WithXAxis3DOpts(opts.XAxis3D{Data: Labels[:lenData], Name: "Month Joined"}),
		charts.WithYAxis3DOpts(opts.YAxis3D{Data: Labels[:lenData], Name: "Month Last Battle"}),
		charts.WithZAxis3DOpts(opts.ZAxis3D{Type: "log", Show: false}),
	)
	bar3d.AddSeries("bar3d", data, charts.WithBar3DChartOpts(opts.Bar3DChart{Shading: "lambert"}))
	return bar3d

}

func generateColorRange() []string {
	var colorRange []string

	// RGB values for white
	r := 255
	g := 255
	b := 255

	// Calculate the color increment
	increment := 255 / 100

	// Generate the color range
	for i := 0; i < 100; i++ {
		// Convert RGB values to hexadecimal string
		hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)

		// Add the color to the range
		colorRange = append(colorRange, hexColor)

		// Decrement the red value
		r -= increment
	}

	return colorRange
}
