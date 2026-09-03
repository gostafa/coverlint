// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"path"
	"strings"
)

type (
	globPattern = struct {
		segments []string
	}

	globMatcher = struct {
		memo         map[[indexPairSize]int]bool
		visited      map[[indexPairSize]int]bool
		segments     []string
		pathSegments []string
	}
)

func compileGlob(pattern string) (globPattern, error) {
	if pattern == emptyString {
		return globPattern{}, errEmptyPattern
	}

	segments := strings.Split(pattern, pathSeparator)

	for i := range segments {
		err := validateGlobSegment(segments[i])
		if err != nil {
			return globPattern{}, fmt.Errorf("validate glob segment: %w", err)
		}
	}

	return globPattern{segments: segments}, nil
}

func globMatch(matcher *globMatcher, patternIndex, pathIndex int) bool {
	current := [indexPairSize]int{patternIndex, pathIndex}

	if matcher.visited[current] {
		return matcher.memo[current]
	}

	matcher.visited[current] = true

	matched := globMatchAt(matcher, patternIndex, pathIndex)

	matcher.memo[current] = matched

	return matched
}

func globMatchAt(matcher *globMatcher, patternIndex, pathIndex int) bool {
	switch {
	case patternIndex == len(matcher.segments):
		return pathIndex == len(matcher.pathSegments)
	case matcher.segments[patternIndex] == doubleStar:
		return globMatchDoubleStar(matcher, patternIndex, pathIndex)
	case pathIndex < len(matcher.pathSegments):
		return globMatchSegment(matcher, patternIndex, pathIndex)
	default:
		return false
	}
}

func globMatchDoubleStar(matcher *globMatcher, patternIndex, pathIndex int) bool {
	if globMatch(matcher, patternIndex+one, pathIndex) {
		return true
	}

	return pathIndex < len(matcher.pathSegments) && globMatch(matcher, patternIndex, pathIndex+one)
}

func globMatchSegment(matcher *globMatcher, patternIndex, pathIndex int) bool {
	segmentMatched, err := path.Match(
		matcher.segments[patternIndex],
		matcher.pathSegments[pathIndex],
	)
	if err != nil {
		return false
	}

	return segmentMatched && globMatch(matcher, patternIndex+one, pathIndex+one)
}

func globSegmentShape(segment string) (bool, error) {
	switch {
	case segment == emptyString:
		return false, errEmptyPathSegment
	case segment == doubleStar:
		return true, nil
	case strings.Contains(segment, doubleStar):
		return false, errPartialDoubleStar
	default:
		return false, nil
	}
}

func matchGlob(pattern globPattern, importPath string) bool {
	return globMatch(newGlobMatcher(pattern, importPath), zero, zero)
}

func newGlobMatcher(pattern globPattern, importPath string) *globMatcher {
	return &globMatcher{
		segments:     pattern.segments,
		pathSegments: strings.Split(importPath, pathSeparator),
		memo:         make(map[[indexPairSize]int]bool),
		visited:      make(map[[indexPairSize]int]bool),
	}
}

func validateGlobSegment(segment string) error {
	skip, err := globSegmentShape(segment)
	if err != nil {
		return fmt.Errorf("validate glob segment shape: %w", err)
	}

	if skip {
		return nil
	}

	err = validateGlobSegmentSyntax(segment)
	if err != nil {
		return fmt.Errorf("validate glob segment syntax: %w", err)
	}

	return nil
}

func validateGlobSegmentSyntax(segment string) error {
	matched, err := path.Match(segment, emptyString)
	if err != nil {
		return fmt.Errorf("compile glob segment: %w", err)
	}

	if matched {
		return nil
	}

	return nil
}
