package scan

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/amir20/dozzle/internal/trivy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubHostService struct {
	containerService *container_support.ContainerService
	hosts            []container.Host
}

func (s *stubHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	return s.containerService, nil
}

func (s *stubHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	return []container.Container{s.containerService.Container}, nil
}

func (s *stubHostService) Dispatchers() []notification.DispatcherConfig {
	return nil
}

func (s *stubHostService) Templates() []*notification.NotificationTemplate {
	return nil
}

func (s *stubHostService) Hosts() []container.Host {
	return s.hosts
}

type stubScanner struct {
	calls  int
	result *trivy.Result
}

func (s *stubScanner) ScanImage(ctx context.Context, image string) (*trivy.Result, error) {
	s.calls++
	return s.result, nil
}

type stubClientService struct {
	runScanResult *trivy.Result
	runScanCalls  int
}

func (s *stubClientService) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return container.Container{}, nil
}

func (s *stubClientService) ListContainers(ctx context.Context, filter container.ContainerLabels) ([]container.Container, error) {
	return nil, nil
}

func (s *stubClientService) Host(ctx context.Context) (container.Host, error) {
	return container.Host{}, nil
}

func (s *stubClientService) ContainerAction(ctx context.Context, c container.Container, action container.ContainerAction) error {
	return nil
}

func (s *stubClientService) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	return false, nil
}

func (s *stubClientService) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	return nil, nil
}

func (s *stubClientService) RawLogs(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return nil, nil
}

func (s *stubClientService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
}

func (s *stubClientService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
}

func (s *stubClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
}

func (s *stubClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	return nil
}

func (s *stubClientService) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
	return nil
}

func (s *stubClientService) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	return nil
}

func (s *stubClientService) RunContainerScan(ctx context.Context, id string) (*trivy.Result, error) {
	s.runScanCalls++
	return s.runScanResult, nil
}

func TestManagerRunScanUsesAgentExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	expected := &trivy.Result{
		Image: "private/image:1.0.0",
		Summary: trivy.Summary{
			High:  2,
			Total: 2,
		},
	}
	client := &stubClientService{runScanResult: expected}
	containerValue := container.Container{
		ID:    "123",
		Name:  "api",
		Image: expected.Image,
		Host:  "agent-1",
		State: "running",
	}
	hostService := &stubHostService{
		containerService: container_support.NewContainerService(client, containerValue),
		hosts: []container.Host{{
			ID:   "agent-1",
			Name: "agent-1",
			Type: "agent",
		}},
	}
	localScanner := &stubScanner{
		result: &trivy.Result{Image: "wrong"},
	}

	manager, err := NewManager(hostService, localScanner)
	require.NoError(t, err)

	state, err := manager.RunScan(context.Background(), "agent-1", "123", true)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, 1, client.runScanCalls)
	assert.Equal(t, 0, localScanner.calls)
	assert.Equal(t, expected.Image, state.Result.Image)
	assert.Equal(t, 2, state.Summary.High)

	_, err = os.Stat(filepath.Join(tmpDir, "data", "scans.db"))
	assert.NoError(t, err)
}

func TestNotifyOnManualDefaultsToTrue(t *testing.T) {
	alert := &ScanAlert{}
	assert.True(t, notifyOnManual(alert))
	value := false
	alert.NotifyOnManual = &value
	assert.False(t, notifyOnManual(alert))
}

func TestManagerMigratesLegacyJSONToSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "data"), 0o755))
	legacyState := persistedState{
		Scans: map[string]*ContainerScanState{
			"agent-1:123": {
				Container: ContainerRef{ID: "123", Host: "agent-1", Name: "api", Image: "private/image:1.0.0"},
				Summary:   trivy.Summary{High: 1, Total: 1},
				Result:    &trivy.Result{Image: "private/image:1.0.0", Summary: trivy.Summary{High: 1, Total: 1}},
				Schedule:  ScanSchedule{Enabled: true, IntervalMinutes: 60},
			},
		},
		AlertNextID: 1,
	}
	file, err := os.Create(filepath.Join(tmpDir, "data", "scans.json"))
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(file).Encode(legacyState))
	require.NoError(t, file.Close())

	hostService := &stubHostService{
		containerService: container_support.NewContainerService(&stubClientService{}, container.Container{
			ID:    "123",
			Name:  "api",
			Image: "private/image:1.0.0",
			Host:  "agent-1",
			State: "running",
		}),
	}
	manager, err := NewManager(hostService, &stubScanner{})
	require.NoError(t, err)

	state := manager.GetState("agent-1", "123")
	require.NotNil(t, state)
	assert.Equal(t, "private/image:1.0.0", state.Result.Image)

	_, err = os.Stat(filepath.Join(tmpDir, "data", "scans.db"))
	assert.NoError(t, err)
}
