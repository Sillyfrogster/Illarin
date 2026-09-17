package http

type ProfileLink struct {
	Address string `json:"address"`
	Label   string `json:"label"`
}

type SaveProfileRequest struct {
	Biography    string        `json:"biography"`
	ContactEmail string        `json:"contactEmail"`
	DisplayName  string        `json:"displayName"`
	Links        []ProfileLink `json:"links"`
}
