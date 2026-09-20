import { useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { SearchPanel } from './SearchPanel'
import { useClaimHistorySearch, useOpenClaimHistory } from './api'
import {
  ClaimHistoryError,
  EMPTY_FORM,
  type ClaimHistory,
  type ExtraColumn,
  type PageInfo,
  type SearchForm,
  type SearchType,
} from './types'

/**
 * View History Claim — menu `MENU_ID 76`, pengganti harness `PNCSearchKlaim`.
 *
 * Judul yang dibaca pengguna di sistem lama adalah "VIEW HISTORY CLAIM", dan isinya
 * pencarian riwayat klaim lintas seluruh klaim. Ia BUKAN inbox: barisnya bukan pekerjaan,
 * tidak hilang setelah dikerjakan, dan tidak punya tenggat (`D-79`).
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/PNCSearchKlaim-Section.xml` apa adanya: judul, tiga isian yang
 * bergantian mengikuti tipe pencarian, tombol Cari, lalu grid berhalaman. Nama kolom
 * TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak
 * perlu belajar ulang, dan itulah teks yang selama ini mereka baca.
 *
 * # Satu hal yang TIDAK ada di sini, dan itu bukan kelalaian
 *
 * Tombol "Lihat Detail Klaim". Di sistem lama ia membuka layar rincian lewat
 * `setDataViewKlaim_Act` — dan layar itu punya menunya sendiri, `MENU_ID 75` "View Claim"
 * (`PNCViewClaim`), sehingga ia modul tersendiri yang belum dibangun. Kunci yang
 * dibutuhkannya sudah ikut dikirim server pada setiap baris (`referensi`), sehingga
 * menyalakannya kelak tidak menuntut perubahan kontrak.
 */
export function ClaimHistoryPage() {
  const [form, setForm] = useState<SearchForm>(EMPTY_FORM)
  const [submitted, setSubmitted] = useState<SearchForm | null>(null)
  const [page, setPage] = useState(1)

  const portal = useSelectedPortal((state) => state.alias)
  const opened = useOpenClaimHistory()
  const search = useClaimHistorySearch(submitted ?? EMPTY_FORM, page, submitted !== null)

  const types: SearchType[] = opened.data?.tipe_pencarian ?? []
  const selected = types.find((t) => t.kode === (submitted ?? form).tipe)

  /** Mengubah tipe pencarian MEMBERSIHKAN isian lain. */
  function change(patch: Partial<SearchForm>) {
    setForm((previous) =>
      patch.tipe !== undefined && patch.tipe !== previous.tipe
        ? { ...EMPTY_FORM, tipe: patch.tipe }
        : { ...previous, ...patch },
    )
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Riwayat klaim milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  // Gerbang proteksi data menutup SELURUH layar, bukan sekadar tombolnya: pengguna yang
  // belum terdaftar atau jatahnya habis tidak boleh melihat satu baris pun.
  if (opened.isError) {
    return (
      <PageFrame>
        <GateBlocked error={opened.error} />
      </PageFrame>
    )
  }

  return (
    <PageFrame>
      {opened.data && <QuotaNotice remaining={opened.data.proteksi.jatah_sisa} />}

      <SearchPanel
        types={types}
        form={form}
        onChange={change}
        onSubmit={() => {
          setSubmitted(form)
          setPage(1)
        }}
        busy={opened.isPending || search.isFetching}
        fieldError={fieldErrorOf(search.error)}
      />

      {submitted !== null && (
        <div className="mt-4">
          <DataTable<ClaimHistory>
            columns={columnsFor(selected?.kolom_tambahan ?? [])}
            rows={search.data?.klaim ?? []}
            rowKey={(row) => row.referensi || row.nomor_klaim}
            title="Hasil pencarian"
            description={descriptionFor(selected)}
            // Kotak cari bawaan disembunyikan: layar ini sudah punya formulir
            // pencariannya sendiri di atas, dan yang kedua hanya akan menyaring halaman
            // yang sedang terbuka — hasilnya menyesatkan pada data berhalaman.
            hideSearch
            isLoading={search.isPending}
            error={
              search.isError && !isValidationError(search.error) ? (
                <ErrorMessage
                  title="Pencarian tidak dapat dijalankan"
                  description={messageOf(search.error)}
                  tone="gangguan"
                />
              ) : undefined
            }
            emptyMessage="Tidak ada klaim yang cocok dengan pencarian ini."
          />

          {search.data && search.data.halaman.total > 0 && (
            <Pagination
              info={search.data.halaman}
              visible={search.data.klaim.length}
              onMove={setPage}
              loading={search.isFetching}
            />
          )}
        </div>
      )}

      <p className="mt-4 text-xs text-slate-500">
        Membuka rincian sebuah klaim belum tersedia di sini: layarnya adalah menu{' '}
        <span className="font-medium">View Claim</span> yang belum dibangun.
      </p>
    </PageFrame>
  )
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">View History Claim</h1>
        <p className="mt-1 text-sm text-slate-600">
          Pencarian riwayat klaim menurut nomor polis, nama, tanggal, dan sembilan acuan
          lain.
        </p>
      </header>
      {children}
    </div>
  )
}

/**
 * GateBlocked menjelaskan penolakan gerbang proteksi data.
 *
 * Kedua penolakan dibedakan karena tindakannya berbeda: yang satu menuntut pendaftaran,
 * yang lain menuntut penambahan jatah. Menyatukannya menjadi satu pesan membuat pengguna
 * menebak mana yang berlaku padanya.
 */
function GateBlocked({ error }: { error: unknown }) {
  const code = error instanceof APIError ? error.kode : ''

  if (code === ClaimHistoryError.quotaExhausted) {
    return (
      <ErrorMessage
        title="Jatah pencarian data Anda sudah habis"
        description={
          'Layar ini membatasi berapa kali seorang pengguna dapat membuka data klaim. ' +
          'Minta penambahan jatah di Master Proteksi Data, lalu buka layar ini lagi.'
        }
        tone="penolakan"
      />
    )
  }

  if (code === ClaimHistoryError.notRegistered) {
    return (
      <ErrorMessage
        title="Anda belum terdaftar di Master Proteksi Data"
        description={
          'Layar ini hanya dapat dibuka pengguna yang terdaftar beserta jatah ' +
          'pencariannya. Hubungi pengelola proteksi data sebelum membukanya.'
        }
        tone="penolakan"
      />
    )
  }

  return (
    <ErrorMessage
      title="Layar tidak dapat dibuka"
      description={messageOf(error)}
      tone="gangguan"
    />
  )
}

/**
 * QuotaNotice menyebutkan sisa jatah SEBELUM habis.
 *
 * Tanpa ini, satu-satunya cara pengguna mengetahui jatahnya adalah dengan kehabisannya.
 * Nadanya berubah saat tinggal sedikit — dan pembedaannya tidak hanya lewat warna
 * (`D-12`: bukan hanya warna), melainkan lewat kalimatnya.
 */
function QuotaNotice({ remaining }: { remaining: number }) {
  const low = remaining <= 3

  return (
    <p
      className={[
        'mt-4 rounded-kartu border px-4 py-3 text-sm',
        low
          ? 'border-amber-200 bg-amber-50/80 text-amber-900'
          : 'border-slate-200 bg-slate-50 text-slate-700',
      ].join(' ')}
    >
      {low ? 'Jatah pencarian Anda tinggal ' : 'Sisa jatah pencarian Anda '}
      <span className="font-medium">{remaining}</span>
      {low
        ? ' kali lagi. Setiap kali layar ini dibuka, satu jatah terpakai.'
        : ' kali. Setiap kali layar ini dibuka, satu jatah terpakai.'}
    </p>
  )
}

/**
 * Paginasi "sebelumnya / berikutnya", bukan nomor halaman.
 *
 * Bentuknya sama dengan layar Pelaporan Klaim supaya kedua layar tidak terasa dirakit
 * dari dua aplikasi berbeda. Ia hidup di sini, bukan di dalam `DataTable`, karena
 * komponen tabel baku belum mengenal paginasi server — itu lingkup `TKT-U2-001`, dan
 * menambahkannya sepihak demi satu layar akan mendahului keputusan pustaka tabel yang
 * sengaja ditinggalkan terbuka (`TKT-U2-005`).
 */
function Pagination({
  info,
  visible,
  onMove,
  loading,
}: {
  info: PageInfo
  visible: number
  onMove: (page: number) => void
  loading: boolean
}) {
  const first = visible === 0 ? 0 : (info.halaman - 1) * info.ukuran + 1
  const last = (info.halaman - 1) * info.ukuran + visible

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-slate-600" role="status">
        Menampilkan {first}–{last} dari {info.total} klaim.
      </p>
      <div className="flex gap-2">
        <Button
          tone="kedua"
          onClick={() => onMove(Math.max(1, info.halaman - 1))}
          disabled={info.halaman <= 1 || loading}
        >
          Sebelumnya
        </Button>
        <Button
          tone="kedua"
          onClick={() => onMove(info.halaman + 1)}
          disabled={info.halaman >= info.total_halaman || loading}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}

/**
 * columnsFor menyusun kolom tabel.
 *
 * Sebelas kolom selalu ada; empat sisanya hanya digambar bila tipe pencarian yang dipakai
 * membawanya. Daftar itu datang dari SERVER, bukan ditebak dari isi baris — menebaknya
 * akan membuat kolom hilang saat seluruh barisnya kebetulan kosong.
 *
 * Urutannya mengikuti grid sistem lama (`Section/PNCSearchKlaim-Section.xml`), dengan
 * kolom tambahan disisipkan di tempat yang sama seperti di sana.
 */
function columnsFor(extra: ExtraColumn[]): Column<ClaimHistory>[] {
  const has = (column: ExtraColumn) => extra.includes(column)

  const columns: Column<ClaimHistory>[] = [
    { key: 'nomor_klaim', title: 'No Klaim', value: (r) => r.nomor_klaim },
    { key: 'nomor_polis', title: 'No Polis', value: (r) => r.nomor_polis },
  ]

  if (has('nomor_akseptasi')) {
    columns.push({
      key: 'nomor_akseptasi',
      title: 'No Akseptasi',
      value: (r) => r.nomor_akseptasi,
    })
  }
  if (has('nomor_balai_lelang')) {
    columns.push({
      key: 'nomor_balai_lelang',
      title: 'No Balai Lelang',
      value: (r) => r.nomor_balai_lelang,
    })
  }

  columns.push({
    key: 'nama_tertanggung',
    title: 'Nama Tertanggung',
    value: (r) => r.nama_tertanggung,
  })

  if (has('nama_objek')) {
    columns.push({ key: 'nama_objek', title: 'Nama Objek', value: (r) => r.nama_objek })
  }
  if (has('tanggal_lahir')) {
    columns.push({
      key: 'tanggal_lahir',
      title: 'Tanggal Lahir',
      value: (r) => dateText(r.tanggal_lahir),
    })
  }

  columns.push(
    { key: 'tanggal_kejadian', title: 'Tgl Kejadian', value: (r) => dateText(r.tanggal_kejadian) },
    { key: 'bisnis', title: 'Bisnis', value: (r) => r.bisnis },
    { key: 'cabang', title: 'Cabang', value: (r) => r.cabang },
    { key: 'status', title: 'Status', value: (r) => r.status },
    { key: 'posisi_klaim', title: 'Posisi Klaim', value: (r) => r.posisi_klaim || '—' },
    { key: 'tanggal_close', title: 'Tanggal Close', value: (r) => dateText(r.tanggal_close) },
    { key: 'catatan_close', title: 'Catatan Close', value: (r) => r.catatan_close },
    { key: 'pic_teknis', title: 'PIC Teknis', value: (r) => r.pic_teknis },
  )

  return columns
}

/**
 * descriptionFor menyatakan keterbatasan yang berlaku pada tipe pencarian terpilih.
 *
 * Kedua keterbatasan di bawah adalah cacat sistem lama yang Work Owner putuskan untuk
 * DIREPLIKASI (2026-09-20). Menyatakannya di layar bukan pembelaan diri — tanpa itu,
 * hasil yang selalu kosong dan kolom yang selalu kosong akan dilaporkan berulang kali
 * sebagai kerusakan modul.
 *
 * Mengembalikan teks KOSONG, bukan `undefined`, karena `DataTable.description` bertipe
 * `string` dan berkas ini dikompilasi dengan `exactOptionalPropertyTypes`. Tabelnya
 * memang tidak menggambar apa pun untuk teks kosong.
 */
function descriptionFor(selected: SearchType | undefined): string {
  if (!selected) return ''

  if (selected.kode === '9') {
    return (
      'Pencarian Tanggal Lahir mengikuti perilaku sistem lama, yang tidak membawa ' +
      'tanggal yang diisi ke dalam kuerinya — hasilnya karena itu selalu kosong.'
    )
  }
  if (selected.kode === '3') {
    return 'Pada pencarian Nama Objek, kolom Posisi Klaim kosong — sama seperti sistem lama.'
  }
  return ''
}

function dateText(iso: string | null): string {
  return iso ? formatDate(iso) : '—'
}

function isValidationError(error: unknown): boolean {
  return error instanceof APIError && error.detail.length > 0
}

/**
 * fieldErrorOf memetakan `detail` galat validasi menjadi pesan per isian.
 *
 * Nama isiannya dibaca dari `field` MAUPUN `kolom`. Keduanya diperiksa karena kontrak
 * galat belum seragam antarmodul — `api/client.ts` mencatat ketiga bentuk yang hidup hari
 * ini, dan penyeragamannya adalah `TKT-F1-004` yang masih terhalang. Modul ini mengirim
 * `field`; membaca keduanya membuat layar tidak ikut rusak bila kontraknya kelak berubah
 * ke bentuk yang lain.
 */
function fieldErrorOf(error: unknown): Record<string, string> {
  if (!(error instanceof APIError)) return {}

  const result: Record<string, string> = {}
  for (const violation of error.detail) {
    const name = violation.field ?? violation.kolom
    if (name) result[name] = violation.pesan
  }
  return result
}

function messageOf(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  return 'Terjadi kesalahan pada sistem.'
}
