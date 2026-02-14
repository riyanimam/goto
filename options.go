package awsmock

// Option configures a [MockServer].
type Option func(*serverConfig)

type serverConfig struct {
	services []Service
}

func defaultConfig() serverConfig {
	return serverConfig{}
}

// WithService registers an additional [Service] with the mock server.
// Use this to add custom service implementations or override built-in ones.
func WithService(svc Service) Option {
	return func(c *serverConfig) {
		c.services = append(c.services, svc)
	}
}
