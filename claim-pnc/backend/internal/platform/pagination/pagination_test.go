package pagination

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	page, size := Normalize(0, 0, 25, 100)
	require.Equal(t, []int{1, 25}, []int{page, size})
	page, size = Normalize(3, 500, 25, 100)
	require.Equal(t, []int{3, 100}, []int{page, size})
	page, size = Normalize(2, 10, 25, 100)
	require.Equal(t, []int{2, 10}, []int{page, size})
}

func TestOffsetAndPages(t *testing.T) {
	require.Equal(t, 0, Offset(1, 25))
	require.Equal(t, 50, Offset(3, 25))
	require.Equal(t, 1, Pages(0, 25))
	require.Equal(t, 1, Pages(25, 25))
	require.Equal(t, 2, Pages(26, 25))
}

func TestWindow(t *testing.T) {
	all := []int{1, 2, 3, 4, 5}
	require.Equal(t, []int{1, 2}, Window(all, 0, 2))
	require.Equal(t, []int{5}, Window(all, 4, 2))
	empty := Window(all, 5, 2)
	require.NotNil(t, empty)
	require.Empty(t, empty)
}

type sizes25 struct{}

func (sizes25) Default() int { return 25 }
func (sizes25) Max() int     { return 100 }

func TestGenericRequestAndPage(t *testing.T) {
	r := Request[sizes25]{}.Normalize()
	require.Equal(t, Request[sizes25]{Page: 1, Size: 25}, r)
	require.Equal(t, 25, Request[sizes25]{Page: 2}.Offset())
	page := Slice([]int{1, 2, 3}, Request[sizes25]{Page: 1, Size: 2})
	require.Equal(t, []int{1, 2}, page.Items)
	require.Equal(t, 3, page.Total)
	require.Equal(t, 2, page.TotalPages())
	require.Empty(t, Slice([]int{1}, Request[sizes25]{Page: 5}).Items)
}

func TestLimitOffset(t *testing.T) {
	l, o := LimitOffset(0, -5, 25, 100)
	require.Equal(t, []int{25, 0}, []int{l, o})
	l, o = LimitOffset(500, 10, 25, 100)
	require.Equal(t, []int{100, 10}, []int{l, o})
}
