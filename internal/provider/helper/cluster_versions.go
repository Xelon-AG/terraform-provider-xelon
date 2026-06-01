package helper

import (
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// SortVersions sorts s in place, newest first. Invalid versions are placed
// at the end, ordered lexically among themselves for stability.
func SortVersions[T any](s []T, key func(T) string) {
	type parsed struct {
		keyStr string
		orig   T
		valid  bool
		ver    *semver.Version
	}

	decorated := make([]parsed, len(s))
	for i, v := range s {
		k := key(v)
		decorated[i].orig = v
		decorated[i].keyStr = k
		if ver, err := semver.NewVersion(k); err == nil {
			decorated[i].ver = ver
			decorated[i].valid = true
		}
	}

	slices.SortStableFunc(decorated, func(first, second parsed) int {
		switch {
		case !first.valid && !second.valid:
			return strings.Compare(first.keyStr, second.keyStr)
		case !first.valid:
			return 1
		case !second.valid:
			return -1
		default:
			return second.ver.Compare(first.ver)
		}
	})

	for i, d := range decorated {
		s[i] = d.orig
	}
}
