# B-4 · Spreading Reasuransi & Koasuransi

| | |
|---|---|
| **Nama di sistem lama** | bagian layar **Input Register** — blok spreading pada `Activity/InputRegister_act-Act.xml`; inbox `InboxClaimTreaty_Harness`, `InboxClaimNonProp_Harness`, `Inbox_XOL_Harness` |
| **Kode modul** | `B-4` |
| **Gelombang** | 3 — Jalur klaim inti |
| **Ukuran** | bagian dari 69 activity PLA/DLA/spreading |
| **Bergantung pada** | `B-3` Objek & Coverage |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Membagi risiko satu Coverage kepada para penanggung: treaty, Fac Out, XOL, koasuransi, dan share
ASM sendiri. **Total pembagiannya wajib 100%.**

## Aturan yang mengikat

| Aturan | Isi |
|---|---|
| Total 100% | diuji `ROUND(SUM(share),4) BETWEEN 99.9999 AND 100.0001` (`ADR-0016`) |
| Fac Out | bila ada spreading Fac Out, data Fac Offer **wajib ada** |
| Group Panel `003` | Fac Offer wajib menyertakan Object Name |
| Ex-Gratia | klaim bertanda Ex-Gratia mengubah treaty `OR` menjadi `ORS` otomatis |
| Baris terhapus | spreading bertanda hapus **dibuang sebelum** perhitungan |

## Cacat yang ditutup modul ini

Toleransi 100% di sistem lama adalah **pencocokan substring**, bukan perbandingan angka:

```
@contains(local.totalspreading, 100.0) || local.totalspreading == 100 || @contains(local.totalspreading, 99.99)
```
`Activity/InputRegister_act-Act.xml:13183`

Akibatnya total **`199.99`** dan **`1100.0`** ikut lolos — keduanya memuat potongan teks yang
dicari. Ini butir 1 pada 13 perbaikan eksplisit `P-5`.

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Keputusan** | Berapa desimal share yang sah? | **Work Owner** |
| **Keputusan** | Group Panel `003`: cek **semua baris** Fac Offer atau hanya baris pertama? | **Work Owner** |
| **Keputusan** | Pemetaan kode `10001` / `10007` / `10015` | **Work Owner** |
| **Keputusan** | **`UPDATEREAS` ber-4 `COMMIT` — boleh ditulis ulang?** | **Work Owner** (`D-68` menjawab prinsipnya; konfirmasi per objek) |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B04-001](issues/01-pembagian-share-dan-validasi-total-100.md) | Pembagian share dan validasi total 100% | `needs-info` |
| [TKT-B04-002](issues/02-fac-out-fac-offer-dan-ex-gratia.md) | Fac Out, Fac Offer, dan Ex-Gratia | `needs-info` |
| [TKT-B04-003](issues/03-penulisan-spreading-atomik.md) | Penulisan spreading sebagai satu transaksi | `needs-info` |
