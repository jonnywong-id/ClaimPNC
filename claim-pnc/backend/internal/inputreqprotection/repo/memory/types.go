package memory

import (
	"context"
	"sync"
	"time"

	"claim-pnc/internal/inputreqprotection"
)

// TypeRepo memenuhi seam inputreqprotection.TypeRepo di dalam proses.
//
// Dipakai mode `PENYIMPANAN=memori` dan seluruh pengujian.
type TypeRepo struct {
	mu    sync.Mutex
	types []inputreqprotection.ProtectionType
}

// NewTypeRepo membentuk repo master kosong.
//
// Kosong, bukan berisi contoh: uji yang perlu memeriksa perilaku saat master kosong harus
// dapat memulainya dari keadaan itu, dan menyaringnya kembali dari daftar berisi jauh lebih
// merepotkan daripada menambahkannya.
func NewTypeRepo() *TypeRepo { return &TypeRepo{} }

// NewTypeRepoWithSamples membentuk repo berisi kesembilan tipe.
//
// # Isinya BUKAN karangan, dan itu disengaja
//
// Kesembilan baris di bawah disalin dari `POOLDATA.M_CLAIM_PROTECTION_TYPE` yang Work Owner
// isi pada 2026-09-24. `D-69` melarang data NASABAH ditulis di berkas yang di-commit; tipe
// proteksi bukan data nasabah melainkan data acuan, sehingga tidak termasuk larangan itu.
//
// Menyalinnya membuat mode memori memperlihatkan nama yang SAMA dengan produksi. Fake yang
// memakai nama karangan akan membuat kesalahan pemetaan kode ke nama baru terlihat setelah
// Oracle dinyalakan — yaitu di tempat yang paling mahal untuk menemukannya.
//
// # Ia SALINAN, bukan cadangan
//
// Tidak ada satu pun jalur yang jatuh ke daftar ini ketika master di Oracle kosong atau
// gagal dibaca. Master yang kosong harus TERLIHAT kosong; menutupinya dengan daftar di kode
// akan membuat master yang belum diisi tampak sudah beres.
func NewTypeRepoWithSamples() *TypeRepo {
	return &TypeRepo{types: []inputreqprotection.ProtectionType{
		{ID: "1", Name: "General"},
		{ID: "2", Name: "Premi Belum Lunas"},
		{ID: "3", Name: "Asuransi Kredit"},
		{ID: "4", Name: "Pengkinian Data"},
		{ID: "5", Name: "Currency Klaim"},
		{ID: "6", Name: "Klaim >= 50 M"},
		{ID: "7", Name: "Perubahan DOL"},
		{ID: "8", Name: "Perubahan COL"},
		{ID: "9", Name: "Nama Rekening Tidak Sesuai"},
	}}
}

// Add menambahkan tipe apa adanya. Dipakai pengujian.
func (r *TypeRepo) Add(types ...inputreqprotection.ProtectionType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types = append(r.types, types...)
}

// ListTypes mengembalikan seluruh tipe.
//
// Salinannya dikembalikan, bukan slice aslinya: pemanggil yang mengubah hasilnya tidak
// boleh ikut mengubah isi repo — kelas cacat yang hanya muncul pada adapter memori dan
// karena itu tidak pernah tertangkap uji terhadap Oracle.
func (r *TypeRepo) ListTypes(ctx context.Context) ([]inputreqprotection.ProtectionType, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]inputreqprotection.ProtectionType, len(r.types))
	copy(out, r.types)
	return out, nil
}

// NewClaimRepoWithSamples membentuk repo klaim berisi contoh.
//
// # Seluruh nilainya KARANGAN
//
// `D-69` melarang nomor polis, nama tertanggung, dan nomor klaim sungguhan ditulis di
// berkas yang di-commit. Berbeda dari master tipe — yang isinya data acuan dan boleh
// disalin — data klaim adalah data nasabah.
//
// Ketiganya sengaja berbeda keadaan, supaya ketiga jalur form dapat dicoba tanpa Oracle:
//
//	PNCN.26.0007  lengkap          panel DOL dan COL terisi keduanya
//	PNCN.26.0008  tanpa DOL        menguji klaim yang sah tetapi DOL-nya tidak tercatat
//	PNCN.26.0009  tanpa objek      menguji klaim tanpa objek pertanggungan
//
// Keadaan kedua bukan mengada-ada: hanya 1.740 dari 2.166 baris `T_CLAIM_PNC` punya
// `DATEOFLOSS`. Form harus tetap dapat dibuka untuk sisanya.
func NewClaimRepoWithSamples() *ClaimRepo {
	dol := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)

	r := NewClaimRepo()
	r.Add(
		inputreqprotection.Claim{
			Number:       "PNCN.26.0007",
			PolicyNumber: "99.001.2026.00000001",
			InsuredName:  "TERTANGGUNG CONTOH SATU",
			LossDate:     &dol,
			CauseOfLoss:  "Kebakaran",
			ObjectName:   "OBJEK CONTOH SATU",
			BranchName:   "CABANG CONTOH",
		},
		inputreqprotection.Claim{
			Number:       "PNCN.26.0008",
			PolicyNumber: "99.001.2026.00000002",
			InsuredName:  "TERTANGGUNG CONTOH DUA",
			CauseOfLoss:  "Banjir",
			ObjectName:   "OBJEK CONTOH DUA",
			BranchName:   "CABANG CONTOH",
		},
		inputreqprotection.Claim{
			Number:       "PNCN.26.0009",
			PolicyNumber: "99.001.2026.00000003",
			InsuredName:  "TERTANGGUNG CONTOH TIGA",
			LossDate:     &dol,
			BranchName:   "CABANG CONTOH",
		},
		// Klaim WARISAN Pega: satu-satunya yang ber-PegaID.
		//
		// Ia ada justru supaya percabangan `ClaimReferenceOf` benar-benar terlewati saat
		// pengembangan lokal. Contoh yang seluruhnya klaim sistem baru akan membuat cabang
		// IDPEGA tidak pernah dijalankan, dan cacatnya baru muncul di produksi — tempat
		// mayoritas klaim justru berasal dari Pega.
		inputreqprotection.Claim{
			Number:       "PNC-1865",
			PegaID:       "ASM-FW-GCNMFW-WORK PNC-1865",
			PolicyNumber: "99.001.2026.00000004",
			InsuredName:  "TERTANGGUNG CONTOH EMPAT",
			LossDate:     &dol,
			CauseOfLoss:  "Kebakaran",
			ObjectName:   "OBJEK CONTOH EMPAT",
			BranchName:   "CABANG CONTOH",
		},
	)

	// Coverage contoh, untuk panel "Detail Perubahan Cause Of Loss".
	//
	// BENTUKNYA meniru data nyata, dan itu yang membuat contoh ini berguna: nama objek dan
	// nama coverage BERULANG, persis seperti klaim `PNC-1452` yang kuerinya mengembalikan
	// `JackHugh / Resiko A` tiga kali dengan Penyebab Kerugian berbeda.
	//
	// Contoh yang setiap barisnya bernama unik akan membuat pemilihan lewat nama tampak
	// berhasil — dan cacatnya baru muncul di produksi, tempat nama memang berulang.
	r.AddCoverages("PNCN.26.0008",
		inputreqprotection.CoverageRow{
			ObjectID: "1", ObjectCoverageID: "1",
			ObjectName: "OBJEK CONTOH DUA", CoverageName: "Resiko A",
			CauseOfLoss: "ILLNESS", CauseOfLossID: "12001",
		},
		inputreqprotection.CoverageRow{
			ObjectID: "1", ObjectCoverageID: "2",
			ObjectName: "OBJEK CONTOH DUA", CoverageName: "Resiko A",
			CauseOfLoss: "STORM", CauseOfLossID: "12002",
		},
		inputreqprotection.CoverageRow{
			ObjectID: "1", ObjectCoverageID: "3",
			ObjectName: "OBJEK CONTOH DUA", CoverageName: "Katastropi",
			CauseOfLoss: "WINDSTORM", CauseOfLossID: "12003",
		},
		inputreqprotection.CoverageRow{
			ObjectID: "1", ObjectCoverageID: "4",
			ObjectName: "OBJEK CONTOH DUA", CoverageName: "Resiko A",
			CauseOfLoss: "HURRICANE", CauseOfLossID: "12004",
		},
	)

	// Klaim warisan ikut diberi coverage, supaya jalur klaim Pega dapat dicoba utuh — dan
	// supaya kunci yang dipakai memang berasal dari klaim yang nomornya berawalan.
	r.AddCoverages("PNC-1865",
		inputreqprotection.CoverageRow{
			ObjectID: "1", ObjectCoverageID: "1",
			ObjectName: "OBJEK CONTOH EMPAT", CoverageName: "Resiko A",
			CauseOfLoss: "FIRE - OPEN FLAME", CauseOfLossID: "11997",
		},
	)

	return r
}

// NewCauseRepoWithSamples membentuk master penyebab kerugian berisi contoh.
//
// Data contohnya KARANGAN (`D-69`), tetapi BENTUKNYA meniru yang nyata: beberapa pilihan
// hanya berlaku bagi satu lini bisnis, dan sebagian berlaku bagi semuanya — itulah yang
// `$.BISNISID` nyatakan di dokumen aslinya.
func NewCauseRepoWithSamples() *CauseRepo {
	r := NewCauseRepo()

	// Berlaku bagi SELURUH lini — baris master yang tidak menyebut lini mana pun.
	r.Add("",
		inputreqprotection.CauseOfLossOption{ID: "12001", Description: "ILLNESS", LossCode: "ILL"},
		inputreqprotection.CauseOfLossOption{ID: "12002", Description: "STORM", LossCode: "STM"},
	)

	// Hanya untuk satu lini bisnis, supaya penyaring BISNISID benar-benar teruji — bukan
	// sekadar dilewati karena setiap pilihan kebetulan berlaku di mana saja.
	r.Add("10166",
		inputreqprotection.CauseOfLossOption{ID: "12003", Description: "WINDSTORM", LossCode: "WND"},
		inputreqprotection.CauseOfLossOption{ID: "12004", Description: "HURRICANE", LossCode: "HUR"},
	)

	return r
}
