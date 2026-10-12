package registrasi

import (
	"testing"
	"time"
)

// Nomor PLA_DLA.prc: KODE || TAHUN || ID_SITE || LPAD(COUNT,15,'0').
func TestPLANumberFollowsProcedure(t *testing.T) {
	if got := PLANumber("J", 2026, "1", 12345); got != "J261000000000012345" {
		t.Fatalf("nomor = %s", got)
	}
}

// Penerima COINS: Sinar Mas leader wajib; Sinar Mas sendiri, anggota terhapus, dan share 0
// tidak menerima PLA. Portal ASI memakai Simas Insurtech sebagai perusahaan sendiri.
func TestCoinsRecipients(t *testing.T) {
	members := []PLACoinsMember{
		{ID: "1", Name: "ASURANSI SINAR MAS - KANTOR PUSAT", Leader: true, Share: 570_000, HasShare: true},
		{ID: "2", Name: "ANGGOTA A", Share: 300_000, HasShare: true},
		{ID: "3", Name: "ANGGOTA B", Share: 130_000, HasShare: true, Deleted: true},
		{ID: "4", Name: "ANGGOTA C", Share: 0, HasShare: true},
	}
	got, err := CoinsPLARecipients("ASM", members)
	if err != nil || len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("penerima = %+v, %v", got, err)
	}
	if _, err := CoinsPLARecipients("ASI", members); err == nil {
		t.Fatalf("di portal ASI Sinar Mas bukan perusahaan sendiri; leader-nya bukan kita")
	}
	if _, err := CoinsPLARecipients("ASM", nil); err == nil {
		t.Fatalf("tanpa CoinsList harus ditolak")
	}
}

// ResultPLA = reserve × % anggota, seperti JSON_PLA historis (135 jt × 7,5% = 10,125 jt).
func TestCoinsAmounts(t *testing.T) {
	a := CoinsPLAAmounts([]FaceSheetAmount{{Currency: "IDR", Value: Rupiah(135_000_000)}}, map[string]string{"IDR": "10026"}, 75_000, 552_500)
	if a[0].Result != Rupiah(10_125_000) || a[0].Base != Rupiah(135_000_000) || a[0].CurrencyID != "10026" {
		t.Fatalf("nilai = %+v", a[0])
	}
}

func TestPLANoteFollowsInsertPLADLA(t *testing.T) {
	if PLANote(PLAPrevious{}, false) != plaFirstNote {
		t.Fatalf("PLA pertama harus Estimation only")
	}
	n := PLANote(PLAPrevious{Number: "J1", Date: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)}, true)
	if n != "- Please see our PLA No.:J1 with DD: 09/07/2026" {
		t.Fatalf("catatan = %s", n)
	}
}
