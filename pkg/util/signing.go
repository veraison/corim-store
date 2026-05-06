package util

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
	"github.com/veraison/go-cose"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrBadCert     = errors.New("invalid X.509 cert")
)

// KeyStoreEntry encapsulates a key obtained from a KeyStore that may be used
// to verify signatures on CoRIMs going into the store.
type KeyStoreEntry interface {
	// PublicKey returns crypto.PublicKey extracted from the contained key
	PublicKey() crypto.PublicKey
	// Authority returns a *comid.CryptoKey encapsulating the contained key
	Authority() *comid.CryptoKey
}

// KeyStore is a repository of keys that may be used to verify CoRIM
// signatures.
type KeyStore interface {
	// Get matches the provided CoRIM to an entry in the store and
	// returns that entry.
	Get(sc *corim.SignedCorim) (KeyStoreEntry, error)
}

// CompositeKeyStore wraps multiple other KeyStore's. It returns the first
// matching entry form the contained store. The stores are checked in the order
// they were added.
type CompositeKeyStore struct {
	stores []KeyStore
}

// NewCompositeKeyStore creates a new CompositeKeyStore containing the provided
// stores.
func NewCompositeKeyStore(stores ...KeyStore) *CompositeKeyStore {
	return &CompositeKeyStore{stores}
}

// Add the provided store to the CompositeKeyStore. The store is added at the
// end of the list, making it lowest priority when looking for matches.
func (o *CompositeKeyStore) Add(store KeyStore) *CompositeKeyStore {
	o.stores = append(o.stores, store)
	return o
}

// Insert the store at the front of the list of contained stores, making it
// highest priority when looking for matches
func (o *CompositeKeyStore) Insert(store KeyStore) *CompositeKeyStore {
	o.stores = append([]KeyStore{store}, o.stores...)
	return o
}

func (o *CompositeKeyStore) Get(sc *corim.SignedCorim) (KeyStoreEntry, error) {
	for _, store := range o.stores {
		entry, err := store.Get(sc)
		if err == nil {
			return entry, nil
		} else if !errors.Is(err, ErrKeyNotFound) {
			// coverage:ignore
			return nil, err
		}
	}

	return nil, ErrKeyNotFound
}

// KeyEntry wraps a crypto.PublicKey and a *comid.CryptoKey obtained from the
// same underying key, and exposes them via the KeyStoreEntry interface.
type KeyEntry struct {
	key  crypto.PublicKey
	auth *comid.CryptoKey
}

func (o *KeyEntry) PublicKey() crypto.PublicKey {
	return o.key
}

func (o *KeyEntry) Authority() *comid.CryptoKey {
	return o.auth
}

// SingleKeyStore wraps a single KeyEntry and matches it to every CoRIM it is
// given.
type SingleKeyStore struct {
	entry KeyEntry
}

func (o *SingleKeyStore) Get(*corim.SignedCorim) (KeyStoreEntry, error) {
	return &o.entry, nil
}

// KeyStoreFromPublicKey returns a *SingleKeyStore that contains an entry for
// the specified crypto.PublicKey.
func KeyStoreFromPublicKey(pub crypto.PublicKey) (*SingleKeyStore, error) {
	entry, err := KeyEntryFromPublicKey(pub)
	if err != nil {
		return nil, err
	}

	return &SingleKeyStore{*entry}, nil
}

// KeyEntryFromPublicKey returns a KeyEntry for the specified crypto.PublicKey.
func KeyEntryFromPublicKey(pub crypto.PublicKey) (*KeyEntry, error) {
	coseKey, err := cose.NewKeyFromPublic(pub)
	if err != nil {
		return nil, err
	}

	keyBytes, err := coseKey.MarshalCBOR()
	if err != nil {
		// coverage:ignore
		return nil, err
	}

	return &KeyEntry{pub, comid.MustNewCOSEKey(keyBytes)}, nil
}

// KeyStoreFromJWKPath returns a *SingleKeyStore that contains an entry for the
// JWK at the specified path.
func KeyStoreFromJWKPath(path string) (*SingleKeyStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return KeyStoreFromJWKBytes(data)
}

// KeyStoreFromJWKBytes returns a *SingleKeyStore that contains an entry for the
// JWK in the provided buffer.
func KeyStoreFromJWKBytes(data []byte) (*SingleKeyStore, error) {
	key, err := jwk.ParseKey(data)
	if err != nil {
		return nil, err
	}

	jpub, err := key.PublicKey()
	if err != nil {
		// coverage:ignore
		return nil, err
	}

	var pub crypto.PublicKey
	if err := jpub.Raw(&pub); err != nil {
		// coverage:ignore
		return nil, err
	}

	return KeyStoreFromPublicKey(pub)
}

// KeyStoreFromPEMPath returns a *SingleKeyStore that contains an entry for the
// PEM PUBLIC KEY block at the specified path.
func KeyStoreFromPEMPath(path string) (*SingleKeyStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return KeyStoreFromPEMBytes(data)
}

// KeyStoreFromPEMBytes returns a *SingleKeyStore that contains an entry for the
// PEM PUBLIC KEY block in the provided buffer.
func KeyStoreFromPEMBytes(data []byte) (*SingleKeyStore, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	var pub crypto.PublicKey
	var err error
	switch block.Type {
	case "PUBLIC KEY":
		pub, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}

	return KeyStoreFromPublicKey(pub)
}

// X5ChainKeyStore produces a KeyEntry for the key extracted from the CoRIM's
// x5chain header, if one is set. The x5chain is verified beforehand.
type X5ChainKeyStore struct {
	rootCerts *x509.CertPool
}

// NewX5ChainKeyStore returns a new X5ChainKeyStore with the specified pool.
func NewX5ChainKeyStore(pool *x509.CertPool) *X5ChainKeyStore {
	if pool == nil {
		pool = x509.NewCertPool()
	}

	return &X5ChainKeyStore{pool}
}

// NewX5ChainKeyStoreWithSystemCerts returns a new X5ChainKeyStore whose root certs are
// populated from the system certificate pool.
func NewX5ChainKeyStoreWithSystemCerts() (*X5ChainKeyStore, error) {
	ret := &X5ChainKeyStore{}

	var err error
	ret.rootCerts, err = x509.SystemCertPool()
	if err != nil {
		// coverage:ignore
		return nil, err
	}

	return ret, nil
}

// SetCertPool replaces the internal root cert pool with the provided one, and
// returns the old pool.
func (o *X5ChainKeyStore) SetCertPool(pool *x509.CertPool) *x509.CertPool {
	old := o.rootCerts
	o.rootCerts = pool
	return old
}

// AddCertFromPath parses a certificate from the specified paths, attempting
// to guess the format. ErrBadCert is returned if the format cannot be
// established.
func (o *X5ChainKeyStore) AddCertFromPath(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return o.AddCertFromBytes(bytes)
}

// AddCertFromBytes parses a certificate from the specified buffers, attempting
// to guess the format. ErrBadCert is returned if the format cannot be
// established.
func (o *X5ChainKeyStore) AddCertFromBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return fmt.Errorf("%w: empty", ErrBadCert)
	}

	if bytes[0] == 0x30 {
		return o.AddCertFromDERBytes(bytes)
	}

	trimmed := strings.TrimSpace(string(bytes))
	if strings.HasPrefix(trimmed, "-----BEGIN CERTIFICATE-----") {
		return o.AddCertFromPEMBytes(bytes)
	}

	return ErrBadCert
}

// AddCertFromPEMPath parses a certificate from the file at the specified path
// and adds it to its root certs.
func (o *X5ChainKeyStore) AddCertFromPEMPath(path string) error {
	pem, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return o.AddCertFromPEMBytes(pem)
}

// AddCertFromPEMBytes parses a certificate from the specified buffer and adds
// it to its root certs.
func (o *X5ChainKeyStore) AddCertFromPEMBytes(pem []byte) error {
	if !o.rootCerts.AppendCertsFromPEM(pem) {
		return ErrBadCert
	}

	return nil
}

// AddCertFromPEMPath parses a certificate from the file at the specified path
// and adds it to its root certs.
func (o *X5ChainKeyStore) AddCertFromDERPath(path string) error {
	der, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return o.AddCertFromDERBytes(der)
}

// AddCertFromDERBytes parses a certificate from the specified buffer and adds
// it to its root certs.
func (o *X5ChainKeyStore) AddCertFromDERBytes(der []byte) error {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrBadCert, err)
	}

	o.rootCerts.AddCert(cert)

	return nil
}

func (o *X5ChainKeyStore) Get(signed *corim.SignedCorim) (KeyStoreEntry, error) {
	if signed.SigningCert == nil {
		return nil, ErrKeyNotFound
	}

	intermediateCertPool := x509.NewCertPool()
	for _, cert := range signed.IntermediateCerts {
		intermediateCertPool.AddCert(cert)
	}

	verifyOpts := x509.VerifyOptions{
		Roots:         o.rootCerts,
		Intermediates: intermediateCertPool,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	_, err := signed.SigningCert.Verify(verifyOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: cert chain verification failed: %w", ErrKeyNotFound, err)
	}

	return KeyEntryFromPublicKey(signed.SigningCert.PublicKey)
}
