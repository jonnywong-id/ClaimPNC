# 0023 — Tegakkan otorisasi berbasis menu di server, dengan 22 peran dan tanpa pemisahan tugas

Status: Accepted
Tanggal keputusan: 2026-09-12    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Work Owner
Jejak bukti: `D-58`, `D-59`, `D-07`, `FR-R1` | `docs/verifikasi-bukti-adr.md` §10.3 (T-10) | `Navigation/pyCaseWorkerNavigation-Navigation.xml` | `POOLDATA.T_ACCESS_GROUP_PNC`
Terkait: ADR-0024, ADR-0026, modul `F-3`, `U-6`

## Konteks

Otorisasi di sistem lama adalah **penyembunyian menu semata**. `pyPrivilegeName` terisi pada
**1 dari 902 activity**, dan yang satu itu privilege bawaan Pega untuk ekspor ruleset — bukan
aturan bisnis (T-10). Tidak ada satu pun pemeriksaan izin di sisi server untuk tindakan bisnis.

Yang tersedia sebagai bahan adalah **22 access group Pega**, seluruhnya terverifikasi ada di
export sebagai literal `GCNMFW:<nama>`:

`Administrators` · `CaseManager` · `PncAdmin` · `PncManagerAdmin` · `PncPICTeknik` ·
`PNCKomiteTeknik` · `PNCKomite` · `PncRCLPUCL` · `PncAnalystDoctor` · `PncComplience` ·
`PncInvestigator` · `PNCSurveyor` · `PncPLADLA` · `PncReceive` · `PncManagerReceive` ·
`PncCollection` · `PncOPCGeneral` · `PNCServiceCenter` · `TreatyIn` · `ViewClaimPNC` ·
`PNCReportClaimInternal` · `PNCReportClaimEksternal`

Pemetaan peran → **51 item menu** tidak hidup sebagai data, melainkan di dalam **34 When rule** +
`Navigation/pyCaseWorkerNavigation-Navigation.xml`.

## Opsi yang dipertimbangkan

Untuk jumlah peran:
1. **22 peran, satu-untuk-satu dengan access group**, hanya dinamai ulang.
2. Lebih sedikit — sebagian access group adalah varian teknis dari peran yang sama.
3. Lebih banyak — ada peran bisnis yang berbagi satu access group.

Untuk satuan izin:
1. Pemisahan tugas nyata — pelaksana tidak boleh menyetujui.
2. Tidak ada pemisahan formal; kontrolnya prosedural.
3. **Sama dengan sistem lama** — siapa pun yang punya akses menu dapat melakukannya.

## Keputusan

**Peran bisnis berjumlah 22, satu-untuk-satu dengan access group Pega**, hanya dinamai ulang agar
terbaca manusia. Tidak ada penggabungan dan tidak ada pemecahan.

**Satuan izin adalah menu, bukan tindakan individual.** Pengguna yang memiliki akses ke sebuah
menu berwenang atas seluruh tindakan yang dijangkau menu itu — termasuk membatalkan klaim,
mengubah nilai setelah persetujuan komite, dan menyetujui komite. **Tidak ada pemisahan tugas
formal.**

Perbedaan dengan sistem lama bukan pada satuan izinnya, melainkan pada **tempat penegakannya**:
dulu hanya disembunyikan di antarmuka, sekarang **ditegakkan di server pada setiap endpoint** —
yang diperiksa adalah "apakah peran pemanggil memiliki menu yang memberi akses ke endpoint ini".
Dengan bacaan itu, `BRD §21.2` kriteria #8 dan `FR-R1` tetap terpenuhi.

## Rationale

Merancang pemisahan tugas dari nol berarti menetapkan aturan kewenangan yang belum pernah ada,
pada saat yang sama dengan memigrasikan seluruh aplikasi. Itu dua perubahan besar sekaligus, dan
yang kedua tidak diminta siapa pun.

Menegakkan di server tetap menutup celah terpenting yang ada sekarang: hari ini, siapa pun yang
mengetahui alamat sebuah endpoint dapat memanggilnya tanpa pemeriksaan apa pun.

## Konsekuensi

### Positif

- Celah "menu disembunyikan tetapi endpoint terbuka" tertutup sepenuhnya.
- Model izin tetap dikenali pengguna dan administrator — tidak ada konsep baru yang harus
  dipelajari.
- Peran menjadi data di tabel, bukan logika tersebar di 34 When rule.

### Negatif / utang teknis

- **Orang yang sama dapat membuat, menyetujui, dan membayarkan satu klaim** bila perannya memiliki
  ketiga menu itu. Tidak ada kontrol teknis yang mencegahnya.
- **Jejak audit menjadi satu-satunya kontrol pengimbang.** Ini menaikkan `S-5` dari modul pendukung
  menjadi kontrol utama — dan `S-5` adalah kemampuan **baru 100% tanpa baseline** (ADR-0028).
- **Tiga nama access group muncul dalam dua kapitalisasi** — `ViewClaimPNC`/`VIEWCLAIMPNC` dan
  `PncReceive`/`PNCRECEIVE`. Perbandingan di rule lama tidak konsisten soal huruf besar-kecil;
  sistem baru wajib menormalkannya menjadi satu identitas per peran.
- **Lima When rule hilang dari export** — `IsGCNMReport`, `IsKomite`, `IsNotViewClaim`,
  `IsPNCBonding`, `IsSurvey` — masing-masing mengendalikan satu item menu.
- **Penugasan operator ke peran tidak ada di database.** `POOLDATA.T_ACCESS_GROUP_PNC` hanya
  memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`. Tabel izin dapat dibangun tetapi **tidak dapat
  diisi** tanpa artefak ini.

### Risiko yang diterima secara sadar

- `AutoAcceptKomite` (ADR-0022) menyetujui komite tanpa pengguna sama sekali, sehingga bahkan
  kontrol berbasis menu tidak berlaku padanya.
- Bila kelak ada temuan audit atau pentest yang menuntut pemisahan tugas, perubahannya menyentuh
  model izin `F-3` — bukan penyesuaian kecil.
- `BRD §21.2` kriteria #9 (jejak audit untuk setiap perubahan bernilai bisnis) menjadi **wajib
  tanpa pengecualian** pada seluruh tiket modul bisnis, karena ia satu-satunya kontrol yang
  tersisa.

## Pertanyaan terbuka

- **Dari mana daftar operator per peran diperoleh?** Tanpa sumbernya, `F-3` dapat membangun
  tabelnya tetapi tidak dapat mengisinya. Pemilik: Work Owner + DBA.
- Lima When rule yang hilang — diminta ke Tim Pega, atau aturannya ditetapkan ulang? Pemilik:
  Work Owner.
