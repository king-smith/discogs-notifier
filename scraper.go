package notifier

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

type WantListItem struct {
	ID                 string
	BlockedSellers     []string
	MaxPrice           int
	MinMediaCondition  int
	MinSleeveCondition int
	PreviousResults    []ListedItem
}

type WantList struct {
	ID        string
	Items     map[string]WantListItem
	Frequency int
}

type ListedItem struct {
	ID              string
	Seller          string
	Location        string
	Price           int
	MediaCondition  int
	SleeveCondition int
}

var ConditionMap = map[string]int{
	"Generic":        1,
	"Poor":           2,
	"Fair":           3,
	"Good Plus":      5,
	"Good":           4,
	"Very Good Plus": 7,
	"Very Good":      6,
	"Near Mint":      8,
	"Mint":           9,
}

func FetchListedItemDocument(url string) (*goquery.Document, error) {
	// Request the HTML page.
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	return doc, err
}

func StringToCondition(input string) int {
	for text, condition := range ConditionMap {
		if strings.Contains(input, text) {
			return condition
		}
	}

	return 0
}

func StringToPrice(input string) (int, error) {

	re := regexp.MustCompile(`[0-9]+(\.[0-9][0-9]?)?`)

	priceString := strings.ReplaceAll(re.FindString(input), ".", "")

	price, err := strconv.Atoi(priceString)
	if err != nil {
		return 0, err
	}

	return price, nil
}

func FindPriceFromSelection(s *goquery.Selection) (int, error) {
	rawPrice, err := StringToPrice(s.Find(".price").Text())
	if err != nil {
		return 0, err
	}

	shippingPrice, err := StringToPrice(s.Find(".item_shipping").Text())
	if err != nil {
		return 0, err
	}

	convertedPrice, err := StringToPrice(s.Find(".converted_price").Text())
	if err != nil {
		return 0, err
	}

	floatPrice := (float32(rawPrice) / (float32(rawPrice) + float32(shippingPrice))) * float32(convertedPrice)

	return int(floatPrice), nil
}

func FindItemsFromDoc(doc *goquery.Document) []ListedItem {

	items := []ListedItem{}

	doc.Find(".shortcut_navigable:not(.unavailable)").Each(func(i int, s *goquery.Selection) {
		idHref, ok := s.Find(".item_description_title").Attr("href")
		if !ok {
			log.Warn("Missing href when finding item ID")
			return
		}

		id := strings.ReplaceAll(idHref, "/sell/item/", "")

		mediaCondition := StringToCondition(s.Find(".item_condition span:nth-child(3)").Text())
		sleeveCondition := StringToCondition(s.Find(".item_condition span:nth-child(7)").Text())

		price, err := FindPriceFromSelection(s)
		if err != nil {
			log.Warn(err)
			return
		}

		seller := s.Find(".seller_info li:nth-child(1) strong").Text()
		location := strings.ReplaceAll(s.Find(".seller_info li:nth-child(3)").Text(), "Ships From:", "")

		item := ListedItem{
			ID:              id,
			MediaCondition:  mediaCondition,
			SleeveCondition: sleeveCondition,
			Price:           price,
			Seller:          seller,
			Location:        location,
		}

		items = append(items, item)
	})

	return items
}

func ScrapeListedItems(id string) ([]ListedItem, error) {

	doc, err := FetchListedItemDocument("https://www.discogs.com/sell/release/" + id)
	if err != nil {
		return nil, err
	}

	return FindItemsFromDoc(doc), nil
}
