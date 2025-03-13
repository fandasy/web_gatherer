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
	m.mu.RLock()

	if exMap, exists := m.maps[name]; exists {
		m.mu.RUnlock()

		exMap.mu.Lock()
		defer exMap.mu.Unlock()

		delete(exMap.m, key)

		return
	}

	m.mu.RUnlock()
}

func (m *MapManager) SetToMap(name, key string, value any, TTL time.Duration) bool {
	m.mu.RLock()

	if existingMap, exists := m.maps[name]; exists {
		m.mu.RUnlock()

		existingMap.mu.Lock()
		defer existingMap.mu.Unlock()

		if TTL <= 0 {
			existingMap.m[key] = valueWithTimer{
				value: value,
				timer: nil,
			}

		} else {
			if existingValue, exists := existingMap.m[key]; exists && existingValue.timer != nil {

				existingValue.timer.Reset(TTL)

				existingMap.m[key] = valueWithTimer{
					value: value,
					timer: existingValue.timer,
				}

			} else {
				timer := time.AfterFunc(TTL, func() {
					m.DeleteFromMap(name, key)
				})

				existingMap.m[key] = valueWithTimer{
					value: value,
					timer: timer,
				}
			}
		}

		return true

	} else {
		m.mu.RUnlock()

		return false
	}
}

func (m *MapManager) SetToMapWithFunc(name, key string, value any, TTL time.Duration, fn func()) bool {
	m.mu.RLock()

	if existingMap, exists := m.maps[name]; exists {
		m.mu.RUnlock()

		existingMap.mu.Lock()
		defer existingMap.mu.Unlock()

		if TTL <= 0 {
			existingMap.m[key] = valueWithTimer{
				value: value,
				timer: nil,
			}

		} else {
			if existingValue, exists := existingMap.m[key]; exists && existingValue.timer != nil {

				if fn != nil {
					existingValue.timer.Stop()

					timer := time.AfterFunc(TTL, func() {
						fn()
						m.DeleteFromMap(name, key)
					})

					existingMap.m[key] = valueWithTimer{
						value: value,
						timer: timer,
					}

				} else {
					existingValue.timer.Reset(TTL)

					existingMap.m[key] = valueWithTimer{
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

				existingMap.m[key] = valueWithTimer{
					value: value,
					timer: timer,
				}
			}
		}

		return true

	} else {
		m.mu.RUnlock()

		return false
	}
}

func (m *MapManager) GetFromMap(name, key string) (any, bool) {
	m.mu.RLock()

	if exMap, exists := m.maps[name]; exists {
		m.mu.RUnlock()

		exMap.mu.RLock()
		defer exMap.mu.RUnlock()

		if val, exists := exMap.m[key]; exists {
			return val.value, true
		}

		return nil, false
	}

	m.mu.RUnlock()

	return nil, false
}

const (
	mutexLockTime   = 5 * time.Second
	rwmutexLockTime = 5 * time.Second
)

func (m *MapManager) Mutex(name string, fn func()) {
	m.mu.RLock()

	if exMap, exists := m.maps[name]; exists {
		m.mu.RUnlock()

		exMap.mu.Lock()

		timer := time.AfterFunc(mutexLockTime, func() {
			exMap.mu.Unlock()
		})

		fn()
		if timer.Stop() {
			exMap.mu.Unlock()
		}

		return
	}

	m.mu.RUnlock()
}

func (m *MapManager) RWMutex(name string, fn func()) {
	m.mu.RLock()

	if exMap, exists := m.maps[name]; exists {
		m.mu.RUnlock()

		exMap.mu.RLock()

		timer := time.AfterFunc(rwmutexLockTime, func() {
			exMap.mu.RUnlock()
		})

		fn()
		if timer.Stop() {
			exMap.mu.RUnlock()
		}

		return
	}

	m.mu.RUnlock()
}
