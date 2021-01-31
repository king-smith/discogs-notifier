package notifier

type ListItem struct {
	ID          int    `json:"id"`
	Title       string `json:"display_title"`
	URL         string `json:"uri"`
	ResourceURL string `json:"resource_url"`
	Comment     string `json:"comment"`
	Type        string `json:"type"`
}

type ListResponse struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	ResourceURL string     `json:"resource_url"`
	Description string     `json:"description"`
	DateAdded   string     `json:"created_ts"`
	DateChanged string     `json:"modified_ts"`
	Items       []ListItem `json:"items"`
}

type URLs struct {
	First string `json:"first,omitempty"`
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	Last  string `json:"last,omitempty"`
}

type Pagination struct {
	PerPage int  `json:"per_page"`
	Items   int  `json:"items"`
	Page    int  `json:"page"`
	Urls    URLs `json:"urls"`
	Pages   int  `json:"pages"`
}

type UserList struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"uri"`
	ResourceURL string `json:"resource_url"`
	Description string `json:"description"`
	DateAdded   string `json:"date_added"`
	DateChanged string `json:"date_changed"`
	Public      bool   `json:"public"`
}

type UserListsResponse struct {
	Pagination Pagination `json:"pagination"`
	Lists      []UserList `json:"lists"`
}

type LowestPrice struct {
	Currency string  `json:"currency"`
	Value    float64 `json:"value"`
}

type MarketResponse struct {
	LowestPrice LowestPrice `json:"lowest_price"`
	NumForSale  int         `json:"num_for_sale"`
	Blocked     bool        `json:"blocked_from_sale"`
}

type MarketItem struct {
	ID           int
	NumForSale   int
	MinimumPrice float64
	LowestPrice  float64
	Name         string
	URL          string
	Currency     string
}

func (item MarketItem) CreateEmailMessage() ([]byte, error) {
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	subject := "Subject: New " + item.Name + " listed!" + "\n"

	templateData := NotifyTemplate{
		Name: item.Name,
		URL:  item.URL,
	}

	body, err := ParseTemplate("email_template.html", templateData)
	if err != nil {
		return nil, err
	}

	msg := []byte(subject + mime + "\n" + body)

	return msg, nil
}
