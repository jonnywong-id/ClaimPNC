package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	list := []Violation{{"nama", "wajib diisi"}, {"kode", "sudah dipakai"}}
	require.Equal(t, "m: validasi gagal — nama: wajib diisi; kode: sudah dipakai",
		Format(list, "m: validasi gagal — ", ": ", "; ", ""))
	require.Equal(t, "m: isian tidak sah (nama: wajib diisi; kode: sudah dipakai)",
		Format(list, "m: isian tidak sah (", ": ", "; ", ")"))
	require.Equal(t, "m: ", Format(nil, "m: ", ": ", "; ", ""))
}
