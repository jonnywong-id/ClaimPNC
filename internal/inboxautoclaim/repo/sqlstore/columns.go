package sqlstore

import "claim-pnc/internal/inboxautoclaim"

// tableColumn memetakan placeholder kolom ke ekspresi SQL per tab.
//
// # Kenapa ini ada
//
// Ketiga tabel batch TIDAK sekerabat. Katalog Oracle (diperiksa `-periksa`, 2026-09-28)
// membuktikannya:
//
//	TMP_BATCH_AUTO_CLAIM   (ANEKA)   TGLKEJADIAN, TGLLAPOR, COL_ID, NOTE, KEYWORD,
//	                                 OBJECTNAME, NOAKSEPTASI, FLAGTIDAKBAYAR
//	TMP_BATCH_AUTO_TRAVEL  (Travel)  TGLKEJADIAN, NOAKSEPTASI, FLAGTIDAKBAYAR,
//	                                 REPORTDESCRIPTION — tanpa TGLLAPOR, COL_ID, NOTE,
//	                                 KEYWORD, OBJECTNAME
//	TMP_BATCH_CLAIM_KREDIT (Kredit)  ACCEPTNO, NOASURANSI, TYPEKLAIM, TANGGALBAYAR — tanpa
//	                                 satu pun kolom tanggal kejadian/lapor
//
// Versi sebelumnya menulis kueri rincian, ekspor, dan sisip dengan kolom ANEKA untuk
// ketiga tab, sehingga tombol Detail, Export, dan unggahan di tab Kredit dan Travel gagal
// dengan ORA-00904 — dan itu tertutup karena data ujinya hanya punya satu bentuk tabel.
//
// Isinya DISALIN dari kueri rincian Pega per tab, bukan dikarang:
//
//	ANEKA   BrowseClaimSPK_detail_AutoClaim       col_id      AS "DistrictID", noakseptasi AS "CaseID"
//	Kredit  BrowseClaimSPK_detail_AsuransiKredit  noasuransi  AS "DistrictID", acceptno    AS "CaseID"
//	Travel  BrowseClaimSPK_detail_Travel          prodke      AS "DistrictID", noakseptasi AS "CaseID"
//
// Kolom yang tidak ada di sebuah tabel diisi NULL, bukan dibuang dari kueri: bentuk
// barisnya tetap satu sehingga pembacanya (scanLine) juga satu.
//
// Seperti {{TABEL}}, nilainya datang dari enum tertutup di dalam kode dan diisi SEKALI saat
// paket dimuat — tidak ada jalur dari masukan pengguna ke sini.
var tableColumn = map[inboxautoclaim.Source]map[string]string{
	inboxautoclaim.SourceAneka: {
		"{{K_AKSEP}}":   "A.NOAKSEPTASI",
		"{{K_REF}}":     "A.COL_ID",
		"{{K_DOL}}":     "A.TGLKEJADIAN",
		"{{K_LAPOR}}":   "A.TGLLAPOR",
		"{{K_NOTE}}":    "A.NOTE",
		"{{K_KEYWORD}}": "A.KEYWORD",
		"{{K_OBJEK}}":   "A.OBJECTNAME",
		"{{K_FLAG}}":    "A.FLAGTIDAKBAYAR",
		// Kunci primer (NOPOLIS, TGLPROSES, TGLKEJADIAN) — DDL yang diterima 2026-09-19.
		"{{K_URUT}}": "A.TGLKEJADIAN",
	},
	inboxautoclaim.SourceTravel: {
		"{{K_AKSEP}}":   "A.NOAKSEPTASI",
		"{{K_REF}}":     "A.PRODKE",
		"{{K_DOL}}":     "A.TGLKEJADIAN",
		"{{K_LAPOR}}":   "NULL",
		"{{K_NOTE}}":    "A.REPORTDESCRIPTION",
		"{{K_KEYWORD}}": "NULL",
		"{{K_OBJEK}}":   "NULL",
		"{{K_FLAG}}":    "A.FLAGTIDAKBAYAR",
		"{{K_URUT}}":    "A.TGLKEJADIAN",
	},
	inboxautoclaim.SourceKredit: {
		"{{K_AKSEP}}":   "A.ACCEPTNO",
		"{{K_REF}}":     "A.NOASURANSI",
		"{{K_DOL}}":     "NULL",
		"{{K_LAPOR}}":   "NULL",
		"{{K_NOTE}}":    "NULL",
		"{{K_KEYWORD}}": "NULL",
		"{{K_OBJEK}}":   "NULL",
		"{{K_FLAG}}":    "NULL",
		// DDL tabel ini tidak ada di repo. NOASURANSI dipakai sebagai pemecah seri karena
		// Pega sendiri memakainya sebagai kunci baris saat menandai hasil
		// (`GetMaxBatchAsuransiKredit`: `WHERE NOPOLIS ... AND upper(NOASURANSI)=...`).
		"{{K_URUT}}": "A.NOASURANSI",
	},
}
