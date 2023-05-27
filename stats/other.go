package stats

const (
	OtherNotes = `
## Data Collection
### Code

You can find the code in the following file: [wows.go](https://github.com/wows-tools/wows-stats/blob/main/backend/wows.go).

### Methodology

To collect player statistics, the code utilizes the [WoWs API provided by Wargaming](https://developers.wargaming.net/).

However, due to limitations in the API, it is not possible to retrieve a complete list of all players directly. Instead, a search endpoint is available, which allows searching for players based on a nickname prefix. The prefix must have a minimum length of 3 characters, and the API returns a maximum of 100 profiles sorted alphabetically.

To overcome these limitations, a creative algorithm is implemented with the following steps:

1. Start with the shortest prefix '000' (the first prefix alphabetically).
2. Perform a search using the current prefix.
3. If the search result contains fewer than 100 players:
   - If the prefix ends with the last character in the alphabet ("` + "`" + `_` + "`" + `"), reduce the prefix size by 1 and move to the next prefix.
   - If the prefix does not end with the last character, set the prefix to the first len(current_prefix) + 1 characters of the last entry's nickname from the search result and move to the next prefix.
4. Repeat steps 2-3 until the prefix loops back to '000\*'.

Once all the players are retrieved using this method, an iteration is performed to gather additional details such as the number of battles, account creation date, etc. Clans are also scanned.

Additionally, the code includes a clean-up process to identify and remove potential QA/test accounts created by Wargaming. These accounts exhibit similar nickname patterns and identical statistics. Examples of these patterns are "pt\*tpt," "lp\_ru\_prod\*," and "auto\_\*." On the European server alone, there are approximately 330,000 such accounts.

The implementation of this algorithm is complex and error-prone. However, the code has successfully collected the following player counts (as of 2023/05/24) which looks consistent:

- EU: 7,058,091 players
- NA: 3,576,287 players
- ASIA: 3,894,958 players

Please note that running the complete data collection process for each server takes a considerable amount of time, approximately 1 to 2 days per server.

### Collect/Report Genation Frequency

This project collects data and generate the present report once per month, toward the end of the month.

## Final Notes

This project is on going and still early on. New charts will be added, problematic charts will be deleted and existing charts tweaked/improved.

This project will also improve over time by having a data history. One of its big flaw currently is the lack of historic data, which leads to numerous approximations. With historic data, these approximations will partially disappear over time.
`
)
