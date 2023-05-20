package stats

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/wows-tools/wows-stats/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"sort"
)

type StatsServer struct {
	Logger *zap.SugaredLogger
	DB     *gorm.DB
	Listen string
	Realm  string
}

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

func (server *StatsServer) PlayerGainLossBar() []*charts.Bar {
	// Get the player count per month using LastBattleDate
	var result []struct {
		Month string
		Count int
	}

	resMap := make(map[string]*dataPlusMinus)

	server.DB.Model(&model.Player{}).
		Select("strftime('%Y-%m', account_creation_date) AS Month, COUNT(*) AS Count").
		Where("last_battle_date > '2000-01-01 00:00:00+00:00' AND hidden_profile = 0").
		Group("Month").
		Order("Month").
		Find(&result)

	// We skip the last 2 months
	for _, entry := range result[0:len(result)-2] {
		if _, ok := resMap[entry.Month]; !ok {
			resMap[entry.Month] = &dataPlusMinus{0, 0}
		}
		resMap[entry.Month].Plus = entry.Count
	}

	server.DB.Model(&model.Player{}).
		Select("strftime('%Y-%m', last_battle_date) AS Month, COUNT(*) AS Count").
		Where("last_battle_date > '2000-01-01 00:00:00+00:00' AND hidden_profile = 0").
		Group("Month").
		Order("Month").
		Find(&result)

	// We skip the last 2 months
	for _, entry := range result[0:len(result)-2] {
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

	barGL := charts.NewBar()
	barGL.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Player Gain/Loss per Month",
			Subtitle: "Based on account creation date & last battle date",
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
				//BarWidth:  20,
				Stack:     "stack",
				//ItemStyle: &opts.ItemStyle{Color: gainsColor},
			}),
		).
		AddSeries("Player loss", losses, charts.WithBarChartOpts(
			opts.BarChart{
				//BarWidth:  20,
				Stack:     "stack",
				//ItemStyle: &opts.ItemStyle{Color: lossesColor},
			}),
		)
	barNet := charts.NewBar()
	barNet.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Player Net Gain/Loss per Month",
			Subtitle: "Based on account creation date & last battle date",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: true,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: false,
		}),
	)

	barNet.SetXAxis(keys).
		AddSeries("Player net gain/loss", net, charts.WithBarChartOpts(
			opts.BarChart{
				//BarWidth:  20,
				Stack:     "stack",
				//ItemStyle: &opts.ItemStyle{Color: gainsColor},
			}),
		)

	return []*charts.Bar{barGL, barNet}

}

func (server *StatsServer) All(w http.ResponseWriter, r *http.Request) {
	// Set up the Go-Echarts components
	page := components.NewPage()

	// Create a FlexLayout component
	page.SetLayout(components.PageFlexLayout)
	gainloss := server.PlayerGainLossBar()
	page.AddCharts(gainloss[0])
	page.AddCharts(gainloss[1])

	page.Render(w)
	return
	//return page.Render(w)
}

func NewStatsServer(listen string, logger *zap.SugaredLogger, db *gorm.DB, realm string) *StatsServer {
	return &StatsServer{
		Logger: logger,
		DB:     db,
		Listen: listen,
		Realm:  realm,
	}
}

func (server *StatsServer) Server() {
	http.HandleFunc("/", server.All)
	http.ListenAndServe(server.Listen, nil)
}
