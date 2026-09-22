package integration

// DiscordChannel says whether a Discord channel is saved; the address itself never leaves the server
type DiscordChannel struct {
	Connected bool `json:"connected"`
}

type DiscordChannelRequest struct {
	Address string `json:"address"`
}
