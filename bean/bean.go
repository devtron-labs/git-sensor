package bean

type StartupConfig struct {
	RestPort int `env:"SERVER_REST_PORT" envDefault:"8080"`
	GrpcPort int `env:"SERVER_GRPC_PORT" envDefault:"8081"`
}
