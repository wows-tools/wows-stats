package stats

import (
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

const (
	DistributionByWinRateMethodology = `
### Description

The blue curve in the graph represents the distribution of players based on their win rate, considering only players who have reached a minimum number of battles.

Each point (x, y) on the graph represents the following:

- **x**: The win rate range of the players.
- **y**: The number of players whose win rate falls between x and x + "step".

The green curve represents the [Normal Distribution](https://en.wikipedia.org/wiki/Normal_distribution), which is obtained using the population's average win rate and its standard deviation.

The graph's subtitle displays the following information:

* Mean: The average win rate of the player population.
* Standard Deviation: The measure of the spread or dispersion of win rates within the population.
* Population Size: The total number of players considered in the analysis.
* Step: The interval used for dividing win rates into ranges.

By visualizing this graph, you can observe the distribution of players' win rates and compare it to the expected Normal Distribution based on the population's average win rate and standard deviation.

### Code

[/stats/player_winrate.go](https://github.com/wows-tools/wows-stats/blob/main/stats/player_winrate.go)

### Methodology

* Take all the intervals (wr-1, wr-2) of length "step" between 0% and 100% (e.g. 50.5% -> 51.0%).
* For each interval, count the number of players with a win rate between wr-1 and wr-2.
* From this, compute the Normal Distribution and we display the 2 curves.

### Caveats

#### Spikiness

Due to a large number of players having a low battle count, the distribution of Win Rates is not evenly spread.

For instance, if many players have only 4 battles, the graph will show spikes at 0% (0/4), 25% (1/4), 50% (2/4), 75% (3/4), and 100% (4/4) win rates.
As a result, when considering low minimum battle thresholds, the charts may appear "spiky" around these win rate values.
To partially compensate for this effect, wider steps are used in the charts with lower battle thresholds. This helps to smooth out the spikes and provide a clearer representation of the distribution.

With higher minimum battle thresholds, this issue disappears.

#### Other

Keep in mind it's a simple player distribution.

As it's more likely to encounter a frequent player with 2000, 5000 or +10000 battles, specially at high tiers, this chart doesn't reflect what your MM looks like.

This is especially true for the chart using the lowest threshold (1 battle) as it contains all the players trying the game for a few battles and leaving after that (not to mention protected MM bellow 200 battles).
`
)

// minBattles: Minimum number of battles
// step: size for win rate increments
func (server *StatsServer) DistributionByWinRate(minBattles int, step float64) *charts.Line {

	// Define the win rate ranges and counts
	winRateRanges := make([]string, 0)
	winRateRangeValues := make([]float64, 0)
	counts := make([]opts.LineData, 0)

	// Iterate over win rate ranges with the specified step size
	start := 0.0
	end := start + step
	population := 0.0
	for end <= 1 {
		// Get the count of players within the win rate range, excluding those with hidden profiles and below the minimum battles criteria
		winRateCount, _ := server.getPlayerCountByWinRate(start, end, minBattles)

		// Add the win rate range and count to the chart data
		winRateRange := fmt.Sprintf("%.2f%%", start*100)
		winRateRanges = append(winRateRanges, winRateRange)
		winRateRangeValues = append(winRateRangeValues, end)
		counts = append(counts, opts.LineData{Value: winRateCount})
		population += float64(winRateCount)

		// Increment the win rate range
		start += step
		end = start + step
	}

	// Compute the normal distribution
	mean, stdDev := computeNormalDistribution(winRateRangeValues, counts)

	// Create a line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    generateChartTitle(minBattles),
			Subtitle: fmt.Sprintf("Mean: %.5f, Std Dev: %.5f, Pop: %d, Step: %.1f%%", mean, stdDev, int(population), step*100),
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
			AxisPointer: &opts.AxisPointer{
				Type: "line",
			},
		}),
	)

	// Add data to the line chart
	line.SetXAxis(winRateRanges).
		AddSeries("Player Distribution", counts)

	// Add the normal distribution series to the line chart
	normalDist := generateNormalDistributionSeries(population, mean, stdDev, winRateRangeValues, step)
	line.AddSeries("Normal Distribution", normalDist)

	return line
}

func computeNormalDistribution(winRateRanges []float64, counts []opts.LineData) (float64, float64) {
	x := make([]float64, len(counts))
	y := make([]float64, len(counts))
	for i, count := range counts {
		x[i] = winRateRanges[i]
		y[i] = float64(count.Value.(int))
	}

	mean, stdDev := stat.PopMeanStdDev(winRateRanges, y)
	return mean, stdDev
}

func generateNormalDistributionSeries(population, mean, stdDev float64, xLabels []float64, step float64) []opts.LineData {
	dist := distuv.Normal{
		Mu:    mean,
		Sigma: stdDev,
	}
	y := make([]opts.LineData, len(xLabels))
	for i, x := range xLabels {
		prob := dist.Prob(x)
		step_count := 1 / step
		y[i] = opts.LineData{Value: prob * population / step_count}
	}
	return y
}

func generateChartTitle(minBattles int) string {
	title := "Win Rate Dist."
	if minBattles > 0 {
		title += fmt.Sprintf(" (Battles > %d)", minBattles)
	}
	return title
}

func (server *StatsServer) getPlayerCountByWinRate(start, end float64, minBattles int) (int, error) {
	var count int64
	query := server.DB.Table("players").
		Where("random_win_rate >= ? AND random_win_rate < ?", start, end).
		Where("random_battles >= ?", minBattles)
	err := query.Count(&count).Error
	return int(count), err
}
