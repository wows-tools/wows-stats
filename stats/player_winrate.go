package stats

import (
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func (server *StatsServer) DistributionByWinRate(minBattles int) *charts.Line {
	// Define the step size for win rate increments
	step := 0.01

	// Define the win rate ranges and counts
	winRateRanges := make([]string, 0)
	counts := make([]opts.LineData, 0)

	// Iterate over win rate ranges with the specified step size
	start := 0.1
	end := start + step
	for end <= 0.9 {
		// Get the count of players within the win rate range, excluding those with hidden profiles and below the minimum battles criteria
		winRateCount, _ := server.getPlayerCountByWinRate(start, end, minBattles)

		// Add the win rate range and count to the chart data
		winRateRange := fmt.Sprintf("%.2f%%", start*100)
		winRateRanges = append(winRateRanges, winRateRange)
		counts = append(counts, opts.LineData{Value: winRateCount})

		// Increment the win rate range
		start += step
		end = start + step
	}

	// Create a line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: generateChartTitle(minBattles),
		}),
	)

	// Add data to the line chart
	line.SetXAxis(winRateRanges).
		AddSeries("Player Distribution", counts)

	return line
}

func generateChartTitle(minBattles int) string {
	title := "Player Distribution by Win Rate"
	if minBattles > 0 {
		title += fmt.Sprintf(" (Min Battles: %d)", minBattles)
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
