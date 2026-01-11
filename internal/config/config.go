package config

type Config struct {
	BotToken string

	AssetsDir string
	TempDir   string

	BackgroundFile      string
	BackgroundStatsFile string
	OverlayFile         string
	FontFile            string

	MaxFileSize int64
}
