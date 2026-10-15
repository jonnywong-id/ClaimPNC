import { useMemo } from 'react'

import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useData } from './api'
import type { Baris, Kolom, Penyaring, Segmen } from './types'

type Props = {
  segmen: Segmen
  penyaring: Penyaring
  halaman: number
  onHalamanChange: (halaman: number) => void

  /** Pencarian sudah ditekan sekali; sebelum itu tabel tidak menembak permintaan. */
  aktif: boolean

  kunciBaris: string
}

/**
 * SegmentPanel menggambar grid satu segmen dari KATALOG kolom, bukan dari daftar kolom
 * yang ditulis di sini.
 *
 * # Kenapa kolomnya tidak ditulis di berkas ini
 *
 * Segmen F06 punya 38 kolom dan segmen D01 punya 20. Menuliskannya di TypeScript berarti
 * 58 definisi yang harus berubah bersamaan dengan katalog Go — dan yang tertinggal tidak
 * menghasilkan galat apa pun, hanya kolom laporan regulator yang diam-diam bergeser.
 *
 * Dengan katalog, satu perubahan di `columns.go` sampai ke layar tanpa menyentuh berkas
 * ini sama sekali.
 */
export function SegmentPanel({
  segmen,
  penyaring,
  halaman,
  onHalamanChange,
  aktif,
  kunciBaris,
}: Props) {
  const query = useData(penyaring, halaman, aktif)

  const columns = useMemo(
    () => segmen.kolom.map((kolom) => toColumn(kolom)),
    [segmen.kolom],
  )

  const rows = query.data?.baris ?? []
  const meta = query.data?.meta

  return (
    <div className="mt-4 space-y-3">
      <DataTable<Baris>
        columns={columns}
        rows={rows}
        rowKey={(row) => row[kunciBaris] ?? ''}
        label={`Tabel ${segmen.label}`}
        title={segmen.label}
        description={keterangan(segmen)}
        isLoading={query.isLoading || query.isFetching}
        // Pencarian dan pengurutan bawaan DIMATIKAN: datanya dipaginasi server, dan
        // menyaring satu halaman dari sepuluh akan memberi tahu pengguna bahwa sebuah
        // klaim tidak ada padahal ia di halaman lain.
        hideSearch
        showHeaderWhenEmpty
        emptyMessage={
          aktif
            ? 'Tidak ada data yang cocok dengan penyaring ini.'
            : 'Tekan "Cari Data" untuk menampilkan isinya.'
        }
        error={
          query.isError ? (
            <ErrorMessage
              title="Data tidak dapat dimuat"
              description={
                query.error instanceof Error
                  ? query.error.message
                  : 'Terjadi kesalahan saat membaca data.'
              }
              tone="gangguan"
            />
          ) : undefined
        }
        // Bilah halaman muncul hanya SESUDAH pencarian pertama. Sebelum itu ia akan
        // menampilkan "0–0 dari 0" pada layar yang memang belum menampilkan apa pun —
        // terbaca sebagai hasil kosong, padahal belum ada yang dicari.
        //
        // Disebar (`...`), bukan dikirim bernilai `undefined`: `exactOptionalPropertyTypes`
        // membedakan "prop tidak dikirim" dari "prop dikirim bernilai undefined", dan
        // DataTable hanya menerima yang pertama.
        {...(meta
          ? {
              pagination: {
                page: meta.halaman,
                size: meta.ukuran,
                total: meta.total,
                totalPage: meta.total_halaman,
                onPageChange: onHalamanChange,
                isLoading: query.isFetching,
              },
            }
          : {})}
      />
    </div>
  )
}

/**
 * toColumn mengubah satu entri katalog menjadi kolom tabel.
 *
 * # Kolom tanpa sumber TIDAK ditandai — dan itu keputusan, bukan kelalaian
 *
 * Versi pertama menggambar sel kosongnya sebagai tanda pisah redup beserta keterangan
 * properti asalnya, supaya "datanya memang kosong" dapat dibedakan dari "kolomnya belum
 * punya sumber".
 *
 * Work Owner memutuskan pada 2026-09-26 bahwa penanda itu **dibuang**: layarnya mengikuti
 * Pega apa adanya (`D-13`), dan di Pega sel itu kosong biasa.
 *
 * Konsekuensinya diterima secara sadar: 30 dari 38 kolom segmen F06 akan tampil kosong
 * tanpa ada apa pun di layar yang menyatakan sebabnya. Keterangannya pindah ke
 * dokumentasi — `docs/catatan-pengembangan.md` §55.4 — bukan hilang.
 */
function toColumn(kolom: Kolom): Column<Baris> {
  const column: Column<Baris> = {
    key: kolom.kunci,
    title: kolom.judul,
    value: (row) => row[kolom.kunci] ?? '',
    // Pengurutan dimatikan seluruhnya: barisnya dipaginasi server, dan mengurutkan satu
    // halaman dari sepuluh bukan pengurutan.
    noSort: true,
  }

  if (kolom.angka) column.alignRight = true

  return column
}

/**
 * keterangan menjelaskan apa yang dibaca segmen ini.
 *
 * Kedua teks ini sempat TERTUKAR, mengikuti pemetaan segmen yang keliru. Struktur
 * pelaporan SLIK OJK — dan layar Pega produksi — menempatkan D01 sebagai data DEBITUR
 * dan F06 sebagai data FASILITAS.
 */
function keterangan(segmen: Segmen): string {
  return segmen.kode === 'D01'
    ? 'Data identitas debitur perorangan — segmen D01 laporan SLIK OJK.'
    : 'Data fasilitas kredit yang sudah tersusun sebagai laporan — segmen F06.'
}
