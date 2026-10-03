package inputreqprotection_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
)

// Uji tambahan atas inti domain: galat validasi, penormalan filter, dan kunci ganda.

func TestNeedsChangeDetailOnlyForTypesSevenAndEight(t *testing.T) {
	// Panel detail perubahan hanya untuk tipe '7' dan '8'; spasi di tepi diabaikan.
	require.True(t, inputreqprotection.NeedsChangeDetail("7"))
	require.True(t, inputreqprotection.NeedsChangeDetail(" 8 "))
	for _, other := range []string{"", "1", "2", "9", "78"} {
		require.False(t, inputreqprotection.NeedsChangeDetail(other), "tipe %q", other)
	}
}

func TestFilterNormalizeAppliesDefaultsAndBounds(t *testing.T) {
	// Batas kosong menjadi bawaan, melebihi maksimum dipangkas, offset negatif menjadi nol.
	f := inputreqprotection.Filter{Search: "  OPCN  ", Limit: 0, Offset: -5}.Normalize()
	require.Equal(t, "OPCN", f.Search)
	require.Equal(t, inputreqprotection.DefaultLimit, f.Limit)
	require.Equal(t, 0, f.Offset)

	f = inputreqprotection.Filter{Limit: inputreqprotection.MaxLimit + 1, Offset: 7}.Normalize()
	require.Equal(t, inputreqprotection.MaxLimit, f.Limit)
	require.Equal(t, 7, f.Offset)

	f = inputreqprotection.Filter{Limit: 20}.Normalize()
	require.Equal(t, 20, f.Limit)
}

func TestValidationErrorMessages(t *testing.T) {
	// Galat tanpa pelanggaran tetap punya pesan umum, dan OrNil mengembalikan nil.
	empty := &inputreqprotection.ValidationError{}
	require.Equal(t, "inputreqprotection: validasi gagal", empty.Error())
	require.False(t, empty.Failed())
	require.NoError(t, empty.OrNil())

	// Seluruh pelanggaran digabung berurutan dengan pemisah titik koma.
	v := &inputreqprotection.ValidationError{}
	v.Add("no_klaim", "wajib")
	v.Add("keterangan", "wajib juga")
	require.True(t, v.Failed())
	require.Equal(t, "inputreqprotection: no_klaim: wajib; keterangan: wajib juga", v.Error())
	require.Equal(t, "no_klaim: wajib", v.Errors[0].Error())
	require.Same(t, v, v.OrNil())
}

func TestDuplicateKeyForNilLocationFallsBackToUTC(t *testing.T) {
	// Tanpa zona, hari dihitung dalam UTC: 22.30 UTC tetap tanggal yang sama.
	at := time.Date(2026, time.September, 22, 22, 30, 0, 0, time.UTC)
	key := inputreqprotection.DuplicateKeyFor("  POL-1  ",
		inputreqprotection.Draft{Type: " 7 "}, at, nil)

	require.Equal(t, "POL-1", key.PolicyNumber)
	require.Equal(t, "7", key.Type)
	require.Equal(t, time.Date(2026, time.September, 22, 0, 0, 0, 0, time.UTC), key.Day)
	require.Equal(t, time.UTC, key.Day.Location())
}

func TestValidateReportsEveryTooLongField(t *testing.T) {
	// Seluruh field yang melebihi lebar kolom dilaporkan bersamaan.
	long := func(n int) string {
		b := make([]rune, n)
		for i := range b {
			b[i] = 'a'
		}
		return string(b)
	}
	d := inputreqprotection.Draft{
		ClaimNumber: "PNCN.26.0007",
		Type:        "8x",
		Note:        long(inputreqprotection.MaxNoteLength + 1),
		Change: inputreqprotection.ChangeRequest{
			CauseOfLossAfter: long(inputreqprotection.MaxChangeDataLength + 1),
		},
	}
	// Tipe "8x" panjangnya 2 (masih sah) — tetapi bukan tipe '8', sehingga panel diabaikan.
	err := d.Validate()
	require.Error(t, err)
	v, ok := err.(*inputreqprotection.ValidationError)
	require.True(t, ok)

	fields := map[string]string{}
	for _, fe := range v.Errors {
		fields[fe.Field] = fe.Message
	}
	require.Len(t, fields, 2)
	require.Contains(t, fields[inputreqprotection.FieldNote], "Keterangan terlalu panjang: 2001 karakter, maksimal 2000.")
	require.Contains(t, fields[inputreqprotection.FieldCauseOfLossMst], "maksimal 400")
}
