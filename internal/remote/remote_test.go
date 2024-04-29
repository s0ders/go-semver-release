package remote

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/go-git/go-git/v5/config"

	"github.com/s0ders/go-semver-release/v2/internal/semver"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRemote_New(t *testing.T) {
	assert := assert.New(t)

	fakeLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	token := ""
	remoteName := "origin"

	expected := Remote{
		logger:     fakeLogger,
		remoteName: remoteName,
		auth: &http.BasicAuth{
			Username: "go-semver-release",
			Password: token,
		},
	}

	actual := New(fakeLogger, token, remoteName)

	assert.Equal(expected, actual)
}

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Push(o *git.PushOptions) error {
	args := m.Called(o)
	return args.Error(0)
}

func TestRemote_PushTagToRemote(t *testing.T) {
	assert := assert.New(t)

	fakeLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	token := ""
	remoteName := "origin"

	remote := New(fakeLogger, token, remoteName)
	version, err := semver.New(1, 0, 0, "")
	assert.NoError(err, "failed to create semver")

	po := &git.PushOptions{
		Auth:       remote.auth,
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		RemoteName: remote.remoteName,
	}

	mockRepo := new(MockRepository)
	mockRepo.On("Push", po).Return(nil)

	err = remote.PushTagToRemote(mockRepo, version)
	assert.NoError(err, "failed to push tag to remote")
}

func TestRemote_PushTagToRemoteFailure(t *testing.T) {
	assert := assert.New(t)

	fakeLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	token := ""
	remoteName := "origin"

	remote := New(fakeLogger, token, remoteName)
	version, err := semver.New(1, 0, 0, "")
	assert.NoError(err, "failed to create semver")

	po := &git.PushOptions{
		Auth:       remote.auth,
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		RemoteName: remote.remoteName,
	}

	mockRepo := new(MockRepository)
	mockRepo.On("Push", po).Return(fmt.Errorf("something went wrong"))

	err = remote.PushTagToRemote(mockRepo, version)
	assert.Error(err, "should have failed pushing tag to remote")
}
