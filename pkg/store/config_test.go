package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		title string
		dbms  string
		opts  []ConfigOption
		err   string
	}{
		{
			title: "ok multi-opt",
			dbms:  "mysql",
			opts: []ConfigOption{
				OptionMD5,
				OptionTraceSQL,
				OptionInsecure,
				OptionForce,
				OptionRequireLabel,
			},
		},
		{
			title: "ok sha256",
			dbms:  "mysql",
			opts:  []ConfigOption{OptionSHA256},
		},
		{
			title: "ok sha512",
			dbms:  "mysql",
			opts:  []ConfigOption{OptionSHA512},
		},
		{
			title: "invalid DBMS",
			dbms:  "foo",
			err:   "invalid DBMS: foo",
		},
		{
			title: "invalid hash alg",
			dbms:  "mysql",
			opts: []ConfigOption{func(c *Config) {
				c.HashAlg = "bar"
			}},
			err: "invalid hash algorithm: bar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			testConfig := NewConfig(tc.dbms, "", tc.opts...)
			err := testConfig.Validate()

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestConfig_DB(t *testing.T) {
	dbConfig := NewConfig("postgres", "foo").DB()
	assert.Equal(t, "postgres", dbConfig.DBMS)
	assert.Equal(t, "foo", dbConfig.DSN)
	assert.False(t, dbConfig.TraceSQL)

	dbConfig = NewConfig("mysql", "bar", OptionTraceSQL).DB()
	assert.Equal(t, "mysql", dbConfig.DBMS)
	assert.Equal(t, "bar", dbConfig.DSN)
	assert.True(t, dbConfig.TraceSQL)
}
