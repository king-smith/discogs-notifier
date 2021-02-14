package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	rate "go.uber.org/ratelimit"
)

// Tag to read in list description
var notifyTag = "notify_me"

// Use rate limiter here to avoid triggering discogs limit and take
// before each Get method. (Should take after successful API call)
var limiter = rate.New(60, rate.Per(60*time.Second))

// AuthenticatedRequest takes a url and makes a request
// with authorization added to the header
// Authorization token 'DISCOGS_TOKEN' is set by .env file
func AuthenticatedRequest(url string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	discogsToken := os.Getenv("DISCOGS_TOKEN")

	req.Header.Set("Authorization", "Discogs token="+discogsToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Take from limiter after successful request
	limiter.Take()

	return resp, nil
}

// GetFilteredUserLists takes a user list api url and returns a
// slice of lists.
// The response is paginated so we loop and append
// until no further pagination is available
// We then filter the results by which we want notifications for
func GetFilteredUserLists(url string) ([]UserList, error) {

	userLists := []UserList{}

	for true {
		resp, err := AuthenticatedRequest(url)
		if err != nil {
			return nil, err
		}

		var data UserListsResponse

		// Decode request response into UserListsResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return nil, err
		}

		userLists = append(userLists, data.Lists...)

		// Check if pagination provides a Next url to use
		nextURL := data.Pagination.Urls.Next
		if nextURL != "" {
			url = nextURL
		} else {
			// Exit loop if not
			break
		}

	}

	// Return filtered results
	return FilterNotifyUserLists(userLists), nil
}

// GetListItems takes a list api url and returns a slice
// of items in this list (ListItem)
func GetListItems(url string) ([]ListItem, error) {
	var data ListResponse

	resp, err := AuthenticatedRequest(url)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data.Items, nil
}

// GetMarketItem takes a list item and url prefix and returns the marketplace
// statistics for this item.
func GetMarketItem(listItem ListItem, urlPrefix string) (*MarketItem, error) {
	var data MarketResponse

	url := fmt.Sprintf("%s%d?%s", urlPrefix, listItem.ID, os.Getenv("CURRENCY"))

	resp, err := AuthenticatedRequest(url)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	minimumPrice, err := ParseComment(listItem.Comment)
	if err != nil {
		return nil, err
	}

	item := MarketItem{
		ID:           listItem.ID,
		NumForSale:   data.NumForSale,
		MinimumPrice: minimumPrice,
		LowestPrice:  data.LowestPrice.Value,
		Name:         listItem.Title,
		URL:          listItem.URL,
		Currency:     data.LowestPrice.Currency,
	}

	return &item, nil
}

// ParseComment takes a string and parses it for a float value
// to be used as our minimum price (defaults to 0)
func ParseComment(comment string) (minimumPrice float64, err error) {

	// Parse entire comment as float
	// TODO: Add more robust parsing
	if comment != "" {
		minimumPrice, err = strconv.ParseFloat(comment, 32)
	}

	return
}

// FilterNotifyUserLists takes a slice of userLists and returns a
// a filtered slice where the description contains our notify tag
func FilterNotifyUserLists(userLists []UserList) []UserList {
	filteredUserLists := []UserList{}

	for _, list := range userLists {
		if strings.Contains(list.Description, notifyTag) {
			filteredUserLists = append(filteredUserLists, list)
		}
	}

	return filteredUserLists
}

// NotifyCheck takes a marketItem and the previous version of the
// marketItem and returns a boolean of whether the user should be
// notified of a new market listing
func NotifyCheck(marketItem, previousMarketItem MarketItem) bool {

	// Check if number of items for sale has increased
	if marketItem.NumForSale <= previousMarketItem.NumForSale {
		return false
	}

	// Check if a minimum price threshold is set and if the lowest price meets it
	if marketItem.MinimumPrice > 0 && marketItem.LowestPrice > marketItem.MinimumPrice {
		return false
	}

	return true
}

// Notify creates and sends an email to the user of a new item
func Notify(marketItem MarketItem) error {
	log.Infof("New listing found for %s", marketItem.Name)

	msg, err := marketItem.CreateEmailMessage()
	if err != nil {
		return err
	}

	return SendEmail(msg)
}

type Notifier struct {
	previousMarketItems map[int]MarketItem
	rateLimiter         rate.Limiter
}

// RunNotifier is the main logic loop for the program.
//
// It first finds all the lists of the user that they want notifications for
// 		For each list it retrieves the items in that list
//			For each list item it retrieves its marketplace stats
//          If this item satisfies our notify conditions, notify user
//          Store these marketplace stats to compare with our next loop
func RunNotifier() error {

	userListsURL := fmt.Sprintf("https://api.discogs.com/users/%s/lists", os.Getenv("DISCOGS_USERNAME"))

	previousMarketItems := map[int]MarketItem{}

	log.Debugf("Running notifier for '%s'", os.Getenv("DISCOGS_USERNAME"))

	for true {
		// Get lists from the user we want to be notified by
		userLists, err := GetFilteredUserLists(userListsURL)
		if err != nil {
			return err
		}

		for _, list := range userLists {
			log.Debugf("Fetching list '%s'", list.Name)

			// Get the items found in each list
			items, err := GetListItems(list.ResourceURL)
			if err != nil {
				log.Errorf("Error getting list items for %s due to %v", list.Name, err)
				continue
			}

			for _, item := range items {
				log.Debugf("Fetching item '%s'", item.Title)

				// Get the marketplace statistics for each item in the list
				marketItem, err := GetMarketItem(item, "https://api.discogs.com/marketplace/stats/")
				if err != nil {
					log.Errorf("Error getting market items for %s due to %v", marketItem.Name, err)
					continue
				}

				// Compare marketplace statistics with previous stats
				// Only compare and notify if a previous item exists in the map
				// (don't notify on first run)
				if previousMarketItem, ok := previousMarketItems[marketItem.ID]; ok {
					if NotifyCheck(*marketItem, previousMarketItem) {

						go func() {
							err = Notify(*marketItem)
							if err != nil {
								log.Errorf("Unable to notify new listing for %s due to %v", marketItem.Name, err)
							}
						}()

					}
				}

				// Add new marketplace statistics regardless of previous logic outcome
				previousMarketItems[marketItem.ID] = *marketItem

				log.Debugf("Updated market item %v", *marketItem)
			}
		}
	}

	return nil
}
