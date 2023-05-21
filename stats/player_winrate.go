package stats

import (
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

func (server *StatsServer) DistributionByWinRate(minBattles int) *charts.Line {
	// Define the step size for win rate increments
	step := 0.001

	// Define the win rate ranges and counts
	winRateRanges := make([]string, 0)
	winRateRangeValues := make([]float64, 0)
	counts := make([]opts.LineData, 0)

	// Iterate over win rate ranges with the specified step size
	start := 0.1
	end := start + step
	population := 0.0
	for end <= 0.9 {
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
			Subtitle: fmt.Sprintf("Mean: %.5f, StdDev: %.5f, Pop: %d, Steps: 0.1%%", mean, stdDev, int(population)),
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
	normalDist := generateNormalDistributionSeries(population, mean, stdDev, winRateRangeValues)
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

func generateNormalDistributionSeries(population, mean, stdDev float64, xLabels []float64) []opts.LineData {
	dist := distuv.Normal{
		Mu:    mean,
		Sigma: stdDev,
	}
	y := make([]opts.LineData, len(xLabels))
	for i, x := range xLabels {
		prob := dist.Prob(x)
		y[i] = opts.LineData{Value: prob * population / 1000.0}
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
