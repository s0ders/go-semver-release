// Package gpg provides function to deal with GPG armored keys.
package gpg

import (
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// FromArmored reads an armored keyring buffer and returns the first key pair.
func FromArmored(reader io.Reader) (*openpgp.Entity, error) {
	entities, err := openpgp.ReadArmoredKeyRing(reader)
	if err != nil {
		return nil, err
	}

	return entities[0], nil
}
