package http

import (
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/google/uuid"
)

func announcementOf(
	destinations *[]uuid.UUID,
	roles *[]uuid.UUID,
	note *string,
) publication.Announcement {
	made := publication.Announcement{Ping: readIDs(roles)}
	if destinations != nil {
		chosen := readIDs(destinations)
		made.Destinations = &chosen
	}
	if note != nil {
		made.Note = *note
	}
	return made
}

func readIDs(listed *[]uuid.UUID) []uuid.UUID {
	if listed == nil {
		return nil
	}
	held := make([]uuid.UUID, 0, len(*listed))
	for _, one := range *listed {
		held = append(held, one)
	}
	return held
}
