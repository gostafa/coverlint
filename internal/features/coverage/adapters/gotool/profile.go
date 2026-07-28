package gotool

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func parseProfile(reader io.Reader) ([]domain.Block, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read coverage profile: %w", err)
		}
		return nil, fmt.Errorf("coverage profile is empty")
	}
	if !strings.HasPrefix(scanner.Text(), "mode: ") {
		return nil, fmt.Errorf("coverage profile has no mode header")
	}

	blocks := make([]domain.Block, 0, 256)
	lineNumber := 1
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		colon := strings.LastIndexByte(line, ':')
		if colon <= 0 || colon == len(line)-1 {
			return nil, fmt.Errorf("coverage profile line %d is malformed", lineNumber)
		}

		fields := strings.Fields(line[colon+1:])
		if len(fields) != 3 || !validPosition(fields[0]) {
			return nil, fmt.Errorf("coverage profile line %d is malformed", lineNumber)
		}
		statements, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || statements < 0 {
			return nil, fmt.Errorf("coverage profile line %d has invalid statement count", lineNumber)
		}
		count, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("coverage profile line %d has invalid execution count", lineNumber)
		}

		blocks = append(blocks, domain.Block{
			File:       line[:colon],
			Position:   fields[0],
			Statements: statements,
			Covered:    count > 0,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read coverage profile: %w", err)
	}
	return blocks, nil
}

func validPosition(value string) bool {
	comma := strings.IndexByte(value, ',')
	return comma > 0 && comma < len(value)-1 && strings.Contains(value[:comma], ".") && strings.Contains(value[comma+1:], ".")
}
