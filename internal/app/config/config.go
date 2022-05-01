package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
)

// Провайдеры хранилища
const (
	DBInmem    = "inmem"
	DBPostgres = "postgres"
)

// Значения по умолчанию
const (
	defaultInmemFlushInterval = 10 * time.Second

	defaultDeleteFlushInterval = time.Millisecond

	fileStorageDefault = "localhost.db"
	baseURLDefault     = "http://localhost:8080"
	srvAddrDefault     = ":8080"

	dsnDefault = "host=localhost port=5432 user=postgres password=qwe123 dbname=postgres"

	DefaultCfgFileName = "config.json"
)

// Config определяет базовую конфигурацию сервиса.
type Config struct {
	BaseURL             string        `json:"base_url"`
	SrvAddr             string        `json:"server_address"`
	Secret              string        `json:"secret"`
	DBType              string        `json:"db_type"`
	StorageFileName     string        `json:"file_storage_path"`
	DSN                 string        `json:"database_dsn"`
	EnableHTTPS         bool          `json:"enable_https"`
	InmemFlushInterval  time.Duration `json:"inmem_flush_interval"`
	DeleteFlushInterval time.Duration `json:"delete_flush_interval"`
	TrustedSubnet       string        `json:"trusted_subnet"`
	PprofAddress        string        `json:"pprof_address"`
	GRPCPort            string        `json:"grpc_port"`
}

func (cfg Config) String() string {
	var b strings.Builder
	b.WriteString("baseURL='" + cfg.BaseURL + "'")
	b.WriteString(" srvAddr='" + cfg.SrvAddr + "'")
	b.WriteString(" secret='*****'")
	b.WriteString(" dbType='" + cfg.DBType + "'")
	b.WriteString(" deleteFlushInterval=" + cfg.DeleteFlushInterval.String())
	if cfg.StorageFileName != "" {
		b.WriteString(" fileName='" + cfg.StorageFileName + "'")
	}
	if cfg.InmemFlushInterval != 0 {
		b.WriteString(" inmemFlushInterval=" + cfg.InmemFlushInterval.String())
	}
	if cfg.DSN != "" {
		b.WriteString(" dsn='" + cfg.DSN + "'")
	}
	if cfg.TrustedSubnet != "" {
		b.WriteString(" trustedSubnet=" + cfg.TrustedSubnet)
	}
	if cfg.PprofAddress != "" {
		b.WriteString(" pprofAddress=" + cfg.PprofAddress)
	}
	if cfg.GRPCPort != "" {
		b.WriteString(" gRPCAddress=" + cfg.GRPCPort)
	}
	if cfg.EnableHTTPS {
		b.WriteString(" enableHTTPS: yes")
	} else {
		b.WriteString(" enableHTTPS: no")
	}

	return b.String()
}

// Validate проверяет конфигурацию и выдает ошибку, если обнаруживает пустые поля.
func (cfg Config) Validate() (retErr error) {
	if cfg.BaseURL == "" {
		retErr = multierror.Append(retErr, errors.New("missing base URL"))
	}
	if cfg.SrvAddr == "" {
		retErr = multierror.Append(retErr, errors.New("mising server address"))
	}
	if cfg.DBType != DBInmem && cfg.DBType != DBPostgres {
		retErr = multierror.Append(retErr, errors.New("invalid storage type"))
	}
	if cfg.TrustedSubnet != "" {
		if _, _, err := net.ParseCIDR(cfg.TrustedSubnet); err != nil {
			retErr = multierror.Append(retErr, fmt.Errorf("incorrect subnet: %s", err))
		}
	}

	return
}

type Option func(*Config)

// NewConfig формирует конфигурацию из значений по умолчанию, затем опционально меняет
// поля при помощи функций configOption.
func NewConfig(opts ...Option) Config {
	cfg := Config{
		BaseURL:             baseURLDefault,
		SrvAddr:             srvAddrDefault,
		StorageFileName:     fileStorageDefault,
		InmemFlushInterval:  defaultInmemFlushInterval,
		DeleteFlushInterval: defaultDeleteFlushInterval,
		DSN:                 "", // значения по умолчанию будут внесены функцией newConfig.
		EnableHTTPS:         false,
	}

	for _, fn := range opts {
		fn(&cfg)
	}

	// Если пользователь передал DSN для Postgres, используем Postgres, игнорируя флаги.
	if cfg.DSN != "" {
		cfg.DBType = DBPostgres
	}

	switch cfg.DBType {
	case DBPostgres:
		if cfg.DSN == "" {
			cfg.DSN = dsnDefault
		}
		cfg.StorageFileName = ""
		cfg.InmemFlushInterval = 0
	case "":
		cfg.DBType = DBInmem
		cfg.DSN = ""
	case DBInmem:
		cfg.DSN = ""
	}

	return cfg
}

// WithFile считывает конфигурацию из файла (по умолчанию - config.json).
func WithFile(filename string) Option {
	return func(cfg *Config) {
		log.Printf("open file %s", filename)
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("config: could not read file %s: %s", filename, err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			log.Fatalf("config: could not decode file %s: %s", filename, err)
		}
	}
}

// WithFlags устанавливает значения конфигурации в соответствии с переданными флагами.
func WithFlags(f AppFlags) Option {
	return func(cfg *Config) {
		if f.BaseURL != nil {
			cfg.BaseURL = *f.BaseURL
		}
		if f.SrvAddr != nil {
			cfg.SrvAddr = *f.SrvAddr
		}
		if f.Secret != nil {
			cfg.Secret = *f.Secret
		}
		if f.DBType != nil {
			cfg.DBType = *f.DBType
		}
		if f.StorageFileName != nil {
			cfg.StorageFileName = *f.StorageFileName
		}
		if f.DSN != nil {
			cfg.DSN = *f.DSN
		}
		if f.EnableHTTPS != nil {
			cfg.EnableHTTPS = *f.EnableHTTPS
		}
		if f.DeleteFlushInterval != nil {
			cfg.DeleteFlushInterval = *f.DeleteFlushInterval
		}
		if f.TrustedSubnet != nil {
			cfg.TrustedSubnet = *f.TrustedSubnet
		}
	}
}

// WithEnv задаёт значения конфигурции в соответствии с переменными окруэения.
func WithEnv() Option {
	return func(cfg *Config) {
		env := map[string]*string{
			"BASE_URL":          &cfg.BaseURL,
			"SERVER_ADDRESS":    &cfg.SrvAddr,
			"FILE_STORAGE_PATH": &cfg.StorageFileName,
			"DATABASE_DSN":      &cfg.DSN,
			"HASH_KEY":          &cfg.Secret,
			"TRUSTED_SUBNET":    &cfg.TrustedSubnet,
		}

		for v := range env {
			if envVal, ok := os.LookupEnv(v); ok {
				*env[v] = envVal
			}
		}
		if _, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
			cfg.EnableHTTPS = true
		}
	}
}
