package memory

import (
	"time"
)

// SampleOwner adalah nama komite pemilik baris contoh.
//
// Kedua tab menyaring `NAMAKOMITE` menurut pengguna yang login, sehingga tanpa nilai ini
// tidak ada satu baris pun yang tampil saat pengembangan lokal. Pada basis data sungguhan,
// nilainya adalah Operator ID komite yang benar-benar ditunjuk memutus banding itu.
const SampleOwner = "KOMITESALVAGE"

// sampleOther adalah komite lain, dipakai membuktikan penyaring kepemilikan bekerja.
const sampleOther = "KOMITELAIN"

// sampleDaniel dan sampleBambang adalah kedua nama yang menyalakan penyaring GILIRAN komite.
//
// Keduanya nyata dan tertanam di `Activity/SetReqSalvage_Act-Act.xml` langkah 9; di sini
// keduanya dipakai supaya aturan itu benar-benar teruji, bukan sekadar ada di kode. Nilainya
// sengaja sama dengan konstanta di komite.go — bila salah satunya kelak berubah, baris contoh
// ini ikut kehilangan maknanya dan ujinya akan menyatakannya.
const (
	sampleDaniel  = "DANIELLISWANDI"
	sampleBambang = "BAMBANGSETIADJIGUNAWAN"
)

// daysAgo membentuk tanggal sekian hari lalu, dipotong ke harinya.
//
// Relatif terhadap hari ini, bukan tanggal tetap: umur baris contoh karena itu tetap sama
// berapa lama pun berkas ini tidak disentuh — tanggal tetap akan membuat seluruh contoh
// tampak berumur bertahun-tahun setelah beberapa bulan.
func daysAgo(days int) *time.Time {
	at := truncateDay(time.Now().UTC()).AddDate(0, 0, -days)
	return &at
}

// SampleCheckers adalah baris `T_CLAIM_CHEKER_SALVAGE` contoh untuk pengembangan lokal.
//
// # Kenapa seluruh isinya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor klaim,
// nama barang, dan angka harga di sini seluruhnya karangan yang bentuknya saja menyerupai
// aslinya.
//
// # Bentuk ketiga ID — ditiru dari produksi, dan BUKAN sekadar kerapian
//
// Work Owner memperlihatkan isi `T_CLAIM_CHEKER_SALVAGE` pada 2026-09-30, dan ketiga ID-nya
// ternyata tidak saling bebas. Ditulis sebagai POLA, bukan sebagai nilai — nomor klaim
// sungguhan tidak pernah masuk berkas yang di-commit (`D-69`):
//
//	NOKLAIM          PNC-<n>              nomor klaim warisan
//	IDSALVAGE        <angka>              ANGKA POLOS, disimpan sebagai teks
//	IDDETAILSALVAGE  PNC-<n>/<m>          MEMUAT nomor klaim di dalamnya
//	                 PNC-<n>/<salvage>/<m>  varian tiga ruas, juga ada
//
// Bentuk terakhir itu yang menjelaskan cacat `UpdateDokReqSalvage`: pernyataan itu
// membandingkan kolom `NOKLAIM` dengan sebuah IDDETAILSALVAGE, dan sebuah nilai berbentuk
// `PNC-<n>/<m>` tidak akan pernah sama dengan `PNC-<n>`. Sebelum bentuknya terlihat,
// ketidakcocokan itu hanya dapat DIDUGA dari beda nama kolom; sesudahnya ia terbaca dari
// bentuk datanya sendiri.
//
// Karena itu baris contoh di bawah memakai bentuk yang sama. Data contoh yang bentuknya
// salah menyembunyikan justru hal yang paling perlu diketahui pembacanya.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": tiap penyaring memperoleh baris yang membuktikannya
// bekerja, dan satu pasangan baris dibuat khusus untuk membuktikan selisih terencana.
//
//   - Umur 30 hari dan 9 hari BERDAMPINGAN. Bila urutan umur kembali menjadi teks seperti di
//     Pega, yang 9 hari akan naik ke atas — dan ujinya langsung menyatakannya.
//   - Satu baris ber-NAMAKOMITE ORANG LAIN, sehingga penyaring kepemilikan yang lupa dipasang
//     akan terlihat sebagai baris yang bocor.
//   - Satu baris ber-HARGAREQUEST kosong, yakni barang yang belum dibanding sama sekali.
//   - Satu baris yang IDDETAILSALVAGE-nya TIDAK ada di DETAIL_PNC_SALVAGE.
//   - Satu baris yang SUDAH diputus, sehingga ia hilang dari Request dan klaimnya muncul di
//     History.
//   - Satu baris TANPA tanggal request, supaya umur nol tidak menjadi galat.
//   - Sepasang baris untuk penyaring GILIRAN: satu tertahan karena komite sebelumnya belum
//     memutus, satu lolos karena sudah.
func SampleCheckers() []CheckerRecord {
	return []CheckerRecord{
		// ------------------------------------------------ milik SampleOwner, menunggu putusan
		{
			ClaimNo:         "PNCN.26.0451",
			SalvageID:       "451",
			DetailObject:    "PNC-0451/1",
			CommitteeName:   SampleOwner,
			ItemName:        "Mesin Genset Bekas",
			ItemPrice:       "12500000.00",
			RequestPrice:    "9750000.00",
			RequestNote:     "Kondisi mesin di bawah taksiran, butuh turun mesin.",
			CheckerNote:     "",
			RequestDate:     daysAgo(30),
			InDetailSalvage: true,
		},
		{
			ClaimNo:         "PNCN.26.0452",
			SalvageID:       "452",
			DetailObject:    "PNC-0452/1",
			CommitteeName:   SampleOwner,
			ItemName:        "Panel Listrik 3 Fasa",
			ItemPrice:       "4200000.00",
			RequestPrice:    "3900000.00",
			RequestNote:     "Selisih tipis, peminat terbatas di lokasi.",
			CheckerNote:     "Sudah dibandingkan dengan lelang sebelumnya.",
			RequestDate:     daysAgo(9),
			InDetailSalvage: true,
		},
		{
			// Tanpa tanggal request — umurnya nol, dan itu bukan galat.
			ClaimNo:         "PNCN.26.0453",
			SalvageID:       "453",
			DetailObject:    "PNC-0453/1",
			CommitteeName:   SampleOwner,
			ItemName:        "Kabel Tembaga Sisa",
			ItemPrice:       "1750000.00",
			RequestPrice:    "1500000.00",
			RequestNote:     "Berat aktual kurang dari catatan.",
			RequestDate:     nil,
			InDetailSalvage: true,
		},

		// ------------------------------------------------ baris yang HARUS tersaring keluar
		{
			// Belum dibanding: HARGAREQUEST kosong.
			ClaimNo:         "PNCN.26.0454",
			SalvageID:       "454",
			DetailObject:    "PNC-0454/1",
			CommitteeName:   SampleOwner,
			ItemName:        "Rangka Baja Ringan",
			ItemPrice:       "8100000.00",
			RequestPrice:    "",
			RequestDate:     daysAgo(4),
			InDetailSalvage: true,
		},
		{
			// Barangnya tidak ada di DETAIL_PNC_SALVAGE.
			ClaimNo:         "PNCN.26.0455",
			SalvageID:       "455",
			DetailObject:    "PNC-0455/99",
			CommitteeName:   SampleOwner,
			ItemName:        "Barang Yatim",
			ItemPrice:       "500000.00",
			RequestPrice:    "450000.00",
			RequestDate:     daysAgo(15),
			InDetailSalvage: false,
		},
		{
			// Milik komite lain.
			ClaimNo:         "PNCN.26.0456",
			SalvageID:       "456",
			DetailObject:    "PNC-0456/1",
			CommitteeName:   sampleOther,
			ItemName:        "Pompa Air Industri",
			ItemPrice:       "6300000.00",
			RequestPrice:    "5100000.00",
			RequestDate:     daysAgo(21),
			InDetailSalvage: true,
		},

		// ------------------------------------------------ sudah diputus -> pindah ke History
		{
			ClaimNo:         "PNCN.26.0440",
			SalvageID:       "440",
			DetailObject:    "PNC-0440/1",
			CommitteeName:   SampleOwner,
			ItemName:        "Forklift Rusak",
			ItemPrice:       "27000000.00",
			RequestPrice:    "21000000.00",
			RequestNote:     "Unit tidak dapat dinyalakan saat pemeriksaan.",
			CheckerNote:     "Disetujui, selisih wajar.",
			RequestDate:     daysAgo(48),
			ApprovedAt:      daysAgo(40),
			ApprovalStatus:  "1",
			InDetailSalvage: true,
		},

		// ------------------------------------------------ penyaring GILIRAN komite
		{
			// Tertahan: komite sebelumnya belum memutus barang yang sama.
			ClaimNo:         "PNCN.26.0460",
			SalvageID:       "460",
			DetailObject:    "PNC-0460/1",
			CommitteeName:   sampleDaniel,
			ItemName:        "Trafo Distribusi",
			ItemPrice:       "33000000.00",
			RequestPrice:    "28500000.00",
			RequestNote:     "Belitan perlu digulung ulang.",
			RequestDate:     daysAgo(6),
			InDetailSalvage: true,
		},
		{
			// Barang yang sama, giliran komite sebelumnya — BELUM diputus.
			ClaimNo:         "PNCN.26.0460",
			SalvageID:       "460",
			DetailObject:    "PNC-0460/1",
			CommitteeName:   sampleBambang,
			ItemName:        "Trafo Distribusi",
			ItemPrice:       "33000000.00",
			RequestPrice:    "28500000.00",
			RequestDate:     daysAgo(6),
			InDetailSalvage: true,
		},
		{
			// Lolos: giliran komite sebelumnya atas barang ini SUDAH lewat.
			ClaimNo:         "PNCN.26.0461",
			SalvageID:       "461",
			DetailObject:    "PNC-0461/1",
			CommitteeName:   sampleDaniel,
			ItemName:        "Kompresor Angin",
			ItemPrice:       "9800000.00",
			RequestPrice:    "8200000.00",
			RequestNote:     "Tabung berkarat.",
			RequestDate:     daysAgo(11),
			InDetailSalvage: true,
		},
		{
			ClaimNo:         "PNCN.26.0461",
			SalvageID:       "461",
			DetailObject:    "PNC-0461/1",
			CommitteeName:   sampleBambang,
			ItemName:        "Kompresor Angin",
			ItemPrice:       "9800000.00",
			RequestPrice:    "8200000.00",
			RequestDate:     daysAgo(11),
			ApprovedAt:      daysAgo(8),
			ApprovalStatus:  "1",
			InDetailSalvage: true,
		},
	}
}

// SampleSalvages adalah baris `PNC_SALVAGE` contoh untuk grid History.
//
// Satu klaim sengaja punya DUA pengajuan salvage. Kueri History tidak ber-`DISTINCT`,
// sehingga klaim itu memang muncul dua kali — dan contoh ini membuat perilaku itu teruji
// alih-alih tidak sengaja "diperbaiki" menjadi satu baris.
//
// Ada pula satu baris milik klaim yang BELUM diputus komite mana pun; ia tidak boleh muncul
// di History, dan keberadaannya membuktikan sub-kueri penyaringnya benar-benar dipakai.
func SampleSalvages() []SalvageRecord {
	return []SalvageRecord{
		{
			ClaimNo:         "PNCN.26.0440",
			SalvageID:       "440",
			SalvageType:     "Alat Berat",
			SalvageLocation: "Gudang Cakung",
			PIC:             "PICTEKNIKSATU",
		},
		{
			ClaimNo:         "PNCN.26.0440",
			SalvageID:       "441",
			SalvageType:     "Suku Cadang",
			SalvageLocation: "Gudang Cakung",
			PIC:             "PICTEKNIKSATU",
		},
		{
			// Klaim yang belum diputus siapa pun — tidak boleh muncul di History.
			ClaimNo:         "PNCN.26.0451",
			SalvageID:       "451",
			SalvageType:     "Mesin",
			SalvageLocation: "Gudang Bekasi",
			PIC:             "PICTEKNIKDUA",
		},
	}
}

// SampleDocuments adalah dokumen banding contoh untuk pengembangan lokal.
//
// Isinya karangan (`D-69`). Bentuk `DetailObject`-nya mengikuti produksi — ia dibandingkan
// dengan kolom `SALAVAGEDOCUMENT.NOKLAIM`, dan nilainya adalah id DETAIL salvage meski nama
// kolomnya menyatakan nomor klaim.
//
// Tiga keadaan sengaja diwakili:
//
//   - Dua dokumen pada satu banding, supaya urutan "terbaru di atas" benar-benar teruji.
//   - Satu dokumen yang SUDAH ditandai ditolak — ia tidak boleh muncul di daftar.
//   - Satu dokumen milik banding komite LAIN — ia tidak boleh terbaca komite mana pun
//     selain pemiliknya.
func SampleDocuments() []DocumentRecord {
	isi := []byte("%PDF-1.4 berkas contoh\n")

	return []DocumentRecord{
		{
			ID:           "9001",
			DetailObject: "PNC-0451/1",
			SalvageID:    "451",
			Name:         "penawaran-balai-lelang.pdf",
			MIMEType:     "application/pdf",
			Content:      isi,
			UploadedAt:   daysAgo(28),
		},
		{
			ID:           "9002",
			DetailObject: "PNC-0451/1",
			SalvageID:    "451",
			Name:         "foto-kondisi-barang.jpg",
			MIMEType:     "image/jpeg",
			Content:      isi,
			UploadedAt:   daysAgo(30),
		},
		{
			// Sudah ditandai ditolak — tersaring keluar oleh `IDBALAILELANG IS NULL`.
			ID:           "9003",
			DetailObject: "PNC-0451/1",
			SalvageID:    "451",
			Name:         "banding-sebelumnya.pdf",
			MIMEType:     "application/pdf",
			Content:      isi,
			UploadedAt:   daysAgo(60),
			Rejected:     true,
		},
		{
			// Milik banding komite lain — tersaring keluar oleh penyaring kepemilikan.
			ID:           "9004",
			DetailObject: "PNC-0456/1",
			SalvageID:    "456",
			Name:         "dokumen-komite-lain.pdf",
			MIMEType:     "application/pdf",
			Content:      isi,
			UploadedAt:   daysAgo(20),
		},
	}
}

// NewSampleDocumentStore membentuk pembaca dokumen berisi baris contoh.
func NewSampleDocumentStore(store *Store) *DocumentStore {
	return NewDocumentStore(store, SampleDocuments()...)
}
