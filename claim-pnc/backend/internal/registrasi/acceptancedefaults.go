package registrasi

import (
	"sort"
	"strings"
)

// # Nilai awal form AcceptationLOD
//
// Pengganti `Activity/AcceptationLOD_PreAct-act.xml`, pre-activity flow action
// AcceptationLOD. Yang dibawa:
//
//   - langkah 7–8: pilihan Tipe Akseptasi Klaim — 1 Indemnity, 2 Interim, 3 Reinstatement,
//     4 Adjuster Fee;
//   - langkah 11–13: Tipe Akseptasi = Tipe Pembayaran baris. Langkah 12 mengisi "1" untuk
//     tipe 1/3/5, tetapi transisinya "continue" (bukan lompatan — `WhenTruePrms=Currency`
//     tersisa tanpa aksi), sehingga langkah 13 selalu menimpanya dengan PaymentType;
//   - langkah 14–16 (hanya Non-MBU): Nama Komite Akseptasi = anggota komite TERAKHIR yang
//     menyetujui (`KomiteAproval == "1"`); bila komite baris itu beranggota dua atau lebih,
//     diganti satu Operator ID tetap — di Pega tertulis di dalam rule, di sini pengaturan
//     AKSEPTASI_KOMITE_BERJENJANG (`D-15`). Nilai LOD = GrossValue × ShareASM / 100;
//   - langkah 17 (hanya Non-MBU): Lokasi kejadian kosong menolak akseptasi.
//
// Yang TIDAK dibawa: langkah 9–10 (isian SLIK OJK `SurveyOJKKlaim` — belum ada di form ini;
// lihat `acceptance.go`), langkah 18 `CekProteksiCurreny` (HILANG dari export), dan langkah
// 19–20 (daftar lampiran tersimpan pada form — lampiran klaim tampil di tab Unggah Dokumen).

// AcceptanceTypeOption adalah satu pilihan dropdown Tipe Akseptasi Klaim.
type AcceptanceTypeOption struct {
	ID   string
	Name string
}

// AcceptanceTypeOptions adalah pilihan `TipeAkseptasi` langkah 8, dalam urutan Pega.
func AcceptanceTypeOptions() []AcceptanceTypeOption {
	return []AcceptanceTypeOption{
		{ID: "1", Name: "Indemnity"},
		{ID: "2", Name: "Interim"},
		{ID: "3", Name: "Reinstatement"},
		{ID: "4", Name: "Adjuster Fee"},
	}
}

// AcceptanceDefaults adalah isian yang sudah terisi saat form AcceptationLOD dibuka.
type AcceptanceDefaults struct {
	Type          string
	CommitteeName string
	LODValue      Money
	HasLODValue   bool
	// LocationMissing menandai langkah 17: lokasi kejadian kosong, akseptasi tidak dapat
	// dilakukan.
	LocationMissing bool
}

// MsgAcceptanceLocation adalah pesan langkah 11/17, disalin apa adanya.
const MsgAcceptanceLocation = "Location kosong !! Tidak dapat melakukan akseptasi"

// ViolationAcceptanceLocation menolak akseptasi Non-MBU yang lokasi kejadiannya kosong.
const ViolationAcceptanceLocation ViolationCode = "akseptasi_lokasi"

// DefaultAcceptance menghitung nilai awal form untuk satu baris adjustment.
//
// members adalah anggota kasus komite baris itu (CASEIDKOMITE); kosong bila baris tidak
// pernah ditransfer ke komite. multiLevel adalah pengganti Nama Komite untuk komite
// beranggota dua atau lebih; kosong berarti tidak ada pengganti dan anggota terakhir yang
// menyetujui tetap dipakai.
func DefaultAcceptance(line SettlementLine, p Policy, location string, members []CommitteeMember, multiLevel string) AcceptanceDefaults {
	d := AcceptanceDefaults{Type: strings.TrimSpace(line.PaymentType)}
	if !p.Line.IsNonMBU() {
		return d
	}

	ordered := append([]CommitteeMember(nil), members...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Level < ordered[j].Level })
	for _, m := range ordered {
		if strings.TrimSpace(m.Decision) == DecisionApprove {
			d.CommitteeName = strings.TrimSpace(m.Operator)
		}
	}
	if len(ordered) >= 2 && strings.TrimSpace(multiLevel) != "" {
		d.CommitteeName = strings.TrimSpace(multiLevel)
	}

	d.LODValue, d.HasLODValue = line.Gross.Share(line.ShareASM), true
	d.LocationMissing = strings.TrimSpace(location) == ""
	return d
}
