package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/store"
)

type Config struct {
	NoColor      bool
	Insecure     bool
	Force        bool
	HashAlg      string
	RequireLabel bool

	DBMS     string
	DSN      string
	TraceSQL bool

	err error
}

func NewConfig() *Config {
	return &Config{}
}

func (o *Config) Check() error {
	if o == nil {
		return errors.New("nil config")
	}

	return o.err
}

func (o *Config) Store() *store.Config {
	return &store.Config{
		Insecure:     o.Insecure,
		Force:        o.Force,
		HashAlg:      o.HashAlg,
		RequireLabel: o.RequireLabel,
		Config: db.Config{
			DBMS:     o.DBMS,
			DSN:      o.DSN,
			TraceSQL: o.TraceSQL,
		},
	}
}

func (o *Config) DB() *db.Config {
	return &db.Config{
		DBMS:     o.DBMS,
		DSN:      o.DSN,
		TraceSQL: o.TraceSQL,
	}
}

func (o *Config) Init(path string) {
	v := viper.GetViper()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		wd, err := os.Getwd()
		if err != nil {
			o.err = err
			return
		}

		userConfigDir, err := os.UserConfigDir()
		if err == nil {
			v.AddConfigPath(userConfigDir)
		}
		v.AddConfigPath(wd)
		v.SetConfigType("yaml")
		v.SetConfigName("corim-store.yaml")
	}

	v.SetEnvPrefix("corim_store")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	err := v.ReadInConfig()
	if errors.As(err, &viper.ConfigFileNotFoundError{}) {
		err = nil
	}

	if err != nil {
		o.err = err
		return
	}

	o.NoColor = v.GetBool("no-color")
	o.Insecure = v.GetBool("insecure")
	o.Force = v.GetBool("force")
	o.RequireLabel = v.GetBool("require-label")
	o.DBMS = v.GetString("dbms")
	o.DSN = v.GetString("dsn")
	o.TraceSQL = v.GetBool("trace-sql")

	o.HashAlg = v.GetString("hash-alg")
	if o.HashAlg == "" {
		o.HashAlg = "sha256"
	}
}
