package trivy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var ErrImageRequired = errors.New("image is required")

type Scanner struct {
	binary string
}

func NewScanner(binary string) *Scanner {
	if strings.TrimSpace(binary) == "" {
		binary = "trivy"
	}

	return &Scanner{binary: binary}
}

type Result struct {
	Image       string         `json:"image"`
	GeneratedAt time.Time      `json:"generatedAt"`
	Summary     Summary        `json:"summary"`
	Results     []TargetResult `json:"results"`
}

type Summary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
	Total    int `json:"total"`
}

type TargetResult struct {
	Target          string          `json:"target"`
	Class           string          `json:"class,omitempty"`
	Type            string          `json:"type,omitempty"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

type Vulnerability struct {
	ID               string `json:"id"`
	PackageName      string `json:"packageName"`
	InstalledVersion string `json:"installedVersion,omitempty"`
	FixedVersion     string `json:"fixedVersion,omitempty"`
	Severity         string `json:"severity"`
	Title            string `json:"title,omitempty"`
	PrimaryURL       string `json:"primaryUrl,omitempty"`
}

type rawReport struct {
	Results []rawTargetResult `json:"Results"`
}

type rawTargetResult struct {
	Target          string             `json:"Target"`
	Class           string             `json:"Class"`
	Type            string             `json:"Type"`
	Vulnerabilities []rawVulnerability `json:"Vulnerabilities"`
}

type rawVulnerability struct {
	ID               string `json:"VulnerabilityID"`
	PackageName      string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title"`
	PrimaryURL       string `json:"PrimaryURL"`
}

func (s *Scanner) ScanImage(ctx context.Context, image string) (*Result, error) {
	if strings.TrimSpace(image) == "" {
		return nil, ErrImageRequired
	}

	cmd := exec.CommandContext(ctx, s.binary, "image", "--quiet", "--format", "json", "--scanners", "vuln", image)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("trivy scan failed: %s: %w", message, err)
	}

	var report rawReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		return nil, fmt.Errorf("decode trivy output: %w", err)
	}

	result := &Result{
		Image:       image,
		GeneratedAt: time.Now().UTC(),
		Results:     make([]TargetResult, 0, len(report.Results)),
	}

	for _, raw := range report.Results {
		target := TargetResult{
			Target:          raw.Target,
			Class:           raw.Class,
			Type:            raw.Type,
			Vulnerabilities: make([]Vulnerability, 0, len(raw.Vulnerabilities)),
		}

		for _, vuln := range raw.Vulnerabilities {
			result.Summary.Total++
			switch strings.ToUpper(vuln.Severity) {
			case "CRITICAL":
				result.Summary.Critical++
			case "HIGH":
				result.Summary.High++
			case "MEDIUM":
				result.Summary.Medium++
			case "LOW":
				result.Summary.Low++
			default:
				result.Summary.Unknown++
			}

			target.Vulnerabilities = append(target.Vulnerabilities, Vulnerability{
				ID:               vuln.ID,
				PackageName:      vuln.PackageName,
				InstalledVersion: vuln.InstalledVersion,
				FixedVersion:     vuln.FixedVersion,
				Severity:         vuln.Severity,
				Title:            vuln.Title,
				PrimaryURL:       vuln.PrimaryURL,
			})
		}

		if len(target.Vulnerabilities) > 0 {
			result.Results = append(result.Results, target)
		}
	}

	return result, nil
}
