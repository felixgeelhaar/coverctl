package coverprofile

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type Parser struct{}

func (Parser) Parse(path string) (map[string]domain.CoverageStat, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	stats := make(map[string]domain.CoverageStat)
	lineNo := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNo++
		if lineNo == 1 {
			if !strings.HasPrefix(line, "mode:") {
				return nil, fmt.Errorf("invalid coverage mode line")
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		filePath, covered, total, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		stat := stats[filePath]
		stat.Covered += covered
		stat.Total += total
		stats[filePath] = stat
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func parseLine(line string) (string, int, int, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return "", 0, 0, fmt.Errorf("invalid coverage line")
	}
	filePart := parts[0]
	stmtPart := parts[1]
	countPart := parts[2]

	filePath := strings.SplitN(filePart, ":", 2)[0]
	stmtCount, err := strconv.Atoi(stmtPart)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid statement count")
	}
	count, err := strconv.ParseInt(countPart, 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid count")
	}

	covered := 0
	if count > 0 {
		covered = stmtCount
	}
	return filePath, covered, stmtCount, nil
}
