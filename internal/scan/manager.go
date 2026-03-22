package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/amir20/dozzle/internal/trivy"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/rs/zerolog/log"
)

type HostService interface {
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	Dispatchers() []notification.DispatcherConfig
	Hosts() []container.Host
}

type Scanner interface {
	ScanImage(ctx context.Context, image string) (*trivy.Result, error)
}

type Severity string

const (
	SeverityUnknown  Severity = "UNKNOWN"
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

type ScanSchedule struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"intervalMinutes"`
}

type ContainerRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	Host  string `json:"host"`
	State string `json:"state,omitempty"`
}

type ContainerScanState struct {
	Container      ContainerRef  `json:"container"`
	Summary        trivy.Summary `json:"summary"`
	Result         *trivy.Result `json:"result,omitempty"`
	LastStartedAt  *time.Time    `json:"lastStartedAt,omitempty"`
	LastFinishedAt *time.Time    `json:"lastFinishedAt,omitempty"`
	LastSuccessAt  *time.Time    `json:"lastSuccessAt,omitempty"`
	LastError      string        `json:"lastError,omitempty"`
	Running        bool          `json:"running"`
	Schedule       ScanSchedule  `json:"schedule"`
	PackageTypes   []string      `json:"packageTypes,omitempty"`
	Severities     []string      `json:"severities,omitempty"`
}

type SummaryItem struct {
	Container     ContainerRef  `json:"container"`
	Summary       trivy.Summary `json:"summary"`
	LastSuccessAt *time.Time    `json:"lastSuccessAt,omitempty"`
	Running       bool          `json:"running"`
	LastError     string        `json:"lastError,omitempty"`
	Schedule      ScanSchedule  `json:"schedule"`
}

type DashboardSummary struct {
	Containers        int           `json:"containers"`
	ScannedContainers int           `json:"scannedContainers"`
	RunningScans      int           `json:"runningScans"`
	Critical          int           `json:"critical"`
	High              int           `json:"high"`
	Medium            int           `json:"medium"`
	Low               int           `json:"low"`
	Unknown           int           `json:"unknown"`
	Total             int           `json:"total"`
	Items             []SummaryItem `json:"items"`
}

type ScanAlert struct {
	ID                  int        `json:"id"`
	Name                string     `json:"name"`
	Enabled             bool       `json:"enabled"`
	DispatcherID        int        `json:"dispatcherId"`
	ContainerExpression string     `json:"containerExpression"`
	MinSeverity         Severity   `json:"minSeverity"`
	PackageTypes        []string   `json:"packageTypes,omitempty"`
	CooldownMinutes     int        `json:"cooldownMinutes,omitempty"`
	TriggerCount        int64      `json:"triggerCount"`
	LastTriggeredAt     *time.Time `json:"lastTriggeredAt,omitempty"`

	ContainerProgram *vm.Program `json:"-" yaml:"-"`
}

type persistedState struct {
	Scans       map[string]*ContainerScanState `json:"scans"`
	Alerts      []*ScanAlert                   `json:"alerts"`
	AlertNextID int                            `json:"alertNextId"`
}

type Manager struct {
	hostService HostService
	scanner     Scanner
	filePath    string
	mu          sync.RWMutex
	state       persistedState
	inFlight    map[string]struct{}
}

func NewManager(hostService HostService, scanner Scanner) (*Manager, error) {
	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, err
	}

	m := &Manager{
		hostService: hostService,
		scanner:     scanner,
		filePath:    filepath.Clean("./data/scans.json"),
		inFlight:    make(map[string]struct{}),
		state: persistedState{
			Scans:  make(map[string]*ContainerScanState),
			Alerts: []*ScanAlert{},
		},
	}

	if err := m.load(); err != nil {
		return nil, err
	}

	return m, nil
}

func scanKey(host, id string) string {
	return host + ":" + id
}

func normalizeInterval(minutes int) int {
	if minutes <= 0 {
		return 60
	}
	if minutes < 5 {
		return 5
	}
	if minutes > 24*60 {
		return 24 * 60
	}
	return minutes
}

func severityRank(sev string) int {
	switch strings.ToUpper(sev) {
	case "CRITICAL":
		return 5
	case "HIGH":
		return 4
	case "MEDIUM":
		return 3
	case "LOW":
		return 2
	default:
		return 1
	}
}

func packageTypesFromResult(result *trivy.Result) []string {
	seen := map[string]struct{}{}
	types := make([]string, 0)
	for _, item := range result.Results {
		if item.Type == "" {
			continue
		}
		if _, ok := seen[item.Type]; ok {
			continue
		}
		seen[item.Type] = struct{}{}
		types = append(types, item.Type)
	}
	slices.Sort(types)
	return types
}

func severitiesFromResult(result *trivy.Result) []string {
	seen := map[string]struct{}{}
	values := make([]string, 0)
	for _, item := range result.Results {
		for _, vuln := range item.Vulnerabilities {
			sev := strings.ToUpper(vuln.Severity)
			if _, ok := seen[sev]; ok {
				continue
			}
			seen[sev] = struct{}{}
			values = append(values, sev)
		}
	}
	slices.SortFunc(values, func(a, b string) int {
		return severityRank(b) - severityRank(a)
	})
	return values
}

func cloneState(in *ContainerScanState) *ContainerScanState {
	if in == nil {
		return nil
	}
	out := *in
	if in.Result != nil {
		result := *in.Result
		result.Results = append([]trivy.TargetResult(nil), in.Result.Results...)
		out.Result = &result
	}
	out.PackageTypes = append([]string(nil), in.PackageTypes...)
	out.Severities = append([]string(nil), in.Severities...)
	return &out
}

func (m *Manager) load() error {
	file, err := os.Open(m.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	var state persistedState
	if err := json.NewDecoder(file).Decode(&state); err != nil {
		return err
	}
	if state.Scans == nil {
		state.Scans = make(map[string]*ContainerScanState)
	}
	for _, alert := range state.Alerts {
		if err := compileAlert(alert); err != nil {
			return err
		}
	}
	m.state = state
	return nil
}

func (m *Manager) saveLocked() error {
	file, err := os.Create(m.filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(m.state); err != nil {
		return err
	}
	return file.Sync()
}

func compileAlert(alert *ScanAlert) error {
	if alert.ContainerExpression == "" {
		alert.ContainerExpression = "true"
	}
	program, err := expr.Compile(alert.ContainerExpression, expr.Env(types.NotificationContainer{}))
	if err != nil {
		return fmt.Errorf("compile container expression: %w", err)
	}
	alert.ContainerProgram = program
	alert.MinSeverity = Severity(strings.ToUpper(string(alert.MinSeverity)))
	if alert.MinSeverity == "" {
		alert.MinSeverity = SeverityHigh
	}
	alert.CooldownMinutes = normalizeInterval(alert.CooldownMinutes)
	slices.Sort(alert.PackageTypes)
	return nil
}

func (m *Manager) Start(ctx context.Context) {
	m.syncContainers()
	ticker := time.NewTicker(time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.runDueScans(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *Manager) runDueScans(ctx context.Context) {
	m.syncContainers()

	m.mu.RLock()
	candidates := make([]string, 0)
	for key, state := range m.state.Scans {
		if !state.Schedule.Enabled || state.Running {
			continue
		}
		if state.LastFinishedAt == nil || time.Since(*state.LastFinishedAt) >= time.Duration(state.Schedule.IntervalMinutes)*time.Minute {
			candidates = append(candidates, key)
		}
	}
	m.mu.RUnlock()

	for _, key := range candidates {
		host, id, ok := strings.Cut(key, ":")
		if !ok {
			continue
		}
		go func(host, id string) {
			runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()
			if _, err := m.RunScan(runCtx, host, id, true); err != nil {
				log.Warn().Err(err).Str("host", host).Str("container", id).Msg("scheduled container scan failed")
			}
		}(host, id)
	}
}

func (m *Manager) GetState(host, id string) *ContainerScanState {
	m.syncContainers()
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneState(m.state.Scans[scanKey(host, id)])
}

func (m *Manager) SetSchedule(host, id string, schedule ScanSchedule) (*ContainerScanState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := scanKey(host, id)
	current := m.state.Scans[key]
	if current == nil {
		current = &ContainerScanState{
			Container: ContainerRef{ID: id, Host: host},
		}
		m.state.Scans[key] = current
	}
	current.Schedule.Enabled = schedule.Enabled
	current.Schedule.IntervalMinutes = normalizeInterval(schedule.IntervalMinutes)
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return cloneState(current), nil
}

func (m *Manager) Summary() DashboardSummary {
	m.syncContainers()
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := DashboardSummary{
		Containers: len(m.state.Scans),
		Items:      make([]SummaryItem, 0, len(m.state.Scans)),
	}

	for _, state := range m.state.Scans {
		if state.Container.State != "running" {
			continue
		}
		if state.Result != nil {
			summary.ScannedContainers++
			summary.Critical += state.Summary.Critical
			summary.High += state.Summary.High
			summary.Medium += state.Summary.Medium
			summary.Low += state.Summary.Low
			summary.Unknown += state.Summary.Unknown
			summary.Total += state.Summary.Total
		}
		if state.Running {
			summary.RunningScans++
		}
		summary.Items = append(summary.Items, SummaryItem{
			Container:     state.Container,
			Summary:       state.Summary,
			LastSuccessAt: state.LastSuccessAt,
			Running:       state.Running,
			LastError:     state.LastError,
			Schedule:      state.Schedule,
		})
	}

	slices.SortFunc(summary.Items, func(a, b SummaryItem) int {
		if a.Summary.Critical != b.Summary.Critical {
			return b.Summary.Critical - a.Summary.Critical
		}
		if a.Summary.High != b.Summary.High {
			return b.Summary.High - a.Summary.High
		}
		return strings.Compare(a.Container.Name, b.Container.Name)
	})

	return summary
}

func (m *Manager) RunScan(ctx context.Context, host, id string, force bool) (*ContainerScanState, error) {
	m.syncContainers()
	key := scanKey(host, id)

	m.mu.Lock()
	if existing := m.state.Scans[key]; existing != nil && existing.Result != nil && !force && !existing.Running {
		out := cloneState(existing)
		m.mu.Unlock()
		return out, nil
	}
	if _, ok := m.inFlight[key]; ok {
		out := cloneState(m.state.Scans[key])
		m.mu.Unlock()
		if out != nil {
			return out, nil
		}
		return nil, fmt.Errorf("scan already running")
	}
	m.inFlight[key] = struct{}{}
	state := m.state.Scans[key]
	if state == nil {
		state = &ContainerScanState{
			Container: ContainerRef{ID: id, Host: host},
			Schedule:  ScanSchedule{IntervalMinutes: 60},
		}
		m.state.Scans[key] = state
	}
	now := time.Now().UTC()
	state.Running = true
	state.LastStartedAt = &now
	_ = m.saveLocked()
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.inFlight, key)
		m.mu.Unlock()
	}()

	containerService, err := m.hostService.FindContainer(host, id, container.ContainerLabels{})
	if err != nil {
		m.finishError(key, err)
		return nil, err
	}

	result, err := m.executeScan(ctx, host, containerService)
	if err != nil {
		m.finishError(key, err)
		return nil, err
	}

	stateOut, alerts, err := m.finishSuccess(key, containerService.Container, result)
	if err != nil {
		return nil, err
	}

	go m.dispatchAlerts(context.Background(), stateOut, alerts)
	return stateOut, nil
}

func (m *Manager) executeScan(ctx context.Context, host string, containerService *container_support.ContainerService) (*trivy.Result, error) {
	if m.isAgentHost(host) {
		result, err := containerService.RunScan(ctx)
		if err == nil {
			return result, nil
		}
		if err != container_support.ErrContainerScanNotSupported {
			return nil, err
		}
	}

	return m.scanner.ScanImage(ctx, containerService.Container.Image)
}

func (m *Manager) isAgentHost(hostID string) bool {
	for _, host := range m.hostService.Hosts() {
		if host.ID == hostID {
			return host.Type == "agent"
		}
	}
	return false
}

func (m *Manager) finishError(key string, scanErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.state.Scans[key]
	if state == nil {
		return
	}
	now := time.Now().UTC()
	state.Running = false
	state.LastFinishedAt = &now
	state.LastError = scanErr.Error()
	_ = m.saveLocked()
}

func (m *Manager) finishSuccess(key string, c container.Container, result *trivy.Result) (*ContainerScanState, []*ScanAlert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.state.Scans[key]
	if state == nil {
		return nil, nil, fmt.Errorf("scan state not found")
	}

	now := time.Now().UTC()
	state.Running = false
	state.Container = ContainerRef{
		ID:    c.ID,
		Name:  c.Name,
		Image: c.Image,
		Host:  c.Host,
		State: c.State,
	}
	state.Result = result
	state.Summary = result.Summary
	state.PackageTypes = packageTypesFromResult(result)
	state.Severities = severitiesFromResult(result)
	state.LastFinishedAt = &now
	state.LastSuccessAt = &now
	state.LastError = ""
	if state.Schedule.IntervalMinutes == 0 {
		state.Schedule.IntervalMinutes = 60
	}

	alerts := m.matchingAlertsLocked(c, result)
	if err := m.saveLocked(); err != nil {
		return nil, nil, err
	}
	return cloneState(state), alerts, nil
}

func (m *Manager) matchingAlertsLocked(c container.Container, result *trivy.Result) []*ScanAlert {
	if result == nil || result.Summary.Total == 0 {
		return nil
	}

	hostName := c.Host
	for _, host := range m.hostService.Hosts() {
		if host.ID == c.Host {
			hostName = host.Name
			break
		}
	}
	notificationContainer := notification.FromContainerModel(c, container.Host{ID: c.Host, Name: hostName})

	var matched []*ScanAlert
	for _, alert := range m.state.Alerts {
		if !alert.Enabled || alert.ContainerProgram == nil {
			continue
		}
		value, err := expr.Run(alert.ContainerProgram, notificationContainer)
		if err != nil {
			continue
		}
		ok, _ := value.(bool)
		if !ok {
			continue
		}
		if !matchesResult(alert, result) {
			continue
		}
		if alert.LastTriggeredAt != nil && time.Since(*alert.LastTriggeredAt) < time.Duration(alert.CooldownMinutes)*time.Minute {
			continue
		}
		now := time.Now().UTC()
		alert.LastTriggeredAt = &now
		alert.TriggerCount++
		matched = append(matched, alert)
	}
	return matched
}

func matchesResult(alert *ScanAlert, result *trivy.Result) bool {
	if result == nil {
		return false
	}
	minRank := severityRank(string(alert.MinSeverity))
	typeSet := make(map[string]struct{}, len(alert.PackageTypes))
	for _, t := range alert.PackageTypes {
		typeSet[strings.ToLower(t)] = struct{}{}
	}
	for _, item := range result.Results {
		if len(typeSet) > 0 {
			if _, ok := typeSet[strings.ToLower(item.Type)]; !ok {
				continue
			}
		}
		for _, vuln := range item.Vulnerabilities {
			if severityRank(vuln.Severity) >= minRank {
				return true
			}
		}
	}
	return false
}

func (m *Manager) dispatchAlerts(ctx context.Context, state *ContainerScanState, alerts []*ScanAlert) {
	if len(alerts) == 0 || state == nil {
		return
	}
	dispatcherConfigs := m.hostService.Dispatchers()
	dispatcherByID := make(map[int]notification.DispatcherConfig, len(dispatcherConfigs))
	for _, cfg := range dispatcherConfigs {
		dispatcherByID[cfg.ID] = cfg
	}

	for _, alert := range alerts {
		cfg, ok := dispatcherByID[alert.DispatcherID]
		if !ok {
			log.Warn().Int("dispatcher", alert.DispatcherID).Msg("scan alert dispatcher not found")
			continue
		}
		d, err := createDispatcher(cfg)
		if err != nil {
			log.Warn().Err(err).Int("dispatcher", alert.DispatcherID).Msg("scan alert dispatcher init failed")
			continue
		}
		notificationPayload := types.Notification{
			ID:        fmt.Sprintf("scan-%s-%d", state.Container.ID, time.Now().Unix()),
			Type:      types.ScanNotification,
			Detail:    fmt.Sprintf("Trivy scan found %d vulnerabilities in %s (%d critical, %d high, %d medium, %d low)", state.Summary.Total, state.Container.Image, state.Summary.Critical, state.Summary.High, state.Summary.Medium, state.Summary.Low),
			Container: types.NotificationContainer{ID: state.Container.ID, Name: state.Container.Name, Image: state.Container.Image, HostID: state.Container.Host, HostName: state.Container.Host},
			Scan:      buildScanNotification(state.Result),
			Subscription: types.SubscriptionConfig{
				ID:                  alert.ID,
				Name:                alert.Name,
				ContainerExpression: alert.ContainerExpression,
			},
			Timestamp: time.Now().UTC(),
		}
		if err := d.Send(ctx, notificationPayload); err != nil {
			log.Warn().Err(err).Str("alert", alert.Name).Msg("scan alert dispatch failed")
		}
	}
}

func buildScanNotification(result *trivy.Result) *types.NotificationScan {
	if result == nil {
		return nil
	}

	scanNotification := &types.NotificationScan{
		Image:       result.Image,
		GeneratedAt: result.GeneratedAt,
		Summary: types.NotificationScanSummary{
			Critical: result.Summary.Critical,
			High:     result.Summary.High,
			Medium:   result.Summary.Medium,
			Low:      result.Summary.Low,
			Unknown:  result.Summary.Unknown,
			Total:    result.Summary.Total,
		},
		PackageTypes:    packageTypesFromResult(result),
		Vulnerabilities: make([]types.NotificationScanVulnerability, 0),
	}

	for _, item := range result.Results {
		for _, vuln := range item.Vulnerabilities {
			scanNotification.Vulnerabilities = append(scanNotification.Vulnerabilities, types.NotificationScanVulnerability{
				ID:               vuln.ID,
				Severity:         strings.ToUpper(vuln.Severity),
				Title:            vuln.Title,
				PrimaryURL:       vuln.PrimaryURL,
				PackageName:      vuln.PackageName,
				PackageType:      item.Type,
				Target:           item.Target,
				InstalledVersion: vuln.InstalledVersion,
				FixedVersion:     vuln.FixedVersion,
			})
		}
	}

	return scanNotification
}

func (m *Manager) syncContainers() {
	containers, _ := m.hostService.ListAllContainers(container.ContainerLabels{})
	current := make(map[string]container.Container, len(containers))
	for _, c := range containers {
		current[scanKey(c.Host, c.ID)] = c
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for key, c := range current {
		state := m.state.Scans[key]
		if state == nil {
			state = &ContainerScanState{
				Container: ContainerRef{
					ID:    c.ID,
					Name:  c.Name,
					Image: c.Image,
					Host:  c.Host,
					State: c.State,
				},
				Schedule: ScanSchedule{IntervalMinutes: 60},
			}
			m.state.Scans[key] = state
			continue
		}

		state.Container.ID = c.ID
		state.Container.Name = c.Name
		state.Container.Image = c.Image
		state.Container.Host = c.Host
		state.Container.State = c.State
		if state.Schedule.IntervalMinutes == 0 {
			state.Schedule.IntervalMinutes = 60
		}
	}

	for key := range m.state.Scans {
		if _, ok := current[key]; !ok {
			delete(m.state.Scans, key)
		}
	}

	_ = m.saveLocked()
}

func createDispatcher(config notification.DispatcherConfig) (dispatcher.Dispatcher, error) {
	switch config.Type {
	case "webhook":
		return dispatcher.NewWebhookDispatcher(config.Name, config.URL, config.Template, config.Headers)
	case "cloud":
		return dispatcher.NewCloudDispatcher(config.Name, config.APIKey, config.Prefix, config.ExpiresAt)
	default:
		return nil, fmt.Errorf("unknown dispatcher type: %s", config.Type)
	}
}

func (m *Manager) Alerts() []*ScanAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*ScanAlert, 0, len(m.state.Alerts))
	for _, alert := range m.state.Alerts {
		copyAlert := *alert
		copyAlert.PackageTypes = append([]string(nil), alert.PackageTypes...)
		out = append(out, &copyAlert)
	}
	return out
}

func (m *Manager) AddAlert(alert *ScanAlert) (*ScanAlert, error) {
	if err := compileAlert(alert); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.AlertNextID++
	alert.ID = m.state.AlertNextID
	if !alert.Enabled {
		alert.Enabled = true
	}
	m.state.Alerts = append(m.state.Alerts, alert)
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	out := *alert
	return &out, nil
}

func (m *Manager) UpdateAlert(id int, next *ScanAlert) (*ScanAlert, error) {
	if err := compileAlert(next); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, alert := range m.state.Alerts {
		if alert.ID != id {
			continue
		}
		next.ID = id
		next.TriggerCount = alert.TriggerCount
		next.LastTriggeredAt = alert.LastTriggeredAt
		m.state.Alerts[i] = next
		if err := m.saveLocked(); err != nil {
			return nil, err
		}
		out := *next
		return &out, nil
	}
	return nil, fmt.Errorf("scan alert not found")
}

func (m *Manager) DeleteAlert(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Alerts = slices.DeleteFunc(m.state.Alerts, func(alert *ScanAlert) bool {
		return alert.ID == id
	})
	_ = m.saveLocked()
}
