package evidence

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CaseResult struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Message  string        `json:"message,omitempty"`
}

type RunResult struct {
	RunID       string        `json:"run_id"`
	Name        string        `json:"name"`
	Status      string        `json:"status"`
	StartedAt   time.Time     `json:"started_at"`
	Duration    time.Duration `json:"duration"`
	Cases       []CaseResult  `json:"cases"`
	EvidenceDir string        `json:"evidence_directory"`
}

type junitSuite struct {
	XMLName  xml.Name    `xml:"testsuite"`
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Time     string      `xml:"time,attr"`
	Cases    []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name    string        `xml:"name,attr"`
	Time    string        `xml:"time,attr"`
	Failure *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func CreateRunDirectory(root, name string, now time.Time) (string, string, error) {
	runID := now.UTC().Format("20060102-150405") + "-" + slug(name)
	directory := filepath.Join(root, runID)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", "", fmt.Errorf("create evidence directory: %w", err)
	}
	return runID, directory, nil
}

func WriteManifest(directory string, raw []byte) error {
	if err := os.WriteFile(filepath.Join(directory, "manifest.yaml"), raw, 0o644); err != nil {
		return fmt.Errorf("write evidence manifest: %w", err)
	}
	return nil
}

func WriteResult(result RunResult) error {
	resultPath := filepath.Join(result.EvidenceDir, "result.json")
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode result JSON: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(resultPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write result JSON: %w", err)
	}

	suite := junitSuite{
		Name:  result.Name,
		Tests: len(result.Cases),
		Time:  seconds(result.Duration),
	}
	for _, item := range result.Cases {
		entry := junitCase{Name: item.Name, Time: seconds(item.Duration)}
		if item.Status != "passed" {
			suite.Failures++
			entry.Failure = &junitFailure{Message: item.Message, Body: item.Message}
		}
		suite.Cases = append(suite.Cases, entry)
	}

	xmlBody, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JUnit XML: %w", err)
	}
	xmlBody = append([]byte(xml.Header), xmlBody...)
	xmlBody = append(xmlBody, '\n')
	if err := os.WriteFile(filepath.Join(result.EvidenceDir, "test-results.xml"), xmlBody, 0o644); err != nil {
		return fmt.Errorf("write JUnit XML: %w", err)
	}
	return nil
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func seconds(duration time.Duration) string {
	return fmt.Sprintf("%.3f", duration.Seconds())
}
