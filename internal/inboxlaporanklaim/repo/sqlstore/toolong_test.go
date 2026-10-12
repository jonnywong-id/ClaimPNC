package sqlstore

import (
	"errors"
	"strings"
	"testing"

	"claim-pnc/internal/inboxlaporanklaim"
)

// ORA-12899 diterjemahkan menjadi pelanggaran isian yang jelas, bukan galat sistem
// (RCVN.26.90/.91, 2026-10-10).
func TestValueTooLargeNamesTheField(t *testing.T) {
	err := valueTooLarge(errors.New(`ORA-12899: value too large for column "POOLDATA"."T_CLAIM_RECIVEDCLAIM"."NAMAPELAPOR" (actual: 219, maximum: 100)`))
	var v *inboxlaporanklaim.ValidationError
	if !errors.As(err, &v) || len(v.Violation) != 1 {
		t.Fatalf("bukan pelanggaran isian: %v", err)
	}
	if v.Violation[0].Field != "nama_pelapor" ||
		v.Violation[0].Message != "Nama Pengirim / Pelapor Dokumen terlalu panjang: 219 karakter, paling banyak 100." {
		t.Fatalf("pesan tidak jelas: %+v", v.Violation[0])
	}
	if valueTooLarge(errors.New("ORA-00942: table or view does not exist")) != nil || valueTooLarge(nil) != nil {
		t.Fatal("galat lain ikut diterjemahkan")
	}
}

// Nama tertanggung dipotong ke lebar kolom byte tanpa memecah karakter UTF-8.
func TestCutBytes(t *testing.T) {
	if got := cutBytes(strings.Repeat("a", 336), 100); len(got) != 100 {
		t.Fatalf("panjang %d", len(got))
	}
	if got := cutBytes("PT ÉMAS", 4); got != "PT" {
		t.Fatalf("memecah karakter: %q", got)
	}
	if got := cutBytes("pendek", 100); got != "pendek" {
		t.Fatalf("%q", got)
	}
}
