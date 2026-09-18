package autoupdate

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v80/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
)

func TestGithubRelease(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		type testCase struct {
			Message       string
			GithubRelease *GithubRelease
			ExpectedError string
		}
		testCases := []testCase{
			{
				Message: "should return nil for a valid GithubRelease using LatestOnly",
				GithubRelease: &GithubRelease{
					Owner:      "test-owner",
					Repository: "test-repo",
					Artifacts:  []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					LatestOnly: true,
				},
				ExpectedError: "",
			},
			{
				Message: "should return nil for a valid GithubRelease using VersionConstraint",
				GithubRelease: &GithubRelease{
					Owner:             "test-owner",
					Repository:        "test-repo",
					Artifacts:         []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionConstraint: ">3.5.10",
				},
				ExpectedError: "",
			},
			{
				Message: "should return error if using both LatestOnly and VersionConstraint",
				GithubRelease: &GithubRelease{
					Owner:             "test-owner",
					Repository:        "test-repo",
					Artifacts:         []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					LatestOnly:        true,
					VersionConstraint: ">3.5.10",
				},
				ExpectedError: "must not specify VersionConstraint when LatestOnly=true",
			},
			{
				Message: "should return error for empty Owner",
				GithubRelease: &GithubRelease{
					Repository: "test-repo",
					Artifacts:  []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
				},
				ExpectedError: "must specify Owner",
			},
			{
				Message: "should return error for empty Repository",
				GithubRelease: &GithubRelease{
					Owner:     "test-owner",
					Artifacts: []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
				},
				ExpectedError: "must specify Repository",
			},
			{
				Message: "should return error for nil Artifacts",
				GithubRelease: &GithubRelease{
					Owner:      "test-owner",
					Repository: "test-repo",
				},
				ExpectedError: "must specify Artifacts",
			},
			{
				Message: "should return error for empty Artifacts",
				GithubRelease: &GithubRelease{
					Owner:      "test-owner",
					Repository: "test-repo",
					Artifacts:  []AutoupdateArtifactRef{},
				},
				ExpectedError: "must specify at least one element for Artifacts",
			},
			{
				Message: "should return error for invalid version constraint",
				GithubRelease: &GithubRelease{
					Owner:             "test-owner",
					Repository:        "test-repo",
					Artifacts:         []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionConstraint: "InvalidVersionConstraint",
				},
				ExpectedError: "invalid VersionConstraint: improper constraint: InvalidVersionConstraint",
			},
			{
				Message: "should return error for invalid version regex",
				GithubRelease: &GithubRelease{
					Owner:        "test-owner",
					Repository:   "test-repo",
					Artifacts:    []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionRegex: "v[asdf[",
				},
				ExpectedError: "invalid VersionRegex: error parsing regexp: missing closing ]: `[asdf[`",
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.Message, func(t *testing.T) {
				err := testCase.GithubRelease.Validate()
				if testCase.ExpectedError == "" {
					assert.Nil(t, err)
				} else {
					assert.EqualError(t, err, testCase.ExpectedError)
				}
			})
		}

		t.Run("should set compiledVersionConstraint", func(t *testing.T) {
			constraintString := ">=1.2.3"
			githubRelease := &GithubRelease{
				Owner:             "test-owner",
				Repository:        "test-repo",
				Artifacts:         []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
				VersionConstraint: constraintString,
			}
			err := githubRelease.Validate()
			assert.NoError(t, err)
			compiledConstraint, err := semver.NewConstraint(constraintString)
			assert.NoError(t, err)
			assert.Equal(t, compiledConstraint, githubRelease.compiledVersionConstraint)
		})

		t.Run("should set compiledVersionRegex", func(t *testing.T) {
			regexString := "v([a-zA-Z0-9]+)"
			githubRelease := &GithubRelease{
				Owner:        "test-owner",
				Repository:   "test-repo",
				Artifacts:    []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
				VersionRegex: regexString,
			}
			err := githubRelease.Validate()
			assert.NoError(t, err)
			assert.Equal(t, regexp.MustCompile(regexString), githubRelease.compiledVersionRegex)
		})
	})

	t.Run("processTagToVersion", func(t *testing.T) {
		type testCase struct {
			Message         string
			GithubRelease   *GithubRelease
			Tag             string
			ExpectedVersion string
			ExpectedError   string
		}
		testCases := []testCase{
			{
				Message: "should not return passed tag if constraint and regex are not defined",
				GithubRelease: &GithubRelease{
					Owner:      "test-owner",
					Repository: "test-repo",
					Artifacts:  []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
				},
				Tag:             "v1.2.3",
				ExpectedVersion: "v1.2.3",
			},
			{
				Message: "should process tag according to regex capture group",
				GithubRelease: &GithubRelease{
					Owner:        "test-owner",
					Repository:   "test-repo",
					Artifacts:    []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionRegex: "^v(.*)$",
				},
				Tag:             "v1.2.3",
				ExpectedVersion: "1.2.3",
			},
			{
				Message: "should return empty version if passed tag does not match regex",
				GithubRelease: &GithubRelease{
					Owner:        "test-owner",
					Repository:   "test-repo",
					Artifacts:    []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionRegex: "v(asdf)",
				},
				Tag:             "v1.2.3",
				ExpectedVersion: "",
			},
			{
				Message: "should return version if passed tag satisfies version constraint",
				GithubRelease: &GithubRelease{
					Owner:             "test-owner",
					Repository:        "test-repo",
					Artifacts:         []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionConstraint: ">=1.0.0",
				},
				Tag:             "v1.2.3",
				ExpectedVersion: "v1.2.3",
			},
			{
				Message: "should return empty version if passed tag does not satisfy version constraint",
				GithubRelease: &GithubRelease{
					Owner:             "test-owner",
					Repository:        "test-repo",
					Artifacts:         []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionConstraint: "<1.0.0",
				},
				Tag:             "v1.2.3",
				ExpectedVersion: "",
			},
			{
				Message: "should return error if version constraint is specified and found version is not valid regex",
				GithubRelease: &GithubRelease{
					Owner:             "test-owner",
					Repository:        "test-repo",
					Artifacts:         []AutoupdateArtifactRef{{SourceArtifact: "k3s-io/k3s"}},
					VersionConstraint: "<1.0.0",
					VersionRegex:      "^v(.*)$",
				},
				Tag:           "v1.2asdf",
				ExpectedError: "error parsing release version: invalid semantic version",
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.Message, func(t *testing.T) {
				err := testCase.GithubRelease.Validate()
				assert.NoError(t, err)
				version, err := testCase.GithubRelease.processTagToVersion(testCase.Tag)
				assert.Equal(t, testCase.ExpectedVersion, version)
				if testCase.ExpectedError != "" {
					assert.EqualError(t, err, testCase.ExpectedError)
				}
			})
		}
	})

	t.Run("getVersionFromLatestRelease", func(t *testing.T) {
		t.Run("should return the tag name for the latest GitHub release", func(t *testing.T) {
			tagname := "v1.0.0"
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetReposReleasesLatestByOwnerByRepo,
					github.RepositoryRelease{
						TagName:    github.Ptr(tagname),
						Draft:      github.Ptr(false),
						Prerelease: github.Ptr(false),
					},
				),
			)

			client := github.NewClient(mockedHTTPClient)
			gr := &GithubRelease{Owner: "kubernetes-sigs", Repository: "metrics-server"}

			latestTag, err := gr.getVersionFromLatestRelease(client)
			assert.NoError(t, err)
			assert.Equal(t, tagname, latestTag, "versions should be equal")
		})

		t.Run("should return error if failed to fetch latest GithubRelease", func(t *testing.T) {
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetReposReleasesLatestByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mock.WriteError(
							w,
							http.StatusInternalServerError,
							"no server is currently available to service your request.",
						)
					}),
				),
			)

			client := github.NewClient(mockedHTTPClient)
			gr := &GithubRelease{Owner: "kubernetes-sigs", Repository: "metrics-server"}

			latestTag, err := gr.getVersionFromLatestRelease(client)
			assert.Equal(t, "", latestTag, "should be an empty string")
			assert.ErrorContains(t, err, "failed to get latest release: ")
		})
	})

	t.Run("getVersionsFromAllReleases", func(t *testing.T) {
		t.Run("should return tag names for all GitHub releases", func(t *testing.T) {
			mockedResponse := []*github.RepositoryRelease{
				{
					TagName: github.Ptr("v0.1.0-rc.1"),
				},
				{
					TagName: github.Ptr("v0.1.1-rc.2"),
				},
				{
					TagName: github.Ptr("v1.0.0-beta"),
				},
				{
					TagName: github.Ptr("v1.0.0"),
				},
			}

			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetReposReleasesByOwnerByRepo,
					mockedResponse,
				),
			)

			client := github.NewClient(mockedHTTPClient)
			gr := &GithubRelease{Owner: "kubernetes-sigs", Repository: "metrics-server"}

			ghTags, err := gr.getVersionsFromAllReleases(client)
			assert.NoError(t, err)
			assert.Equal(t, []string{"v0.1.0-rc.1", "v0.1.1-rc.2", "v1.0.0-beta", "v1.0.0"}, ghTags, "versions should be equal")
		})

		t.Run("should not return draft or pre-release GithubReleases", func(t *testing.T) {
			mockedResponse := []*github.RepositoryRelease{
				{
					TagName:    github.Ptr("v0.1.0-rc.1"),
					Draft:      github.Ptr(true),
					Prerelease: github.Ptr(false),
				},
				{
					TagName:    github.Ptr("v0.1.1-rc.2"),
					Draft:      github.Ptr(true),
					Prerelease: github.Ptr(false),
				},
				{
					TagName:    github.Ptr("v1.0.0-beta"),
					Draft:      github.Ptr(false),
					Prerelease: github.Ptr(true),
				},
				{
					TagName: github.Ptr("v1.0.0"),
				},
			}

			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetReposReleasesByOwnerByRepo,
					mockedResponse,
				),
			)

			client := github.NewClient(mockedHTTPClient)
			gr := &GithubRelease{Owner: "kubernetes-sigs", Repository: "metrics-server"}

			ghTags, err := gr.getVersionsFromAllReleases(client)
			assert.NoError(t, err)
			assert.Equal(t, []string{"v1.0.0"}, ghTags, "versions should be equal")
		})

		t.Run("should return error if failed to fetch GithubReleases", func(t *testing.T) {
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetReposReleasesByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mock.WriteError(
							w,
							http.StatusInternalServerError,
							"no server is currently available to service your request.",
						)
					}),
				),
			)

			client := github.NewClient(mockedHTTPClient)
			gr := &GithubRelease{Owner: "kubernetes-sigs", Repository: "metrics-server"}

			ghTags, err := gr.getVersionsFromAllReleases(client)
			assert.Nil(t, ghTags, "should be nil on error")
			assert.ErrorContains(t, err, "failed to get releases: ")
		})
	})
}
