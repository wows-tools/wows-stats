// Auto generated file!

package wargaming

import (
	"context"
	"github.com/IceflowRE/go-wargaming/v4/wargaming/internal"
	"github.com/IceflowRE/go-wargaming/v4/wargaming/wot"
	"strconv"
	"strings"
)

// RatingsAccounts returns player ratings by specified IDs.
//
// https://developers.wargaming.net/reference/all/wot/ratings/accounts
//
// Deprecated: Attention! The method is deprecated.
//
// realm:
//     Valid realms: RealmAsia, RealmEu, RealmNa
// accountId:
//     IDs of player accounts. Maximum limit: 100.
// typ:
//     Rating period. For valid values, check the Types of ratings method.
func (service *WotService) RatingsAccounts(ctx context.Context, realm Realm, accountId []int, typ string, options *wot.RatingsAccountsOptions) (*wot.RatingsAccounts, error) {
	if err := validateRealm(realm, []Realm{RealmAsia, RealmEu, RealmNa}); err != nil {
		return nil, err
	}

	reqParam := map[string]string{
		"account_id": internal.SliceIntToString(accountId, ","),
		"type":       typ,
	}

	if options != nil {
		if options.BattleType != nil {
			reqParam["battle_type"] = *options.BattleType
		}
		if options.Date != nil {
			reqParam["date"] = strconv.FormatInt(options.Date.Unix(), 10)
		}
		if options.Fields != nil {
			reqParam["fields"] = strings.Join(options.Fields, ",")
		}
		if options.Language != nil {
			reqParam["language"] = *options.Language
		}
	}

	var data *wot.RatingsAccounts
	err := service.client.getRequest(ctx, sectionWot, realm, "/ratings/accounts/", reqParam, &data, nil)
	return data, err
}
