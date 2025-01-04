// steam provides access to Steam API wrappers.
package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type App struct {
	Name        string
	Appid       int
	Free        bool
	Description string
	Image       string
	Reviews     int
	ComingSoon  bool
	Price
}

type Price struct {
	Discount int    `json:"discount_percent"`
	Initial  string `json:"initial_formatted"`
	Final    string `json:"final_formatted"`
}

type appDetails struct {
	Success bool `json:"success"`

	Data struct {
		Name             string `json:"name"`
		SteamAppid       int    `json:"steam_appid"`
		IsFree           bool   `json:"is_free"`
		ShortDescription string `json:"short_description"`
		HeaderImage      string `json:"header_image"`

		Recommendations struct {
			Total int `json:"total"`
		} `json:"recommendations"`

		ReleaseDate struct {
			ComingSoon bool `json:"coming_soon"`
		} `json:"release_date"`

		// Field may be missing, represented by nil
		PriceOverview Price `json:"price_overview"`
	} `json:"data"`
}

type SearchResult struct {
	Appid int
	Name  string
}

type searchApp struct {
	Appid string `json:"appid"`
	Name  string `json:"name"`
}

// Client is the HTTP client used for requests to the Steam API.
var client = http.Client{
	Timeout: 10 * time.Second,
}

func (a *App) Url() string {
	return "https://store.steampowered.com/app/" + fmt.Sprint(a.Appid)
}

func newAppFrom(d appDetails) App {
	return App{
		Name:        d.Data.Name,
		Appid:       d.Data.SteamAppid,
		Free:        d.Data.IsFree,
		Description: d.Data.ShortDescription,
		Image:       d.Data.HeaderImage,
		Reviews:     d.Data.Recommendations.Total,
		ComingSoon:  d.Data.ReleaseDate.ComingSoon,
		Price:       d.Data.PriceOverview,
	}
}

func newSearchResultFrom(s searchApp) (SearchResult, error) {
	appid, err := strconv.Atoi(s.Appid)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Appid: appid, Name: s.Name}, nil
}

// NewApp calls the Steam API with appid to retrieve information on that app.
// Fields may be unset, and Steam rate limits requests.
//
// More information: https://github.com/Revadike/InternalSteamWebAPI/wiki/Get-App-Details
func NewApp(appid int) (App, error) {
	aid := fmt.Sprint(appid)
	endpoint :=
		"https://store.steampowered.com/api/appdetails" +
			"?filters=basic,price_overview,recommendations,release_date&" +
			url.Values{
				"appids": {aid},
				"cc":     {"US"},
			}.Encode()

	resp, err := client.Get(endpoint)
	if err != nil {
		return App{}, err
	}
	defer resp.Body.Close()

	details := make(map[string]appDetails, 1)
	err = json.NewDecoder(resp.Body).Decode(&details)
	if err != nil {
		return App{}, err
	}

	if !details[aid].Success || details[aid].Data.SteamAppid != appid {
		return App{}, errors.New("Invalid appid " + aid)
	}

	return newAppFrom(details[aid]), nil
}

func Search(query string) ([]SearchResult, error) {
	endpoint :=
		"https://steamcommunity.com/actions/SearchApps/" + url.QueryEscape(query)

	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sApps := []searchApp{}
	err = json.NewDecoder(resp.Body).Decode(&sApps)
	if err != nil {
		return nil, err
	}

	sResults := []SearchResult{}
	for _, sApp := range sApps {
		sResult, err := newSearchResultFrom(sApp)
		if err != nil {
			continue
		}

		sResults = append(sResults, sResult)
	}

	return sResults, nil
}
