package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	version := VersionDescriptor{
		Major: 1,
		Minor: 2,
		Patch: 3,
	}

	assert.Equal(t, "1.2.3", version.String())

	Build = "deadbeef"

	assert.Equal(t, "1.2.3 (deadbeef)", version.String())
}
