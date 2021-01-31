package notifier

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func MockJsonHandler(t *testing.T, v interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("Authorization")
		if authToken != tokenHeader {
			t.Fatalf("Expected header 'Authorization' to be %s, got %s", tokenHeader, authToken)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(v)
	}
}

func MockServerWithURL(URL string, handler http.Handler) (*httptest.Server, error) {
	ts := httptest.NewUnstartedServer(handler)
	if URL != "" {
		l, err := net.Listen("tcp", URL)
		if err != nil {
			return nil, err
		}
		ts.Listener.Close()
		ts.Listener = l
	}

	ts.Start()

	return ts, nil
}

var token = "MY_TOKEN"
var tokenHeader = "Discogs token=" + token

func TestAuthenticatedRequest(t *testing.T) {
	os.Setenv("DISCOGS_TOKEN", token)

	ts := httptest.NewServer(MockJsonHandler(t, ""))

	defer ts.Close()

	_, err := AuthenticatedRequest(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetListItems(t *testing.T) {
	os.Setenv("DISCOGS_TOKEN", token)

	responseData := ListResponse{
		ID:          0,
		Name:        "Test List",
		URL:         "https://discogs.com",
		ResourceURL: "https://api.discogs.com",
		Description: "",
		DateAdded:   "2021-01-31T10:00:17+00:00",
		Items: []ListItem{
			ListItem{
				ID:          1,
				Title:       "Test Item 1",
				URL:         "https://discogs.com/item1",
				ResourceURL: "https://api.discogs.com/item1",
				Comment:     "",
				Type:        "",
			},
			ListItem{
				ID:          2,
				Title:       "Test Item 2",
				URL:         "https://discogs.com/item2",
				ResourceURL: "https://api.discogs.com/item2",
				Comment:     "",
				Type:        "",
			},
		},
	}

	// Create mock server which will return desired json response
	ts := httptest.NewServer(MockJsonHandler(t, responseData))
	defer ts.Close()

	// Call server to get list items
	items, err := GetListItems(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Check if response items equal our input response data
	if !cmp.Equal(items, responseData.Items) {
		t.Errorf("Expected items %v, got %v", responseData.Items, items)
	}
}

// TestGetFilteredUserLists tests the response and pagination of retrieving
// user lists data from a url
func TestGetFilteredUserLists(t *testing.T) {
	url := "127.0.0.1:35357"
	nextPath := "/item2"
	os.Setenv("DISCOGS_TOKEN", token)

	responseData1 := UserListsResponse{
		Pagination: Pagination{
			Urls: URLs{
				Next: "http://" + url + nextPath,
			},
		},
		Lists: []UserList{
			UserList{
				ID:          1,
				Name:        "Test List 1",
				Description: "notify_me",
			},
			UserList{
				ID:          2,
				Name:        "Test List 2",
				Description: "notify_me",
			},
		},
	}

	responseData2 := UserListsResponse{
		Lists: []UserList{
			UserList{
				ID:          3,
				Name:        "Test List 3",
				Description: "notify_me",
			},
			UserList{
				ID:          4,
				Name:        "Test List 4",
				Description: "notify_me",
			},
		},
	}

	// Create handlers for each individual pagination path
	firstResponseHandler := MockJsonHandler(t, responseData1)
	secondResponseHandler := MockJsonHandler(t, responseData2)

	// Create mux for each path
	mux := http.NewServeMux()
	mux.Handle("/", firstResponseHandler)
	mux.Handle(nextPath, secondResponseHandler)

	// Create a server with a preset url so we can specify the pagination url
	ts, err := MockServerWithURL(url, mux)
	defer ts.Close()

	if err != nil {
		log.Fatal(err)
	}

	// Call server to get and filter lists
	userLists, err := GetFilteredUserLists(ts.URL)
	if err != nil {
		log.Fatal(err)
	}

	// Append our response input lists and check if this equals our response lists
	lists := append(responseData1.Lists, responseData2.Lists...)
	if !cmp.Equal(userLists, lists) {
		t.Errorf("Expected lists %v, got %v", lists, userLists)
	}
}

func TestFilterNotifyUserLists(t *testing.T) {
	// Create lists with notifyTag in the description
	notifyUserLists := []UserList{
		UserList{
			ID:          1,
			Name:        "Test List 1",
			Description: notifyTag,
		},
		UserList{
			ID:          2,
			Name:        "Test List 2",
			Description: "hello please " + notifyTag,
		},
		UserList{
			ID:          3,
			Name:        "Test List 3",
			Description: "a " + notifyTag + " b",
		},
	}

	// Create lists without notifyTag in the description
	baseUserLists := []UserList{
		UserList{
			ID:          4,
			Name:        "Test List 4",
			Description: "hello",
		},
		UserList{
			ID:          5,
			Name:        "Test List 5",
			Description: "do not notify me",
		},
	}

	// Concat slices to pass in full user lists
	userLists := append(baseUserLists, notifyUserLists...)

	filteredUserLists := FilterNotifyUserLists(userLists)

	if !cmp.Equal(filteredUserLists, notifyUserLists) {
		t.Errorf("Expected lists %v, got %v", notifyUserLists, filteredUserLists)
	}
}

func TestGetMarketItem(t *testing.T) {
	currency := "AUD"

	os.Setenv("CURRENCY", currency)
	os.Setenv("DISCOGS_TOKEN", token)

	// Create response
	responseData := MarketResponse{
		LowestPrice: LowestPrice{
			Currency: currency,
			Value:    30,
		},
		NumForSale: 10,
		Blocked:    false,
	}

	// Create input
	listItem := ListItem{
		ID:          1,
		Title:       "Test Item 1",
		URL:         "https://discogs.com/item1",
		ResourceURL: "https://api.discogs.com/item1",
		Comment:     "35",
		Type:        "",
	}

	handler := MockJsonHandler(t, responseData)

	// Create mux route for our ID item path
	mux := http.NewServeMux()
	mux.Handle(fmt.Sprintf("/%d", listItem.ID), handler)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Use ts.URL as the prefix
	marketItem, err := GetMarketItem(listItem, ts.URL+"/")
	if err != nil {
		t.Fatal(err)
	}

	// Parse the comment to use for our expected output
	minPrice, err := ParseComment(listItem.Comment)
	if err != nil {
		t.Fatal(err)
	}

	// Expected item is the combination of data of our list item
	// and the market response stats
	expectedMarketItem := MarketItem{
		ID:           listItem.ID,
		NumForSale:   responseData.NumForSale,
		MinimumPrice: minPrice,
		LowestPrice:  responseData.LowestPrice.Value,
		Name:         listItem.Title,
		URL:          listItem.URL,
		Currency:     responseData.LowestPrice.Currency,
	}

	if !cmp.Equal(*marketItem, expectedMarketItem) {
		t.Errorf("Expected item %v, got %v", expectedMarketItem, *marketItem)
	}
}

func TestNotifyCheck(t *testing.T) {
	previousMarketItem := MarketItem{
		NumForSale: 10,
	}

	marketItem := MarketItem{
		NumForSale:   10,
		MinimumPrice: 0,
		LowestPrice:  30,
	}

	if NotifyCheck(marketItem, previousMarketItem) {
		t.Error("Expected false result with same NumForSale")
	}

	marketItem.NumForSale++

	if !NotifyCheck(marketItem, previousMarketItem) {
		t.Error("Expected true result with larger marketItem NumForSale")
	}

	marketItem.MinimumPrice = 30

	if !NotifyCheck(marketItem, previousMarketItem) {
		t.Error("Expected true result when lowest price is the minimum price")
	}

	marketItem.MinimumPrice = 29

	if NotifyCheck(marketItem, previousMarketItem) {
		t.Error("Expected false result with lower minimum price than lowest price")
	}
}

type CommentCase struct {
	Comment  string
	Expected float64
	Valid    bool
}

func TestParseComment(t *testing.T) {
	cases := []CommentCase{
		CommentCase{
			Comment:  "30",
			Expected: 30,
			Valid:    true,
		},
		CommentCase{
			Comment:  "30.5",
			Expected: 30.5,
			Valid:    true,
		},
		CommentCase{
			Comment:  "",
			Expected: 0,
			Valid:    true,
		},
		CommentCase{
			Comment:  "hello 30",
			Expected: 0,
			Valid:    false,
		},
	}

	for _, _case := range cases {
		minPrice, err := ParseComment(_case.Comment)
		if err != nil {
			if _case.Valid {
				t.Error(err)
			} else {
				continue
			}
		}

		if minPrice != _case.Expected {
			t.Errorf("Expected min price %f, got %f", _case.Expected, minPrice)
		}
	}
}
