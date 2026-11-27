package gpg

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGPG_FromArmored(t *testing.T) {
	want, err := openpgp.NewEntity("John Doe", "", "john.doe@example.com", &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA})
	require.NoError(t, err)

	var armored bytes.Buffer
	armorWriter, err := armor.Encode(&armored, openpgp.PrivateKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, want.SerializePrivate(armorWriter, nil))
	require.NoError(t, armorWriter.Close())

	got, err := FromArmored(&armored)
	require.NoError(t, err)

	assert.Equal(t, want.PrimaryKey.Fingerprint, got.PrimaryKey.Fingerprint, "public keys fingerprints should be equal")
	assert.Equal(t, want.PrivateKey.Fingerprint, got.PrivateKey.Fingerprint, "private keys fingerprints should be equal")
}

func TestGPG_FromArmored_Invalid(t *testing.T) {
	for name, content := range map[string]string{
		"empty":       "",
		"not armored": "this is not a key",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := FromArmored(strings.NewReader(content))
			assert.Error(t, err)
		})
	}
}
