package steam

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type mockClient struct {
	resp *http.Response
	err  error
}

func (c *mockClient) Get(string) (*http.Response, error) {
	return c.resp, c.err
}

func (c *mockClient) setBody(v any) {
	json, err := json.Marshal(v)
	if err != nil {
		log.Fatal("Failed to marshal v")
	}

	c.resp = &http.Response{
		Body: io.NopCloser(bytes.NewBuffer(json)),
	}
}

type newAppShould struct {
	suite.Suite
	arbitraryAppid      int
	arbitraryAppDetails appDetails
	client              mockClient
}

func (s *newAppShould) setReturnedDetails(d appDetails) {
	s.client.setBody(
		map[string]appDetails{
			strconv.Itoa(d.Data.SteamAppid): d,
		},
	)
}

func (s *newAppShould) SetupTest() {
	s.arbitraryAppid = 1

	s.arbitraryAppDetails = appDetails{
		Success: true,
		Data: appDetailsData{
			Name:             "Name",
			SteamAppid:       s.arbitraryAppid,
			IsFree:           false,
			ShortDescription: "Description",
			HeaderImage:      "Image",

			Recommendations: recommendations{
				Total: 1,
			},

			ReleaseDate: releaseDate{
				ComingSoon: false,
			},

			PriceOverview: Price{
				Discount: 1,
				Initial:  "Initial",
				Final:    "Final",
			},
		},
	}

	s.client = mockClient{
		resp: nil,
		err:  nil,
	}
	client = &s.client
}

func TestNewAppShould(t *testing.T) {
	suite.Run(t, new(newAppShould))
}

func (s *newAppShould) TestErrOnFalseSuccess() {
	s.setReturnedDetails(appDetails{
		Success: false,
		Data: appDetailsData{
			SteamAppid: s.arbitraryAppid, // must be set or it can also err from being unequal
		},
	})

	app, err := NewApp(s.arbitraryAppid)

	s.Error(err)
	s.Equal(app, App{})
}

func (s *newAppShould) TestErrOnDifferentAppid() {
	differentAppid := s.arbitraryAppid + 1
	s.setReturnedDetails(appDetails{
		Success: true, // must be true or it can also err from this being false
		Data: appDetailsData{
			SteamAppid: differentAppid,
		},
	})

	app, err := NewApp(s.arbitraryAppid)

	s.Error(err)
	s.Equal(app, App{})
}

func (s *newAppShould) TestNoErrOnTrueSuccessAndSameAppid() {
	s.setReturnedDetails(appDetails{
		Success: true,
		Data: appDetailsData{
			SteamAppid: s.arbitraryAppid,
		},
	})

	app, err := NewApp(s.arbitraryAppid)

	s.Nil(err)
	s.NotEqual(app, App{})
}

func (s *newAppShould) TestAppEqualToApp() {
	s.setReturnedDetails(s.arbitraryAppDetails)

	app, err := NewApp(s.arbitraryAppDetails.Data.SteamAppid)

	s.Nil(err)
	s.Equal(app, newAppFrom(s.arbitraryAppDetails))
}

type searchShould struct {
	suite.Suite
	client mockClient
}

func (s *searchShould) SetupTest() {
	s.client = mockClient{
		resp: nil,
		err:  nil,
	}
	client = &s.client
}

func TestSearchShould(t *testing.T) {
	suite.Run(t, new(searchShould))
}

func (s *searchShould) setReturnedResults(r []rawSearchResult) {
	s.client.setBody(r)
}

func (s *searchShould) TestResultsEmptyWhenResultNotConvToInt() {
	s.setReturnedResults([]rawSearchResult{
		{
			Appid: "NotConvertibleToInt",
		},
	})

	res, err := Search("")

	s.Nil(err)
	s.Empty(res)
}

func (s *searchShould) TestResultsEqualToResults() {
	rawRes := []rawSearchResult{
		{
			Name:  "Name",
			Appid: "1",
		},
	}
	s.setReturnedResults(rawRes)

	actualRes, err := Search("")

	s.Nil(err)
	expectedRes := []SearchResult{}
	for _, r := range rawRes {
		r, err := newSearchResultFrom(r)
		if err != nil {
			log.Fatal("Failed to convert to searchResult")
		}
		expectedRes = append(expectedRes, r)
	}
	s.Equal(actualRes, expectedRes)
}

func (s *searchShould) TestFiltersResultsThatArentConvToInt() {
	badRaw := rawSearchResult{
		Appid: "NotConvertibleToInt",
	}
	okRaw := rawSearchResult{
		Appid: "1",
	}
	rawRes := []rawSearchResult{badRaw, okRaw, badRaw, okRaw}
	s.setReturnedResults(rawRes)

	actualRes, err := Search("")

	s.Nil(err)
	expectedRes := []SearchResult{}
	for _, r := range rawRes {
		r, err := newSearchResultFrom(r)
		if err != nil {
			continue
		}
		expectedRes = append(expectedRes, r)
	}
	s.Equal(actualRes, expectedRes)
}
