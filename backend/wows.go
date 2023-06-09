package backend

import (
	"context"
	"errors"
	"github.com/alitto/pond"
	"github.com/wows-tools/wows-stats/model"
	"github.com/wows-tools/wows-stats/pester"
	"github.com/wows-tools/wows-stats/wargaming"
	"github.com/wows-tools/wows-stats/wargaming/wows"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ErrWrongLengthPrefix = errors.New("The search should use a 3 letters prefix")
	ErrUnknownRealm      = errors.New("Unknown Wows realm/server")
	prefixOrder          = "0123456789_abcdefghijklmnopqrstuvwxyz"
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
	client         *wargaming.Client
	ShipMapping    map[int]int
	Realm          wargaming.Realm
	Logger         *zap.SugaredLogger
	DB             *gorm.DB
	APICallCounter int
	PrefixBreak    int // Number of trigram prefixes scanned before breaking, usefull when testing
	ClanBreak      int // Number of clan to scan before braking, once again for testing
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

func (backend *Backend) pesterLogger(e pester.ErrEntry) {
	if e.Err != nil {
		backend.Logger.Infof("Attempt %d/%d failed for [%s] %s | error: %s", e.Retry, e.Request, e.Verb, e.URL, e.Err)
	} else {
		backend.Logger.Infof("Attempt %d/%d failed for [%s] %s", e.Retry, e.Request, e.Verb, e.URL)
	}
}

func NewBackend(key string, realm string, logger *zap.SugaredLogger, db *gorm.DB) *Backend {
	wReam, err := WowsRealm(realm)
	if err != nil {
		return nil
	}

	client := pester.New()
	client.MaxRetries = 5
	client.SetTimeout(20 * time.Second)
	client.Backoff = pester.ExponentialJitterBackoff
	client.SetRetryOnHTTP429(true)
	client.Ratelimiter = rate.NewLimiter(rate.Every(time.Millisecond*105), 1)

	backend := &Backend{
		client:      wargaming.NewClient(key, &wargaming.ClientOptions{client}),
		ShipMapping: make(map[int]int),
		Realm:       wReam,
		Logger:      logger,
		DB:          db,
	}
	client.LogHook = backend.pesterLogger
	return backend
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
		backend.APICallCounter++
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
	backend.APICallCounter++

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
		Fields: []string{"account_id", "created_at", "hidden_profile", "last_battle_time", "logout_at", "nickname",
			"statistics.pvp.wins", "statistics.pvp.battles", "statistics.pvp.xp",
			"statistics.pvp_solo.wins", "statistics.pvp_solo.battles", "statistics.pvp_solo.xp",
			"statistics.pvp_div2.wins", "statistics.pvp_div2.battles", "statistics.pvp_div2.xp",
			"statistics.pvp_div3.wins", "statistics.pvp_div3.battles", "statistics.pvp_div3.xp",
			"statistics.rank_solo.wins", "statistics.rank_solo.battles", "statistics.rank_solo.xp",
			"statistics.pve.wins", "statistics.pve.battles", "statistics.pve.xp",
			"statistics.oper_solo.battles", "statistics.oper_solo.wins", "statistics.oper_solo.xp",
		},
		Extra: []string{
			"statistics.pve", "statistics.rank_solo", "statistics.pvp_solo", "statistics.pvp_div2", "statistics.pvp_div3", "statistics.oper_solo",
		},
	})
	if err != nil {
		return nil, err
	}
	backend.APICallCounter++

	clanPlayers, _, err := client.Wows.ClansAccountinfo(context.Background(), realm, playerIds, &wows.ClansAccountinfoOptions{})
	if err != nil {
		return nil, err
	}
	backend.APICallCounter++

	for _, playerData := range players {
		if playerData == nil {
			continue
		}

		T10Count := 0
		JoinDate := time.Now()

		if len(clanPlayers) != 0 {
			if clanPlayer, ok := clanPlayers[*playerData.AccountId]; ok && clanPlayer != nil && clanPlayer.JoinedAt != nil {
				JoinDate = clanPlayer.JoinedAt.Time
			}
		}

		if withT10 {
			T10Count, err = backend.GetPlayerT10Count(*playerData.AccountId)
			if err != nil {
				T10Count = 0
			}
		}
		var battles int
		var win int
		var wr float64
		if playerData.Statistics == nil || playerData.Statistics.Pvp == nil || playerData.Statistics.Pvp.Battles == nil || *playerData.Statistics.Pvp.Battles == 0 || playerData.Statistics.Pvp.Wins == nil {
			battles = 0
			win = 0
			wr = 0
			backend.Logger.Debugf("no pvp stats for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			battles = *playerData.Statistics.Pvp.Battles
			win = *playerData.Statistics.Pvp.Wins
			wr = float64(win) / float64(battles)
		}
		var coopbattles int
		var coopwin int
		var coopwr float64
		if playerData.Statistics == nil || playerData.Statistics.Pve == nil || playerData.Statistics.Pve.Battles == nil || *playerData.Statistics.Pve.Battles == 0 || playerData.Statistics.Pve.Wins == nil {
			coopbattles = 0
			coopwin = 0
			coopwr = 0
			backend.Logger.Debugf("no pve stats for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			coopbattles = *playerData.Statistics.Pve.Battles
			coopwin = *playerData.Statistics.Pve.Wins
			coopwr = float64(coopwin) / float64(coopbattles)
		}
		var operbattles int
		var operwin int
		var operwr float64
		if playerData.Statistics == nil || playerData.Statistics.OperSolo == nil || playerData.Statistics.OperSolo.Battles == nil || *playerData.Statistics.OperSolo.Battles == 0 || playerData.Statistics.OperSolo.Wins == nil {
			operbattles = 0
			operwin = 0
			operwr = 0
			backend.Logger.Debugf("no oper stats for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			operbattles = *playerData.Statistics.OperSolo.Battles
			operwin = *playerData.Statistics.OperSolo.Wins
			operwr = float64(operwin) / float64(operbattles)
		}
		var rankedbattles int
		var rankedwin int
		var rankedwr float64
		if playerData.Statistics == nil || playerData.Statistics.RankSolo == nil || playerData.Statistics.RankSolo.Battles == nil || *playerData.Statistics.RankSolo.Battles == 0 || playerData.Statistics.RankSolo.Wins == nil {
			rankedbattles = 0
			rankedwin = 0
			rankedwr = 0
			backend.Logger.Debugf("no oper stats for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			rankedbattles = *playerData.Statistics.RankSolo.Battles
			rankedwin = *playerData.Statistics.RankSolo.Wins
			rankedwr = float64(rankedwin) / float64(rankedbattles)
		}

		var divbattles int
		var divwin int
		var divwr float64
		if playerData.Statistics == nil || playerData.Statistics.PvpDiv2 == nil || playerData.Statistics.PvpDiv2.Battles == nil || *playerData.Statistics.PvpDiv2.Battles == 0 || playerData.Statistics.PvpDiv2.Wins == nil {
			divbattles = 0
			divwin = 0
			divwr = 0
			backend.Logger.Debugf("no pvp div2 stats for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			divbattles += *playerData.Statistics.PvpDiv2.Battles
			divwin += *playerData.Statistics.PvpDiv2.Wins
		}
		if playerData.Statistics == nil || playerData.Statistics.PvpDiv3 == nil || playerData.Statistics.PvpDiv3.Battles == nil || *playerData.Statistics.PvpDiv3.Battles == 0 || playerData.Statistics.PvpDiv3.Wins == nil {
			divbattles = 0
			divwin = 0
			backend.Logger.Debugf("no pvp div3 for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			divbattles += *playerData.Statistics.PvpDiv3.Battles
			divwin += *playerData.Statistics.PvpDiv3.Wins
		}
		if divbattles != 0 {
			divwr = float64(divwin) / float64(divbattles)
		}

		player := &model.Player{
			ID:                  *playerData.AccountId,
			Nick:                playerData.Nickname,
			AccountCreationDate: &playerData.CreatedAt.Time,
			LastBattleDate:      &playerData.LastBattleTime.Time,
			LastLogoutDate:      &playerData.LogoutAt.Time,
			RandomBattles:       &battles,
			RandomWinRate:       &wr,
			RankedBattles:       &rankedbattles,
			RankedWinRate:       &rankedwr,
			RandomDivBattles:    &divbattles,
			RandomDivWinRate:    &divwr,
			CoopBattles:         &coopbattles,
			CoopWinRate:         &coopwr,
			OperBattles:         &operbattles,
			OperWinRate:         &operwr,
			NumberT10:           &T10Count,
			HiddenProfile:       playerData.HiddenProfile,
			ClanJoinDate:        &JoinDate,
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
	res, _, err := client.Wows.ClansList(context.Background(), backend.Realm, &wows.ClansListOptions{
		Limit:  &limit,
		PageNo: &page,
		Fields: []string{"clan_id"},
	})
	if err != nil {
		return nil, err
	}
	backend.APICallCounter++
	for _, clan := range res {
		ret = append(ret, *clan.ClanId)
	}
	backend.Logger.Debugf("Finish listing clans page[%d]", page)
	return ret, nil
}

func (backend *Backend) GetClansDetails(clanIDs []int) (ret []*model.Clan, err error) {
	client := backend.client
	clanInfo, _, err := client.Wows.ClansInfo(context.Background(), backend.Realm, clanIDs, &wows.ClansInfoOptions{
		Extra:  []string{"members"},
		Fields: []string{"description", "name", "tag", "clan_id", "created_at", "is_clan_disbanded", "updated_at", "members_ids", "leader_id"},
	})
	if err != nil {
		return nil, err
	}
	backend.APICallCounter++

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
		if len(clanIDs) == 0 {
			return nil
		}
		clanDetails, err := backend.GetClansDetails(clanIDs[0:(min(pageSize, len(clanIDs)))])
		if err != nil {
			return err
		}

		for _, clan := range clanDetails {
			var clanPrev model.Clan
			clanPrev.ID = clan.ID
			err = backend.DB.Preload("Players").FirstOrInit(&clanPrev).Error
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
							JoinDate:  *player.ClanJoinDate,
							LeaveDate: time.Now(),
							ClanID:    clanPrev.ID,
							PlayerID:  player.ID,
						}
						backend.DB.Create(prevClanEntry)
					}
					backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(diff, 100)
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
				player.ClanID = &clan.ID
			}
			backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(players, 100)
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
	pool := pond.New(20, 10, pond.MinWorkers(20))

	backend.Logger.Infof("Start updating details for all players in DB")
	offset := 0
	for {
		backend.Logger.Debugf("updating details for players %d to %d", offset, offset+pageSize-1)
		var accountList []model.Player
		backend.DB.Offset(offset).Limit(pageSize).Find(&accountList)

		playerIDs := make([]int, len(accountList))
		var cur_offset int
		cur_offset = offset
		for i, account := range accountList {
			playerIDs[i] = account.ID
		}
		pool.Submit(func() { backend.UpdatePlayerBatch(playerIDs, cur_offset) })

		// If we go less than the pageSize in the last query, it's time to stop
		if len(playerIDs) < pageSize {
			break
		}
		offset += pageSize

	}
	pool.StopAndWait()
	backend.Logger.Infof("Finish updating details for all players in DB")
	return nil
}

func (backend *Backend) UpdatePlayerBatch(playerIDs []int, offset int) {
	players, err := backend.GetPlayerDetails(playerIDs, false)
	if err != nil {
		backend.Logger.Infof("Failed to get Players: %s", err.Error())
	}
	backend.DB.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(players, 100)
	if (backend.APICallCounter % 100) == 0 {
		backend.Logger.Infof("updated players %d to %d, %d api calls made", offset, offset+pageSize, backend.APICallCounter)
	}

}

func (backend *Backend) ScrapAllClans() (err error) {
	backend.Logger.Infof("Start scrapping all clans")
	pool := pond.New(20, 10, pond.MinWorkers(20))
	page := 1
	for {
		backend.Logger.Infof("Start scrapping clan page [%d]", page)
		var clanIDs []int
		clanIDs, err = backend.ListClansIds(page)
		if err != nil {
			return err
		}

		pool.Submit(func() { backend.UpdateClans(clanIDs) })

		if len(clanIDs) < 100 {
			break
		}
		if backend.ClanBreak != 0 && page > backend.ClanBreak {
			backend.Logger.Infof("[DEV MODE] breaking scan loop after ~%d clan pages", backend.ClanBreak)
			break
		}
		page++
	}
	pool.StopAndWait()
	backend.Logger.Infof("Finish scrapping all clans")
	return nil
}

func (backend *Backend) ScrapAll() (err error) {
	backend.Logger.Infof("Start scrapping all players")
	start := time.Now()
	err = backend.ScanAllPlayers()
	if err != nil {
		return err
	}

	err = backend.UpdateDetailsAllPlayers()
	if err != nil {
		return err
	}

	err = backend.ScrapAllClans()
	if err != nil {
		return err
	}
	end := time.Now()

	scan := &model.Scan{
		StartDate:    start,
		EndDate:      end,
		ApiCallCount: backend.APICallCounter,
	}
	backend.DB.Create(scan)

	// Clean-up, there are ~100k of these tests accounts all created the 2023-05-19 with 1927 random battles
	backend.DB.Where("nick LIKE 'pt%tpt' or nick LIKE 'lp_ru_prod%' OR nick LIKE 'auto_%'").Where("random_battles = ?", 1927).Delete(&model.Player{})
	backend.DB.Where("nick LIKE 'pt%tpt' or nick LIKE 'lp_ru_prod%' OR nick LIKE 'auto_%'").Where("random_battles = ?", 92).Delete(&model.Player{})
	backend.DB.Where("nick LIKE 'pt%tpt' or nick LIKE 'lp_ru_prod%' OR nick LIKE 'auto_%'").Where("random_battles = ?", 0).Delete(&model.Player{})
	backend.DB.Where("nick LIKE 'pt%tpt' or nick LIKE 'lp_ru_prod%' OR nick LIKE 'auto_%'").Where("random_battles = ?", 5).Delete(&model.Player{})

	backend.Logger.Infof("Finish scrapping all players")
	return nil
}

func (backend *Backend) ScanAllPlayers() (err error) {
	prefix := "000"
	trigramPrefixCount := 0
	backend.Logger.Infof("Start scanning all players")
	pool := pond.New(20, 10, pond.MinWorkers(20))
	for {
		curPrefix := prefix
		pool.Submit(func() { backend.ScanAllPlayersTrigram(curPrefix, &trigramPrefixCount) })
		prefix = backend.getNextPrefix(prefix, false, prefix)
		if prefix == "000" {
			break
		}
		// If we reached the total number of prefixes, we stop
		if backend.PrefixBreak != 0 && trigramPrefixCount > backend.PrefixBreak {
			backend.Logger.Infof("[DEV MODE] breaking scan loop after ~%d player trigram prefixes", backend.PrefixBreak)
			break
		}
	}
	pool.StopAndWait()
	backend.Logger.Infof("Finish scanning all players")
	return nil
}

func (backend *Backend) ScanAllPlayersTrigram(startingTrigramPrefix string, trigramPrefixCount *int) (err error) {
	if len(startingTrigramPrefix) != 3 {
		return ErrWrongLengthPrefix
	}

	trigramPrefixAll := len(prefixOrder) * len(prefixOrder) * len(prefixOrder)
	prefix := startingTrigramPrefix

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
		backend.APICallCounter++
		for _, playerData := range res {
			player := &model.Player{
				ID:   *playerData.AccountId,
				Nick: playerData.Nickname,
			}

			backend.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(player)
		}
		backend.Logger.Debugf("scrapped %d entries with prefix '%s'", len(res), prefix)
		morePrecisionRequired := (len(res) == pageSize)
		var lastNick string
		if len(res) != 0 {
			lastNick = *res[len(res)-1].Nickname
		} else {
			lastNick = prefix
		}
		prefix = backend.getNextPrefix(prefix, morePrecisionRequired, lastNick)

		// If the next prefix has a different starting Trigram, it's time to stop
		if prefix[0:3] != startingTrigramPrefix {
			*trigramPrefixCount++
			break
		}
		// Every 100 API call, log progress
		if (backend.APICallCounter % 100) == 0 {
			backend.Logger.Infof("scrapped %d/%d trigrams, %d api calls made, current prefix: %s", *trigramPrefixCount, trigramPrefixAll, backend.APICallCounter, prefix)
		}
	}
	return nil
}
