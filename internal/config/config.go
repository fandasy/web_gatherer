package config

import (
	"io"
	"os"
	"project/pkg/e"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Telegram  *Telegram  `yaml:"telegram"`
	WebServer *WebServer `yaml:"web_server"`
	VkApi     *VkApi     `yaml:"vk_api"`
	Slog      *Slog      `yaml:"slog"`
	MPath     string     `yaml:"migrations_path"`
	Storage   *DB        `yaml:"storage"`
	Files     *Files     `yaml:"file_storage"`
	Redis     *Redis     `yaml:"redis"`
}

type Telegram struct {
	Host    string `yaml:"host"`
	Token   string `yaml:"token"`
	Timeout int    `yaml:"update_timeout"`
}

type WebServer struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type VkApi struct {
	Token string `yaml:"token"`
}

type Slog struct {
	Env    string `yaml:"env"`
	Output string `yaml:"output"`
}

type DB struct {
	DBHost     string `yaml:"DB_HOST"`
	DBPort     int    `yaml:"DB_PORT"`
	DBName     string `yaml:"DB_NAME"`
	DBUser     string `yaml:"DB_USER"`
	DBPassword string `yaml:"DB_PASSWORD"`
}

type Files struct {
	Addr   string `yaml:"addr"`
	KeyID  string `yaml:"key_id"`
	Secret string `yaml:"secret_key"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func Load(path string) (*Config, error) {
	const fn = "config.Load"

	var cfg Config

	file, err := os.Open(path)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, e.Wrap(fn, err)
	}

	return &cfg, nil
}
