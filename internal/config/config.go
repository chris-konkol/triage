package config

import "github.com/kelseyhightower/envconfig"

type TicketSvc struct {
	GRPCPort     string `envconfig:"GRPC_PORT" default:"50051"`
	DatabaseURL  string `envconfig:"DATABASE_URL" required:"true"`
	KafkaBrokers string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	OTELEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"localhost:4317"`
	ServiceName  string `envconfig:"OTEL_SERVICE_NAME" default:"ticket-svc"`
}

type ConsumerSvc struct {
	KafkaBrokers string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	KafkaGroup   string `envconfig:"KAFKA_GROUP" required:"true"`
	DatabaseURL  string `envconfig:"DATABASE_URL"`
	OTELEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"localhost:4317"`
	ServiceName  string `envconfig:"OTEL_SERVICE_NAME"`
}

func LoadConsumerSvc() (*ConsumerSvc, error) {
	var cfg ConsumerSvc
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadTicketSvc() (*TicketSvc, error) {
	var cfg TicketSvc
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type AnalyticsSvc struct {
	GRPCPort     string `envconfig:"GRPC_PORT" default:"50052"`
	DatabaseURL  string `envconfig:"DATABASE_URL" required:"true"`
	KafkaBrokers string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	KafkaGroup   string `envconfig:"KAFKA_GROUP" default:"analytics-svc"`
	OTELEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"localhost:4317"`
	ServiceName  string `envconfig:"OTEL_SERVICE_NAME" default:"analytics-svc"`
}

func LoadAnalyticsSvc() (*AnalyticsSvc, error) {
	var cfg AnalyticsSvc
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Gateway struct {
	Port              string `envconfig:"PORT" default:"8080"`
	DatabaseURL       string `envconfig:"DATABASE_URL" required:"true"`
	TicketSvcAddr     string `envconfig:"TICKET_SVC_ADDR" default:"localhost:50051"`
	AnalyticsSvcAddr  string `envconfig:"ANALYTICS_SVC_ADDR" default:"localhost:50052"`
	JWTSecret         string `envconfig:"JWT_SECRET" default:"dev-secret-change-in-prod"`
	OTELEndpoint      string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"localhost:4317"`
	ServiceName       string `envconfig:"OTEL_SERVICE_NAME" default:"gateway"`
}

func LoadGateway() (*Gateway, error) {
	var cfg Gateway
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
