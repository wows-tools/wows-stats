package backend

import (
	"context"
	"errors"
	"github.com/IceflowRE/go-wargaming/v4/wargaming"
	"github.com/IceflowRE/go-wargaming/v4/wargaming/wows"
	"github.com/wows-tools/wows-stats/model"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"time"
)

var (
	EURealm   = wargaming.RealmEu
	NARealm   = wargaming.RealmNa
	AsiaRealm = wargaming.RealmAsia
)

var (
	ErrShipReturnInvalid = errors.New("Invalid return size for ship listing")
	ErrUnknownRealm      = errors.New("Unknown Wows realm/server")
)

func WowsRealm(realmStr string) (wargaming.Realm, error) {
	switch realmStr {
	case "eu":
		return EURealm, nil
	case "na":
		return NARealm, nil
	case "asia":
		return AsiaRealm, nil
	default:
		return nil, ErrUnknownRealm
	}
}

type Backend struct {
	client         *wargaming.Client
	ShipMapping    map[int]int
	Realm          wargaming.Realm
	Logger         *zap.SugaredLogger
	DB             *gorm.DB
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func difference(a, b []*model.Player) []*model.Player {
	mb := make(map[int]*model.Player, len(b))
	for _, x := range b {
		mb[x.ID] = x
	}
	var diff []*model.Player
	for _, x := range a {
		if _, found := mb[x.ID]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func NewBackend(key string, realm string, logger *zap.SugaredLogger, db *gorm.DB) *Backend {
	wReam, err := WowsRealm(realm)
	if err != nil {
		return nil
	}
	return &Backend{
		client:         wargaming.NewClient(key, &wargaming.ClientOptions{HTTPClient: &http.Client{Timeout: 10 * time.Second}}),
		ShipMapping:    make(map[int]int),
		Realm:          wReam,
		Logger:         logger,
		DB:             db,
	}
}

func (backend *Backend) FillShipMapping() error {
	backend.Logger.Debugf("Start filling ship mapping")
	client := backend.client
	respSize := 9999
	pageNo := 1
	for respSize != 0 {
		res, _, err := client.Wows.EncyclopediaShips(context.Background(), wargaming.RealmEu, &wows.EncyclopediaShipsOptions{
			Fields: []string{"ship_id", "tier"},
			PageNo: &pageNo,
		})
		if err != nil && pageNo == 1 {
			return err
		}
		if err != nil {
			// FIXME the go-wargaming library doesn't provide the "meta" part of the response
			// (containing the number of pages and number of ships)
			// so for now, we stop on the first error which is not ideal...
			return nil
		}
		respSize = len(res)
		pageNo++
		for _, ship := range res {
			backend.ShipMapping[*ship.ShipId] = *ship.Tier
		}
	}
	backend.Logger.Debugf("Finish filling ship mapping")
	return nil

}

func (backend *Backend) GetPlayerT10Count(playerId int) (int, error) {
	backend.Logger.Debugf("Start getting T10 ship count for player %d", playerId)
	realm := backend.Realm
	client := backend.client
	ret := 0
	inGarage := "1"
	res, _, err := client.Wows.ShipsStats(context.Background(), realm, playerId, &wows.ShipsStatsOptions{
		Fields:   []string{"ship_id"},
		InGarage: &inGarage,
	})
	if err != nil {
		return 0, err
	}

	if len(res) != 1 {
		return 0, ErrShipReturnInvalid
	}
	shipList, ok := res[playerId]

	if !ok {
		return 0, ErrShipReturnInvalid
	}

	for _, ship := range shipList {
		shipTier, ok := backend.ShipMapping[*ship.ShipId]
		if !ok {
			continue
		}
		if shipTier == 10 {
			ret++
		}
	}
	backend.Logger.Debugf("Finish getting T10 ship count for player %d", playerId)
	return ret, nil
}

func (backend *Backend) GetPlayerDetails(playerIds []int, withT10 bool) ([]*model.Player, error) {
	backend.Logger.Debugf("Start getting player details for players %v", playerIds)
	realm := backend.Realm
	client := backend.client
	var ret []*model.Player
	players, err := client.Wows.AccountInfo(context.Background(), realm, playerIds, &wows.AccountInfoOptions{
		Fields: []string{"account_id", "created_at", "hidden_profile", "last_battle_time", "logout_at", "nickname", "statistics.pvp.wins", "statistics.pvp.battles", "statistics.battles"},
	})
	if err != nil {
		return nil, err
	}
	clanPlayers, err := client.Wows.ClansAccountinfo(context.Background(), realm, playerIds, &wows.ClansAccountinfoOptions{})
	if err != nil {
		return nil, err
	}

	for _, playerData := range players {
		if playerData == nil {
			continue
		}

		T10Count := 0
		JoinDate := time.Now()
		if clanPlayer, ok := clanPlayers[*playerData.AccountId]; ok {
			JoinDate = clanPlayer.JoinedAt.Time
		}

		if withT10 {
			T10Count, err = backend.GetPlayerT10Count(*playerData.AccountId)
			if err != nil {
				T10Count = 0
			}
		}
		var battles int
		var win int
		if playerData.Statistics == nil || playerData.Statistics.Pvp == nil || playerData.Statistics.Pvp.Battles == nil || playerData.Statistics.Pvp.Wins == nil {
			battles = 1
			win = 0
			backend.Logger.Debugf("no stats for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			battles = *playerData.Statistics.Pvp.Battles
			win = *playerData.Statistics.Pvp.Wins
		}
		player := &model.Player{
			ID:                  *playerData.AccountId,
			Nick:                *playerData.Nickname,
			AccountCreationDate: playerData.CreatedAt.Time,
			LastBattleDate:      playerData.LastBattleTime.Time,
			LastLogoutDate:      playerData.LogoutAt.Time,
			Battles:             battles,
			WinRate:             float64(win) / float64(battles),
			NumberT10:           T10Count,
			HiddenProfile:       *playerData.HiddenProfile,
			Tracked:             false,
			ClanJoinDate:        JoinDate,
		}
		ret = append(ret, player)
	}
	backend.Logger.Debugf("Finish getting player details for players %v", playerIds)
	return ret, nil
}

func (backend *Backend) ListClansIds(page int) ([]int, error) {
	backend.Logger.Debugf("Start listing clans page[%d]", page)
	client := backend.client
	var ret []int
	limit := 100
	res, err := client.Wows.ClansList(context.Background(), EURealm, &wows.ClansListOptions{
		Limit:  &limit,
		PageNo: &page,
		Fields: []string{"clan_id"},
	})
	if err != nil {
		return nil, err
	}
	for _, clan := range res {
		ret = append(ret, *clan.ClanId)
	}
	backend.Logger.Debugf("Finish listing clans page[%d]", page)
	return ret, nil
}

func (backend *Backend) GetClansDetails(clanIDs []int) (ret []*model.Clan, err error) {
	client := backend.client
	clanInfo, err := client.Wows.ClansInfo(context.Background(), EURealm, clanIDs, &wows.ClansInfoOptions{
		Extra:  []string{"members"},
		Fields: []string{"description", "name", "tag", "clan_id", "created_at", "is_clan_disbanded", "updated_at", "members_ids", "leader_id"},
	})
	if err != nil {
		return nil, err
	}

	for _, clan := range clanInfo {
		// Clan doesn't actually exist
		if clan == nil {
			continue
		}
		// Clan is disbanded, ignore
		if clan.IsClanDisbanded != nil && *clan.IsClanDisbanded {
			continue
		}

		clanString := *clan.Name

		var players []*model.Player
		for _, memberId := range clan.MembersIds {
			players = append(players, &model.Player{ID: memberId})
		}
		ret = append(ret, &model.Clan{
			ID:           *clan.ClanId,
			Name:         *clan.Name,
			Tag:          *clan.Tag,
			Players:      players,
			PlayerIDs:    clan.MembersIds,
			PlayerID:     *clan.LeaderId,
			CreationDate: clan.CreatedAt.Time,
			UpdatedDate:  clan.UpdatedAt.Time,
			Tracked:      false,
		})

	}
	return ret, nil
}

func (backend *Backend) UpdatePlayerListT10(playerList []*model.Player) ([]*model.Player, error) {
	var ids []int
	for _, player := range playerList {
		ids = append(ids, player.ID)
	}
	return backend.GetPlayerDetails(ids, true)
}

func (backend *Backend) UpdateClans(clanIDs []int) error {
	for {
		clanDetails, err := backend.GetClansDetails(clanIDs[0:(min(100, len(clanIDs)))])
		if err != nil {
			return err
		}

		for _, clan := range clanDetails {
			var clanPrev model.Clan
			clanPrev.ID = clan.ID
			err = backend.DB.Preload("Players").First(&clanPrev).Error
			if err == nil {
				// If the clan was previously tracked, we need to keep it tracked
				if clanPrev.Tracked {
					clan.Tracked = true

				}
				prevPlayersList := make([]int, len(clanPrev.Players))
				backend.Logger.Debugf("Clan [%s] already present, computing player diff", clan.Tag)
				for i, player := range clanPrev.Players {
					prevPlayersList[i] = player.ID
				}
				diff := difference(clanPrev.Players, clan.Players)
				if len(diff) != 0 {
					diff, err := backend.UpdatePlayerListT10(diff)
					if err != nil {
						backend.Logger.Infof("Failed to update players: %s", err.Error())
						continue
					}

					for _, player := range diff {
						backend.Logger.Infof("player '%s' left clan [%s]", player.Nick, clan.Tag)
						prevClanEntry := &model.PreviousClan{
							JoinDate:  player.ClanJoinDate,
							LeaveDate: time.Now(),
							ClanID:    clanPrev.ID,
							PlayerID:  player.ID,
						}
						backend.DB.Create(prevClanEntry)
						backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(player)
					}
				}
				backend.DB.Model(&clanPrev).Association("Players").Delete(diff)
			}

			// Upsert the clan informations
			backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(clan)
			backend.Logger.Debugf("Start getting player details for clan [%s]", clan.Tag)

			// Upsert the players information
			players, err := backend.GetPlayerDetails(clan.PlayerIDs, false)
			if err != nil {
				backend.Logger.Infof("Failed to get Players: %s", err.Error())
			}
			for _, player := range players {
				player.ClanID = clan.ID
				backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(player)
			}
			backend.Logger.Debugf("Finish getting player details for clan [%s]", clan.Tag)
		}
		if len(clanIDs) < 100 {
			break
		}
		clanIDs = clanIDs[100:]
	}
	return nil
}

func (backend *Backend) ScrapMonitoredClans() (err error) {
	backend.Logger.Infof("start scrapping monitored clans")
	var clans []model.Clan
	backend.DB.Where("tracked = true").Find(&clans)
	var ids []int
	for _, clan := range clans {
		ids = append(ids, clan.ID)
	}
	err = backend.UpdateClans(ids)
	backend.Logger.Infof("finish scrapping %d monitored clans", len(ids))
	return err
}

func (backend *Backend) ScrapAllClans() (err error) {
	backend.Logger.Infof("Start scrapping all clans")
	page := 1
	for {
		backend.Logger.Infof("Start scrapping clan page [%d]", page)
		clanIDs, err := backend.ListClansIds(page)
		if err != nil {
			return err
		}

		backend.UpdateClans(clanIDs)

		backend.Logger.Infof("Finish scrapping clan page [%d]", page)
		if len(clanIDs) < 100 {
			break
		}
		page++
	}
	backend.Logger.Infof("Finish scrapping all clans")
	return nil
}
