package inboxpladlapredlahttp

import (
	"encoding/csv"
	"net/http"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/usecase"
	"claim-pnc/internal/platform/csvexport"
)

// exportChunk adalah banyaknya baris yang diambil sekali jalan saat mengekspor.
//
// Ia sengaja SEBESAR batas halaman maksimum, bukan sebesar halaman layar: hasilnya DITULIS
// langsung ke jawaban setiap kali satu potong selesai dibaca, sehingga memori tetap datar
// berapa pun jumlah barisnya (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6).
// Memakai 10 — ukuran halaman layar ini — akan menempuh seratus kali lebih banyak
// perjalanan ke basis data tanpa menghemat memori sedikit pun.
const exportChunk = inboxpladlapredla.MaxPageSize

// exportLimit membatasi banyaknya baris pada satu berkas ekspor.
//
// # Kenapa ada batas
//
// Karena berapa baris yang wajib dilayani satu ekspor adalah pertanyaan terbuka `ADR-0011`
// yang belum dijawab: `pyMaxRecords=500` terpasang pada 54 dari 56 laporan Pega, sehingga
// kebutuhan ekspor bervolume besar belum pernah benar-benar dilayani.
//
// Berkas yang menyentuhnya diberi tanda di baris terakhir — bukan dipotong tanpa satu pun
// pemberitahuan, yang persis cacat sistem lama. Nilainya sama dengan modul inbox lain supaya
// tidak ada dua batas berbeda tanpa alasan.
const exportLimit = 50_000

// Export menangani GET /api/inbox-pla-dla-pre-dla/ekspor — tombol "Export To Excel".
//
// # Apa yang diekspor, dan di mana ia berbeda dari Pega
//
// Salinan antrean yang SEDANG DILIHAT, beserta seluruh penyaringnya. Pega menempuhnya
// dengan tiga cara yang berbeda-beda, dan tidak satu pun sama dengan yang lain:
//
//	tab PLA      ExportDataPLA         kueri TERSENDIRI, 11 kolom, memakai rentang tanggal
//	tab DLA      ExportDataDLA         kueri TERSENDIRI, 16 kolom, memakai rentang tanggal
//	tab Pre DLA  GetExportDataPreDLA   kueri tersendiri, 7 kolom, TANPA penyaring apa pun
//
// Ketiganya diseragamkan di sini menjadi "ekspor apa yang terlihat", dan dua selisihnya
// dinyatakan di PlannedDifferences:
//
//   - Kolom tambahan pada ekspor PLA dan DLA — nomor akseptasi, nilai DLA, nama
//     koasuransi, bulan sebagai angka — TIDAK ikut. Kolomnya sama dengan grid.
//   - Ekspor Pre DLA kini MENGIKUTI penyaring. Di Pega ia menarik seluruh isi
//     `T_PREDLALIST` tanpa satu pun penyaring — berkas yang isinya berbeda dari layar di
//     atasnya, dan pemindaian penuh atas tabel yang tumbuh terus. Keputusan Work Owner
//     2026-09-26.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	filter, err := readFilter(r)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	page := inboxpladlapredla.Pagination{Page: 1, Size: exportChunk}

	// Potongan pertama diambil SEBELUM satu byte pun ditulis. Setelah header terkirim,
	// galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke pengguna akan berupa
	// berkas separuh jadi tanpa satu pun keterangan. Daftar yang tidak dikenal, rentang
	// tanggal terbalik, dan sesi yang tidak lengkap karena itu tetap dijawab sebagai
	// galat yang terbaca.
	first, err := h.Service.List(r.Context(), active.Alias, caller, filter, page)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.exportGrid(w, r, active.Alias, caller, filter, first)
}

// exportGrid menulis berkas berisi salinan antrean yang sedang dilihat.
func (h *Handler) exportGrid(
	w http.ResponseWriter,
	r *http.Request,
	portalAlias string,
	caller inboxpladlapredla.Caller,
	filter inboxpladlapredla.QueryInput,
	first usecase.Listed,
) {
	tab := first.Query.Tab
	header := columnTitles(tab.Columns)

	h.beginDownload(w, exportFilename(tab))

	page := inboxpladlapredla.Pagination{Page: 1, Size: exportChunk}

	csvexport.Paged[inboxpladlapredla.Row]{
		Header: header,
		Limit:  exportLimit,
		Notice: truncationNotice,
		Row: func(item inboxpladlapredla.Row) []string {
			return gridRow(tab, item)
		},
		Next: func(n int) ([]inboxpladlapredla.Row, int, error) {
			page.Page = n
			next, err := h.Service.List(r.Context(), portalAlias, caller, filter, page)
			return next.Page.Items, next.Page.Total, err
		},
		Fail: func(err error) { h.logExportFailure(r, err) },
	}.Write(w, first.Page.Items, first.Page.Total)
}

// beginDownload memasang header unduhan.
//
// `no-store` bukan kehati-hatian berlebih: berkas ini memuat nama tertanggung dan nomor
// polis, dan ia tidak boleh mengendap di cache perantara mana pun.
func (h *Handler) beginDownload(w http.ResponseWriter, filename string) {
	csvexport.BeginDownload(w, filename)
}

// flush mendorong isi yang sudah tertulis keluar setiap potong, bukan menahannya sampai
// akhir. Itulah yang membuat unduhan besar mulai mengalir segera dan memori tidak
// menumpuk.
func (h *Handler) flush(w http.ResponseWriter, writer *csv.Writer, r *http.Request) bool {
	if err := csvexport.Flush(w, writer); err != nil {
		h.logExportFailure(r, err)
		return false
	}
	return true
}

// columnTitles menyusun baris judul dari senarai kolom.
//
// Judulnya sama persis dengan judul kolom di grid. Orang yang mencocokkan berkas dengan
// layar berdampingan tidak boleh menemukan kolom yang tergeser.
func columnTitles(columns []inboxpladlapredla.Column) []string {
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, column.Title)
	}
	return header
}

// gridRow menyusun satu baris berkas ekspor.
//
// Urutannya WAJIB sama dengan columnTitles, dan keduanya dibangun dari senarai kolom yang
// SAMA — bukan dari dua daftar yang kebetulan sejalan. Dengan begitu penambahan kolom di
// tab.go tidak dapat menggeser isi berkas tanpa menggeser judulnya sekaligus.
func gridRow(tab inboxpladlapredla.Tab, row inboxpladlapredla.Row) []string {
	cells := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		cells = append(cells, gridCell(row, column.Key))
	}
	return cells
}

// gridCell mengambil isi satu sel menurut nama kolomnya.
//
// Ia memakai konstanta Field… yang sama dengan tab.go dan dengan nama field JSON di dto.go.
// Kolom yang tidak dikenali menghasilkan teks kosong, bukan panik: berkas ekspor yang
// kehilangan satu kolom masih dapat dipakai, sedangkan permintaan yang gagal di tengah
// unduhan tidak.
func gridCell(row inboxpladlapredla.Row, key string) string {
	switch key {
	case inboxpladlapredla.FieldClaimNo:
		return row.ClaimNo
	case inboxpladlapredla.FieldPolicyNo:
		return row.PolicyNo
	case inboxpladlapredla.FieldInsured:
		return row.Insured
	case inboxpladlapredla.FieldRegisterDate:
		return row.RegisterDate
	case inboxpladlapredla.FieldLossDate:
		return row.LossDate
	case inboxpladlapredla.FieldPICTeknik:
		return row.PICTeknik
	case inboxpladlapredla.FieldAdviceDate:
		return row.AdviceDate
	default:
		return ""
	}
}

// exportFilename menyusun nama berkas yang menyebut daftar asalnya.
//
// Tanpa itu, unduhan dari ketiga tab menghasilkan berkas bernama sama di folder unduhan —
// dan yang berikutnya menimpa yang sebelumnya pada sebagian peramban. Di layar ini
// akibatnya lebih buruk daripada biasa: ketiga tab punya kolom yang IDENTIK kecuali judul
// satu kolom, sehingga berkas yang tertimpa hampir tidak dapat dikenali dari isinya.
//
// Kode tab dipakai apa adanya sebagai bagian nama berkas. Ia sudah berbentuk `kebab-case`
// dan tidak memuat satu pun karakter yang perlu dilepaskan pada nama berkas.
func exportFilename(tab inboxpladlapredla.Tab) string {
	if tab.Code == "" {
		return "inbox-pla-dla-pre-dla.csv"
	}
	return "inbox-" + tab.Code + ".csv"
}

// truncationNotice menyusun baris penanda bahwa berkasnya tidak lengkap.
func truncationNotice(width, total int) []string {
	return csvexport.TruncationNotice(width, exportLimit, total, " Persempit rentang tanggalnya.")
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Ia tidak dapat lagi dijawab sebagai galat HTTP — status sudah 200 dan sebagian berkas
// sudah sampai ke pengguna. Yang dapat dilakukan hanyalah menghentikan penulisan dan
// meninggalkan jejak, supaya unduhan yang terpotong di sisi pengguna punya pasangan
// keterangan di sisi peladen.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	csvexport.LogFailure(r, h.Logger, "ekspor inbox PLA/DLA/Pre DLA terputus", err)
}
