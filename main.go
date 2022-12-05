package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Regular expression to match valid semantic version number
var semverRegex = regexp.MustCompile("v[0-9]+.[0-9]+.[0-9]+")

type Semver struct {
	Major int
	Minor int
	Patch int
}

func (s *Semver) IncrPatch() {
	s.Patch++
}

func (s *Semver) IncrMinor() {
	s.Patch = 0
	s.Minor++
}

func (s *Semver) IncrMajor() {
	s.Patch = 0
	s.Minor = 0
	s.Patch++
}

func (s Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", s.Major, s.Minor, s.Patch)
}

// NewSemver returns a semver struct corresponding to
// the tag used as an input.
func NewSemver(tag *object.Tag) (*Semver, error) {

	semver := strings.Replace(tag.Name, "v", "", 1)
	components := strings.Split(semver, ".")

	if len(components) != 3 {
		return nil, errors.New("Invalid semantic version number")
	}

	major, err := strconv.Atoi(components[0])
	failOnError(err)
	minor, err := strconv.Atoi(components[1])
	failOnError(err)
	patch, err := strconv.Atoi(components[2])
	failOnError(err)

	return &Semver{major, minor, patch}, nil
}

// compareSemver returns an integer representing
// which of the two versions v1 or v2 passed is
// the most recent. 1 meaning v1 is the most recent,
// -1 that it is v2 and 0 that they are equal.
func CompareSemver(v1, v2 Semver) int {
	switch {
	case v1.Major > v2.Major:
		return 1
	case v1.Major < v2.Major:
		return -1
	case v1.Minor > v2.Minor:
		return 1
	case v1.Minor < v2.Minor:
		return -1
	case v1.Patch > v2.Patch:
		return 1
	case v1.Patch < v2.Patch:
		return -1
	default:
		return 0
	}
}

func main() {
	gitUrl := flag.String("url", "", "The Git repository to work on")
	flag.Parse()

	if *gitUrl == "" {
		log.Fatalf("--url cannot be empty\n")
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: *gitUrl,
	})
	failOnError(err)

	// Fetch all tags from Git repository
	tagrefs, err := r.TagObjects()
	failOnError(err)

	// Stores all tags matching a semver
	semverTags := make([]*object.Tag, 0)

	var latestSemverTag *object.Tag

	// Fetch all tags matching a semver
	tagrefs.ForEach(func(tag *object.Tag) error {
		if semverRegex.MatchString(string(tag.Name)) {
			semverTags = append(semverTags, tag)
		}
		return nil
	})

	if len(semverTags) < 1 {
		// TODO: create and push annotated tag v0.0.0
		log.Fatalf("No tags on repository")
	} else if len(semverTags) < 2 {
		latestSemverTag = semverTags[0]
	} else {
		for i := 0; i < len(semverTags)-1; i++ {
			v1, err := NewSemver(semverTags[i])
			failOnError(err)
			v2, err := NewSemver(semverTags[i+1])
			failOnError(err)
	
			comparison := CompareSemver(*v1, *v2)
	
			switch comparison {
			case 1:
				latestSemverTag = semverTags[i]
			case -1:
				latestSemverTag = semverTags[i+1]
			default:
				latestSemverTag = semverTags[i]
			}
		}
	}

	commitHistory, err := r.Log(&git.LogOptions{Since: &latestSemverTag.Tagger.When})
	failOnError(err)

	semverToApply, err := NewSemver(latestSemverTag)
	failOnError(err)
	
	// ... just iterates over the commits, printing it
	err = commitHistory.ForEach(func(c *object.Commit) error {
		
		switch {
			case regexp.Match()
		}



		return nil
	})
}

func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}
