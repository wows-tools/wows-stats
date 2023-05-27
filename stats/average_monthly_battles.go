package stats

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/wows-tools/wows-stats/model"
	"sort"
)

const (
	MonthlyBattleEstimationMethodology = `
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

func (server *StatsServer) MonthlyBattleEstimation() *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Estimated Monthly Battles",
			Subtitle: "Sum of monthly average battle per player per month (warning: assumes player plays a constant number of battles per day)",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    true,
			Trigger: "axis",
			AxisPointer: &opts.AxisPointer{
				Type: "line",
			},
		}),
	)

	pageSize := 100000
	offset := 0
	data := make(map[string]float64)

	for {
		var players []model.Player
		result := server.DB.
			Limit(pageSize).
			Offset(offset).
			Find(&players)

		if result.Error != nil {
			// Handle the error
			return line
		}

		if result.RowsAffected == 0 {
			// No more players to process, exit the loop
			break
		}

		for _, player := range players {
			days := int(player.LastBattleDate.Sub(player.AccountCreationDate).Hours()/24) + 1
			avgDailyBattles := float64(player.RandomBattles) / float64(days)
			date := player.AccountCreationDate
			for date.Before(player.LastBattleDate) {
				month := date.Format("2006-01")
				nextDate := date.AddDate(0, 1, 0)
				if nextDate.Day() != 1 {
					nextDate = nextDate.AddDate(0, 0, -nextDate.Day()+1)
				}
				if nextDate.After(player.LastBattleDate) {
					nextDate = player.LastBattleDate
				}
				daysInMonth := int(nextDate.Sub(date).Hours() / 24)
				if _, ok := data[month]; !ok {
					data[month] = 0
				}
				data[month] += float64(daysInMonth) * avgDailyBattles
				date = nextDate
			}
		}

		offset += pageSize
	}

	// Sort the keys of the data map
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	res := make([]opts.LineData, len(data))
	for i, key := range keys {
		res[i] = opts.LineData{Value: float64(int(data[key]))}
	}

	line.SetXAxis(keys[:len(keys)-2]).AddSeries("Monthly Battles", res[:len(res)-2])

	return line
}
