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

package util

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/go-resty/resty/v2"
)

func ParseKubernetesMarker(version string) (string, error) {
	if _, err := semver.ParseTolerant(version); err == nil {
		return version, nil
	}
	if u, err := url.Parse(version); err == nil {
		resp, err := resty.New().R().Get(version)
		if err != nil {
			return "", err
		}

		// Replace the last part of the version URL path with the contents of the URL's body
		// Example:
		// https://storage.googleapis.com/k8s-release-dev/ci/latest.txt -> v1.21.0-beta.1.112+576aa2d2470b28%0A
		// becomes https://storage.googleapis.com/k8s-release-dev/ci/v1.21.0-beta.1.112+576aa2d2470b28%0A
		pathParts := strings.Split(u.Path, "/")
		pathParts[len(pathParts)-1] = resp.String()
		u.Path = strings.Join(pathParts, "/")
		return strings.TrimSpace(u.String()), nil
	}
	return "", fmt.Errorf("unexpected kubernetes version: %v", version)
}

// PseudoUniqueSubstring returns a substring of a UUID
// that can be reasonably used in resource names
// where length is constrained
// e.g https://cloud.google.com/compute/docs/naming-resources
// but still retain as much uniqueness as possible
// also easily lets us tie it back to a run
func PseudoUniqueSubstring(uuid string) string {
	// both KUBETEST2_RUN_ID and PROW_JOB_ID uuids are generated
	// following RFC 4122 https://tools.ietf.org/html/rfc4122
	// e.g. 09a2565a-7ac6-11eb-a603-2218f636630c
	// extract the first 13 characters (09a2565a-7ac6) as they are the ones that depend on
	// timestamp and has the best avalanche effect (https://en.wikipedia.org/wiki/Avalanche_effect)
	// as compared to the other bytes
	// 13 characters is also <= the no. of character being used previously
	const maxResourceNamePrefixLength = 13
	if len(uuid) <= maxResourceNamePrefixLength {
		return uuid
	}
	return uuid[:maxResourceNamePrefixLength]
}
