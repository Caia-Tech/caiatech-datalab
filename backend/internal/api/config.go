package api

import "os"

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	MigrationsDir string
	AdminToken    string
}

func LoadConfigFromEnv() Config {
	listenAddr := getenvDefault("DATALAB_LISTEN_ADDR", ":8080")
	databaseURL := getenvDefault("DATALAB_DATABASE_URL", "postgres://datalab:datalab@localhost:5432/datalab?sslmode=disable")
	migrationsDir := getenvDefault("DATALAB_MIGRATIONS_DIR", "./migrations")
	adminToken := getenvDefault("DATALAB_ADMIN_TOKEN", "")

	return Config{
		ListenAddr:    listenAddr,
		DatabaseURL:   databaseURL,
		MigrationsDir: migrationsDir,
		AdminToken:    adminToken,
	}
}

func getenvDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
