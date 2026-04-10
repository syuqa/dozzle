package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRedirectContainerByLabel(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	mockedClient.On("FindContainer", mock.Anything, "f0181ef6ccb6").Return(container.Container{
		ID:    "f0181ef6ccb6",
		Name:  "udg-backend-1",
		Host:  "localhost",
		State: "running",
		Labels: map[string]string{
			containerLinkLabel: "udg-backend-1-prod",
		},
	}, nil)
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{
			ID:    "f0181ef6ccb6",
			Name:  "udg-backend-1",
			Host:  "localhost",
			State: "running",
			Labels: map[string]string{
				containerLinkLabel: "udg-backend-1-prod",
			},
		},
	}, nil)

	router := createDefaultHandler(mockedClient)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/container/ref/udg-backend-1-prod", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "/container/f0181ef6ccb6", rec.Header().Get("Location"))
}

func TestRedirectContainerDetailsByLabel(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	mockedClient.On("FindContainer", mock.Anything, "f0181ef6ccb6").Return(container.Container{
		ID:    "f0181ef6ccb6",
		Name:  "udg-backend-1",
		Host:  "localhost",
		State: "running",
		Labels: map[string]string{
			containerLinkLabel: "udg-backend-1-prod",
		},
	}, nil)
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{
			ID:    "f0181ef6ccb6",
			Name:  "udg-backend-1",
			Host:  "localhost",
			State: "running",
			Labels: map[string]string{
				containerLinkLabel: "udg-backend-1-prod",
			},
		},
	}, nil)

	router := createDefaultHandler(mockedClient)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/container/ref/udg-backend-1-prod/details", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "/container/f0181ef6ccb6/details", rec.Header().Get("Location"))
}

func TestRedirectContainerByLabelNotFound(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	mockedClient.On("FindContainer", mock.Anything, mock.Anything).Return(container.Container{}, nil)
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{}, nil)

	router := createDefaultHandler(mockedClient)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/container/ref/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
