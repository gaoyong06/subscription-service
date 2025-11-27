package conf

import (
	"fmt"
)

type Bootstrap struct {
	Server *Server `yaml:"server" json:"server"`
	Data   *Data   `yaml:"data" json:"data"`
	Client *Client `yaml:"client" json:"client"`
	Log    *Log    `yaml:"log" json:"log"`
}

type Server struct {
	Http struct {
		Addr    string `yaml:"addr" json:"addr"`
		Timeout string `yaml:"timeout" json:"timeout"`
	} `yaml:"http" json:"http"`
	Grpc struct {
		Addr    string `yaml:"addr" json:"addr"`
		Timeout string `yaml:"timeout" json:"timeout"`
	} `yaml:"grpc" json:"grpc"`
}

type Data struct {
	Database struct {
		Driver string `yaml:"driver" json:"driver"`
		Source string `yaml:"source" json:"source"`
	} `yaml:"database" json:"database"`
	Redis struct {
		Addr         string `yaml:"addr" json:"addr"`
		Password     string `yaml:"password" json:"password"`
		Db           int32  `yaml:"db" json:"db"`
		ReadTimeout  string `yaml:"read_timeout" json:"read_timeout"`
		WriteTimeout string `yaml:"write_timeout" json:"write_timeout"`
	} `yaml:"redis" json:"redis"`
}

type Client struct {
	PaymentService      *PaymentService      `yaml:"payment_service" json:"payment_service"`
	SubscriptionService *SubscriptionService `yaml:"subscription_service" json:"subscription_service"`
}

type PaymentService struct {
	Addr string `yaml:"addr" json:"addr"`
}

type SubscriptionService struct {
	ReturnURL           string `yaml:"return_url" json:"return_url"`
	AutoRenewDaysBefore int    `yaml:"auto_renew_days_before" json:"auto_renew_days_before"`
	ExpiryCheckDays     int    `yaml:"expiry_check_days" json:"expiry_check_days"`
}

type Log struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"`
	Output     string `yaml:"output" json:"output"`
	FilePath   string `yaml:"file_path" json:"file_path"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	Compress   bool   `yaml:"compress" json:"compress"`
}

// Validate validates the configuration
func (b *Bootstrap) Validate() error {
	if b.Server == nil {
		return fmt.Errorf("server configuration is required")
	}
	if b.Server.Http.Addr == "" {
		return fmt.Errorf("server.http.addr is required")
	}
	if b.Server.Grpc.Addr == "" {
		return fmt.Errorf("server.grpc.addr is required")
	}
	if b.Data == nil {
		return fmt.Errorf("data configuration is required")
	}
	if b.Data.Database.Source == "" {
		return fmt.Errorf("data.database.source is required")
	}
	if b.Client == nil {
		return fmt.Errorf("client configuration is required")
	}
	if b.Client.PaymentService == nil || b.Client.PaymentService.Addr == "" {
		return fmt.Errorf("client.payment_service.addr is required")
	}
	if b.Client.SubscriptionService == nil {
		return fmt.Errorf("client.subscription_service configuration is required")
	}
	if b.Log == nil {
		return fmt.Errorf("log configuration is required")
	}
	return nil
}
