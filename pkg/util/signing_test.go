package util

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
)

func TestKeyStoreFromJWKPath(t *testing.T) {
	ks, err := KeyStoreFromJWKPath("../../sample/corim/key.pub.jwk")
	assert.NoError(t, err)

	entry, err := ks.Get(nil)
	assert.NoError(t, err)

	_, ok := entry.PublicKey().(*ecdsa.PublicKey)
	assert.True(t, ok)

	auth := entry.Authority()
	assert.NotNil(t, auth)

	_, err = KeyStoreFromJWKPath("does not exist")
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestKeyStoreFromJWKBytes_bad(t *testing.T) {
	_, err := KeyStoreFromJWKBytes([]byte(`{ "kty": "RSA", "n": "deadbeef", "e": "AQAB" }`))
	assert.ErrorContains(t, err, "invalid public key")

	_, err = KeyStoreFromJWKBytes([]byte(`}`))
	assert.ErrorContains(t, err, "invalid character '}'")

	_, err = KeyStoreFromJWKBytes([]byte(`{ "kty": "EC", "crv": "P-256", "d": "deadbeef" }`))
	assert.ErrorContains(t, err, "required field x is missing")
}

func TestKeyStoreFromPEMPath(t *testing.T) {
	ks, err := KeyStoreFromPEMPath("../../sample/corim/key.pub.pem")
	assert.NoError(t, err)

	entry, err := ks.Get(nil)
	assert.NoError(t, err)

	_, ok := entry.PublicKey().(*ecdsa.PublicKey)
	assert.True(t, ok)

	auth := entry.Authority()
	assert.NotNil(t, auth)

	_, err = KeyStoreFromPEMPath("does not exist")
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestKeyStoreFromPEMBytes_bad(t *testing.T) {
	_, err := KeyStoreFromPEMBytes([]byte("bad"))
	assert.ErrorContains(t, err, "failed to parse PEM block")

	_, err = KeyStoreFromPEMBytes([]byte("-----BEGIN BAD-----\nYmFkCg==\n-----END BAD-----"))
	assert.ErrorContains(t, err, "unsupported PEM block type")

	_, err = KeyStoreFromPEMBytes([]byte("-----BEGIN PUBLIC KEY-----\nYmFkCg==\n-----END PUBLIC KEY-----"))
	assert.ErrorContains(t, err, "asn1: structure error")
}

type fakeEntry struct {
	val string
}

func (o *fakeEntry) PublicKey() crypto.PublicKey {
	return o.val
}

func (o *fakeEntry) Authority() *comid.CryptoKey {
	return nil
}

type fakeKeyStore struct {
	entry fakeEntry
}

func (o *fakeKeyStore) Get(*corim.SignedCorim) (KeyStoreEntry, error) {
	return &o.entry, nil
}

type errKeyStore struct{}

func (o *errKeyStore) Get(*corim.SignedCorim) (KeyStoreEntry, error) {
	return nil, ErrKeyNotFound
}

func TestCompositeKeyStore(t *testing.T) {
	ks := NewCompositeKeyStore(
		&errKeyStore{},
	)

	_, err := ks.Get(nil)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	ks.Add(&fakeKeyStore{fakeEntry{"foo"}}).
		Add(&fakeKeyStore{fakeEntry{"bar"}})

	entry, err := ks.Get(nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "foo", entry.PublicKey())

	ks.Insert(&fakeKeyStore{fakeEntry{"baz"}})

	entry, err = ks.Get(nil)
	assert.NoError(t, err)
	assert.EqualValues(t, "baz", entry.PublicKey())
}

func TestX5ChainKeyStore(t *testing.T) {
	signed := readCoRIM(t, "../../sample/corim/signed-cca-ref-plat.cose")
	signedWithCert := readCoRIM(t, "../../sample/corim/signed-with-cert-cca-ref-plat.cose")

	ks, err := NewX5ChainKeyStoreWithSystemCerts()
	assert.NoError(t, err)

	_, err = ks.Get(signedWithCert)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	err = ks.AddCertFromPEMPath("../../sample/corim/certs/leaf.cert.pem")
	assert.NoError(t, err)

	ke, err := ks.Get(signedWithCert)
	assert.NoError(t, err)
	assert.NoError(t, signed.Verify(ke.PublicKey()))

	_, err = ks.Get(signed)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	err = ks.AddCertFromDERPath("../../sample/corim/certs/leaf.cert.der")
	assert.NoError(t, err)

	err = ks.AddCertFromDERPath("invalid")
	assert.ErrorContains(t, err, "no such file")

	err = ks.AddCertFromPEMPath("invalid")
	assert.ErrorContains(t, err, "no such file")

	err = ks.AddCertFromPEMPath("../../sample/corim/key.pub.pem")
	assert.ErrorIs(t, err, ErrBadCert)

	err = ks.AddCertFromPath("../../sample/corim/certs/leaf.cert.pem")
	assert.NoError(t, err)

	err = ks.AddCertFromPath("../../sample/corim/certs/leaf.cert.der")
	assert.NoError(t, err)

	err = ks.AddCertFromPath("../../sample/corim/key.pub.pem")
	assert.ErrorIs(t, err, ErrBadCert)

	err = ks.AddCertFromPath("invalid")
	assert.ErrorContains(t, err, "no such file")

	err = ks.AddCertFromBytes(nil)
	assert.ErrorIs(t, err, ErrBadCert)
	assert.ErrorContains(t, err, "empty")

	err = ks.AddCertFromDERBytes(nil)
	assert.ErrorIs(t, err, ErrBadCert)
}

func TestX5ChainKeyStore_SetCertPool(t *testing.T) {
	sysPool, err := x509.SystemCertPool()
	assert.NoError(t, err)

	ks := NewX5ChainKeyStore(nil)
	ks.SetCertPool(sysPool)
	assert.Equal(t, sysPool, ks.rootCerts)
}

func readCoRIM(t *testing.T, path string) *corim.SignedCorim {
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	signed, err := corim.UnmarshalAndValidateSignedCorimFromCBOR(buf)
	if err != nil {
		t.Fatal(err)
	}

	return signed
}
