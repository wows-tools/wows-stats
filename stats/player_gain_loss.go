package stats

import (
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/wows-tools/wows-stats/model"
	"sort"
)

const (
	PlayerGainLossBarMethodology = `
### Description

The chart displays the following counts for each month:

* The number of accounts created (shown in blue)
* The number of players who played their last battle this month (shown in red)

### Code

[stats/player_gain_loss.go](https://github.com/wows-tools/wows-stats/blob/main/stats/player_gain_loss.go)

### Methodology

The number are computed as follows:

* number of accounts created: Take the account creation date, keep only the the year and the month (e.g. '2020-07'), group by month, count the number of players each month.
* number of players who played their last battle this month: Take the last battle date, keep only the the year and the month (e.g. '2020-07'), group by month, count the number of players each month.

The last 2 months are not displayed.

### Caveats

A player who played his last battle this month (red) does not equate to a player leaving the game.

A reasonable assumption would be that players who have not been active for just a few months are likely to come back.
On the other hand, a player who last played 2 or 3 years ago has likely left the game for good (but returning players do exist).

Please keep these caveats in mind when interpreting the red portion of the graph, as it does not directly indicate a definitive loss of players.
`
)

const (
	PlayerGainLossBarNetMethodology = `
### Description

This chart illustrates the net difference between the number of accounts created and the number of players who played their last battle in each month.

### Code

[stats/player_gain_loss.go](https://github.com/wows-tools/wows-stats/blob/main/stats/player_gain_loss.go)

### Methodology

The numbers displayed on the chart are calculated using the following methodology:

* Determine the number of accounts created each month by extracting the account creation date, keeping only the year and month (e.g., '2020-07'). Group the accounts by month and count the number of accounts in each month.
* Calculate the number of players who played their last battle in each month by extracting the last battle date, keeping only the year and month (e.g., '2020-07'). Group the players by month and count the number of accounts in each month.
* Compute the net difference between the number of accounts created and the number of players who played their last battle for each month.

The last 2 months are not displayed.

### Caveats

As for the previous graph, "last battle date" doesn't equate to "player left the game date".

This means that:

* net gains are underestimated
* net losses are overestimated

This is especially true for more recent months.

Please keep these caveats in mind when interpreting this chart, as it does not directly indicate a definitive loss of players.
`
)

const (
	PlayerGainLossBarWithBattlesMethodoloy = `
### Description

The chart displays the following counts for each month:

* The number of accounts created with at least the set minimum number of battles at present (shown in blue).
* The number of players with at least the set minimum number of battles at present, who played their last battle in that month (shown in red).

### Code

[stats/player_gain_loss.go](https://github.com/wows-tools/wows-stats/blob/main/stats/player_gain_loss.go)

### Methodology

The number are computed as follows:

* filter the accounts with to get all the accounts with more than the minimum number of battles
* number of accounts created: Take the account creation date, keep only the year and the month (e.g. '2020-07'), group by month, count the number of players each month.
* number of players who played their last battle this month: Take the last battle date, keep only the year and the month (e.g. '2020-07'), group by month, count the number of players each month.

The last 2 months are not displayed.

### Caveats

As for the previously mentioned, "last battle date" doesn't equate to "player left the game date". So "losses" are over-estimated

In addition, players are not "born" with 200 or 2000 battles.

On average a player plays ~1 battle per day (between account creation and last battle date), this means it takes ~6 months to reach 200 battles. 

This means "gains" are under-estimated

This graph **will be removed**.

This graph might get replaced an estimation based on the battle per day distribution.
`
)

const (
	PlayerGainLossBarNetWithBattlesMethodoloy = `
### Description


This chart illustrates the net difference between the number of accounts created and the number of players who played their last battle in each month.

The players must have, presently, the minimum number of battles specified.

### Code

[stats/player_gain_loss.go](https://github.com/wows-tools/wows-stats/blob/main/stats/player_gain_loss.go)

### Methodology

* filter the accounts with to get all the accounts with more than the minimum number of battles
* number of accounts created: Take the account creation date, keep only the year and the month (e.g. '2020-07'), group by month, count the number of players each month.
* number of players who played their last battle this month: Take the last battle date, keep only the year and the month (e.g. '2020-07'), group by month, count the number of players each month.
* Compute the net difference between the number of accounts created and the number of players who played their last battle for each month.

### Caveats

Same as before, but worse, here we under-estimate account creations and over-estimate losses.

This double approximation renders this chart far too approximative.

This graph **will be removed**.
`
)

const (
	PlayerGainLossBarNetMethodology2000 = `
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

const (
	PlayerGainLossBarNetWithBattlesMethodoloy2000 = `
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

type dataPlusMinus struct {
	Plus  int
	Minus int
}

func getSortedKeys(m map[string]*dataPlusMinus) []string {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func (server *StatsServer) PlayerGainLossBar(minBattles int) []*charts.Bar {
	// Get the player count per month using LastBattleDate
	var result []struct {
		Month string
		Count int
	}

	resMap := make(map[string]*dataPlusMinus)

	server.DB.Model(&model.Player{}).
		Select("strftime('%Y-%m', account_creation_date) AS Month, COUNT(*) AS Count").
		Where("last_battle_date > '2000-01-01 00:00:00+00:00' AND hidden_profile = 0 AND random_battles > ?", minBattles).
		Group("Month").
		Order("Month").
		Find(&result)

	// We skip the last 2 months
	for _, entry := range result[0 : len(result)-2] {
		if _, ok := resMap[entry.Month]; !ok {
			resMap[entry.Month] = &dataPlusMinus{0, 0}
		}
		resMap[entry.Month].Plus = entry.Count
	}

	server.DB.Model(&model.Player{}).
		Select("strftime('%Y-%m', last_battle_date) AS Month, COUNT(*) AS Count").
		Where("last_battle_date > '2000-01-01 00:00:00+00:00' AND hidden_profile = 0 AND random_battles > ?", minBattles).
		Group("Month").
		Order("Month").
		Find(&result)

	// We skip the last 2 months
	for _, entry := range result[0 : len(result)-2] {
		if _, ok := resMap[entry.Month]; !ok {
			resMap[entry.Month] = &dataPlusMinus{0, 0}
		}
		resMap[entry.Month].Minus -= entry.Count
	}
	keys := getSortedKeys(resMap)
	var gains []opts.BarData
	var losses []opts.BarData
	var net []opts.BarData

	// Customize bar styles
	gainsColor := "#37a2da"
	lossesColor := "#ff4b3a"

	for _, key := range keys {
		gains = append(gains, opts.BarData{Value: resMap[key].Plus, ItemStyle: &opts.ItemStyle{
			Color: gainsColor,
		}})
		losses = append(losses, opts.BarData{Value: resMap[key].Minus, ItemStyle: &opts.ItemStyle{
			Color: lossesColor,
		}})
		net_entry := resMap[key].Plus + resMap[key].Minus
		color := lossesColor
		if net_entry > 0 {
			color = gainsColor
		} else {
			color = lossesColor
		}
		net = append(net, opts.BarData{Value: net_entry, ItemStyle: &opts.ItemStyle{
			Color: color,
		}})

	}
	titleBoth := "Player Account Creations Vs Player Count by Last Battle each Month"
	if minBattles != 0 {
		titleBoth = titleBoth + fmt.Sprintf(" (players with +%d random battles)", minBattles)
	}

	barGL := charts.NewBar()
	barGL.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    titleBoth,
			Subtitle: "Based on account creation date & last battle date (ignoring last 2 months)",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: false,
		}),
	)

	barGL.SetXAxis(keys).
		AddSeries("Player gain", gains, charts.WithBarChartOpts(
			opts.BarChart{
				Stack: "stack",
			}),
		).
		AddSeries("Player who played their last battle", losses, charts.WithBarChartOpts(
			opts.BarChart{
				Stack: "stack",
			}),
		)
	titleNet := "Player Account Creations Minus Player Count by Last Battle each Month"
	if minBattles != 0 {
		titleNet = titleNet + fmt.Sprintf(" (players with +%d random battles)", minBattles)
	}

	barNet := charts.NewBar()
	barNet.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    titleNet,
			Subtitle: "Based on account creation date & last battle date (ignoring last 2 months)",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: false,
		}),
	)

	barNet.SetXAxis(keys).
		AddSeries("Player difference", net, charts.WithBarChartOpts(
			opts.BarChart{
				Stack: "stack",
			}),
		)

	return []*charts.Bar{barGL, barNet}

}
