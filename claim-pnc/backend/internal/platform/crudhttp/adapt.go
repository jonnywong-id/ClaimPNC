package crudhttp

import (
	"context"
	"net/http"
)

// Load menyusun penangan List dari pemanggilan layanan dan penyusun badan jawabannya.
func Load[T any](load func(ctx context.Context, alias string) (T, error), body func(value T, alias string) any) func(*http.Request, string) (any, error) {
	return func(r *http.Request, alias string) (any, error) {
		value, err := load(r.Context(), alias)
		if err != nil {
			return nil, err
		}
		return body(value, alias), nil
	}
}

// LoadOne menyusun penangan Get dari pemanggilan layanan dan penyusun badan jawabannya.
func LoadOne[T any](load func(ctx context.Context, alias, id string) (T, error), body func(value T, alias string) any) func(*http.Request, string, string) (any, error) {
	return func(r *http.Request, alias, id string) (any, error) {
		value, err := load(r.Context(), alias, id)
		if err != nil {
			return nil, err
		}
		return body(value, alias), nil
	}
}

// Save menyusun penangan Create dari langkah simpan dan penyusun badan jawabannya.
func Save[Req, T any](save func(r *http.Request, alias string, request Req) (T, error), body func(value T, alias string) any) func(*http.Request, string, Req) (any, error) {
	return func(r *http.Request, alias string, request Req) (any, error) {
		value, err := save(r, alias, request)
		if err != nil {
			return nil, err
		}
		return body(value, alias), nil
	}
}

// SaveOne menyusun penangan Update dari langkah simpan dan penyusun badan jawabannya.
func SaveOne[Req, T any](save func(r *http.Request, alias, id string, request Req) (T, error), body func(value T, alias string) any) func(*http.Request, string, string, Req) (any, error) {
	return func(r *http.Request, alias, id string, request Req) (any, error) {
		value, err := save(r, alias, id, request)
		if err != nil {
			return nil, err
		}
		return body(value, alias), nil
	}
}
