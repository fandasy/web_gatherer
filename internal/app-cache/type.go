package app_cache

import (
	"sync"
	"time"
)

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
