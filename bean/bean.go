package bean

type StartupConfig struct {
	Protocol string `env:"SERVER_PROTOCOL" envDefault:"REST"`
	Port     int    `env:"SERVER_PORT" envDefault:"8080"`
}
