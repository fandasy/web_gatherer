package app_cache

import (
	"time"
)

func (m *MapManager) CreateMap(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newMap := &mapOpj{
		m: make(map[string]valueWithTimer),
	}
	m.maps[name] = newMap
}

func (m *MapManager) DeleteFromMap(name, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.maps[name]; exists {
		delete(m.maps[name].m, key)
	}
}

func (m *MapManager) SetToMap(name, key string, value any, TTL time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.maps[name]; exists {
		if TTL <= 0 {
			m.maps[name].m[key] = valueWithTimer{
				value: value,
				timer: nil,
			}

		} else {
			if existingValue, exists := m.maps[name].m[key]; exists && existingValue.timer != nil {

				existingValue.timer.Reset(TTL)

				m.maps[name].m[key] = valueWithTimer{
					value: value,
					timer: existingValue.timer,
				}

			} else {
				timer := time.AfterFunc(TTL, func() {
					m.DeleteFromMap(name, key)
				})

				m.maps[name].m[key] = valueWithTimer{
					value: value,
					timer: timer,
				}
			}
		}

		return true

	} else {
		return false
	}
}

func (m *MapManager) SetToMapWithFunc(name, key string, value any, TTL time.Duration, fn func()) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.maps[name]; exists {
		if TTL <= 0 {
			m.maps[name].m[key] = valueWithTimer{
				value: value,
				timer: nil,
			}

		} else {
			if existingValue, exists := m.maps[name].m[key]; exists && existingValue.timer != nil {

				if fn != nil {
					existingValue.timer.Stop()

					timer := time.AfterFunc(TTL, func() {
						fn()
						m.DeleteFromMap(name, key)
					})

					m.maps[name].m[key] = valueWithTimer{
						value: value,
						timer: timer,
					}

				} else {
					existingValue.timer.Reset(TTL)

					m.maps[name].m[key] = valueWithTimer{
						value: value,
						timer: existingValue.timer,
					}
				}

			} else {
				if fn == nil {
					fn = func() {}
				}

				timer := time.AfterFunc(TTL, func() {
					fn()
					m.DeleteFromMap(name, key)
				})

				m.maps[name].m[key] = valueWithTimer{
					value: value,
					timer: timer,
				}
			}
		}

		return true

	} else {
		return false
	}
}

func (m *MapManager) GetFromMap(name, key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.maps[name]; exists {
		if val, exists := m.maps[name].m[key]; exists {
			return val.value, true
		}
	}

	return nil, false
}

const (
	mutexLockTime   = 5 * time.Second
	rwmutexLockTime = 5 * time.Second
)

func (m *MapManager) Mutex(name string, fn func()) {
	m.maps[name].mu.Lock()

	timer := time.AfterFunc(mutexLockTime, func() {
		m.maps[name].mu.Unlock()
	})

	fn()
	if timer.Stop() {
		m.maps[name].mu.Unlock()
	}
}

func (m *MapManager) RWMutex(name string, fn func()) {
	m.maps[name].mu.RLock()

	timer := time.AfterFunc(rwmutexLockTime, func() {
		m.maps[name].mu.RUnlock()
	})

	fn()
	if timer.Stop() {
		m.maps[name].mu.RUnlock()
	}
}
