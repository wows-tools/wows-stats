# Data Collection

Code: [https://github.com/wows-tools/wows-stats/blob/main/backend/wows.go](https://github.com/wows-tools/wows-stats/blob/main/backend/wows.go)

The data collection uses WoWs APIs ([https://developers.wargaming.net/](https://developers.wargaming.net/))

This part is actually a bit tricky as it's not possible to fully enumerate all the players using the API.

Only a search endpoint is available, it searches by nickname prefix (with a minimum prefix length of 3), it returns at most 100 profiles, sorted alphabetically.

So, I had to be a bit creative to collect (hopefully) all the players stats.

The algorithm used is the following:

* \[1\] start with the shorter prefix '000' (first prefix alphabetically) \[1\]
* \[2\] search using said prefix
* \[3.0\] if the return has less 100 players, it means we got all the players with said prefix, and we can go to the next prefix (ex: 000 -> 001, abcd -> abce) and we go back to step \[2\]
* \[3.1\] if the return has less than 100 players and the last letter of the current search prefix is '\_' (the last alphabetically), we reduce the prefix size by 1, trimming the last letter, and going to the next prefix (ex: 000y\_ -> 000z). and we go to step \[2\]
* \[3.2\] if the return contains 100 players, it means they are additional players with a nickname starting with said prefix, we then take the last entry of the return and set the search prefix to be the first len(current\_prefix) + 1 letters of this last entry's nickname. and we go to step \[2\]
* \[3.3\] if the next prefix has looped back to '000\*', we stop

Once I've recovered all the players, I iterate on them to get the details (number of battles, account creation date, etc). I also scan clans (which can be listed).

Lastly, I'm doing a bit of clean-up of what appears to be qa/test accounts created by WG (they share the same nick name patterns and exactly the same stats), there are \~330k  of these on EU for example (nickname patterns: pt\*tpt, lp\_ru\_prod\* and auto\_\*)

The algorithm is quite complex and tricky/error prone to implement in fairness, but I managed to scrap the following numbers of players (pre clean-up and as of 2023/05/24):

* EU: 7058091 players
* NA: 3576287 players
* ASIA: 3894958 players

These numbers looks reasonably plausible to me.

Note: running the full data collection is quite long, ~1 to 2 days.

# Charts generation

## Player Gain/Loss per Month (players with +%d random battles and +%.1f%% WR)

### Code
[https://github.com/wows-tools/wows-stats/blob/main/stats/player_gain_loss.go](https://github.com/wows-tools/wows-stats/blob/main/stats/player_gain_loss.go)

### Description

These charts are based on the `account creation date` and the `last battle date`.

There are variants of these charts with additional filtering on Win Rate and number of random battles:

### Methodology

General principal:
* Take the `account_creation_date` of each players, trim it to contain only `<year>-<month>`, group by `<year>-<month>` and count the number of players by `<year>-<month>`
* Take the `last_battle_date` of each players, trim it to contain only `<year>-<month>`, group by `<year>-<month>` and count the number of players by `<year>-<month>`
* For the net gain/loss, substract the latter from the first for each month.

Additional filtering:

The code as a few filters to ignore hidden incomplete profiles (`last_battle_date > '2000-01-01 00:00:00+00:00' AND hidden_profile = 0`)

Variants by minimum number of battles and minimum random win rate are generated with a simple filter `random_battles > ? AND random_win_rate > ?`:

We ignore the 2 most recent months.

### Caveats

It's really hard to differenciate player who have left the game for good and players simply taking a pause from the game.

This is especially true for the most recent dates as players who only "left" the game 3 months ago are reasonably likely to return.

Another caveats is when additional filtering is applied. Players are not 'born' with 2k battles or 55% WR.

Players with 2000 battle or more play ~3 battles per day on average, meaning it takes ~2 years to reach this number of battles.
And that's just an average, roughly half the players will take longer (Complete, guesstimate, but I probably around 3 years for 80% of players which will ever reach 2k battles)

For players with 200 battles or more, it's 1.7 battles per day, meaning ~4 months (guesstimate ~6 months for 80% of players). 

The guestimations are, well, guestimations, but they are their to underlying the fact that ignoring the last 2 years or 4 months respectively is not sufficient.

Average battle per day was computed like that:

```sql
SELECT
    AVG(random_battles / (julianday(last_battle_date) - julianday(account_creation_date) + 1)) AS average_battles_per_day
FROM
    players
WHERE random_battles >= 2000;
```


### Possible improvements

Title needs to be changed to reflect what this graph represent.

With more history data, by recording periods of inactivities for each player, it might be possible to get a "return probability" function: given this player has not played in the last N months, he has X% chance of returning.

With that, we could then take the last_battle_date, multiply it with this probability function and get a better estimate of actual player loss.

## Active players in the last 3 months

### Code

[https://github.com/wows-tools/wows-stats/blob/main/stats/active_players.go](https://github.com/wows-tools/wows-stats/blob/main/stats/active_players.go)

### Methodology

Just count the number of players with a last battle date less than 3 month ago

### Possible improvements

Clearly state it is based on the last battle date

## Number of active players each month

### Description

Estimation of the number of active players each month

### Code

[https://github.com/wows-tools/wows-stats/blob/main/stats/active_players_monthly.go](https://github.com/wows-tools/wows-stats/blob/main/stats/active_players_monthly.go)

### Methodology

Take each months an account was created (trick to get all the individual months in the lifespan of the game).

For each month, count the number of players who have created their account before the end of the month and have their last battle after the last day of the month.

### Caveats

This graph is too problematic, counting a player "active" soly based on account creation date and last battle date is too much of an approximation.

It's misleading has it over estimates the number of players active in the 'old' days, making for 

### Possible improvements

This chart **needs to be removed** and replaced with actual historical data.

For the past, this data is not available, so this new chart will only start when this data collection is in placed.

It can also be completed by the number of monthly battles. For these, existing past data sets might exist ([http://maplesyrup.sweet.coocan.jp/wows/](http://maplesyrup.sweet.coocan.jp/wows/)) and if necessary to fill gaps, estimations might prove more accurate (and can be checked against real data).

## Hidden profiles

TODO

## Player In  Clan

TODO

## Win Rate Distribution

TODO

## Average Win Rate by Random Battle

TODO

## Win Rate vs Div Win Rate

TODO

## Player Count by random Battles

TODO
