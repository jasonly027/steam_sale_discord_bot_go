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

// App is appDetails flattened and simplified
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

// appDetails is the form Steam API naturally returns
type appDetails struct {
	Success bool           `json:"success"`
	Data    appDetailsData `json:"data"`
}

type appDetailsData struct {
	Name             string `json:"name"`
	SteamAppid       int    `json:"steam_appid"`
	IsFree           bool   `json:"is_free"`
	ShortDescription string `json:"short_description"`
	HeaderImage      string `json:"header_image"`

	Recommendations recommendations `json:"recommendations"`
	ReleaseDate     releaseDate     `json:"release_date"`
	PriceOverview   Price           `json:"price_overview"`
}

type recommendations struct {
	Total int `json:"total"`
}

type releaseDate struct {
	ComingSoon bool `json:"coming_soon"`
}

// SearchResult is rawSearchResult with its Appid converted to int
type SearchResult struct {
	Appid int
	Name  string
}

// rawSearchResult is the form Steam API naturally returns
type rawSearchResult struct {
	Appid string `json:"appid"`
	Name  string `json:"name"`
}

type httpClient interface {
	Get(string) (*http.Response, error)
}

// client is the client used for requests to the Steam API.
var client httpClient

func init() {
	client = &http.Client{Timeout: 10 * time.Second}
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

func newSearchResultFrom(s rawSearchResult) (SearchResult, error) {
	appid, err := strconv.Atoi(s.Appid)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Appid: appid, Name: s.Name}, nil
}

var ErrNetTryAgainLater = errors.New("too many requests. Try again later")

// apiGet sends a GET request to endpoint and tries to decode it into value
func apiGet[T any](endpoint string, value *T) error {
	resp, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode == http.StatusForbidden {
		return ErrNetTryAgainLater
	}

	err = json.NewDecoder(resp.Body).Decode(&value)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
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

	details := make(map[string]appDetails, 1)
	err := apiGet(endpoint, &details)
	if err != nil {
		return App{}, err
	}

	if !details[aid].Success || details[aid].Data.SteamAppid != appid {
		return App{}, errors.New("Invalid appid " + aid)
	}

	return newAppFrom(details[aid]), nil
}

// Search calls the Steam API with query to find apps.
//
// More information: https://github.com/Revadike/InternalSteamWebAPI/wiki/Search-Apps
func Search(query string) ([]SearchResult, error) {
	endpoint :=
		"https://steamcommunity.com/actions/SearchApps/" + url.QueryEscape(query)

	rawResults := []rawSearchResult{}
	err := apiGet(endpoint, &rawResults)
	if err != nil {
		return nil, err
	}

	results := []SearchResult{}
	for _, raw := range rawResults {
		result, err := newSearchResultFrom(raw)
		if err != nil {
			continue
		}

		results = append(results, result)
	}

	return results, nil
}
