package crudhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdapters(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	broken := errors.New("rusak")
	body := func(v string, alias string) any { return v + "@" + alias }

	list := Load(func(_ context.Context, alias string) (string, error) { return "daftar", nil }, body)
	got, err := list(r, "asm")
	require.NoError(t, err)
	require.Equal(t, "daftar@asm", got)
	_, err = Load(func(context.Context, string) (string, error) { return "", broken }, body)(r, "asm")
	require.ErrorIs(t, err, broken)

	one := LoadOne(func(_ context.Context, alias, id string) (string, error) { return id, nil }, body)
	got, err = one(r, "asm", "7")
	require.NoError(t, err)
	require.Equal(t, "7@asm", got)
	_, err = LoadOne(func(context.Context, string, string) (string, error) { return "", broken }, body)(r, "asm", "7")
	require.ErrorIs(t, err, broken)

	create := Save(func(_ *http.Request, _ string, req int) (string, error) { return "baru", nil }, body)
	got, err = create(r, "asm", 1)
	require.NoError(t, err)
	require.Equal(t, "baru@asm", got)
	_, err = Save(func(*http.Request, string, int) (string, error) { return "", broken }, body)(r, "asm", 1)
	require.ErrorIs(t, err, broken)

	update := SaveOne(func(_ *http.Request, _, id string, _ int) (string, error) { return "ubah" + id, nil }, body)
	got, err = update(r, "asm", "9", 1)
	require.NoError(t, err)
	require.Equal(t, "ubah9@asm", got)
	_, err = SaveOne(func(*http.Request, string, string, int) (string, error) { return "", broken }, body)(r, "asm", "9", 1)
	require.ErrorIs(t, err, broken)
}
