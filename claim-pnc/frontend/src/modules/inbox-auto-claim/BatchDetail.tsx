import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { AutoClaimResult, ErrorCode, type AutoClaimLine } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useAutoClaimLineList } from './api'

type Props = {
  source: string
  company: string
  companyName: string
  batch: string
  onClose: () => void
}

/**
 * Rincian satu batch — pengganti tombol DETAIL pada layar lama.
 *
 * Di Pega ia membuka harness tersendiri (`Detail_AUTOCLAIM_Harness`) lewat aksi
 * `showHarness`. Harness itu TIDAK ADA di export (R-16), sehingga susunan kolomnya tidak
 * diketahui; yang ditampilkan di sini adalah seluruh kolom TMP_BATCH_AUTO_CLAIM yang
 * terbaca dari kueri lain.
 *
 * Ia dibuka sebagai panel DI BAWAH grid, bukan layar terpisah. Alasannya bukan selera:
 * petugas membandingkan rincian dengan angka ringkasnya — berapa berhasil, berapa gagal —
 * dan layar terpisah memaksa ia bolak-balik untuk itu.
 */
export function BatchDetail({ source, company, companyName, batch, onClose }: Props) {
  const [page, setPage] = useState(1)
  const list = useAutoClaimLineList(source, company, batch, page)

  const columns: Column<AutoClaimLine>[] = [
    {
      key: 'nomor_polis',
      title: 'No Polis',
      width: '12rem',
      value: (row) => `${row.nomor_polis} ${row.prod_ke}`,
      render: (row) => (
        <span>
          {row.nomor_polis}
          <span className="ml-2 text-xs text-slate-500">prod ke {row.prod_ke}</span>
        </span>
      ),
    },
    {
      key: 'tanggal',
      title: 'Kejadian / Lapor',
      width: '11rem',
      value: (row) => `${row.tanggal_kejadian} ${row.tanggal_lapor}`,
      render: (row) => (
        <span className="whitespace-nowrap">
          {row.tanggal_kejadian}
          <span className="mx-1 text-slate-400">→</span>
          {row.tanggal_lapor}
        </span>
      ),
    },
    {
      // Kolom kedua pada grid rincian Pega
      // (`InboxAutoClaim/BrowseClaimSPK_detail_AutoClaim-SQL.xml`). Ia juga bagian
      // PRIMARY KEY tabelnya, sehingga dua baris berpolis sama hanya dapat dibedakan
      // lewat kolom ini.
      key: 'tanggal_proses',
      title: 'Tgl Proses',
      width: '7rem',
      value: (row) => row.tanggal_proses,
      render: (row) => <span className="whitespace-nowrap tabular-nums">{row.tanggal_proses}</span>,
    },
    {
      key: 'nilai',
      title: 'Nilai Klaim',
      width: '11rem',
      alignRight: true,
      // `mata_uang` berisi KODE hasil lookup ke POOLDATA.CURRENCY, bukan id yang
      // tersimpan di kolomnya. Versi pertama layar ini menampilkan id-nya — pengguna
      // melihat "1", bukan "IDR".
      value: (row) => `${row.mata_uang} ${row.nilai_klaim}`,
      render: (row) => (
        <span className="whitespace-nowrap tabular-nums">
          <span className="mr-1 text-xs text-slate-500">{row.mata_uang}</span>
          {formatMoney(row.nilai_klaim)}
        </span>
      ),
    },
    {
      key: 'nomor_klaim',
      title: 'No Klaim',
      width: '10rem',
      value: (row) => `${row.nomor_klaim} ${row.nomor_aksep}`,
      render: (row) =>
        row.nomor_klaim === '' ? (
          <span className="text-slate-400">—</span>
        ) : (
          <span>
            {row.nomor_klaim}
            {row.nomor_aksep !== '' && (
              <span className="ml-2 text-xs text-slate-500">{row.nomor_aksep}</span>
            )}
          </span>
        ),
    },
    {
      key: 'hasil',
      title: 'Hasil',
      width: '16rem',
      value: (row) => `${row.hasil} ${row.keterangan}`,
      render: (row) => <ResultBadge line={row} />,
    },
  ]

  return (
    <section className="mt-6">
      <header className="mb-3 flex flex-wrap items-baseline justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-slate-900">
            Rincian batch {batch}
            <span className="ml-2 font-normal text-slate-600">
              · {companyName === '' ? company : companyName}
            </span>
          </h2>
          <p className="text-sm text-slate-600">
            Baris klaim di dalam batch ini beserta hasil pemrosesannya.
          </p>
        </div>
        <Button tone="halus" onClick={onClose}>
          Tutup rincian
        </Button>
      </header>

      {list.isError ? (
        <ErrorMessage
          title={detailErrorTitle(list.error)}
          description={detailErrorDescription(list.error)}
          tone={
            list.error instanceof APIError && list.error.kode === ErrorCode.notFound
              ? 'penolakan'
              : 'gangguan'
          }
        />
      ) : (
        <DataTable
          columns={columns}
          rows={list.data?.baris ?? []}
          // Kunci barisnya mengikuti PRIMARY KEY tabelnya —
          // (NOPOLIS, TGLPROSES, TGLKEJADIAN) menurut Database/CREATE_TABLE_1.sql —
          // bukan susunan kolom yang kebetulan terasa unik.
          rowKey={(row) => `${row.nomor_polis}-${row.tanggal_proses}-${row.tanggal_kejadian}`}
          isLoading={list.isPending}
          // Pencarian disembunyikan: ia hanya akan menyaring HALAMAN YANG SEDANG TAMPIL,
          // dan pengguna mengira ia mencari ke seluruh batch. Baris yang dicarinya ada di
          // halaman lain dan tidak akan pernah muncul.
          hideSearch
          emptyMessage="Batch ini tidak memuat baris."
          pagination={{
            page: list.data?.paginasi.halaman ?? page,
            size: list.data?.paginasi.ukuran ?? 15,
            total: list.data?.paginasi.total ?? 0,
            totalPage: list.data?.paginasi.total_halaman ?? 0,
            onPageChange: setPage,
            isLoading: list.isFetching,
          }}
        />
      )}
    </section>
  )
}

/**
 * Lencana hasil dibedakan WARNA dan TEKS, tidak pernah warna saja.
 *
 * Tiga keadaan ini menentukan tindakan petugas berikutnya, dan pengguna buta warna tidak
 * boleh kehilangan pembedaannya (sistem desain: "pembedaan penting tidak pernah hanya
 * warna").
 */
function ResultBadge({ line }: { line: AutoClaimLine }) {
  if (line.hasil === AutoClaimResult.succeeded) {
    return (
      <span className="inline-flex items-center rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-800 ring-1 ring-emerald-200 ring-inset">
        Berhasil
      </span>
    )
  }
  if (line.hasil === AutoClaimResult.pending) {
    return (
      <span className="inline-flex items-center rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-700 ring-1 ring-slate-300 ring-inset">
        Belum diproses
      </span>
    )
  }
  return (
    <span className="flex flex-col gap-1">
      <span className="inline-flex w-fit items-center rounded-full bg-red-50 px-2.5 py-0.5 text-xs font-medium text-red-800 ring-1 ring-red-200 ring-inset">
        Gagal
      </span>
      {/* Pesan kegagalannya ditampilkan apa adanya: ia ditulis pemrosesan dan itulah satu-
          satunya keterangan tentang apa yang perlu diperbaiki. */}
      {line.keterangan !== '' && <span className="text-xs text-slate-600">{line.keterangan}</span>}
    </span>
  )
}

/**
 * formatMoney menampilkan nilai uang dengan pemisah ribuan.
 *
 * Nilai aslinya TIDAK diubah — yang diubah hanya tampilannya, sesuai `I-12`: nilai uang
 * disimpan presisi penuh dan dibulatkan hanya saat ditampilkan. Bagian desimalnya dibawa
 * apa adanya, bukan dipaksa dua angka, supaya nilai berdesimal panjang tidak terlihat
 * berubah.
 *
 * Nilai yang bukan angka dikembalikan apa adanya: baris lama dapat memuat apa saja, dan
 * menampilkan "NaN" menyembunyikan isinya yang sebenarnya.
 */
function formatMoney(value: string): string {
  const [whole, fraction] = value.split('.')
  if (whole === undefined || !/^-?\d+$/.test(whole)) return value

  const formatted = Number(whole).toLocaleString('id-ID')
  return fraction === undefined ? formatted : `${formatted},${fraction}`
}

function detailErrorTitle(error: unknown): string {
  if (error instanceof NetworkError) return 'Server Claim PNC tidak dapat dihubungi'
  if (error instanceof APIError && error.kode === ErrorCode.notFound) {
    return 'Batch ini tidak ada lagi'
  }
  return 'Rincian batch tidak dapat dimuat'
}

function detailErrorDescription(error: unknown): string {
  if (error instanceof NetworkError) return 'Periksa koneksi jaringan Anda, lalu muat ulang.'
  if (error instanceof APIError && error.kode === ErrorCode.notFound) {
    return 'Mungkin batch-nya sudah dibersihkan. Tutup rincian ini lalu muat ulang daftarnya.'
  }
  return 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.'
}
