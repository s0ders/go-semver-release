// Package gpg provides function to deal with GPG armored keys.
package gpg

import (
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
)

type Options struct {
	Passphrase string
}

// FromArmored reads an armored keyring buffer and returns the first key pair.
func FromArmored(reader io.Reader, opts *Options) (*openpgp.Entity, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(reader)
	if err != nil {
		return nil, err
	}

	entity := entityList[0]

	if opts != nil && opts.Passphrase != "" {
		err = entity.PrivateKey.Decrypt([]byte(opts.Passphrase))
		if err != nil {
			return nil, err
		}
	}

	return entity, nil
}
