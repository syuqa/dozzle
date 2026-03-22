package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/trivy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockedTrivyScanner struct {
	mock.Mock
}

func (m *MockedTrivyScanner) ScanImage(ctx context.Context, image string) (*trivy.Result, error) {
	args := m.Called(ctx, image)
	if result := args.Get(0); result != nil {
		return result.(*trivy.Result), args.Error(1)
	}
	return nil, args.Error(1)
}

func Test_handler_scanContainer(t *testing.T) {
	mockedClient := mockedClient()
	mockedScanner := new(MockedTrivyScanner)
	mockedScanner.On("ScanImage", mock.Anything, "nginx:latest").Return(&trivy.Result{
		Image: "nginx:latest",
		Summary: trivy.Summary{
			Critical: 1,
			High:     2,
			Total:    3,
		},
	}, nil)

	containerValue, err := mockedClient.FindContainer(context.Background(), "123")
	require.NoError(t, err)
	containerValue.Image = "nginx:latest"
	mockedClient.ExpectedCalls = nil
	mockedClient.Calls = nil
	mockedClient.On("FindContainer", mock.Anything, "123").Return(containerValue, nil)
	mockedClient.On("FindContainer", mock.Anything, "456").Return(container.Container{}, errors.New("container not found"))
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{containerValue}, nil)
	mockedClient.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	handler := createHandlerWithScanner(mockedClient, nil, Config{
		Base:                "/",
		EnableContainerScan: true,
		Authorization:       Authorization{Provider: NONE},
	}, mockedScanner)
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/scan", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "\"image\":\"nginx:latest\"")
	assert.Contains(t, rr.Body.String(), "\"critical\":1")
}

func Test_handler_scanContainer_trivyMissing(t *testing.T) {
	mockedClient := mockedClient()
	mockedScanner := new(MockedTrivyScanner)
	mockedScanner.On("ScanImage", mock.Anything, "nginx:latest").Return(nil, exec.ErrNotFound)

	containerValue, err := mockedClient.FindContainer(context.Background(), "123")
	require.NoError(t, err)
	containerValue.Image = "nginx:latest"
	mockedClient.ExpectedCalls = nil
	mockedClient.Calls = nil
	mockedClient.On("FindContainer", mock.Anything, "123").Return(containerValue, nil)
	mockedClient.On("FindContainer", mock.Anything, "456").Return(container.Container{}, errors.New("container not found"))
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{containerValue}, nil)
	mockedClient.On("ContainerEvents", mock.Anything, mock.Anything).Return(nil)

	handler := createHandlerWithScanner(mockedClient, nil, Config{
		Base:                "/",
		EnableContainerScan: true,
		Authorization:       Authorization{Provider: NONE},
	}, mockedScanner)
	req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/scan", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}
