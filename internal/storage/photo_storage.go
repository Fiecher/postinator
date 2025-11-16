package storage

import "sync"

type InMemoryPhotoStorage struct {
	processing map[int64]bool
	mu         sync.RWMutex
}

func NewInMemoryPhotoStorage() *InMemoryPhotoStorage {
	return &InMemoryPhotoStorage{
		processing: make(map[int64]bool),
	}
}

func (ps *InMemoryPhotoStorage) IsProcessing(chatID int64) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.processing[chatID]
}

func (ps *InMemoryPhotoStorage) SetProcessing(chatID int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.processing[chatID] = true
}

func (ps *InMemoryPhotoStorage) ClearProcessing(chatID int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.processing, chatID)
}
