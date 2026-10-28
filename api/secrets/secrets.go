// Package secrets encrypts the credentials LibreQuenda has to keep and use again
// (a council's SMTP password), so the database never holds them in clear.
// AES-256-GCM with SECRETS_KEY, which never goes into the database.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"librequenda/shared"
)

// keyVersion leads every sealed value, so the key can be rotated later.
const keyVersion byte = 1

var ErrNoKey = errors.New("SECRETS_KEY is not set or is not 32 bytes in hex (openssl rand -hex 32)")
var ErrSealed = errors.New("the sealed value is damaged, belongs elsewhere or used another key")

func aead() (cipher.AEAD, error) {
	var key, err = hex.DecodeString(shared.SECRETS_KEY)
	if err != nil || len(key) != 32 {
		return nil, ErrNoKey
	}
	var block cipher.Block
	if block, err = aes.NewCipher(key); err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Seal encrypts plain for its owner (e.g. the council id): it only opens with the same owner.
func Seal(plain string, owner []byte) ([]byte, error) {
	var gcm, err = aead()
	if err != nil {
		return nil, err
	}
	var sealed = make([]byte, 1+gcm.NonceSize(), 1+gcm.NonceSize()+len(plain)+gcm.Overhead())
	sealed[0] = keyVersion
	if _, err = rand.Read(sealed[1:]); err != nil {
		return nil, err
	}
	return gcm.Seal(sealed, sealed[1:], []byte(plain), owner), nil
}

// Open decrypts what Seal gave for the same owner.
func Open(sealed []byte, owner []byte) (string, error) {
	var gcm, err = aead()
	if err != nil {
		return "", err
	}
	if len(sealed) < 1+gcm.NonceSize() || sealed[0] != keyVersion {
		return "", ErrSealed
	}
	var nonce = sealed[1 : 1+gcm.NonceSize()]
	var plain []byte
	if plain, err = gcm.Open(nil, nonce, sealed[1+gcm.NonceSize():], owner); err != nil {
		return "", ErrSealed
	}
	return string(plain), nil
}
