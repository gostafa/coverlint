package domain

import (
	"fmt"
	"path"
	"strings"
)

type globPattern struct {
	segments []string
}

func compileGlob(pattern string) (globPattern, error) {
	if pattern == "" {
		return globPattern{}, fmt.Errorf("pattern is empty")
	}

	segments := strings.Split(pattern, "/")
	for _, segment := range segments {
		if segment == "" {
			return globPattern{}, fmt.Errorf("pattern contains an empty path segment")
		}
		if segment == "**" {
			continue
		}
		if strings.Contains(segment, "**") {
			return globPattern{}, fmt.Errorf("** must be a complete path segment")
		}
		if _, err := path.Match(segment, ""); err != nil {
			return globPattern{}, err
		}
	}

	return globPattern{segments: segments}, nil
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
