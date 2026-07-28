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
		if segment == "" {
			return globPattern{}, errEmptyPathSegment
		}

		if segment == "**" {
			continue
		}

		if strings.Contains(segment, "**") {
			return globPattern{}, errPartialDoubleStar
		}

		_, err := path.Match(segment, "")
		if err != nil {
			return globPattern{}, fmt.Errorf("compile glob segment: %w", err)
		}
	}

	return globPattern{
		segments: segments,
	}, nil
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

		var matched bool

		switch {
		case patternIndex == len(g.segments):
			matched = pathIndex == len(pathSegments)
		case g.segments[patternIndex] == "**":
			matched = match(patternIndex+1, pathIndex) ||
				(pathIndex < len(pathSegments) && match(patternIndex, pathIndex+1))
		case pathIndex < len(pathSegments):
			segmentMatched, _ := path.Match(g.segments[patternIndex], pathSegments[pathIndex])
			matched = segmentMatched && match(patternIndex+1, pathIndex+1)
		}

		memo[current] = matched

		return matched
	}

	return match(0, 0)
}
