---
title: "TKT-S4-002 — Empat layanan REST masuk dan otentikasinya"
labels: [modul::S-4, tipe::keamanan, status::needs-info, prioritas::tinggi, gelombang::5]
milestone: "Gelombang 5 — Nilai dan pihak luar"
epic: "Migrasi Claim PNC"
---

# TKT-S4-002 — Empat layanan REST masuk dan otentikasinya

Status: needs-info
Kesiapan: **terhalang keputusan lingkup**
Modul: **S-4 Integrasi Sistem Luar** · Gelombang: 5 · Bergantung pada: TKT-F1-004, TKT-B07-002
Requirement: FR-S4    Keputusan: D-25    ADR: 0017, 0023    Risiko: R-18
Rule Pega yang digantikan: `Service REST/KomiteAcceptAdjustment-REST.xml` · `Service REST/KomiteAcceptAdjustmentPA-REST.xml` · `Service REST/RecivedDataandAttachmentLelangASMSimasbid-REST.xml` · `Service REST/RequestCreateClaimCredit2-REST.xml`
Peran penguji gerbang 2: **PncManagerAdmin** (komite) dan **PncAdmin**

## Hasil yang diharapkan (dan nilai bisnisnya)

Empat pintu masuk yang selama ini ada tetapi **belum pernah diaudit** menjadi pintu yang diketahui:
siapa boleh mengetuk, apa yang boleh diminta, dan apa yang tercatat.

Nilai bisnisnya terletak pada dua dari empat layanan itu: **`KomiteAcceptAdjustment` dan
`KomiteAcceptAdjustmentPA` menerima persetujuan komite dari sistem lain.** Artinya keputusan
persetujuan klaim — inti kendali `B-7` — **dapat masuk dari luar aplikasi**. Siapa pun yang dapat
memanggil dua layanan itu dapat menyetujui klaim.

## Ruang lingkup

- Empat endpoint masuk dengan kontrak yang **setara dengan yang berlaku sekarang**.
- **Otentikasi dan otorisasi pemanggil**, diperiksa di server (`ADR-0023`).
- Pencatatan setiap panggilan masuk: siapa memanggil, klaim mana, apa yang diubah, kapan —
  tersambung ke jejak audit `S-5`.
- Penegakan aturan komite yang sama seperti jalur layar: persetujuan lewat API **tidak boleh
  melewati** jenjang kumulatif `B-7`.

## Non-goal

- **Tidak** menambah layanan masuk baru.
- **Tidak** mengubah kontrak yang sudah dipakai sistem lawan tanpa kesepakatan.

## Yang kurang dan siapa yang bisa melengkapinya

| Yang kurang | Pemilik | Kenapa menahan |
|---|---|---|
| **Keempat layanan masuk ini masuk lingkup migrasi?** | **Work Owner** | Permukaan ini **belum pernah masuk hitungan** `FR-S4` maupun `D-25`. Bila masuk, lingkup `S-4` bertambah; bila tidak, harus jelas apa yang terjadi pada pemanggilnya saat Pega dimatikan |
| **Siapa sistem pemanggil keempat layanan itu, dan atas dasar apa mereka berwenang?** | **Work Owner + tim integrasi** | Tanpa daftar pemanggil yang sah, otorisasi tidak dapat dirumuskan |
| **Persetujuan komite dari luar tunduk jenjang kumulatif yang sama?** | **Work Owner** (`D-52`, `D-70`) | Bila tidak, ada **dua aturan persetujuan berbeda** untuk klaim yang sama — dan yang satu memintas yang lain |
| **Layanan masuk memakai otentikasi apa sekarang?** | **Tim Infra/Security** (`R-18`) | Rule-nya ada di export, tetapi kelayakan mekanismenya belum dinilai |

## Acceptance criteria

- [ ] Keempat endpoint masuk berfungsi dengan kontrak setara — diuji per endpoint.
- [ ] Panggilan **tanpa otentikasi ditolak** — diuji pada keempatnya: bukan `200`.
- [ ] Panggilan dari pemanggil yang tidak berwenang ditolak — diuji.
- [ ] **Persetujuan komite lewat API tunduk jenjang kumulatif yang sama dengan layar** — diuji:
      klaim yang belum cukup jenjangnya **tidak berpindah status** walau API memanggilnya.
- [ ] Setiap panggilan masuk **tercatat di jejak audit** dengan identitas pemanggil — diuji.
- [ ] Panggilan berulang dengan isi sama **tidak menggandakan efeknya** — diuji.
- [ ] Gerbang 2: UAT **PncManagerAdmin** untuk dua layanan komite, terpisah dari dua lainnya.

## Dependency / Blocked by

`TKT-F1-004` · `TKT-B07-002` (aturan jenjang komite) · `TKT-S5-001` (jejak audit).
**Terhalang keputusan lingkup Work Owner.**

## Constraint keamanan, data, operasional

- **Dua layanan ini dapat menyetujui klaim dari luar aplikasi.** Ini permukaan dengan dampak
  finansial langsung, dan ia **belum pernah diaudit**.
- Otorisasi wajib diperiksa **di server** (`D-59`, `ADR-0023`) — tidak boleh bersandar pada
  anggapan bahwa hanya sistem internal yang tahu alamatnya.
- Selama Pega dan Go berjalan berdampingan (`D-05`), **kedua sistem dapat menerima panggilan yang
  sama**. Aturan penulis tunggal (`P-1`) harus tetap berlaku pada tabel yang mereka ubah.

## Migrasi skema / rollout / rollback

Tidak mengubah skema; memakai tabel klaim dan jejak audit yang ada.

**Rollout:** pengalihan pemanggil dari Pega ke Go menuntut **sistem lawan mengubah alamat** —
ini koordinasi dengan pihak luar, bukan pekerjaan sepihak.

**Rollback:** mengembalikan alamat ke Pega. Persetujuan yang sudah masuk lewat Go **tetap ada** di
data — rollback tidak membatalkannya.

## Rencana verifikasi

> **Rencana — belum dijalankan.**

```bash
go test ./internal/adapter/http/inbound/... -run TestTanpaOtentikasiDitolak
go test ./internal/adapter/http/inbound/... -run TestKomiteAPITundukJenjangKumulatif
go test ./internal/adapter/http/inbound/... -run TestPanggilanBerulangTidakMenggandakan
go test ./internal/app/audit/... -run TestPanggilanMasukTercatat
```

## Bukti ruang lingkup

| Klaim | Sumber |
|---|---|
| Empat layanan REST masuk | direktori `Service REST/` (4 berkas) |
| Dua di antaranya menerima persetujuan komite | `docs/verifikasi-bukti-adr.md:2707-2715` |
| Permukaan masuk belum pernah masuk hitungan `FR-S4`/`D-25` | `docs/verifikasi-bukti-adr.md:2715` |
| Jenjang komite kumulatif; pita nilai hanya Non-MBU | `D-52` · `D-70` |
| Otorisasi diperiksa di server | `D-59` · `ADR-0023` |

## Comments
