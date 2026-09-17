package http

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/profile"
	"github.com/google/uuid"
)

type CreatePublicationGrantRequest struct {
	AppId             uuid.UUID   `json:"appId"`
	CategoryIds       []uuid.UUID `json:"categoryIds"`
	DefaultCategoryId uuid.UUID   `json:"defaultCategoryId"`
	Handle            string      `json:"handle"`
}

type OrderPublicationCategoriesRequest struct {
	CategoryIds []uuid.UUID `json:"categoryIds"`
}

type PublicationApp struct {
	Destinations []PublicationDestinationChoice `json:"destinations"`
	Home         string                         `json:"home"`
	Id           uuid.UUID                      `json:"id"`
	Name         string                         `json:"name"`
	Position     int                            `json:"position"`
	Retired      bool                           `json:"retired"`
	Slug         string                         `json:"slug"`
}

type PublicationCategory struct {
	Id       uuid.UUID `json:"id"`
	Label    string    `json:"label"`
	Position int       `json:"position"`
	Retired  bool      `json:"retired"`
	Slug     string    `json:"slug"`
}

type PublicationCategoryList struct {
	Categories []PublicationCategory `json:"categories"`
}

type PublicationGrant struct {
	Active                bool                           `json:"active"`
	App                   PublicationApp                 `json:"app"`
	Categories            []PublicationCategory          `json:"categories"`
	DefaultCategory       PublicationCategory            `json:"defaultCategory"`
	Destinations          []PublicationDestinationChoice `json:"destinations"`
	DestinationsInherited bool                           `json:"destinationsInherited"`
	GrantedAt             time.Time                      `json:"grantedAt"`
	Holder                PublicationGrantHolder         `json:"holder"`
	Id                    uuid.UUID                      `json:"id"`
	RevokedAt             *time.Time                     `json:"revokedAt,omitempty"`
}

type PublicationGrantHolder struct {
	Avatar      *profile.ProfileAvatar `json:"avatar,omitempty"`
	DisplayName string                 `json:"displayName"`
	Handle      string                 `json:"handle"`
	Restricted  bool                   `json:"restricted"`
}

type PublicationGrantList struct {
	Grants []PublicationGrant `json:"grants"`
}

type PublicationWorkspace struct {
	Admin      bool                  `json:"admin"`
	Apps       []PublicationApp      `json:"apps"`
	Categories []PublicationCategory `json:"categories"`
	Grants     []PublicationGrant    `json:"grants"`
	Handle     string                `json:"handle"`
}

type UpdatePublicationCategoryRequest struct {
	Label   *string `json:"label,omitempty"`
	Retired *bool   `json:"retired,omitempty"`
}

type UpdatePublicationGrantRequest struct {
	CategoryIds       *[]uuid.UUID `json:"categoryIds,omitempty"`
	DefaultCategoryId *uuid.UUID   `json:"defaultCategoryId,omitempty"`
}
