package memory

import (
	"context"
	"strings"
)

// Groups adalah padanan memori M_LOGIN_GROUP_PNC: login → GROUP_ID.
type Groups map[string][]string

// GroupsOf mengembalikan grup sebuah login; login dicocokkan tanpa membedakan huruf.
func (g Groups) GroupsOf(_ context.Context, login string) ([]string, error) {
	for k, v := range g {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(login)) {
			return append([]string(nil), v...), nil
		}
	}
	return nil, nil
}
