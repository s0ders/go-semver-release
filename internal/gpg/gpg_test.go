package gpg

import (
	"bytes"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGPG_FromArmored(t *testing.T) {
	assert := assert.New(t)
	// Creates new armored key file
	dir, err := os.MkdirTemp("", "gpg-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %s", err)
	}

	defer func() {
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("failed to remove temp directory: %s", err)
		}
	}()

	keyFilePath := filepath.Join(dir, "key.asc")
	armoredKeyFile, err := os.Create(keyFilePath)
	if err != nil {
		t.Fatalf("failed to create armored key file: %s", err)
	}
	defer func() {
		err = armoredKeyFile.Close()
		if err != nil {
			t.Fatalf("failed to close armored key file: %s", err)
		}
	}()

	opts := &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA, RSABits: 1024}
	expectedEntity, err := openpgp.NewEntity("John Doe", "", "john.doe@example.com", opts)
	if err != nil {
		t.Fatalf("entity creation failed: %s", err)
	}

	armorWriter, err := armor.Encode(armoredKeyFile, openpgp.PrivateKeyType, nil)
	if err != nil {
		t.Fatalf("armor encoding failed: %s", err)
	}

	if err = expectedEntity.SerializePrivate(armorWriter, nil); err != nil {
		t.Fatalf("serialization failed: %s", err)
	}

	err = armorWriter.Close()
	if err != nil {
		t.Fatalf("failed to close armor writer: %s", err)
	}

	// Reads from armored keyring and produce a new entity
	readFileBuf, err := os.ReadFile(keyFilePath)
	if err != nil {
		t.Fatalf("failed to read %s: %s", keyFilePath, err)
	}

	actualEntity, err := FromArmored(bytes.NewReader(readFileBuf), nil)
	if err != nil {
		t.Fatalf("failed to read from armored: %s", err)
	}

	assert.Equal(expectedEntity.PrimaryKey.Fingerprint, actualEntity.PrimaryKey.Fingerprint, "public keys fingerprints should be equal")

	assert.Equal(expectedEntity.PrivateKey.Fingerprint, actualEntity.PrivateKey.Fingerprint, "private keys fingerprints should be equal")
}

func TestGPG_FromArmoredEmptyReader(t *testing.T) {
	assert := assert.New(t)

	reader := strings.NewReader("")

	_, err := FromArmored(reader, nil)
	t.Log(err)

	assert.Error(err, "should have failed trying to read empty reader")
}

func TestGPG_FromArmoredPassphrase(t *testing.T) {
	assert := assert.New(t)

	dir, err := os.MkdirTemp("", "gpg-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %s", err)
	}

	defer func() {
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("failed to remove temp directory: %s", err)
		}
	}()

	keyFilePath := filepath.Join(dir, "key.asc")
	armoredKeyFile, err := os.Create(keyFilePath)
	if err != nil {
		t.Fatalf("failed to create armored key file: %s", err)
	}
	defer func() {
		err = armoredKeyFile.Close()
		if err != nil {
			t.Fatalf("failed to close armored key file: %s", err)
		}
	}()

	packetOpts := &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA, RSABits: 1024}
	expectedEntity, err := openpgp.NewEntity("John Doe", "", "john.doe@example.com", packetOpts)
	if err != nil {
		t.Fatalf("entity creation failed: %s", err)
	}

	passphrase := "secret"

	err = expectedEntity.PrivateKey.Encrypt([]byte(passphrase))
	if err != nil {
		t.Fatalf("failed to encrypt private key: %s", err)
	}

	armorWriter, err := armor.Encode(armoredKeyFile, openpgp.PrivateKeyType, nil)
	if err != nil {
		t.Fatalf("armor encoding failed: %s", err)
	}

	if err = expectedEntity.SerializePrivate(armorWriter, nil); err != nil {
		t.Fatalf("serialization failed: %s", err)
	}

	err = armorWriter.Close()
	if err != nil {
		t.Fatalf("failed to close armor writer: %s", err)
	}

	// Reads from armored keyring and produce a new entity
	readFileBuf, err := os.ReadFile(keyFilePath)
	if err != nil {
		t.Fatalf("failed to read %s: %s", keyFilePath, err)
	}

	fromArmoredOpts := &Options{
		Passphrase: passphrase,
	}

	actualEntity, err := FromArmored(bytes.NewReader(readFileBuf), fromArmoredOpts)
	if err != nil {
		t.Fatalf("failed to read from armored: %s", err)
	}

	assert.Equal(expectedEntity.PrimaryKey.Fingerprint, actualEntity.PrimaryKey.Fingerprint, "public keys fingerprints should be equal")

	assert.Equal(expectedEntity.PrivateKey.Fingerprint, actualEntity.PrivateKey.Fingerprint, "private keys fingerprints should be equal")
}
