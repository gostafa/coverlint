package gotool

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const (
	initialScannerBufferSize = 64 * 1024
	maxScannerBufferSize     = 4 * 1024 * 1024
	initialProfileBlockCap   = 256
)

var (
	errEmptyCoverageProfile  = errors.New("coverage profile is empty")
	errMissingProfileMode    = errors.New("coverage profile has no mode header")
	errMalformedProfileLine  = errors.New("coverage profile line is malformed")
	errInvalidStatementCount = errors.New("coverage profile line has invalid statement count")
	errInvalidExecutionCount = errors.New("coverage profile line has invalid execution count")
)

// ParseProfile reads Go coverage profile blocks.
func ParseProfile(reader io.Reader) ([]domain.Block, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, initialScannerBufferSize), maxScannerBufferSize)

	if !scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			return nil, fmt.Errorf("read coverage profile: %w", err)
		}

		return nil, errEmptyCoverageProfile
	}

	if !strings.HasPrefix(scanner.Text(), "mode: ") {
		return nil, errMissingProfileMode
	}

	blocks := make([]domain.Block, 0, initialProfileBlockCap)

	lineNumber := 1
	for scanner.Scan() {
		lineNumber++

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		block, err := parseProfileBlock(line, lineNumber)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}

	err := scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("read coverage profile: %w", err)
	}

	return blocks, nil
}

func parseProfileBlock(line string, lineNumber int) (domain.Block, error) {
	colon := strings.LastIndexByte(line, ':')
	if colon <= 0 || colon == len(line)-1 {
		return domain.Block{}, fmt.Errorf("%w: %d", errMalformedProfileLine, lineNumber)
	}

	fields := strings.Fields(line[colon+1:])
	if len(fields) != 3 || !validPosition(fields[0]) {
		return domain.Block{}, fmt.Errorf("%w: %d", errMalformedProfileLine, lineNumber)
	}

	statements, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || statements < 0 {
		return domain.Block{}, fmt.Errorf("%w: %d", errInvalidStatementCount, lineNumber)
	}

	count, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return domain.Block{}, fmt.Errorf("%w: %d", errInvalidExecutionCount, lineNumber)
	}

	return domain.Block{
		File:       line[:colon],
		Position:   fields[0],
		Statements: statements,
		Covered:    count > 0,
	}, nil
}

func validPosition(value string) bool {
	comma := strings.IndexByte(value, ',')

	return comma > 0 && comma < len(value)-1 && strings.Contains(value[:comma], ".") &&
		strings.Contains(value[comma+1:], ".")
}
