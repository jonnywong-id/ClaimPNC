import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { useInboxServiceCenterDetail } from './api'
import type { ClaimDetail, FieldGroup, FieldRef, ProgressNote } from './types'

/**
 * Rincian satu klaim portal rekanan — pengganti `Section/InputClaimServiceCenter-Section.xml`.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Tujuh kelompok, berurutan seperti judul kontainer di section Pega: General Information,
 * Informasi Unit, Informasi Perbaikan, Estimasi Date, Estimasi Biaya, Harga Approve Komite,
 * dan Accessories Unit. Judul isiannya diambil dari `pyCaption` apa adanya (`D-13`), termasuk
 * yang berbahasa Inggris dan yang ditulis huruf besar semua.
 *
 * # Kenapa kelompok dan judulnya datang dari server
 *
 * Karena keduanya hasil pembacaan export Pega, dan tempat pembacaan itu tercatat adalah
 * backend (`internal/inboxservicecenter/detail.go`). Menyalinnya ke sini berarti 83 judul
 * hidup di dua tempat.
 *
 * # Layar ini BACA-SAJA, dan itu bukan kelalaian
 *
 * Jalur tulisnya bermuara pada `POOLDATA.PEGA_PORTAL_REKANAN`, dan source-nya sudah diterima
 * 2026-09-28. Yang menahan bukan lagi artefak melainkan `P-1`: selama masa paralel, satu
 * tabel hanya boleh ditulis satu sistem, dan `T_KLAIM_PORTAL_REKANAN` hari ini ditulis Pega.
 */
export function ServiceCenterDetailPage() {
  const { id = '' } = useParams<{ id: string }>()
  const portal = useSelectedPortal((state) => state.alias)
  const detail = useInboxServiceCenterDetail(id)

  if (portal === null) {
    return (
      <PageFrame id={id}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Klaim portal rekanan milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (detail.isPending) {
    return (
      <PageFrame id={id}>
        <p className="mt-6 text-sm text-slate-600" role="status">
          Memuat rincian klaim…
        </p>
      </PageFrame>
    )
  }

  if (detail.isError) {
    const notFound = detail.error instanceof APIError && detail.error.status === 404

    return (
      <PageFrame id={id}>
        <ErrorMessage
          title={notFound ? 'Klaim tidak ditemukan' : 'Rincian tidak dapat dimuat'}
          description={messageOf(detail.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  const claim = detail.data.klaim

  return (
    <PageFrame id={id} claim={claim}>
      <div className="mt-6 space-y-6">
        {detail.data.kelompok.map((group) => (
          <FieldGroupCard key={group.kode} group={group} claim={claim} />
        ))}

        <ProgressHistory notes={detail.data.riwayat_progres} />

        <PartDetailNote raw={claim['detail_part_json']} />
      </div>
    </PageFrame>
  )
}

function PageFrame({
  id,
  claim,
  children,
}: {
  id: string
  claim?: ClaimDetail
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        {/*
          Tautan kembali digambar SEBELUM judul, bukan sebagai tombol di kanan: layar ini
          hanya dapat dicapai dari daftar, dan jalan pulang adalah hal pertama yang dicari
          pengguna ketika ia membuka klaim yang keliru.
        */}
        <Link
          to="/inbox-service-center"
          className="text-sm text-blue-700 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
        >
          ← Kembali ke Inbox Service Center
        </Link>

        <h1 className="mt-2 text-xl font-semibold text-slate-900">Rincian Klaim {id}</h1>

        {claim && (
          <p className="mt-1 text-sm text-slate-600">
            {textOf(claim['nasabah']) || textOf(claim['customer_name']) || 'Nasabah tidak tercatat'}
            {' · '}
            {textOf(claim['status_approval_label']) || 'Status persetujuan tidak tercatat'}
            {' · '}
            {textOf(claim['repair_status_label']) || 'Status perbaikan tidak tercatat'}
          </p>
        )}
      </header>
      {children}
    </div>
  )
}

/**
 * Satu kelompok isian.
 *
 * Digambar sebagai daftar deskripsi, bukan tabel: isinya pasangan label dan nilai, dan tabel
 * dua kolom yang barisnya tidak dapat diurutkan hanya meniru bentuk tabel tanpa manfaatnya.
 */
function FieldGroupCard({ group, claim }: { group: FieldGroup; claim: ClaimDetail }) {
  return (
    <section className="rounded-kartu border border-slate-200 bg-white">
      <h2 className="border-b border-slate-200 px-4 py-3 text-sm font-semibold text-slate-800">
        {group.judul}
      </h2>

      <dl className="grid grid-cols-1 gap-x-6 gap-y-3 px-4 py-4 sm:grid-cols-2 lg:grid-cols-3">
        {group.isian.map((field) => (
          <Field key={field.kunci} field={field} claim={claim} />
        ))}
      </dl>
    </section>
  )
}

function Field({ field, claim }: { field: FieldRef; claim: ClaimDetail }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium tracking-wide text-slate-500">{field.judul}</dt>
      {/*
        `break-words` supaya isian panjang — deskripsi kerusakan, alasan pembatalan —
        membungkus alih-alih melebarkan kolom dan merusak tata letak kelompoknya.
      */}
      <dd className="mt-0.5 text-sm break-words text-slate-900">{cellText(claim, field)}</dd>
    </div>
  )
}

/**
 * Riwayat catatan progres.
 *
 * Terlama di atas, mengikuti `ORDER BY INSERTDATE ASC` pada kueri lamanya — riwayat memang
 * dibaca dari awal.
 */
function ProgressHistory({ notes }: { notes: ProgressNote[] }) {
  const columns: Column<ProgressNote>[] = [
    {
      key: 'tanggal',
      title: 'Tanggal',
      value: (row) => (row.tanggal ? formatDate(row.tanggal) : '—'),
      width: '12rem',
    },
    { key: 'catatan', title: 'Catatan', value: (row) => row.catatan || '—' },
    { key: 'oleh', title: 'Oleh', value: (row) => row.oleh || '—', width: '14rem' },
  ]

  return (
    <DataTable<ProgressNote>
      columns={columns}
      rows={notes}
      rowKey={(row) => `${row.tanggal ?? ''}|${row.oleh}|${row.catatan}`}
      title="Riwayat Progres"
      label="Riwayat progres klaim"
      hideSearch
      emptyMessage="Belum ada catatan progres untuk klaim ini."
    />
  )
}

/**
 * Isi kolom `DETAILPART`, ditampilkan APA ADANYA.
 *
 * Bentuk JSON-nya belum pernah dibaca dari data sungguhan. Menguraikannya berdasarkan tebakan
 * akan menghasilkan grid yang kosong atau salah kolom tanpa satu pun galat — dan pengguna
 * tidak punya cara menduga bahwa yang ia lihat tidak lengkap. Sampai bentuknya diketahui,
 * isinya ditampilkan mentah supaya datanya tidak hilang dari layar.
 */
function PartDetailNote({ raw }: { raw: string | null | undefined }) {
  const value = textOf(raw)
  if (value === '') return null

  return (
    <section className="rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">Detail Part</h2>
      <p className="mt-1 text-xs text-slate-600">
        Ditampilkan apa adanya. Bentuk datanya belum dibaca dari data sungguhan, sehingga belum
        diuraikan menjadi tabel.
      </p>
      <pre className="mt-2 overflow-x-auto rounded-kontrol bg-white p-3 text-xs text-slate-700">
        {value}
      </pre>
    </section>
  )
}

/**
 * cellText menyusun teks satu isian.
 *
 * Tanggal diformat ke `1 Juni 2026`; sisanya ditampilkan apa adanya. Nilai kosong menjadi
 * tanda pisah — bukan sel kosong yang tidak dapat dibedakan dari isian yang gagal dimuat.
 */
function cellText(claim: ClaimDetail, field: FieldRef): string {
  const value = textOf(claim[field.kunci])
  if (value === '') return '—'
  return isDate(value) ? formatDate(value) : value
}

/** textOf memangkas nilai yang boleh null menjadi teks. */
function textOf(value: string | null | undefined): string {
  return (value ?? '').trim()
}

/** isDate mengenali bentuk `YYYY-MM-DD` yang dikirim server untuk kolom tanggal. */
function isDate(text: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(text)
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
