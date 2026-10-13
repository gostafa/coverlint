// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package failure

import (
	"errors"
)

// Translate retains the diagnostic and stable category while hiding concrete causes.
// Cancellation causes and standard identities remain matchable with [errors.Is].
func Translate(cause, category error) error {
	if cause == nil {
		return nil
	}

	details := boundaryDetails{message: cause.Error(), category: category, cause: cause}

	var result categoryMatcher = boundaryError(func() boundaryDetails { return details })

	return result
}

func matches(details *boundaryDetails, target error) bool {
	return errors.Is(target, details.category) || errors.Is(details.cause, target)
}

// Error returns the translated diagnostic.
func (failure boundaryError) Error() string { return failure().message }

// Is matches the stable category and caller-owned error identities.
func (failure boundaryError) Is(target error) bool {
	details := failure()

	return matches(&details, target)
}
