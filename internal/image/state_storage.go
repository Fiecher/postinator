package image

import "sync"

const (
	ModeNone = iota
	ModeStats
	ModePost
)

type UserSession struct {
	Mode       int
	Processing bool
}

type RenderStateStore struct {
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

func NewRenderStateStore() *RenderStateStore {
	return &RenderStateStore{
		sessions: make(map[int64]*UserSession),
	}
}

func (s *RenderStateStore) SetMode(chatID int64, mode int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[chatID]; !ok {
		s.sessions[chatID] = &UserSession{}
	}
	s.sessions[chatID].Mode = mode
}

func (s *RenderStateStore) GetMode(chatID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sess, ok := s.sessions[chatID]; ok {
		return sess.Mode
	}
	return ModeNone
}

func (s *RenderStateStore) TryStart(chatID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[chatID]
	if !ok {
		s.sessions[chatID] = &UserSession{Processing: true}
		return true
	}

	if sess.Processing {
		return false
	}

	sess.Processing = true
	return true
}

func (s *RenderStateStore) IsProcessing(chatID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sess, ok := s.sessions[chatID]; ok {
		return sess.Processing
	}
	return false
}

func (s *RenderStateStore) Finish(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}
