//go:build test

package store

import (
	_ "embed"
)

var (
	//go:embed  fixtures/digests.yaml
	digestsFixture []byte

	//go:embed  fixtures/integrity_registers.yaml
	integrityRegistersFixture []byte

	//go:embed  fixtures/flags.yaml
	flagsFixture []byte

	//go:embed  fixtures/measurement_values.yaml
	measurementValuesFixture []byte

	//go:embed  fixtures/cryptokeys.yaml
	cryptoKeysFixture []byte

	//go:embed  fixtures/measurements.yaml
	measurementsFixture []byte

	//go:embed  fixtures/locators.yaml
	locatorsFixture []byte

	//go:embed  fixtures/entities.yaml
	entitiesFixture []byte

	//go:embed  fixtures/roles.yaml
	rolesFixture []byte

	//go:embed  fixtures/manifests.yaml
	manifestsFixture []byte

	//go:embed  fixtures/linked_tags.yaml
	linkedTagsFixture []byte

	//go:embed  fixtures/module_tags.yaml
	moduleTagsFixture []byte

	//go:embed  fixtures/environments.yaml
	environmentsFixture []byte

	//go:embed  fixtures/triples.yaml
	triplesFixture []byte
)
