// Auto generated file!

package wows

import (
	"github.com/IceflowRE/go-wargaming/v4/wargaming/wgnTime"
)

type ClansAccountinfoOptions struct {
	// Extra fields that will be added to the response. Valid values:
	//
	// "clan"
	Extra []string `json:"extra,omitempty"`
	// Response field. The fields are separated with commas. Embedded fields are separated with dots. To exclude a field, use "-" in front of its name. In case the parameter is not defined, the method returns all fields. Maximum limit: 100.
	Fields []string `json:"fields,omitempty"`
	// Localization language. Default is "en". Valid values:
	//
	// "cs" - Čeština
	// "de" - Deutsch
	// "en" - English (by default)
	// "es" - Español
	// "fr" - Français
	// "ja" - 日本語
	// "pl" - Polski
	// "ru" - Русский
	// "th" - ไทย
	// "zh-tw" - 繁體中文
	// "tr" - Türkçe
	// "zh-cn" - 中文
	// "pt-br" - Português do Brasil
	// "es-mx" - Español (México)
	Language *string `json:"language,omitempty"`
}

type ClansAccountinfo struct {
	// User ID
	AccountId *int `json:"account_id,omitempty"`
	// Player name
	AccountName *string `json:"account_name,omitempty"`
	// Short info about clan.
	// An extra field.
	Clan *struct {
		// Clan ID
		ClanId *int `json:"clan_id,omitempty"`
		// Clan creation date
		CreatedAt *wgnTime.UnixTime `json:"created_at,omitempty"`
		// Number of clan members
		MembersCount *int `json:"members_count,omitempty"`
		// Clan name
		Name *string `json:"name,omitempty"`
		// Clan tag
		Tag *string `json:"tag,omitempty"`
	} `json:"clan,omitempty"`
	// Clan ID
	ClanId *int `json:"clan_id,omitempty"`
	// Date when player joined clan
	JoinedAt *wgnTime.UnixTime `json:"joined_at,omitempty"`
	// Technical position name
	Role *string `json:"role,omitempty"`
}
