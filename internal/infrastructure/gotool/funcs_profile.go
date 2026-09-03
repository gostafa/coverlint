// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

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

	err := readProfileHeader(scanner)
	if err != nil {
		return nil, err
	}

	blocks, err := readProfileBlocks(scanner)
	if err != nil {
		return nil, err
	}

	err = scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("read coverage profile: %w", err)
	}

	return blocks, nil
}

func readProfileHeader(scanner *bufio.Scanner) error {
	if !scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			return fmt.Errorf("read coverage profile: %w", err)
		}

		return errEmptyCoverageProfile
	}

	if !strings.HasPrefix(scanner.Text(), "mode: ") {
		return errMissingProfileMode
	}

	return nil
}

func readProfileBlocks(scanner *bufio.Scanner) ([]domain.Block, error) {
	blocks := make([]domain.Block, 0, initialProfileBlockCap)

	lineNumber := 1

	for scanner.Scan() {
		lineNumber++

		block, ok, err := parseProfileLine(scanner.Text(), lineNumber)
		if err != nil {
			return nil, err
		}

		if ok {
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

func parseProfileLine(line string, lineNumber int) (domain.Block, bool, error) {
	line = strings.TrimSpace(line)

	if line == "" {
		return domain.Block{
			File:       "",
			Position:   "",
			Statements: 0,
			Covered:    false,
		}, false, nil
	}

	block, err := parseProfileBlock(line, lineNumber)

	return block, true, err
}

func parseProfileBlock(line string, lineNumber int) (domain.Block, error) {
	colon := strings.LastIndexByte(line, ':')

	if invalidColon(colon, line) {
		return domain.Block{}, fmt.Errorf("%w: %d", errMalformedProfileLine, lineNumber)
	}

	fields := strings.Fields(line[colon+1:])

	if invalidProfileFields(fields) {
		return domain.Block{}, fmt.Errorf("%w: %d", errMalformedProfileLine, lineNumber)
	}

	statements, err := parseStatementCount(fields[1])
	if err != nil {
		return domain.Block{}, fmt.Errorf("%w: %d", errInvalidStatementCount, lineNumber)
	}

	covered, err := parseCovered(fields[2])
	if err != nil {
		return domain.Block{}, fmt.Errorf("%w: %d", errInvalidExecutionCount, lineNumber)
	}

	return domain.Block{
		File:       line[:colon],
		Position:   fields[0],
		Statements: statements,
		Covered:    covered,
	}, nil
}

func invalidColon(colon int, line string) bool {
	return colon <= 0 || colon == len(line)-1
}

func invalidProfileFields(fields []string) bool {
	return len(fields) != 3 || !validPosition(fields[0])
}

func parseStatementCount(value string) (int64, error) {
	statements, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse statement count: %w", err)
	}

	if statements < 0 {
		return 0, errInvalidStatementCount
	}

	return statements, nil
}

func parseCovered(value string) (bool, error) {
	count, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return false, fmt.Errorf("parse execution count: %w", err)
	}

	return count > 0, nil
}

func validPosition(value string) bool {
	comma := strings.IndexByte(value, ',')

	return comma > 0 && comma < len(value)-1 && strings.Contains(value[:comma], ".") &&
		strings.Contains(value[comma+1:], ".")
}
