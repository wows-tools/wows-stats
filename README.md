# wows-stats

Tool scrapping player data from World of Warships API and generated interesting graphs about server population with this data.

# Build

## Requirement

You need to have `golang` > 1.19 and `make` installed

## Building

To build wows-stats, run:

```shell
make
```

# Running

To run this tool, you need:
* a WoWs API key
* to be patient 😀

## WoWs API key

To get a WoWs API key, please refer to [the Wargaming Developer Documentation](https://developers.wargaming.net/documentation/guide/getting-started/)

## Notice

Scrapping all the data is rather **slow**, be prepared to let the tool **run for several hours, if not days**.

## Running the tool

Help:

```shell
$ ./wows-stats -help

Usage of ./wows-stats:
  -apikey string
    	Wargaming.net API key
  -debug
    	Enable debug mode
  -output string
    	Output file path (required)
  -server string
    	World of Warships server
  -skip-generation
    	Skip report generation
  -skip-scraping
    	Skip scraping data
```

Scrapping the data, and generating the report:

```shell
./wows-stats -apikey "xxxxx" -output index.html -server eu
```

From there, you can make the report available as a static html file (using nginx for example).

Or you can open it in your browser:

```shell
$ firefox index.html
```

## Optional flags

You can skip the data collection or the report generation if needed:

```shell
# skip scrapping data (useful when tweaking graphs)
./wows-stats -apikey "xxxxx" -output index.html -server eu -skip-scraping

# skip report generation (useful when troubleshooting data collection issues)
./wows-stats -apikey "xxxxx" -output index.html -server eu -skip-generation
```

You can also enable more verbose logs if needed with the `-debug` flag:

```shell
./wows-stats -apikey "xxxxx" -output index.html -server eu -debug
```

# Development

The project is separated in two main parts:

* [Data collection](https://github.com/wows-tools/wows-stats/tree/main/backend)
* [Report Generation](https://github.com/wows-tools/wows-stats/tree/main/stats)

The raw data is stored in an SQLite DB, and its schema is defined [here](https://github.com/wows-tools/wows-stats/tree/main/model)

If you want to access the DB directly (after data collection of course):

```shell
export WOWS_SERVER=eu
sqlite3 "$WOWS_SERVER-stats.db"
```

And then run whatever query you want:
```
SQLite version 3.40.1 2022-12-28 14:03:47
Enter ".help" for usage hints.
sqlite> SELECT AVG(random_win_rate) from players where random_battles > 7000;
```

# Acknowledgement and Copyright

```
© wows-stat. All rights reserved | © Wargaming.net. All rights reserved
This is application is a third-party tool under Wargaming Developers Program.
```
