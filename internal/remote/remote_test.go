package remote

import (
	"testing"

	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v4/internal/gittest"
)

func TestRemote_Clone_HappyScenario(t *testing.T) {
	assert := assertion.New(t)

	remote := New("origin", "password")

	testRepository, err := gittest.NewRepository()
	if err != nil {
		t.Fatalf("creating test repository: %s", err)
	}

	testRepository.StartRemoteWithAuth("username", "password")
	defer testRepository.StopRemote()

	clonedRepository, err := remote.Clone(testRepository.RemoteURL)
	if err != nil {
		t.Fatalf("failed to clone repository: %s", err)
	}

	assert.NoError(err)
	assert.NotNil(clonedRepository)
}

func TestRemote_Clone_WrongToken(t *testing.T) {
	assert := assertion.New(t)

	remote := New("origin", "wrong")

	testRepository, err := gittest.NewRepository()
	if err != nil {
		t.Fatalf("creating test repository: %s", err)
	}

	testRepository.StartRemoteWithAuth("username", "password")
	defer testRepository.StopRemote()

	_, err = remote.Clone(testRepository.RemoteURL)

	assert.Error(err)
}
