package models

import (
	"errors"
	"time"
)

const (
	RoleIDsMapName     = "RoleIDs"
	MediaGroupMapName  = "mediaGroupIDs"
	VkNewsGroupMapName = "vkNewsGroupMap"

	SubUserRole = "sub user"
	AdminRole   = "admin"

	MsgPhoto    = "Photo"
	MsgVideo    = "Video"
	MsgAudio    = "Audio"
	MsgDocument = "Document"
	MsgIframe   = "Iframe"
)

var (
	ErrSkipEvent   = errors.New("skip event")
	ErrBadRequest  = errors.New("bad request")
	ErrUnknownUser = errors.New("unknown user")
)

type User struct {
	UserID    int64
	Username  string
	FirstName string
	LastName  string
	RoleID    int64
}

type Role struct {
	RoleID   int64
	RoleName string
}

type MetaPair struct {
	Url  string `json:"url"`
	Type string `json:"type"`
}

type TgGroup struct {
	GroupID     int64
	Name        string
	Description string
}

type TgMetaPair struct {
	ID   string
	Type string
}

type TgGroupMessage struct {
	MessageID  int
	GroupID    int64
	Username   string
	Text       string
	MetadataID []TgMetaPair
	Metadata   []MetaPair
	CreatedAt  time.Time
}

type TgChannel struct {
	ChannelID   int64
	Name        string
	Description string
}

type TgChMessage struct {
	MessageID  int
	ChannelID  int64
	Text       string
	MetadataID []TgMetaPair
	Metadata   []MetaPair
	CreatedAt  time.Time
}

type VkGroup struct {
	ID     int
	Name   string
	Domain string
}

type VkMessage struct {
	MessageID int
	GroupID   int
	Text      string
	Metadata  []MetaPair
	CreatedAt time.Time
}

type WebMessage struct {
	ID        int64
	GroupName string
	Text      string
	Metadata  []MetaPair
	CreatedAt time.Time
	Type      string
}
