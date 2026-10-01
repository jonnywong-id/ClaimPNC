package spa

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// distBuilt memeriksa isi folder dist di disk. Isinya berbeda antara salinan lokal (sudah
// dibangun: index.html dan versi.txt) dan clone bersih di CI (hanya .gitkeep), sehingga
// test menyesuaikan harapannya dengan keadaan yang benar-benar disematkan saat kompilasi.
func distBuilt(t *testing.T) bool {
	t.Helper()
	_, err := os.Stat("dist/index.html")
	return err == nil
}

// swapEmbedded mengganti FS tersemat selama satu test lalu memulihkannya.
func swapEmbedded(t *testing.T, replacement embed.FS) {
	t.Helper()
	original := embedded
	embedded = replacement
	t.Cleanup(func() { embedded = original })
}

func TestFilesMatchesEmbeddedBuildState(t *testing.T) {
	root, err := Files()
	if !distBuilt(t) {
		// Clone bersih: tidak ada hasil build, galatnya harus ErrNotBuilt.
		require.ErrorIs(t, err, ErrNotBuilt)
		require.Nil(t, root)
		return
	}
	require.NoError(t, err)
	require.NotNil(t, root)

	// FS berakar di dist: index.html dapat dibaca langsung tanpa awalan "dist/".
	want, err := os.ReadFile("dist/index.html")
	require.NoError(t, err)
	got, err := fs.ReadFile(root, "index.html")
	require.NoError(t, err)
	require.Equal(t, want, got)

	_, err = fs.Stat(root, "dist/index.html")
	require.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestVersionMatchesEmbeddedMarker(t *testing.T) {
	raw, err := os.ReadFile("dist/versi.txt")
	if err != nil {
		// Tanpa penanda versi, Version mengembalikan string kosong.
		require.Equal(t, "", Version())
		return
	}
	require.Equal(t, strings.TrimSpace(string(raw)), Version())
}

func TestFilesReturnsErrNotBuiltWhenDistEmpty(t *testing.T) {
	// FS kosong meniru binary yang dikompilasi tanpa hasil build antarmuka.
	swapEmbedded(t, embed.FS{})

	root, err := Files()
	require.ErrorIs(t, err, ErrNotBuilt)
	require.Nil(t, root)
	require.Contains(t, err.Error(), "npm run build")
}

func TestVersionEmptyWithoutMarker(t *testing.T) {
	swapEmbedded(t, embed.FS{})

	require.Equal(t, "", Version())
}
