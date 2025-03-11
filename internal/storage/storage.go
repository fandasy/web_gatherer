package storage

import (
	"errors"
)

var (
	ErrNoRecordsFound     = errors.New("no records found")
	ErrChannelAlreadyOpen = errors.New("channel already open")
	ErrChannelNotFound    = errors.New("channel not found")
)
