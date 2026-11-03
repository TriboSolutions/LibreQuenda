// Package rating: the 1-to-5 star averages shown for facilities and people.
package rating

// Summary: the average stars and how many gave them; never who.
type Summary struct {
	Average float64 `json:"average"` // 1 to 5, one decimal
	Count   int64   `json:"count"`
}

// Of rounds the average to one decimal; nil when nobody rated yet.
func Of(average float64, count int64) *Summary {
	if count <= 0 {
		return nil
	}
	return &Summary{Average: float64(int(average*10+0.5)) / 10, Count: count}
}
