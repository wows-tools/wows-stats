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

**Estimation** of the number of random battles per month.

### Code

[/stats/average_monthly_battles.go](https://github.com/wows-tools/wows-stats/blob/main/stats/average_monthly_battles.go)

### Methodology

This chart is computed as follows:

1. Create a counter for each month
2. For each player:
   - compute the average number of random battles per day between the account creation and the last battle
   - each month between the account creation and the last battle, increment the counter by day_in_month * average_battle_per_day for this player (for the month of account creation and last battle, only the appropriate portion of the month is taken).
3. Plot the month counters on the chart.

### Caveats

It's important to consider the following caveats when interpreting the chart:

* This method provides an estimation and does not rely on actual historic data.
* The estimation assumes an even distribution of battles for each player, whereas players may not play at a constant rate.
* The estimation may under-estimate the number of battles in the most recent months.
* The estimation does not account for players who may have created their account and remained inactive for months or years before actively playing (old WoT players for example).
* This chart appears to be less inaccurate than the "Estimated Active Players Each Month" chart, but further verification is required.

Please keep these caveats in mind when analyzing the chart, as it provides only an estimate and may not reflect the actual distribution of battles.
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
			days := int((*player.LastBattleDate).Sub((*player.AccountCreationDate)).Hours()/24) + 1
			avgDailyBattles := float64(*player.RandomBattles) / float64(days)
			date := *player.AccountCreationDate
			for date.Before(*player.LastBattleDate) {
				month := date.Format("2006-01")
				nextDate := date.AddDate(0, 1, 0)
				if nextDate.Day() != 1 {
					nextDate = nextDate.AddDate(0, 0, -nextDate.Day()+1)
				}
				if nextDate.After(*player.LastBattleDate) {
					nextDate = *player.LastBattleDate
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
