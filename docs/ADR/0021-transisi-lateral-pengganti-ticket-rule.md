# 0021 — Bangun ulang transisi lateral (Ticket rule) sebagai perpindahan tahap yang eksplisit

Status: Proposed
Tanggal keputusan: —    Tanggal dokumen: 2026-09-14
Sifat: retrospective
Pemilik keputusan: Tim Pega (artefak) → Work Owner (perilaku)
Jejak bukti: `FR-W2` (`docs/BRD.md:518`) | `docs/verifikasi-bukti-adr.md` §5 | lihat tabel untuk `berkas:baris`
Terkait: CONTEXT.md#Lompatan-Lateral, ADR-0019, modul `B-7`, `B-11`, `B-13`, `B-14`

> **Belum diputuskan.** Berkas ini memuat konteks, opsi, dan apa yang terhalang — **tanpa bagian
> `Keputusan`**. Jangan dijadikan dasar implementasi.

> **Peringatan istilah.** "Ticket rule" di sini adalah mekanisme Pega. Kata **"tiket"** di seluruh
> dokumen proyek ini berarti **tiket pekerjaan** di `docs/ticketing/`.

## Konteks

Alur klaim tidak selalu berjalan maju. Pada keadaan tertentu klaim **melompat** ke tahap yang
bukan tahap berikutnya: kembali dari Komite ke petugas estimasi begitu seluruh anggota komite
menyetujui, atau berpindah ke PUCL saat ditolak pada Group Panel tertentu.

Di Pega mekanisme ini bernama **Ticket rule**: sebuah nama tujuan dipasang pada satu tahap, lalu
dipicu dari tempat lain.

Empat flow merujuk **17 nama Ticket rule**; folder `Ticket/` hanya berisi **8 rule**.

**Delapan yang ada:** `AcceptanceKomite` · `Akp_PNCInputProtection` · `SendToEstAdmin` ·
`SendToEstTravel` · `SendToEstimatorPA` · `SendToPICTravel` · `SendtoAnalysator` ·
`TC_PNCInputProtection`

**Sembilan yang dirujuk tanpa rule:**

| Ticket rule | Dirujuk dari | Klasifikasi |
|---|---|---|
| `SendToReceiveDocument` | `Flow/InputReceiveDocument.xml:690` | minta ke Tim Pega |
| `komiteAccept_ticket` | `Flow/Komite_Flow.xml:796` | minta ke Tim Pega |
| `KomiteAssign_ticket` | `Flow/Komite_Flow.xml:974` | minta ke Tim Pega |
| `RCLDokter` | `Flow/Register_Flow.xml:2954` | minta ke Tim Pega |
| `setToRegister_ticket` | `Flow/Register_Flow.xml:3052` | minta ke Tim Pega |
| `SendtoPUCL` | `Flow/Register_Flow.xml:3197` | minta ke Tim Pega |
| `SendToInvestigator` | `Flow/Register_Flow.xml:3624` | minta ke Tim Pega |
| `CompliancePNC` | `Flow/Register_Flow.xml:3844` | minta ke Tim Pega |
| `Status-Resolved` | `Flow/Register_Flow.xml:3004` | **bawaan Pega** — dipicu activity OOTB `Work-.Resolve` (`Activity/Resolve-Act.xml:1331`, nama ticket `:1343`); tidak perlu diminta |

Jadi yang benar-benar hilang dan harus digali ulang: **8 rule custom**.

**Masalah kedua lebih berat daripada yang pertama.** Dari 17 nama itu, **hanya tiga transisi yang
punya pemicu** di seluruh 902 activity + 29 flow action:

| Ticket rule | Dipicu dari | Kondisi |
|---|---|---|
| `SendToEstimatorPA` | `Activity/KomitePost_Adjustment-Act.xml:15083` | precondition `IsPA` (`:15033`); deskripsi *"Pindahin ke inputor apabila semua komite sudah aksep"* (`:14993`) |
| `SendToEstimatorPA` | `Activity/SetListComiteeClaimPerObjAdj-Act.xml:28380` | precondition `IsPA` (`:28426`); *"Back to estimator"* |
| `SendtoPUCL` | `Activity/KomitePost_Reject-Act.xml:3172` | `pyWorkCover.Policy.Quotation.GroupPanel=="002"` (`:3121`) |

**Empat belas nama sisanya tidak punya pemicu apa pun di export.** Metode pemicu yang dicari dan
jumlah temuannya: `Obj-Set-Tickets` **1** (hanya `Activity/Resolve-Act.xml:1331`, dan Pega
menandainya *deprecated* di `:153`) · `call SetTicket` **4** · `<Ticket>` sebagai parameter **4** ·
`<SetTicketNames>` **1** · rujukan ticket di `Flow Action/` **0**.

Artinya: meski Tim Pega mengirim 8 rule yang hilang, **kapan** masing-masing dipicu tetap tidak
diketahui.

**Koreksi atas `FR-W2`.** Angka "11 ticket" di `docs/BRD.md:518` benar untuk `Register_Flow` saja
— 12 shape ticket dikurangi `Status-Resolved` yang bawaan. Angka untuk seluruh aplikasi:
**17 dirujuk, 8 ada, 9 tanpa rule.**

## Opsi yang dipertimbangkan

**Opsi 1 — Minta artefak lengkap ke Tim Pega lebih dulu**, baru rancang penggantinya.
Paling akurat; bergantung penuh pada pihak luar dan pada apakah pemicunya memang ada di ruleset
yang tidak ikut export.

**Opsi 2 — Gali ulang perilakunya bersama pengguna bisnis**, lalu tetapkan aturan perpindahan
tahap sebagai aturan baru yang eksplisit.
Tidak bergantung pada Tim Pega, tetapi **melepas jaminan kesetaraan**: perilaku hasil galian belum
tentu sama dengan yang berjalan di produksi, dan gerbang 1 kehilangan pembandingnya.

**Opsi 3 — Bawa hanya tiga transisi yang terbukti punya pemicu**, sisanya dianggap tidak aktif
sampai ada bukti sebaliknya.
Paling jujur terhadap bukti, tetapi berisiko menghapus jalur yang ternyata dipakai — dan
kegagalannya baru terlihat sebagai klaim yang mandek di produksi.

## Konsekuensi bila dibiarkan tidak diputuskan

- Tiket `B-7`, `B-11`, `B-13`, dan `B-14` tidak dapat ditulis lengkap: perpindahan tahap adalah
  bagian inti alur keempat modul itu.
- Uji kesetaraan untuk keempat modul tidak dapat dirancang, karena perilaku yang dibandingkan
  belum diketahui seluruhnya.
- Bila ternyata ada transisi aktif yang tidak terbawa, akibatnya adalah klaim yang berhenti di
  satu tahap tanpa ada yang tahu mengapa — kelas cacat yang paling mahal ditemukan setelah rilis.

## Pertanyaan terbuka

1. **Apakah 8 Ticket rule custom yang hilang tersedia di sistem Pega produksi?** Pemilik: Tim Pega.
   Ini bagian dari permintaan export ulang `D-39`.
2. **Di mana pemicu 14 nama yang tidak punya pemicu berada** — di ruleset lain yang tidak ikut
   export, atau memang sudah mati? Pemilik: Tim Pega.
3. **Bila jawabannya "sudah mati", apakah Work Owner menyetujui transisi itu tidak dibawa?**
   Pemilik: Work Owner.
4. Apakah `CompliancePNC` yang dipakai serentak sebagai **nama Ticket rule dan nama workbasket**
   memang satu hal yang sama? Pemilik: Work Owner.
