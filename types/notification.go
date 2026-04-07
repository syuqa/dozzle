package types

import "time"

// NotificationType indicates whether a notification was triggered by a log event or a metric threshold
type NotificationType string

const (
	LogNotification    NotificationType = "log"
	MetricNotification NotificationType = "metric"
	ScanNotification   NotificationType = "scan"
	StateNotification  NotificationType = "state"
)

// Notification represents a notification event that can be filtered and sent
type Notification struct {
	ID           string                `json:"id"`
	Type         NotificationType      `json:"type"`
	Detail       string                `json:"detail"`
	Container    NotificationContainer `json:"container"`
	Log          *NotificationLog      `json:"log,omitempty"`
	Stat         *NotificationStat     `json:"stat,omitempty"`
	Scan         *NotificationScan     `json:"scan,omitempty"`
	State        *NotificationState    `json:"state,omitempty"`
	Subscription SubscriptionConfig    `json:"subscription"`
	Timestamp    time.Time             `json:"timestamp"`
}

// NotificationContainer represents a simplified container structure for notifications
type NotificationContainer struct {
	ID       string            `json:"id" expr:"id"`
	Name     string            `json:"name" expr:"name"`
	Image    string            `json:"image" expr:"image"`
	State    string            `json:"state" expr:"state"`
	Health   string            `json:"health" expr:"health"`
	HostID   string            `json:"hostId" expr:"hostId"`
	HostName string            `json:"hostName" expr:"hostName"`
	Labels   map[string]string `json:"labels" expr:"labels"`
}

// NotificationLog represents a log entry with message that can be string or object
type NotificationLog struct {
	ID        uint32 `json:"id" expr:"id"`
	Message   any    `json:"message" expr:"message"` // string for simple/grouped logs, map for complex logs
	Timestamp int64  `json:"timestamp" expr:"timestamp"`
	Level     string `json:"level" expr:"level"`
	Stream    string `json:"stream" expr:"stream"`
	Type      string `json:"type" expr:"type"`
}

// NotificationStat represents container resource metrics for metric-based alerts
type NotificationStat struct {
	CPUPercent    float64 `json:"cpu" expr:"cpu"`
	MemoryPercent float64 `json:"memory" expr:"memory"`
	MemoryUsage   float64 `json:"memoryUsage" expr:"memoryUsage"`
}

// NotificationScan represents vulnerability scan details for scan-based alerts
type NotificationScan struct {
	Image           string                          `json:"image"`
	GeneratedAt     time.Time                       `json:"generatedAt"`
	Summary         NotificationScanSummary         `json:"summary"`
	PackageTypes    []string                        `json:"packageTypes,omitempty"`
	Vulnerabilities []NotificationScanVulnerability `json:"vulnerabilities,omitempty"`
}

type NotificationScanSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
	Total    int `json:"total"`
}

type NotificationScanVulnerability struct {
	ID               string `json:"id"`
	Severity         string `json:"severity"`
	Title            string `json:"title,omitempty"`
	PrimaryURL       string `json:"primaryUrl,omitempty"`
	PackageName      string `json:"packageName"`
	PackageType      string `json:"packageType,omitempty"`
	Target           string `json:"target,omitempty"`
	InstalledVersion string `json:"installedVersion,omitempty"`
	FixedVersion     string `json:"fixedVersion,omitempty"`
}

// NotificationState represents container lifecycle or health change details.
type NotificationState struct {
	Trigger       string            `json:"trigger"`
	Event         string            `json:"event,omitempty"`
	PreviousState string            `json:"previousState,omitempty"`
	CurrentState  string            `json:"currentState,omitempty"`
	PreviousImage string            `json:"previousImage,omitempty"`
	CurrentImage  string            `json:"currentImage,omitempty"`
	ExitCode      string            `json:"exitCode,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// SubscriptionConfig represents a notification subscription configuration
type SubscriptionConfig struct {
	ID                  int      `json:"id"`
	Name                string   `json:"name"`
	Enabled             bool     `json:"-"`
	DispatcherID        int      `json:"-"`
	LogExpression       string   `json:"logExpression,omitempty"`
	ContainerExpression string   `json:"containerExpression"`
	MetricExpression    string   `json:"metricExpression,omitempty"`
	Cooldown            int      `json:"cooldown,omitempty"`
	SampleWindow        int      `json:"sampleWindow,omitempty"`
	StateTriggers       []string `json:"stateTriggers,omitempty"`
	HoldoffSeconds      int      `json:"holdoffSeconds,omitempty"`
	Template            string   `json:"template,omitempty"`
	TemplateID          int      `json:"templateId,omitempty"`
}

// SubscriptionStats represents runtime stats for a notification subscription
type SubscriptionStats struct {
	SubscriptionID        int        `json:"subscriptionId"`
	TriggerCount          int64      `json:"triggerCount"`
	LastTriggeredAt       *time.Time `json:"lastTriggeredAt,omitempty"`
	TriggeredContainerIDs []string   `json:"triggeredContainerIds"`
}

// DispatcherConfig represents a notification dispatcher configuration
type DispatcherConfig struct {
	ID              int
	Name            string
	Type            string
	URL             string
	Template        string
	TemplateID      int
	Headers         map[string]string
	APIKey          string
	Prefix          string
	ExpiresAt       *time.Time
	BotToken        string
	ChatID          string
	MessageThreadID string
	ParseMode       string
}

type NotificationTemplate struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Body string `json:"body"`
}
