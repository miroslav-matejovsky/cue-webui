package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMock_LoadReturnsCopy(t *testing.T) {
	store := NewMock(map[string]string{"server.host": "localhost"})

	values, err := store.Load(context.Background())
	require.NoError(t, err)
	values["server.host"] = "changed"

	snapshot := store.Snapshot()
	require.Equal(t, "localhost", snapshot["server.host"])
}

func TestMock_SaveReplacesValues(t *testing.T) {
	store := NewMock(map[string]string{"server.host": "localhost"})

	err := store.Save(context.Background(), map[string]string{"server.port": "8080"})
	require.NoError(t, err)

	snapshot := store.Snapshot()
	require.Equal(t, map[string]string{"server.port": "8080"}, snapshot)
}

func TestMock_ConfiguredErrors(t *testing.T) {
	store := NewMock(nil)
	loadErr := errors.New("load failed")
	saveErr := errors.New("save failed")

	store.SetLoadError(loadErr)
	_, err := store.Load(context.Background())
	require.ErrorIs(t, err, loadErr)

	store.SetLoadError(nil)
	store.SetSaveError(saveErr)
	err = store.Save(context.Background(), map[string]string{"key": "value"})
	require.ErrorIs(t, err, saveErr)
	asserted := store.Snapshot()
	require.Empty(t, asserted)
}
