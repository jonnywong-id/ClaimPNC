# Inventaris 74 Harness — bahan kerja pembagian modul layar

| | |
|---|---|
| **Tabel lengkapnya** | [`../Steering/22-INVENTARIS-HARNESS.md`](../Steering/22-INVENTARIS-HARNESS.md) |
| **Sumber** | direktori `Harness/` — dibaca langsung oleh `docs/tools/build-inventaris-harness.js` |
| **Dasar** | `D-73` · pertanyaan terbuka `U-3` dan `U-4`/`U-5`/`U-6` |

> **Tabel di bawah dibangkitkan generator yang sama** dengan lampiran Steering> (`docs/tools/build-inventaris-harness.js`), langsung dari direktori `Harness/`. Keduanya karena> itu **tidak dapat berbeda**: yang disunting tangan hanya pengantar di atas tabel, bukan tabelnya.
## Kenapa pemisahan ini penting bagi tiket

Empat tiket bergantung langsung pada pembagian yang belum diputuskan:

| Tiket | Bergantung pada |
|---|---|
| [TKT-U3-002](U-3-Layar-Inbox-per-Peran/issues/02-inbox-per-peran-dan-pemetaannya.md) | Mana dari 26 harness bernama *Inbox* yang benar-benar **inbox peran** |
| [TKT-U4-001](U-4-Layar-Transaksi-Klaim/issues/01-kerangka-layar-transaksi-per-tahap.md) | Pembagian harness non-Inbox ke `U-4`/`U-5`/`U-6` |
| [TKT-U5-001](U-5-Layar-Laporan/issues/01-layar-pemilih-laporan-dan-parameter.md) | idem |
| [TKT-U6-002](U-6-Layar-Master-Data/issues/02-kelompok-master-dan-kewenangannya.md) | idem |

## Pemisahan berdasarkan NAMA ternyata bukan pembeda yang berguna

Diuji terhadap export, dan **gagal**:

| Yang diuji | Hasil |
|---|---|
| Harness bernama *Inbox* memakai model penugasan? | **Tidak terbukti.** Penanda yang dipakai ternyata menangkap boilerplate Pega (`pyDashboardMyWorkList`); `InboxRegister_Harness` — inbox sungguhan — justru **nol kecocokan** |
| Nama *Inbox* sejalan dengan kelas harness? | **Tidak.** 23 dari 26 bernama Inbox berkelas `Data-Portal` — sama dengan 35 yang bukan bernama Inbox |

## Yang benar-benar berpengaruh pada implementasi

| Kelas | Jumlah | Akibatnya pada kode |
|---|---:|---|
| **`ASM-FW-GCNMFW-Work*`** | **7** | Terikat pada **satu klaim**; rutenya memuat nomor klaim, dan layarnya tidak berarti tanpa klaim yang dimuat. **Sudah pasti milik `U-4`** |
| **`@baseclass`** | **9** | Generik; konteks ditentukan pemanggil |
| **`Data-Portal`** | **58** | Berdiri sendiri di portal — daftar, master, laporan. Rute tidak memerlukan klaim |

Ketujuh yang sudah pasti `U-4`:

```
InputProgress · ViewDetailHasilSurveyorInternal1 · ViewReceiveDocument
ViewTempDetailAllCase · ViewTempDetailClaim · ViewTempDetailReqDocument
View_DetailKlaimCabang_Harness
```

## Apa yang berubah bila pembagiannya salah

| Terpengaruh | Bagaimana |
|---|---|
| **Ukuran `U-3`** | `U-3` membangun **satu layar inbox dipakai ulang**. Bila yang benar-benar inbox peran hanya 14 dan bukan 26, komponennya lebih kecil dan AC-nya berubah |
| **Urutan kerja** | Layar yang membaca model penugasan **tidak dapat dibangun sebelum `B-6`** |
| **Bentuk rute** | Layar terikat klaim butuh nomor klaim di rutenya; layar portal tidak |
| **Penguji gerbang 2** | Tiap layar diuji pemegang perannya — salah modul berarti salah penguji |
| **Kendali akses** | Pada inbox, penyaringan "hanya pekerjaan saya" **adalah kendali akses**; pada master, kendalinya "siapa boleh mengubah" |

## Definisi Inbox kini menjadi dasar penggolongan (`D-79`)

> **Inbox** = layar berisi **daftar pekerjaan milik pengguna** — Tugas dari Worklist/Workbasket.
> Layar **data acuan** bukan Inbox, sekalipun dapat dicari dan sekalipun namanya mengandung
> kata "Inbox".

Sebelum `D-79`, kolom **Usulan modul** bersandar pada **sidik jari berkas** — dasar yang
korelasional. Sekarang dasarnya **isi layar**; sidik jari tetap dipakai sebagai **alat baca**,
karena isi layar tidak dapat dibaca langsung dari XML. Yang berubah adalah kedudukannya: sidik
jari menjadi cara **menerapkan** definisi, bukan **pengganti** definisi.

Akibatnya **tujuh harness bernama *Inbox* diusulkan ke `U-6`**.

## Yang perlu diputuskan

1. ~~Mana yang benar-benar inbox peran.~~ — **definisinya tertutup `D-79`**. Yang tersisa
   adalah **koreksi atas usulan**: 27 bertanda `DUGAAN` dan 5 bertanda `JANGGAL`.
2. **Pemilik modul untuk 67 harness selain ketujuh yang berkelas Work.**

Kolom **Usulan modul** kini terisi — lihat bagian **Tingkat keyakinan** di atas. Setiap baris
ditandai PASTI / KUAT / DUGAAN / JANGGAL, sehingga usulan tidak dapat tersamar sebagai fakta.

## Sidik jari berkas mempersempit keputusan dari 74 menjadi segelintir

Ukuran berkas harness dan jumlah rujukan Report Definition di dalamnya ternyata mengelompok tajam.
Ini **bukti korelasional, bukan pernyataan tujuan** — tetapi cukup untuk menyingkirkan sebagian
besar dugaan.

| Kelompok | Jumlah | Sidik jari | Tafsiran |
|---|---:|---|---|
| Bernama *Inbox*, berkas **besar** | **10** | 971–1.885 KB · RD 69–276 | **Kandidat kuat inbox peran** |
| Bernama *Inbox*, ukuran **sedang** | 7 | 412–805 KB · RD 27–84 | Perlu ditinjau |
| Bernama *Inbox*, berkas **kecil** | 7 | **267–273 KB · RD 15** | **Hampir pasti BUKAN inbox peran** |
| Pembanding: `MasterRekening` (master murni) | — | **273 KB · RD 15** | — |

Ketujuh yang berkas kecil bersidik jari **nyaris identik dengan layar master**, bukan dengan
`InboxRegister_Harness` (1.330 KB · RD 144):

```
CauseOfLossInbox · CauseOfLossInboxSimasOnline · DetailSurveyorsInbox
ListDocumentTypeInbox · StatusClaimInbox · SurveyorsInbox · UserTeknisInbox
```

**Dua yang menyimpang dari pola dan perlu ditinjau khusus:**

| Harness | Sidik jari | Kenapa janggal |
|---|---|---|
| `inboxCompliance_Harness` | **144 KB · RD 0** | Terkecil dari seluruh 74 dan **tanpa satu pun Report Definition**. Layar antrean kerja tanpa sumber data adalah hal yang tidak masuk akal |
| `inboxAnalystDoctor_Harness` | 348 KB · RD 12 | Di bawah ambang kelompok master, tetapi ia menyangkut **data medis** (`FR-R2`) sehingga salah golong berakibat pada kendali akses |

Enam harness **tidak** bernama *Inbox* tetapi berkas besar — layar kerja sungguhan yang penamaannya
tidak memberi petunjuk: `PNCArchiveDokumen` · `ViewReceiveDocument` · `ReportKPIHarness` ·
`PNCTATReport` · `ProgressClaim_Harness` · `MasterProteksiVisibilityData`.

## Tingkat keyakinan pada kolom usulan

Kolom **Usulan modul** diisi oleh aturan di `docs/tools/build-inventaris-harness.js` (fungsi
`usul()`), bukan diketik tangan — sehingga dasarnya dapat diperiksa dan hasilnya dapat dibangun
ulang. **Seluruhnya usulan, bukan keputusan.**

| Tanda | Artinya | Jumlah |
|---|---|---:|
| **PASTI** | Bukti **struktural** dari export: kelas harness `ASM-FW-GCNMFW-Work*`. Tidak bergantung nama sama sekali | **7** |
| **KUAT** | **Dua bukti sejalan** — topik dari nama *dan* sidik jari berkas menunjuk arah yang sama | **35** |
| **DUGAAN** | **Satu bukti saja**, atau bukti yang saling bertentangan | **27** |
| **JANGGAL** | Sidik jarinya menyimpang dari pola mana pun — **wajib ditinjau manusia** | **5** |

**Cara nama berkas dipakai — dan tidak dipakai.** Nama dipakai untuk menduga **topik** layar
(rekening, sparepart, laporan). Nama **tidak** dipakai untuk menduga **pola interaksi**, karena itu
sudah diuji dan gagal. Pola diambil dari kelas harness dan sidik jari berkas.

Akibatnya, tujuh harness bernama *Inbox* justru diusulkan ke **`U-6`**, bukan `U-3` — sidik jarinya
sama dengan layar master, bukan dengan layar antrean kerja.

### Lima yang JANGGAL — tidak saya beri usulan sama sekali

| Harness | Sidik jari | Kenapa tidak diusulkan |
|---|---|---|
| `inboxCompliance_Harness` | 145 KB · RD 0 | Bernama inbox tetapi **tanpa satu pun sumber data**. Antrean kerja tanpa sumber data tidak masuk akal |
| `RCLPUCL_Harness` | 184 KB · RD 0 | `B-11` menyebutnya layar utama RCL/PUCL, tetapi isinya nyaris kosong |
| `DashboardClaim_Harness` | 155 KB · RD 0 | Dashboard tanpa sumber data |
| `ViewPolis1` | 122 KB · RD 0 | Kembar dengan `ViewPolis`; salah satunya mungkin sudah mati |
| `GCNMCatSparepart` | 85 KB · RD 0 | **Terkecil dari 74** |

Kelimanya berbagi satu pola: **`RD = 0`**. Kemungkinan besar cangkang tipis — layar yang isinya
dipasok dari tempat lain, atau layar yang sudah tidak dipakai. Keduanya berakibat berbeda pada
lingkup, dan **BELUM DIPUTUSKAN — pertanyaan terbuka** (pemilik: **Work Owner + Tim Pega**).

### Yang tetap tidak berubah

Usulan ini **tidak menutup** pertanyaan terbuka `U-3` maupun `U-4`/`U-5`/`U-6`. Ia hanya mengubah
bentuk pekerjaan Work Owner: dari **mengklasifikasi 74 berkas** menjadi **mengoreksi 27 DUGAAN dan
memutuskan 5 JANGGAL**. Yang bertanda **PASTI** tidak perlu ditinjau; yang **KUAT** cukup dibaca
sekilas.

## Inventaris

| Harness | Kelas | KB | RD | Bernama *Inbox* | **Usulan modul** | Keyakinan |
|---|---|---:|---:|---|---|---|
| `AutoKlaim` | Data-Portal | 276 | 9 | — | **U-4** | DUGAAN |
| `BengkelHE` | Data-Portal | 267 | 15 | — | **U-6** | KUAT |
| `BrowseMasterDocumentTravel_Harness` | Data-Portal | 278 | 21 | — | **U-6** | DUGAAN |
| `CauseOfLossInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `CauseOfLossInboxSimasOnline` | Data-Portal | 274 | 15 | ya | **U-6** | KUAT |
| `DashboardClaim_Harness` | @baseclass | 155 | 0 | — | **?** | JANGGAL |
| `DetailCauseOfLoss` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `DetailDominanFactor` | Data-Portal | 316 | 27 | — | **U-6** | DUGAAN |
| `DetailMasterPasalRejected` | Data-Portal | 274 | 15 | — | **U-6** | KUAT |
| `DetailMasterXOL` | Data-Portal | 163 | 9 | — | **U-6** | DUGAAN |
| `DetailSurveyorsInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `DetTypeDocumenBisnis` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `GCNMCatSparepart` | Data-Portal | 85 | 0 | — | **?** | JANGGAL |
| `GCNMMasterSparepartType` | Data-Portal | 419 | 30 | — | **U-6** | DUGAAN |
| `GroupingSparePart_HE` | Data-Portal | 269 | 15 | — | **U-6** | KUAT |
| `Har_LaporanHasilAI` | Data-Portal | 466 | 45 | — | **U-5** | KUAT |
| `Inbox_XOL_Harness` | Data-Portal | 971 | 69 | ya | **U-3** | KUAT |
| `inboxAnalystDoctor_Harness` | Data-Portal | 348 | 12 | ya | **U-3** | DUGAAN |
| `InboxAutoClaim` | Data-Portal | 1664 | 183 | ya | **U-3** | KUAT |
| `InboxClaimNonProp_Harness` | Data-Portal | 1284 | 120 | ya | **U-3** | KUAT |
| `InboxClaimTreaty_Harness` | Data-Portal | 1538 | 69 | ya | **U-3** | KUAT |
| `inboxCompliance_Harness` | Data-Portal | 145 | 0 | ya | **?** | JANGGAL |
| `InboxInvestigator_Harness` | Data-Portal | 456 | 33 | ya | **U-3** | DUGAAN |
| `InboxKomite_Harness` | Data-Portal | 1759 | 216 | ya | **U-3** | KUAT |
| `InboxKomunikasiCabang` | @baseclass | 682 | 84 | ya | **U-3** | DUGAAN |
| `InboxManagerAdmin_Harness` | Data-Portal | 482 | 30 | ya | **U-3** | DUGAAN |
| `InboxPLA_harness` | Data-Portal | 706 | 27 | ya | **U-3** | DUGAAN |
| `InboxPLADLA` | Data-Portal | 1599 | 126 | ya | **U-3** | KUAT |
| `InboxRCVApp_Harness` | @baseclass | 1126 | 126 | ya | **U-3** | KUAT |
| `InboxRegister_Harness` | Data-Portal | 1331 | 144 | ya | **U-3** | KUAT |
| `InboxSalvage` | Data-Portal | 413 | 42 | ya | **U-3** | DUGAAN |
| `InboxSurvey_Harness` | Data-Portal | 1796 | 150 | ya | **U-3** | KUAT |
| `InboxTKA_Harness` | Data-Portal | 774 | 51 | ya | **U-3** | DUGAAN |
| `InputProgress` | Work (konteks klaim) | 251 | 12 | — | **U-4** | PASTI |
| `InputProtection_Harness` | Data-Portal | 483 | 54 | — | **U-4** | DUGAAN |
| `InputReqProtection_Harness` | Data-Portal | 232 | 21 | — | **U-4** | DUGAAN |
| `ListDetTypeDocument` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `ListDocumentObject` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `ListDocumentTravel` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `ListDocumentTypeInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `MasterLoginSurvey` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `MasterPanel_HE` | Data-Portal | 273 | 15 | — | **U-6** | KUAT |
| `MasterProteksiVisibilityData` | Data-Portal | 1011 | 66 | — | **U-6** | DUGAAN |
| `MasterRecovery` | Data-Portal | 361 | 27 | — | **U-6** | DUGAAN |
| `MasterRekening` | Data-Portal | 274 | 15 | — | **U-6** | KUAT |
| `MasterSupplier` | Data-Portal | 322 | 15 | — | **U-6** | DUGAAN |
| `MonitoringSLINKOJK` | Data-Portal | 198 | 3 | — | **U-5** | DUGAAN |
| `OutstandingKlaimperCabang_Harness` | Data-Portal | 469 | 18 | — | **U-5** | KUAT |
| `PNC_MasterTolakKlaim` | Data-Portal | 276 | 15 | — | **U-6** | KUAT |
| `PNCArchiveDokumen` | Data-Portal | 1282 | 135 | — | **U-4** | DUGAAN |
| `PNCInboxAdmin` | @baseclass | 1885 | 276 | ya | **U-3** | KUAT |
| `PNCSearchKlaim` | @baseclass | 574 | 36 | — | **U-4** | DUGAAN |
| `PNCStudyClaim` | Data-Portal | 476 | 42 | — | **U-5** | KUAT |
| `PNCTATReport` | @baseclass | 1127 | 114 | — | **U-5** | KUAT |
| `ProgressClaim_Harness` | @baseclass | 1024 | 93 | — | **U-4** | DUGAAN |
| `RCL_Harness` | Data-Portal | 214 | 9 | — | **U-4** | DUGAAN |
| `RCLPUCL_Harness` | Data-Portal | 184 | 0 | — | **?** | JANGGAL |
| `ReceiveDoucument_Harness` | Data-Portal | 502 | 15 | — | **U-4** | DUGAAN |
| `ReportKPIHarness` | Data-Portal | 1620 | 78 | — | **U-5** | KUAT |
| `SparePart_HE` | Data-Portal | 268 | 15 | — | **U-6** | KUAT |
| `StatusClaimInbox` | Data-Portal | 267 | 15 | ya | **U-6** | KUAT |
| `StatusProgress` | Data-Portal | 267 | 15 | — | **U-5** | DUGAAN |
| `StatusProgress2` | Data-Portal | 267 | 15 | — | **U-5** | DUGAAN |
| `SurveyorsInbox` | Data-Portal | 268 | 15 | ya | **U-6** | KUAT |
| `UserInbox_Harness` | Data-Portal | 805 | 30 | ya | **U-3** | DUGAAN |
| `UserTeknisInbox` | Data-Portal | 270 | 15 | ya | **U-6** | KUAT |
| `View_DetailKlaimCabang_Harness` | Work (konteks klaim) | 734 | 54 | — | **U-4** | PASTI |
| `ViewDetailHasilSurveyorInternal1` | Work (konteks klaim) | 516 | 33 | — | **U-4** | PASTI |
| `ViewPolis` | @baseclass | 431 | 27 | — | **U-4** | DUGAAN |
| `ViewPolis1` | @baseclass | 122 | 0 | — | **?** | JANGGAL |
| `ViewReceiveDocument` | Work (konteks klaim) | 1238 | 132 | — | **U-4** | PASTI |
| `ViewTempDetailAllCase` | Work (konteks klaim) | 493 | 0 | — | **U-4** | PASTI |
| `ViewTempDetailClaim` | Work (konteks klaim) | 537 | 3 | — | **U-4** | PASTI |
| `ViewTempDetailReqDocument` | Work (konteks klaim) | 533 | 0 | — | **U-4** | PASTI |

**Total: 74 harness** — 7 berkelas Work · 9 `@baseclass` · 58 `Data-Portal` · 26 di antaranya bernama *Inbox*.

> Tabel di atas **dibangun ulang** oleh `docs/tools/build-inventaris-harness.js` langsung dari
> direktori `Harness/`, termasuk kolom **Usulan modul** dan **Keyakinan** yang dihitung dari
> aturan di `usul()`. Jangan menyuntingnya dengan tangan — koreksi Work Owner dicatat di
> Decision Log, lalu aturannya disesuaikan di sini agar tabel tetap dapat dibangun ulang.
