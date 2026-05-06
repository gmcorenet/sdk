package gmcore_mailer

import (
	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	Host      string `yaml:"host" json:"host"`
	Port      int    `yaml:"port" json:"port"`
	Username  string `yaml:"username" json:"username"`
	Password  string `yaml:"password" json:"password"`
	From      string `yaml:"from" json:"from"`
	FromName  string `yaml:"from_name" json:"from_name"`
	ReplyTo   string `yaml:"reply_to" json:"reply_to"`
	Encryption string `yaml:"encryption" json:"encryption"`
}

func LoadConfig(appPath string) (*Config, error) {
	l := gmcore_config.NewLoader[Config](appPath)
	for _, name := range []string{"mailer.yaml", "mailer.yml", "email.yaml", "email.yml"} {
		if cfg, err := l.LoadDefault(name); cfg != nil || err != nil {
			return cfg, err
		}
	}
	return nil, nil
}

func (c *Config) Build() *SMTPMailer {
	return NewSMTPMailer(c.Host, c.Port, c.Username, c.Password)
}
