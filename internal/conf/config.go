package conf

import "fmt"

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
}

type Client struct {
	Payment struct {
		Addr string `yaml:"addr" json:"addr"`
	} `yaml:"payment" json:"payment"`
}

type Log struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	Output string `yaml:"output" json:"output"`
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
	if b.Client.Payment.Addr == "" {
		return fmt.Errorf("client.payment.addr is required")
	}
	if b.Log == nil {
		return fmt.Errorf("log configuration is required")
	}
	return nil
}
