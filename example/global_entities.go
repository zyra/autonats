package example

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TotalImages int    `json:"total.images"`
}

type Image struct {
	ID     string `json:"id"`
	Url    string `json:"url"`
	UserId string `json:"userId"`
}
