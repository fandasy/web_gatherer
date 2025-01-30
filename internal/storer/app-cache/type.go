package app_cache

import (
	"sync"
	"time"
)

type AppCache interface {
	CreateMap(name string)
	DeleteFromMap(name, key string)
	SetToMap(name, key string, value any, TTL time.Duration) bool
	SetToMapWithFunc(name, key string, value any, TTL time.Duration, fn func()) bool
	GetFromMap(name, key string) (any, bool)
	Mutex(name string, fn func())
	RWMutex(name string, fn func())
}

type valueWithTimer struct {
	value any
	timer *time.Timer
}

type mapOpj struct {
	mu sync.RWMutex
	m  map[string]valueWithTimer
}

type MapManager struct {
	mu   sync.RWMutex
	maps map[string]*mapOpj
}

func New() *MapManager {
	return &MapManager{
		maps: make(map[string]*mapOpj),
	}
}
