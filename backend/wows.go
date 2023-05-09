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
	"strings"
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
	prefixOrder          = "0123456789abcdefghijklmnopqrstuvwxyz_"
	pageSize             = 100
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
	client      *wargaming.Client
	ShipMapping map[int]int
	Realm       wargaming.Realm
	Logger      *zap.SugaredLogger
	DB          *gorm.DB
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
		client:      wargaming.NewClient(key, &wargaming.ClientOptions{HTTPClient: &http.Client{Timeout: 10 * time.Second}}),
		ShipMapping: make(map[int]int),
		Realm:       wReam,
		Logger:      logger,
		DB:          db,
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
	players, _, err := client.Wows.AccountInfo(context.Background(), realm, playerIds, &wows.AccountInfoOptions{
		Fields: []string{"account_id", "created_at", "hidden_profile", "last_battle_time", "logout_at", "nickname", "statistics.pvp.wins", "statistics.pvp.battles", "statistics.battles"},
	})
	if err != nil {
		return nil, err
	}
	//clanPlayers, _, err := client.Wows.ClansAccountinfo(context.Background(), realm, playerIds, &wows.ClansAccountinfoOptions{})
	if err != nil {
		return nil, err
	}

	for _, playerData := range players {
		if playerData == nil {
			continue
		}

		T10Count := 0
		JoinDate := time.Now()
		//if clanPlayer, ok := clanPlayers[*playerData.AccountId]; ok {
		//	JoinDate = clanPlayer.JoinedAt.Time
		//}

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
	limit := pageSize
	res, _, err := client.Wows.ClansList(context.Background(), EURealm, &wows.ClansListOptions{
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
	clanInfo, _, err := client.Wows.ClansInfo(context.Background(), EURealm, clanIDs, &wows.ClansInfoOptions{
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
		clanDetails, err := backend.GetClansDetails(clanIDs[0:(min(pageSize, len(clanIDs)))])
		if err != nil {
			return err
		}

		for _, clan := range clanDetails {
			var clanPrev model.Clan
			clanPrev.ID = clan.ID
			err = backend.DB.Preload("Players").First(&clanPrev).Error
			if err == nil {
				// If the clan was previously tracked, we need to keep it tracked
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
		if len(clanIDs) < pageSize {
			break
		}
		clanIDs = clanIDs[pageSize:]
	}
	return nil
}

func (backend *Backend) getNextPrefix(currentPrefix string, morePrecisionRequired bool, lastNick string) string {
	// Define the prefix characters in order

	// If we need more precision, we pick the last nick, and use the start of it as our next search prefix
	// The start is set to be one character longer than the current prefix
	if morePrecisionRequired && (len(lastNick) > len(currentPrefix)) {
		return strings.ToLower(lastNick[0 : len(currentPrefix)+1])
	}

	// If we don't require more precision and we reached the last char, we can reduce the size of the prefix
	// We can also start directly at the end of the last nickname obtained
	if len(currentPrefix) > 3 && currentPrefix[len(currentPrefix)-1] == prefixOrder[len(prefixOrder)-1] {
		currentPrefix = currentPrefix[0 : len(currentPrefix)-1]
	}

	// Find the indexes of the current prefix
	indexes := make([]int, len(currentPrefix))
	for i, char := range currentPrefix {
		indexes[i] = strings.IndexRune(prefixOrder, char)

	}

	// Increase the prefix index by one letter.
	// If the last char of the prefix reached the last char of prefixOrder ('_'), set the index to 0 and update the previous prefix char
	// Otherwise, simply increment
	for i := len(indexes) - 1; i >= 0; i-- {
		if indexes[i] >= (len(prefixOrder) - 1) {
			indexes[i] = 0
		} else {
			indexes[i] += 1
			break
		}
	}

	var result string
	//backend.Logger.Debugf("getting next prefix for '%s', indexes: %v", currentPrefix, indexes)
	for _, index := range indexes {
		result += string(prefixOrder[index])
	}
	return result
}

func (backend *Backend) UpdateDetailsAllPlayers() (err error) {
	backend.Logger.Infof("Start updating details for all players in DB")
	offset := 0
	for {
		backend.Logger.Debugf("updating details for players %d to %d", offset, offset+pageSize-1)
		var accountList []model.Player
		backend.DB.Offset(offset).Limit(pageSize).Find(&accountList)

		playerIDs := make([]int, len(accountList))
		for i, account := range accountList {
			playerIDs[i] = account.ID
		}
		players, err := backend.GetPlayerDetails(playerIDs, false)
		if err != nil {
			backend.Logger.Infof("Failed to get Players: %s", err.Error())
		}
		for _, player := range players {
			backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(player)
		}

		// If we go less than the pageSize in the last query, it's time to stop
		if len(accountList) < pageSize {
			break
		}
		offset += pageSize

	}
	backend.Logger.Infof("Finish updating details for all players in DB")
	return nil
}

func (backend *Backend) ScrapAllPlayers() (err error) {
	backend.Logger.Infof("Start scrapping all players")
	err = backend.ScanAllPlayers()
	if err != nil {
		return err
	}

	err = backend.UpdateDetailsAllPlayers()
	if err != nil {
		return err
	}
	backend.Logger.Infof("Finish scrapping all players")
	return nil
}

func (backend *Backend) ScanAllPlayers() (err error) {
	backend.Logger.Infof("Start scanning all players")
	prefix := "000"
	trigramPrefixAll := len(prefixOrder) * len(prefixOrder) * len(prefixOrder)
	trigramPrefixCount := 0
	apiCallCount := 0

	for {
		client := backend.client
		realm := backend.Realm
		limit := pageSize
		res, _, err := client.Wows.AccountList(context.Background(), realm, prefix, &wows.AccountListOptions{
			Fields: []string{"account_id", "nickname"},
			Type:   wargaming.String("startswith"),
			Limit:  &limit,
		})
		if err != nil {
			backend.Logger.Errorf("failed scrapping with prefix %s: %s", prefix, err.Error())
		}
		apiCallCount++
		for _, playerData := range res {
			player := &model.Player{
				ID:   *playerData.AccountId,
				Nick: *playerData.Nickname,
			}

			backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(player)
		}
		backend.Logger.Debugf("scrapped %d entries with prefix '%s'", len(res), prefix)
		morePrecisionRequired := (len(res) == pageSize)
		var lastNick string
		if len(res) != 0 {
			lastNick = *res[len(res)-1].Nickname
		} else {
			lastNick = prefix
		}
		// TODO compare old and new trigram prefix part to count trigrams
		prefix = backend.getNextPrefix(prefix, morePrecisionRequired, lastNick)

		// FIXME, we actually have a few trigrams that never pops as trigrams
		if len(prefix) == 3 {
			trigramPrefixCount++
		}

		// If we got all the trigram prefixes, we can stop
		// FIXME, we actually have a few trigrams that never pops as trigrams
		// So we overscan slightly
		if trigramPrefixCount > trigramPrefixAll {
			break
		}
		// Every 100 API call, log progress
		if (apiCallCount % 100) == 0 {
			backend.Logger.Infof("scrapped %d/%d trigrams, %d api calls made, current prefix: %s", trigramPrefixCount, trigramPrefixAll, apiCallCount, prefix)
		}
	}
	backend.Logger.Infof("Finish scanning all players")
	return nil
}
