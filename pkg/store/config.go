package store

import (
	"fmt"
	"slices"

	"github.com/veraison/corim-store/pkg/db"
)

// Config contains configuration for the Store.
type Config struct {
	db.Config

	// HashAlg specifies the hashing algorithm used for computing manifest
	// digests. This must be either md5, sha256, or sha512.
	HashAlg string
	// Insecure indicates whether insecure opearations are permitted.
	Insecure bool
	// Force, when set, allows potentially unsafe operations such as
	// overwriting existing values.
	Force bool
	// RequireLabel indicates whether a label must be specified when adding
	// or looking up values from the Store.
	RequireLabel bool
}

func NewConfig(dbms, dsn string, options ...ConfigOption) *Config {
	ret := &Config{
		Config: db.Config{
			DBMS:     dbms,
			DSN:      dsn,
			TraceSQL: false,
		},
		HashAlg:      "sha256",
		RequireLabel: false,
		Insecure:     false,
		Force:        false,
	}

	ret.WithOptions(options...)

	return ret
}

func (o *Config) WithOptions(options ...ConfigOption) *Config {
	for _, opt := range options {
		opt(o)
	}

	return o
}

func (o *Config) Validate() error {
	if !slices.Contains([]string{
		"sqlite", "sqlite3", "mysql", "mariadb", "postgres", "pq", "pgx",
	}, o.DBMS) {
		return fmt.Errorf("invalid DBMS: %s", o.DBMS)
	}

	if !slices.Contains([]string{
		"md5", "MD5", "sha256", "SHA256", "sha512", "SHA512",
	}, o.HashAlg) {
		return fmt.Errorf("invalid hash algorithm: %s", o.HashAlg)
	}

	return nil
}

func (o *Config) DB() *db.Config {
	return &o.Config
}

type ConfigOption func(c *Config)

func OptionSHA256(c *Config) {
	c.HashAlg = "sha256"
}

func OptionSHA512(c *Config) {
	c.HashAlg = "sha512"
}

func OptionMD5(c *Config) {
	c.HashAlg = "md5"
}

func OptionTraceSQL(c *Config) {
	c.TraceSQL = true
}

func OptionInsecure(c *Config) {
	c.Insecure = true
}

func OptionForce(c *Config) {
	c.Force = true
}

func OptionRequireLabel(c *Config) {
	c.RequireLabel = true
}
