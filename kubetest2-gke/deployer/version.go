/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/api/container/v1"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

var (
	noneReleaseChannel     = "None"
	rapidReleaseChannel    = "rapid"
	regularReleaseChannel  = "regular"
	stableReleaseChannel   = "stable"
	extendedReleaseChannel = "extended"

	validReleaseChannels = []string{noneReleaseChannel, rapidReleaseChannel, regularReleaseChannel, stableReleaseChannel, extendedReleaseChannel}
)

func validateVersion(version string) error {
	switch version {
	case "latest", "":
		return nil
	default:
		re, err := regexp.Compile(`(\d)\.(\d)+(\.(\d)*(.*))?`)
		if err != nil {
			return err
		}
		if !re.MatchString(version) {
			return fmt.Errorf("unknown version %q", version)
		}
	}
	return nil
}

func validateReleaseChannel(releaseChannel string) error {
	if releaseChannel != "" {
		for _, c := range validReleaseChannels {
			if releaseChannel == c {
				return nil
			}
		}
		return fmt.Errorf("%q is not one of the valid release channels %v", releaseChannel, validReleaseChannels)
	}
	return nil
}

// Resolve the current latest version in the given release channel.
func resolveLatestVersionInChannel(loc, channelName string) (string, error) {
	// Get the server config for the current location.
	cfg, err := getServerConfig(loc)
	if err != nil {
		return "", fmt.Errorf("error getting server config: %w", err)
	}
	for _, channel := range cfg.Channels {
		if strings.EqualFold(channel.Channel, channelName) {
			if len(channel.ValidVersions) == 0 {
				return "", fmt.Errorf("no valid versions for channel %q", channelName)
			}
			return channel.ValidVersions[0], nil
		}
	}

	return "", fmt.Errorf("channel %q does not exist in the server config", channelName)
}

// Resolve the valid release channel for the given cluster version.
func resolveReleaseChannelForClusterVersion(clusterVersion, loc string) (string, error) {
	if clusterVersion == "" || clusterVersion == "latest" {
		// For latest or non cluster version, always use none release channel.
		return noneReleaseChannel, nil
	}

	// Get the server config.
	cfg, err := getServerConfig(loc)
	if err != nil {
		return "", err
	}

	// Look through the versions not associated with a channel.
	for _, v := range cfg.ValidMasterVersions {
		if isClusterVersionMatch(clusterVersion, v) {
			// Use None release channel if there is a match in the valid master versions.
			return noneReleaseChannel, nil
		}
	}

	// Look through all the channels.
	for _, channel := range cfg.Channels {
		for _, v := range channel.ValidVersions {
			if isClusterVersionMatch(clusterVersion, v) {
				return toReleaseChannel(channel.Channel)
			}
		}
	}

	return "", fmt.Errorf("no matched release channel found for cluster version %q", clusterVersion)
}

func toReleaseChannel(channelName string) (string, error) {
	switch channelName {
	case "RAPID":
		return rapidReleaseChannel, nil
	case "REGULAR":
		return regularReleaseChannel, nil
	case "STABLE":
		return stableReleaseChannel, nil
	case "EXTENDED":
		return extendedReleaseChannel, nil
	default:
		return "", fmt.Errorf("selected unknown release channel: %s", channelName)
	}
}

// isClusterVersionMatch compares a cluster version against a match version. The match version can
// be used to match against various patch versions. For example, the match version 1.7 will match
// against versions 1.7.3 and 1.7.10.
func isClusterVersionMatch(match, version string) bool {
	if match == version {
		return true
	}

	matchParts := strings.Split(match, ".")
	// The version is in the format of major.minor.patch-gke.number, here we
	// only need to match the major.minor.patch version.
	if strings.Contains(version, "-") {
		version = strings.Split(version, "-")[0]
	}
	parts := strings.Split(version, ".")

	if len(parts) < len(matchParts) {
		return false
	}

	for i := 0; i < len(matchParts); i++ {
		if matchParts[i] != parts[i] {
			return false
		}
	}
	return true
}

func getServerConfig(loc string) (*container.ServerConfig, error) {
	// List the available versions for each release channel.
	out, err := exec.Output(exec.RawCommand((fmt.Sprintf("gcloud container get-server-config --format=json %s", loc))))
	if err != nil {
		return nil, err
	}

	// Parse the JSON into a struct.
	cfg := &container.ServerConfig{}
	if err := json.Unmarshal(out, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
