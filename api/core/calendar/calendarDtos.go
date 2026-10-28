package calendar

// CalendarFeedDto: the address to subscribe to in a calendar app. Url is empty
// while the feed is off.
type CalendarFeedDto struct {
	Url string `json:"url"`
}
