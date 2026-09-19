import { APIError, NetworkError } from '@/api/client'
import type { KomiteThreshold, KomiteFinding } from '@/api/types'
import { ReloadIcon, WarningIcon } from '@/components/Icon'
import { ErrorMessage } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { formatRupiah } from '@/lib/money'

import { useThresholdList, useIntegrity } from './api'

/**
 * Layar Master Ambang Komite.
 *
 * Menampilkan tangga jenjang persetujuan nilai klaim — siapa yang menyetujui, mulai
 * nilai berapa, dan pada urutan ke berapa.
 *
 * # Kenapa layar ini dibaca saja
 *
 * `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel.
 * `POOLDATA.EMAILKOMITE` masih ditulis Pega dan dibaca 17 kueri di sana. Memindahkan
 * kepemilikannya menuntut prosedur `D-63` — permintaan perubahan skema tertulis,
 * persetujuan Work Owner, pelaksanaan DBA, lalu pengujian dengan menjalankan Pega dan Go
 * bersamaan — dan itu belum ditempuh. Keputusan Work Owner 2026-09-17.
 *
 * Karena itu tidak ada tombol Tambah maupun Ubah di sini, dan ketiadaannya **dijelaskan
 * di layar**, bukan dibiarkan terlihat seperti fitur yang terlupa.
 *
 * # Yang tidak ada di sistem lama
 *
 * Tangga ini tidak pernah punya layarnya sendiri. Di Pega ia hanya terbaca lewat 17
 * kueri yang tersebar, masing-masing dengan penyaring sendiri, dan tidak seorang pun
 * dapat melihatnya utuh dalam satu tampilan. Pemeriksaan integritasnya pun baru —
 * `LIMIT_TOP` tersimpan di sana tetapi tidak pernah dipakai satu kueri pun.
 *
 * Nama prop komponen bersama mengikuti `D-80`: seluruh nama di dalam kode berbahasa
 * Inggris (`columns`, `rows`, `title`, …). Yang tetap berbahasa Indonesia hanyalah nama
 * modul (`ambang-komite`, `D-81`), nama field JSON pada API, dan komentar seperti ini.
 */
export function ThresholdPage() {
  const list = useThresholdList()
  const integrity = useIntegrity()

  const bands = list.data?.kebijakan_pita ?? []

  const columns: Column<KomiteThreshold>[] = [
    {
      key: 'lini',
      title: 'Lini bisnis',
      value: (a) => a.lini,
      width: '13%',
      render: (a) => <span className="font-medium text-slate-900">{a.lini}</span>,
    },
    {
      key: 'jenjang',
      title: 'Jenjang',
      value: (a) => String(a.jenjang),
      width: '8%',
    },
    {
      key: 'batas_bawah',
      title: 'Mulai nilai',
      value: (a) => a.batas_bawah,
      width: '16%',
      render: (a) => (
        <span className="tabular-nums">{formatRupiah(a.batas_bawah)}</span>
      ),
    },
    {
      key: 'batas_atas',
      title: 'Batas atas',
      value: (a) => a.batas_atas,
      width: '16%',
      render: (a) => (
        <span className="tabular-nums text-slate-600">
          {a.batas_atas === '0.00' ? 'tanpa batas' : formatRupiah(a.batas_atas)}
        </span>
      ),
    },
    {
      key: 'penyetuju',
      title: 'Penyetuju',
      value: (a) => `${a.nama} ${a.operator_id}`,
      render: (a) => (
        <span className="min-w-0">
          <span className="block truncate font-medium text-slate-900">{a.nama || '—'}</span>
          <span className="block truncate text-xs text-slate-500">{a.operator_id || '—'}</span>
        </span>
      ),
    },
    {
      key: 'jenis_komite',
      title: 'Jenis',
      value: (a) => a.jenis_komite,
      width: '8%',
      render: (a) => <span className="tabular-nums text-slate-600">{a.jenis_komite || '—'}</span>,
    },
    {
      key: 'status',
      title: 'Status',
      value: (a) => rowNote(a),
      width: '15%',
      render: (a) => <RowBadges threshold={a} />,
    },
  ]

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Master Data</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Ambang Komite</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          Master Ambang Komite
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Tangga jenjang persetujuan nilai klaim. Aturannya <strong>kumulatif</strong>:
          setiap jenjang yang ambang bawahnya sudah terlampaui nilai klaim ikut
          menyetujui — makin besar klaim, makin banyak penyetujunya.
        </p>
      </header>

      <ReadOnlyNote />

      {bands.length > 0 && <BandNote bands={bands} />}

      {integrity.data && integrity.data.temuan.length > 0 && (
        <FindingList findings={integrity.data.temuan} />
      )}

      <DataTable
        columns={columns}
        rows={list.data?.ambang ?? []}
        rowKey={(a) => a.id}
        title="Tangga Ambang Komite"
        description={
          list.data
            ? `${list.data.total_jenjang} jenjang persetujuan aktif dari ${list.data.total} baris master.`
            : 'Memuat tangga ambang…'
        }
        searchLabel="Cari lini, nama penyetuju, atau Operator ID"
        emptyMessage="Master ambang komite belum berisi satu baris pun."
        isLoading={list.isPending}
        error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
        actions={
          <Button
            tone="kedua"
            onClick={() => {
              void list.refetch()
              void integrity.refetch()
            }}
            disabled={list.isFetching}
          >
            <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
            {list.isFetching ? 'Memuat…' : 'Muat ulang'}
          </Button>
        }
      />
    </div>
  )
}

/**
 * Ketiadaan tombol Tambah dan Ubah DIJELASKAN, bukan dibiarkan terlihat seperti fitur
 * yang terlupa.
 *
 * Pengguna yang terbiasa dengan layar master lain di aplikasi ini akan mencari tombolnya;
 * tanpa keterangan, ia akan menyimpulkan layarnya belum selesai.
 */
function ReadOnlyNote() {
  return (
    <div className="mb-6 rounded-kartu border border-slate-200 bg-slate-50/80 p-4">
      <p className="text-sm leading-relaxed text-slate-700">
        <strong className="font-semibold text-slate-900">Layar ini dibaca saja.</strong>{' '}
        Selama Pega dan aplikasi ini berjalan berdampingan, satu tabel hanya boleh ditulis
        satu sistem. Master ambang komite masih ditulis Pega dan dibaca 17 kueri di sana,
        sehingga perubahannya dilakukan di Pega — bukan di sini.
      </p>
    </div>
  )
}

/** Batas pita ditampilkan supaya pengguna dapat melihatnya, bukan menghafalnya. */
function BandNote({ bands }: { bands: { lini: string; batas: string }[] }) {
  return (
    <div className="mb-6 rounded-kartu border border-blue-200 bg-blue-50/70 p-4">
      <h2 className="text-sm font-semibold text-blue-900">Pita nilai</h2>
      <ul className="mt-2 space-y-1 text-sm leading-relaxed text-blue-900/90">
        {bands.map((b) => (
          <li key={b.lini}>
            Untuk lini <strong>{b.lini}</strong>, pita dipilih lebih dulu: sampai{' '}
            <strong className="tabular-nums">{formatRupiah(b.batas)}</strong> masuk pita 1,
            di atasnya masuk pita 2. Akumulasi jenjang kemudian berjalan di dalam pita itu
            saja.
          </li>
        ))}
      </ul>
      <p className="mt-2 text-sm leading-relaxed text-blue-900/80">
        Lini lain <strong>tidak</strong> mengenal pita. Memberlakukannya ke semua lini akan
        membuat sebagian klaim kehilangan seluruh penyetujunya.
      </p>
    </div>
  )
}

/**
 * Temuan pemeriksaan integritas.
 *
 * Cacat dan peringatan dibedakan tegas: cacat berarti hasil penjenjangan tidak dapat
 * dipercaya, peringatan berarti masternya tidak salah tetapi ada yang pantas dilihat
 * sebelum dipakai memutuskan uang.
 */
function FindingList({ findings }: { findings: KomiteFinding[] }) {
  const defects = findings.filter((f) => f.tingkat === 'cacat')
  const warnings = findings.filter((f) => f.tingkat !== 'cacat')

  return (
    <section className="mb-6 space-y-4">
      {defects.length > 0 && <FindingCard title="Cacat pada master" findings={defects} severe />}
      {warnings.length > 0 && (
        <FindingCard title="Yang perlu diperhatikan" findings={warnings} severe={false} />
      )}
    </section>
  )
}

function FindingCard({
  title,
  findings,
  severe,
}: {
  title: string
  findings: KomiteFinding[]
  severe: boolean
}) {
  const style = severe
    ? { box: 'border-red-200 bg-red-50/80', icon: 'text-red-700', title: 'text-red-900' }
    : { box: 'border-amber-200 bg-amber-50/70', icon: 'text-amber-700', title: 'text-amber-900' }

  return (
    <div className={`rounded-kartu border p-4 ${style.box}`}>
      <h2 className={`flex items-center gap-2 text-sm font-semibold ${style.title}`}>
        <WarningIcon className={`h-4 w-4 ${style.icon}`} />
        {title}
        <span className="font-normal opacity-70">({findings.length})</span>
      </h2>
      <ul className="mt-3 space-y-2.5">
        {findings.map((f, i) => (
          <li key={`${f.jenis}-${f.lini}-${f.pita ?? ''}-${i}`} className="text-sm leading-relaxed">
            <span className="font-medium text-slate-900">
              {f.lini}
              {f.pita ? ` · pita ${f.pita}` : ''}
            </span>
            <span className="text-slate-500"> — </span>
            <span className="text-slate-700">{f.pesan}</span>
            {f.id_ambang.length > 0 && (
              <span className="text-slate-500"> (baris {f.id_ambang.join(', ')})</span>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

function rowNote(a: KomiteThreshold): string {
  if (a.jenjang_persetujuan) return a.sedang_absen ? 'Jenjang · sedang absen' : 'Jenjang'
  if (!a.aktif) return 'Tidak aktif'
  if (a.untuk_registrasi) return 'Pemberitahuan registrasi'
  return 'Bukan jenjang'
}

/**
 * Lencana menjelaskan KENAPA sebuah baris tidak ikut menyetujui.
 *
 * Tanpa itu, baris yang tidak aktif dan baris yang hanya penerima pemberitahuan
 * registrasi akan tampak sama — dan pertanyaan "kenapa orang ini tidak ikut" tidak
 * terjawab oleh layarnya sendiri.
 */
function RowBadges({ threshold }: { threshold: KomiteThreshold }) {
  const label = rowNote(threshold)

  const style = threshold.jenjang_persetujuan
    ? threshold.sedang_absen
      ? 'bg-amber-100 text-amber-900'
      : 'bg-emerald-100 text-emerald-900'
    : 'bg-slate-100 text-slate-600'

  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${style}`}
      title={
        threshold.jenjang_persetujuan
          ? 'Baris ini ikut menyetujui nilai klaim.'
          : 'Baris ini tidak ikut menyetujui nilai klaim.'
      }
    >
      {label}
    </span>
  )
}

/**
 * Gagal memuat selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat
 * salah — ia baru membuka layarnya.
 */
function LoadErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Tangga ambang belum dapat dimuat. Periksa koneksi lalu tekan Muat ulang."
        tone="gangguan"
      />
    )
  }
  if (error instanceof APIError) {
    return <ErrorMessage title="Gagal memuat" description={error.message} tone="gangguan" />
  }
  return (
    <ErrorMessage
      title="Gagal memuat"
      description="Tangga ambang belum dapat dimuat. Coba tekan Muat ulang."
      tone="gangguan"
    />
  )
}
