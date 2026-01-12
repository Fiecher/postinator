package config

type Config struct {
	BotToken            string      `yaml:"bot_token"`
	AssetsDir           string      `yaml:"assets_dir"`
	TempDir             string      `yaml:"temp_dir"`
	BackgroundFile      string      `yaml:"background_file"`
	BackgroundStatsFile string      `yaml:"background_stats_file"`
	OverlayFile         string      `yaml:"overlay_file"`
	FontFile            string      `yaml:"font_file"`
	MaxFileSize         int64       `yaml:"max_file_size"`
	TogglToken          string      `yaml:"toggl_token"`
	TogglWorkspaceID    int         `yaml:"toggl_workspace"`
	Stats               StatsConfig `yaml:"stats"`
}

type ProjectMapping struct {
	DisplayName string   `yaml:"display_name"`
	Color       string   `yaml:"color"`
	TogglNames  []string `yaml:"toggl_names"`
}

type StatsConfig struct {
	Mappings []ProjectMapping `yaml:"mappings"`
	Other    ProjectMapping   `yaml:"other"`
}
