package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPublicContainerRefURL(t *testing.T) {
	url, err := buildPublicContainerRefURL("https://dozzle.example.com", "/", "udg-backend-1")
	require.NoError(t, err)
	assert.Equal(t, "https://dozzle.example.com/container/ref/udg-backend-1", url)

	url, err = buildPublicContainerRefURL("https://dozzle.example.com", "/ops/dozzle", "udg backend")
	require.NoError(t, err)
	assert.Equal(t, "https://dozzle.example.com/ops/dozzle/container/ref/udg%20backend", url)
}
