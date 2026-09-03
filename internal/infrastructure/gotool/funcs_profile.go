// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

// ParseProfile reads Go coverage profile blocks.
func ParseProfile(reader io.Reader) ([]domain.Block, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, initialScannerBufferSize), maxScannerBufferSize)

	err := readProfileHeader(scanner)
	if err != nil {
		return nil, fmt.Errorf(errParseCoverageProfileFormat, err)
	}

	blocks, err := finishProfileScan(scanner)
	if err != nil {
		return nil, fmt.Errorf(errParseCoverageProfileFormat, err)
	}

	return blocks, nil
}

func finishProfileScan(scanner *bufio.Scanner) ([]domain.Block, error) {
	blocks, err := readProfileBlocks(scanner)
	if err != nil {
		return nil, fmt.Errorf("finish profile scan: %w", err)
	}

	err = scanner.Err()
	if err != nil {
		return nil, fmt.Errorf(errReadCoverageProfileFormat, err)
	}

	return blocks, nil
}

func invalidColon(colon int, line string) bool {
	return colon <= zero || colon == len(line)-one
}

func invalidProfileFields(fields []string) bool {
	return len(fields) != three || !validPosition(fields[zero])
}

func parseCovered(value string) (bool, error) {
	count, err := strconv.ParseUint(value, decimalBase, intBitSize)
	if err != nil {
		return false, fmt.Errorf("parse execution count: %w", err)
	}

	return count > zero, nil
}

func parseProfileBlock(line string, lineNumber int) (domain.Block, error) {
	colon := strings.LastIndexByte(line, ':')

	if invalidColon(colon, line) {
		return domain.Block{}, fmt.Errorf(errWrapLine, errMalformedProfileLine, lineNumber)
	}

	block, err := parseProfileFields(line, colon, lineNumber)
	if err != nil {
		return domain.Block{}, fmt.Errorf("parse profile block: %w", err)
	}

	return block, nil
}

func parseProfileFields(line string, colon, lineNumber int) (domain.Block, error) {
	fields := strings.Fields(line[colon+one:])

	if invalidProfileFields(fields) {
		return domain.Block{}, fmt.Errorf(errWrapLine, errMalformedProfileLine, lineNumber)
	}

	block, err := buildProfileBlock(line[:colon], fields, lineNumber)
	if err != nil {
		return domain.Block{}, fmt.Errorf("parse profile fields: %w", err)
	}

	return block, nil
}

func buildProfileBlock(file string, fields []string, lineNumber int) (domain.Block, error) {
	statements, err := parseStatementCount(fields[one])
	if err != nil {
		return domain.Block{}, fmt.Errorf(errWrapLine, errInvalidStatementCount, lineNumber)
	}

	covered, err := parseCovered(fields[2])
	if err != nil {
		return domain.Block{}, fmt.Errorf(errWrapLine, errInvalidExecutionCount, lineNumber)
	}

	return domain.Block{
		File:       file,
		Position:   fields[zero],
		Statements: statements,
		Covered:    covered,
	}, nil
}

func parseProfileLine(line string, lineNumber int) (block domain.Block, ok bool, err error) {
	line = strings.TrimSpace(line)

	if line == emptyString {
		return emptyBlock(), false, nil
	}

	block, err = parseProfileBlock(line, lineNumber)
	if err != nil {
		return domain.Block{}, true, fmt.Errorf("parse profile line: %w", err)
	}

	return block, true, nil
}

func emptyBlock() domain.Block {
	return domain.Block{
		File:       emptyString,
		Position:   emptyString,
		Statements: zero,
		Covered:    false,
	}
}

func parseStatementCount(value string) (int64, error) {
	statements, err := strconv.ParseInt(value, decimalBase, intBitSize)
	if err != nil {
		return zero, fmt.Errorf("parse statement count: %w", err)
	}

	if statements < zero {
		return zero, errInvalidStatementCount
	}

	return statements, nil
}

func readProfileBlocks(scanner *bufio.Scanner) ([]domain.Block, error) {
	blocks := make([]domain.Block, zero, initialProfileBlockCap)
	lineNumber := one

	for scanner.Scan() {
		lineNumber++

		block, ok, err := parseProfileLine(scanner.Text(), lineNumber)
		if err != nil {
			return nil, fmt.Errorf("read profile blocks: %w", err)
		}

		if ok {
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

func readProfileHeader(scanner *bufio.Scanner) error {
	if !scanner.Scan() {
		return fmt.Errorf("read profile header: %w", profileHeaderScanError(scanner))
	}

	if !strings.HasPrefix(scanner.Text(), profileModePrefix) {
		return errMissingProfileMode
	}

	return nil
}

func profileHeaderScanError(scanner *bufio.Scanner) error {
	err := scanner.Err()
	if err != nil {
		return fmt.Errorf(errReadCoverageProfileFormat, err)
	}

	return errEmptyCoverageProfile
}

func validPosition(value string) bool {
	comma := strings.IndexByte(value, ',')

	return comma > zero && comma < len(value)-one && strings.Contains(value[:comma], positionDot) &&
		strings.Contains(value[comma+one:], positionDot)
}
