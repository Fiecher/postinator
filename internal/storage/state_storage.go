package storage

import "sync"

type RenderStateStore struct {
	processing map[int64]bool
	mu         sync.RWMutex
}

func NewRenderStateStore() *RenderStateStore {
	return &RenderStateStore{
		processing: make(map[int64]bool),
	}
}

func (s *RenderStateStore) TryStart(chatID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.processing[chatID] {
		return false
	}

	s.processing[chatID] = true
	return true
}

func (s *RenderStateStore) Finish(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.processing, chatID)
}

func (s *RenderStateStore) IsProcessing(chatID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.processing[chatID]
}
