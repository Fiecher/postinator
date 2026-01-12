package services

import (
	"context"
	"fmt"
	"postinator/internal/config"
	"postinator/internal/toggl"
)

type TogglService struct {
	client *toggl.Client
	cfg    config.StatsConfig
}

func NewTogglService(client *toggl.Client, cfg config.StatsConfig) *TogglService {
	return &TogglService{
		client: client,
		cfg:    cfg,
	}
}

func (s *TogglService) GetMonthlyStats(ctx context.Context, caption string) ([]toggl.StatItem, error) {
	start, end, err := s.client.ParseDates(caption)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dates: %w", err)
	}

	mappings := make([]config.ProjectMapping, len(s.cfg.Mappings))
	for i, m := range s.cfg.Mappings {
		mappings[i] = config.ProjectMapping{
			DisplayName: m.DisplayName,
			Color:       m.Color,
			TogglNames:  m.TogglNames,
		}
	}

	otherMapping := config.ProjectMapping{
		DisplayName: s.cfg.Other.DisplayName,
		Color:       s.cfg.Other.Color,
	}

	return s.client.GetStats(ctx, start, end, mappings, otherMapping)
}
