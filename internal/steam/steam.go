// steam provides access to Steam API wrappers.
package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

func (a *App) Url() string {
	return "https://store.steampowered.com/app/" + fmt.Sprint(a.Appid)
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

// Client is the HTTP client used for requests to the Steam API.
var client = http.Client{
	Timeout: 10 * time.Second,
}

// NewApp calls the Steam API with appid to retrieve information on that app.
// Fields may be unset, and Steam rate limits requests.
//
// More information: https://github.com/Revadike/InternalSteamWebAPI/wiki/Get-App-Details
func NewApp(appid int) (App, error) {
	aid := fmt.Sprint(appid)
	url :=
		"https://store.steampowered.com/api/appdetails" +
			"?filters=basic,price_overview,recommendations,release_date&" +
			url.Values{
				"appids": {aid},
				"cc":     {"US"},
			}.Encode()

	resp, err := client.Get(url)
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
