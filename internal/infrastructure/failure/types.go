// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package failure

type (
	boundaryError func() boundaryDetails

	boundaryDetails = struct {
		category error
		cause    error
		message  string
	}

	diagnostic interface{ Error() string }

	identityMatcher interface{ Is(target error) bool }

	categoryMatcher interface {
		diagnostic
		identityMatcher
	}
)
