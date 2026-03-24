package web

import "testing"

func TestNormalizeScanStatusPayload(t *testing.T) {
	payload := map[string]any{
		"overall_status": "warning",
		"gitlab_url":     "https://gitlab.example.com",
		"cve": map[string]any{
			"false_positive": []any{},
			"open": []any{
				map[string]any{
					"iid":          15,
					"status_label": "Open",
					"created_at":   "2026-03-23T10:00:00.000Z",
					"updated_at":   "2026-03-23T10:00:00.000Z",
				},
			},
		},
	}

	normalized, ok := normalizeScanStatusPayload(payload).(map[string]any)
	if !ok {
		t.Fatalf("expected map payload")
	}

	if normalized["overallStatus"] != "warning" {
		t.Fatalf("expected overallStatus to be normalized, got %#v", normalized["overallStatus"])
	}
	if normalized["gitlabUrl"] != "https://gitlab.example.com" {
		t.Fatalf("expected gitlabUrl to be normalized, got %#v", normalized["gitlabUrl"])
	}

	cve, ok := normalized["cve"].(map[string]any)
	if !ok {
		t.Fatalf("expected cve group")
	}
	if _, exists := cve["false_positive"]; exists {
		t.Fatalf("expected false_positive key to be renamed")
	}
	if _, exists := cve["falsePositive"]; !exists {
		t.Fatalf("expected falsePositive key to exist")
	}

	open, ok := cve["open"].([]any)
	if !ok || len(open) != 1 {
		t.Fatalf("expected one open item")
	}
	issue, ok := open[0].(map[string]any)
	if !ok {
		t.Fatalf("expected issue item to be a map")
	}
	if issue["statusLabel"] != "Open" {
		t.Fatalf("expected statusLabel to be normalized, got %#v", issue["statusLabel"])
	}
	if issue["createdAt"] != "2026-03-23T10:00:00.000Z" {
		t.Fatalf("expected createdAt to be normalized, got %#v", issue["createdAt"])
	}
	if issue["updatedAt"] != "2026-03-23T10:00:00.000Z" {
		t.Fatalf("expected updatedAt to be normalized, got %#v", issue["updatedAt"])
	}
}
