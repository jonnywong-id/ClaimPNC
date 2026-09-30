package inboxbandinghargasalvage

import (
	"context"
	"strings"
)

// Berkas ini memuat tombol **Approve** dan **Reject** pada kolom "Action" grid Request.
//
// # Apa yang digantikan
//
//	Section/ButtonApproveRejectedRequest-Section.xml  kedua tombol dan isian catatannya
//	Activity/ApprovalCheckerSalvage-Act.xml           urutan langkahnya
//	RDB List/UpdateDataReqSalvage-SQL.xml             putusan pada tabel checker
//	RDB List/UpdateDokReqSalvage-SQL.xml              penandaan dokumen saat ditolak
//	RDB List/UpdateHargaSalvage-SQL.xml               penerapan harga saat disetujui
//
// # Tiga keputusan Work Owner yang membentuk berkas ini (2026-09-30)
//
//  1. **Layanan REST ke balai lelang TIDAK dipanggil.** `SendData_SalvageSimasBid` menunjuk
//     host DEV dan tanpa autentikasi. Akibatnya diterima sadar: keputusan tersimpan di basis
//     data tetapi TIDAK sampai ke balai lelang. Lihat Limitations.
//  2. **`UpdateDokReqSalvage` ditiru apa adanya**, termasuk penyaringnya yang tampak keliru.
//     Lihat MarkDocument.
//  3. **Modul ini hanya boleh menulis kolom `HARGAITEM`** pada `DETAIL_PNC_SALVAGE`.
//     Kepemilikan tabel itu tetap pada modul Inbox Salvage.

// DecisionInput adalah isian mentah dari layar, belum divalidasi.
type DecisionInput struct {
	// DetailObject adalah `IDDETAILSALVAGE` — barang mana yang diputuskan.
	DetailObject string

	// SalvageID adalah `IDSALVAGE`, dibutuhkan penyaring penerapan harga.
	SalvageID string

	// RequestPrice adalah `HARGAREQUEST` — harga tandingan yang sedang diputuskan.
	//
	// Ia dikirim ULANG oleh layar, bukan dibaca server dari barisnya, karena begitulah
	// tombolnya di Pega: nilainya diambil dari sel yang sedang tergambar. Akibatnya
	// dinyatakan di DecisionCommand.
	RequestPrice string

	// Note adalah catatan komite — ditulis ke `NOTEAPPROVE`.
	Note string

	// Approve membedakan kedua tombol.
	Approve bool
}

// DecisionCommand adalah keputusan yang sudah tervalidasi, lengkap dengan rencana langkahnya.
//
// # Kenapa rencananya ikut dibawa, bukan diputuskan adapter
//
// Karena ketiga langkah tambahan seluruhnya bergantung pada aturan bernama orang, dan aturan
// itu dikurung di komite.go. Menyerahkannya ke adapter berarti nama orang muncul di lapisan
// SQL pula — dan hari `F-4` selesai, yang harus dibongkar menjadi dua tempat.
type DecisionCommand struct {
	DetailObject string
	SalvageID    string
	RequestPrice string
	Note         string

	// Status adalah nilai `STATUSAPPROVE` yang ditulis: DecisionApproved atau
	// DecisionRejected.
	Status string

	// Reviewer menyatakan atas nama SIAPA keputusan ini dicatat.
	Reviewer Reviewer

	// CascadeTo adalah komite LAIN yang barisnya ikut ditutup dengan status yang sama.
	//
	// Kosong berarti tidak ada. Lihat PlanDecision.
	CascadeTo string

	// ApplyPrice menyatakan `DETAIL_PNC_SALVAGE.HARGAITEM` ditimpa RequestPrice.
	ApplyPrice bool

	// MarkDocument menyatakan dokumen banding ditandai "Reject Checker".
	MarkDocument bool
}

// Approved menyatakan keputusan ini menerima harga tandingan balai lelang.
func (c DecisionCommand) Approved() bool { return c.Status == DecisionApproved }

// NewDecisionCommand membentuk keputusan yang sah beserta rencana langkahnya.
func NewDecisionCommand(input DecisionInput, caller Caller) (DecisionCommand, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return DecisionCommand{}, ErrCallerUnknown
	}

	violations := []Violation{}

	detail := strings.TrimSpace(input.DetailObject)
	if detail == "" {
		violations = append(violations, Violation{
			Field:   FieldDetailObject,
			Message: "Barang yang diputuskan tidak terbaca. Muat ulang daftarnya.",
		})
	}

	salvage := strings.TrimSpace(input.SalvageID)
	if salvage == "" {
		violations = append(violations, Violation{
			Field:   FieldSalvageID,
			Message: "Pengajuan salvage tidak terbaca. Muat ulang daftarnya.",
		})
	}

	// Harga hanya dituntut saat MENYETUJUI. Menolak tidak menyentuh harga sama sekali.
	price := strings.TrimSpace(input.RequestPrice)
	if input.Approve && price == "" {
		violations = append(violations, Violation{
			Field:   FieldRequestPrice,
			Message: "Harga request kosong, sehingga tidak ada nilai yang dapat disetujui.",
		})
	}

	// Catatan TIDAK diwajibkan.
	//
	// Ia menggoda untuk diwajibkan — catatan komite adalah satu-satunya jejak alasan sebuah
	// keputusan atas nilai uang. Tetapi di Pega ia isian biasa tanpa penanda wajib, dan
	// mewajibkannya di sini adalah menambah penghalang yang tidak pernah ada: seorang komite
	// yang selama ini menyetujui tanpa menulis apa pun akan mendapati tombolnya menolak,
	// tanpa perubahan aturan yang pernah diputuskan siapa pun.
	//
	// Bila kelak diwajibkan, itu keputusan Work Owner — bukan keputusan yang diselipkan
	// lewat validasi.
	note := strings.TrimSpace(input.Note)

	if len(violations) > 0 {
		return DecisionCommand{}, NewValidationError(violations)
	}

	status := DecisionRejected
	if input.Approve {
		status = DecisionApproved
	}

	return PlanDecision(DecisionCommand{
		DetailObject: detail,
		SalvageID:    salvage,
		RequestPrice: price,
		Note:         note,
		Status:       status,
		Reviewer:     ReviewerFor(cleanCaller),
	}), nil
}

// DecisionResult menyatakan apa yang benar-benar terjadi.
type DecisionResult struct {
	// Recorded menyatakan baris putusannya benar-benar tertulis.
	//
	// Bernilai false bila tidak ada baris yang cocok — barangnya bukan milik komite ini,
	// atau sudah diputus lebih dulu. Keduanya dijawab satu galat; lihat ErrAlreadyDecided.
	Recorded bool

	// PriceApplied dan DocumentMarked menyatakan langkah tambahan mana yang berjalan.
	//
	// Keduanya dikirim balik ke layar, bukan disimpan diam-diam: pada layar ini "disetujui"
	// tidak selalu berarti "harganya berubah", dan pengguna berhak tahu mana yang terjadi.
	PriceApplied   bool
	DocumentMarked bool
}

// Writer adalah seam ke penulisan keputusan banding harga.
//
// Ia TERPISAH dari Repo, dan pemisahan itu disengaja: Repo dipakai seluruh permintaan baca,
// sementara Writer hanya dipakai satu rute. Menyatukannya berarti setiap pengisi Repo —
// termasuk yang dipakai uji baca — harus ikut memikul operasi tulis yang tidak dipakainya.
//
// # Yang dituntut pengisinya
//
// Seluruh langkah berjalan dalam SATU transaksi. Di Pega tiap pernyataan berdiri sendiri dan
// menyimpan seketika, sehingga kegagalan di tengah meninggalkan putusan tanpa harga — atau
// sebaliknya. Kepemilikan transaksi ada di Go (`D-68`), dan di sini itu bukan kerapian:
// ketiga pernyataannya menyentuh nilai uang.
type Writer interface {
	Decide(ctx context.Context, command DecisionCommand) (DecisionResult, error)
}

// WriterSelector memilih Writer milik satu portal entitas.
//
// Alasannya sama dengan RepoSelector, dan di sini taruhannya lebih besar: ia MENULIS.
// Permintaan yang jatuh ke koneksi bawaan tidak sekadar menampilkan data entitas lain — ia
// mengubahnya (`R-20`).
type WriterSelector func(portalAlias string) (Writer, error)
