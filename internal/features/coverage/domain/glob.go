// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

var (
	errEmptyPattern      = errors.New("pattern is empty")
	errEmptyPathSegment  = errors.New("pattern contains an empty path segment")
	errPartialDoubleStar = errors.New("** must be a complete path segment")
)

type globPattern struct {
	segments []string
}

func compileGlob(pattern string) (globPattern, error) {
	if pattern == "" {
		return globPattern{}, errEmptyPattern
	}

	segments := strings.Split(pattern, "/")

	for _, segment := range segments {
		err := validateGlobSegment(segment)
		if err != nil {
			return globPattern{}, err
		}
	}

	return globPattern{
		segments: segments,
	}, nil
}

func validateGlobSegment(segment string) error {
	switch {
	case segment == "":
		return errEmptyPathSegment
	case segment == "**":
		return nil
	case strings.Contains(segment, "**"):
		return errPartialDoubleStar
	}

	_, err := path.Match(segment, "")
	if err != nil {
		return fmt.Errorf("compile glob segment: %w", err)
	}

	return nil
}

func (g globPattern) Match(importPath string) bool {
	pathSegments := strings.Split(importPath, "/")

	type position struct {
		pattern int
		path    int
	}

	memo := make(map[position]bool)
	visited := make(map[position]bool)

	var match func(patternIndex, pathIndex int) bool

	match = func(patternIndex, pathIndex int) bool {
		current := position{pattern: patternIndex, path: pathIndex}

		if visited[current] {
			return memo[current]
		}

		visited[current] = true

		matched := g.matchAt(
			patternSegments{pathSegments: pathSegments, match: match},
			patternIndex,
			pathIndex,
		)

		memo[current] = matched

		return matched
	}

	return match(0, 0)
}

type patternSegments struct {
	match        func(patternIndex, pathIndex int) bool
	pathSegments []string
}

func (g globPattern) matchAt(state patternSegments, patternIndex, pathIndex int) bool {
	switch {
	case patternIndex == len(g.segments):
		return pathIndex == len(state.pathSegments)
	case g.segments[patternIndex] == "**":
		return g.matchDoubleStar(state, patternIndex, pathIndex)
	case pathIndex < len(state.pathSegments):
		return g.matchSegment(state, patternIndex, pathIndex)
	default:
		return false
	}
}

func (g globPattern) matchDoubleStar(state patternSegments, patternIndex, pathIndex int) bool {
	return state.match(patternIndex+1, pathIndex) ||
		(pathIndex < len(state.pathSegments) && state.match(patternIndex, pathIndex+1))
}

func (g globPattern) matchSegment(state patternSegments, patternIndex, pathIndex int) bool {
	segmentMatched, _ := path.Match(g.segments[patternIndex], state.pathSegments[pathIndex])

	return segmentMatched && state.match(patternIndex+1, pathIndex+1)
}
