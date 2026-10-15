package reportklaim

// Catalog adalah ke-23 panel pada layar Report Klaim.
//
// # Urutannya
//
// Dikelompokkan menurut Group supaya 23 kartu tetap terbaca di layar. Urutan ASLI harness
// tetap tersimpan pada pegaOrder dan dikembalikan CatalogInPegaOrder — siapa pun yang
// perlu mencocokkan modul ini dengan layar lama membacanya dari sana, bukan dari sini.
//
// # Kenapa fungsi, bukan variabel paket
//
// Karena isinya memuat slice, dan slice paket yang diekspor dapat diubah pemanggil.
// Satu baris `reportklaim.Catalog[0].Title = "..."` di tempat lain akan mengubah katalog
// bagi seluruh aplikasi, dan tidak ada apa pun yang mencegahnya. Fungsi mengembalikan
// salinan barunya setiap kali.
func Catalog() []Report {
	out := make([]Report, 0, 23)
	out = append(out, catalogKlaim()...)
	out = append(out, catalogReasuransi()...)
	out = append(out, catalogPenyelesaian()...)
	out = append(out, catalogLiniBisnis()...)
	out = append(out, catalogOperasional()...)
	return out
}

// pegaOrder adalah urutan panel sebagaimana tertumpuk di
// `Harness/PNCTATReport-Harness.xml`, dari atas ke bawah.
//
// Ia dicatat karena ia satu-satunya urutan yang DAPAT DIVERIFIKASI terhadap layar lama.
// Pengelompokan pada Catalog adalah penambahan yang disadari (lihat Group); urutan ini
// bukan.
//
// # Urutannya mengalir KIRI-KANAN, bukan ke bawah
//
// Layar Pega menyusun panelnya dalam DUA KOLOM. Urutan di sini urutan baca — kiri, kanan,
// lalu turun — dan itu sebabnya ia cocok dengan layar produksi tanpa diubah sama sekali
// ketika tata letaknya diperbaiki menjadi dua kolom pada 2026-10-13.
//
// # Lima panel yang dicabut
//
// Daftar ini sempat memuat 28. Lima dicabut karena tidak ada di Pega produksi — lihat
// catatan di kepala catalogLiniBisnis.
var pegaOrder = []Code{
	CodeTAT,
	CodeMitra,
	CodeProduksiKlaimPA,
	CodeCompliance,
	CodeAdjuster,
	CodeKlaimHarian,
	CodeKlaimHE,
	CodeRejectKlaim,
	CodeCloseKlaim,
	CodeTemporaryCloseKlaim,
	CodePLA,
	CodeDLA,
	CodePengirimanPLA,
	CodePengirimanDLA,
	CodeAIKlaim,
	CodeKasirSudahBayar,
	CodeKasirBelumBayar,
	CodePendingLOD,
	CodeRegistSimasOnline,
	CodeAkseptasi,
	CodeOSKomite,
	CodeOSBelumKomite,
	CodeKomite,
}

// CatalogInPegaOrder mengembalikan katalog dalam urutan tumpukan panel di harness.
func CatalogInPegaOrder() []Report {
	byCode := make(map[Code]Report, 23)
	for _, r := range Catalog() {
		byCode[r.Code] = r
	}
	out := make([]Report, 0, len(pegaOrder))
	for _, c := range pegaOrder {
		if r, ok := byCode[c]; ok {
			out = append(out, r)
		}
	}
	return out
}

// Find mencari satu laporan menurut kodenya.
func Find(code Code) (Report, bool) {
	for _, r := range Catalog() {
		if r.Code == code {
			return r, true
		}
	}
	return Report{}, false
}

// Lookup mencari laporan dan menolak yang tidak ada atau belum dapat dijalankan.
//
// Keduanya dipisahkan menjadi galat yang berbeda — lihat ErrReportNotReady.
func Lookup(code Code) (Report, error) {
	r, ok := Find(code)
	if !ok {
		return Report{}, ErrUnknownReport
	}
	if !r.Availability.Ready {
		return Report{}, ErrReportNotReady
	}
	return r, nil
}

// personalVariant menyusun syarat "lini bisnis PA atau Travel".
//
// Ia disebut sekali dan dipakai berulang karena syaratnya memang satu dan sama di
// export — `StatusReceiver=="002"||StatusReceiver=="005"` muncul apa adanya pada TAT dan
// Data Komite. Menuliskannya berulang berarti keduanya dapat berbeda saat salah satu
// diubah, dan berkas yang dihasilkan salah susunan kolomnya tanpa satu pun tanda.
func personalVariant(why string, cols []Column) variant {
	return variant{
		when:    func(f Filter) bool { return f.BusinessLine.IsPersonal() },
		why:     why,
		columns: cols,
	}
}

// nonMBUVariant menyusun syarat "lini bisnis Non-MBU".
func nonMBUVariant(why string, cols []Column) variant {
	return variant{
		when:    func(f Filter) bool { return f.BusinessLine == BusinessLineNonMBU },
		why:     why,
		columns: cols,
	}
}

// detailVariant menyusun syarat "kotak centang dicentang".
func detailVariant(why string, cols []Column) variant {
	return variant{
		when:    func(f Filter) bool { return f.Detail },
		why:     why,
		columns: cols,
	}
}

// always menyusun susunan kolom penutup, yang berlaku bila tidak ada syarat yang cocok.
func always(why string, cols []Column) variant {
	return variant{when: nil, why: why, columns: cols}
}

// oneAction menyusun daftar aksi untuk panel yang tombolnya satu.
func oneAction(label string, fixed map[string]string) []Action {
	return []Action{{Code: "", Label: label, FixedParam: fixed}}
}
