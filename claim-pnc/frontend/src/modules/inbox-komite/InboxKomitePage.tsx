import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  KomiteInboxKind,
  KomiteOutcome,
  type KomiteCase,
  type KomiteInboxKind as Kind,
} from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { ReloadIcon, ScaleIcon } from '@/components/Icon'
import { formatRupiah } from '@/lib/money'

import { DecisionPanel } from './DecisionPanel'
import { InboxTabs } from './InboxTabs'
import { useDecide, useInboxList } from './api'

/**
 * Layar Inbox Komite.
 *
 * Menggantikan harness `InboxKomite_Harness` beserta section `InboxKomite_section`
 * (1,8 MB) — MENU_ID 52 pada `POOLDATA.M_MENU_APLIKASI_PNC`.
 *
 * # Yang ditiru dari layar lama
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Tiga kotak: Outstanding, Diterima, Ditolak | `Section/InboxKomite_section-Section.xml` |
 * | Kolom daftar | `GetKomitePAOutstanding`, `ShowKomiteTerimaTolakNonMBU` |
 * | "Cari" berdasarkan No Komite / No Klaim | prompt pencarian pada section |
 * | "Tgl Input Dari" dan "Tgl Input Sampai" | isian tanggal pada section |
 * | Yang paling lama menunggu di atas | `ORDER BY "AgingKomite" DESC` |
 * | Inbox milik SATU orang | `PXASSIGNEDOPERATORID = <yang login>` |
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Nama kolom | `.IBNR`, `.pyScore`, `.DraftWordingID` | nama yang sesuai isinya (`D-19`) |
 * | Tiga grid bertumpuk | seluruhnya tampil sekaligus | tab, supaya pekerjaan terbaca lebih dulu |
 * | Keputusan | `AcceptStatus` tanpa keterangan | tiga pilihan beserta akibatnya |
 * | Catatan pada tolak | boleh kosong | wajib |
 * | Paginasi | `pyMaxRecords=500`, memotong diam-diam | halaman + jumlah seluruhnya |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 *
 * # Satu hal yang WAJIB diketahui pemakainya
 *
 * Keputusan yang dicatat di sini TIDAK menyentuh Pega: `P-1` melarang aplikasi ini
 * menulis ke tabel yang masih ditulis Pega. Selama masa paralel, kasus yang sudah
 * diputuskan di sini tetap terbuka di sana. Itu disebutkan di layar, bukan disembunyikan.
 */
export function InboxKomitePage() {
  const [kind, setKind] = useState<Kind>(KomiteInboxKind.outstanding)
  const [search, setSearch] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [offset, setOffset] = useState(0)
  const [deciding, setDeciding] = useState<KomiteCase | null>(null)

  const list = useInboxList({ kind, search, from, to, offset })
  const decide = useDecide()

  const data = list.data
  const rows = data?.kasus ?? []

  function changeKind(next: Kind) {
    setKind(next)
    setDeciding(null)
    // Halaman dikembalikan ke awal. Tanpa ini, berpindah dari kotak berisi 200 baris di
    // halaman empat ke kotak berisi 3 baris akan menampilkan tabel kosong yang tampak
    // rusak.
    setOffset(0)
  }

  function changeSearch(next: string) {
    setSearch(next)
    setOffset(0)
  }

  function changeRange(nextFrom: string, nextTo: string) {
    setFrom(nextFrom)
    setTo(nextTo)
    setOffset(0)
  }

  const columns: Column<KomiteCase>[] = [
    {
      key: 'nomor',
      title: 'No case & klaim',
      width: '12rem',
      value: (c) => `${c.nomor_case} ${c.nomor_klaim}`,
      render: (c) => (
        <div className="min-w-0">
          <p className="truncate font-mono text-xs font-medium text-slate-900">{c.nomor_case}</p>
          <p className="truncate font-mono text-xs text-slate-500">{c.nomor_klaim || '—'}</p>
        </div>
      ),
    },
    {
      key: 'tertanggung',
      title: 'Polis & tertanggung',
      value: (c) => `${c.nomor_polis} ${c.nama_tertanggung}`,
      render: (c) => (
        <div className="min-w-0">
          <p className="truncate text-slate-900">{c.nama_tertanggung || '—'}</p>
          <p className="truncate font-mono text-xs text-slate-500">{c.nomor_polis || '—'}</p>
        </div>
      ),
    },
    {
      key: 'bisnis',
      title: 'Bisnis & cabang',
      value: (c) => `${c.nama_bisnis} ${c.sumber_bisnis} ${c.cabang}`,
      render: (c) => (
        <div className="min-w-0">
          <p className="truncate text-slate-900">{c.nama_bisnis || '—'}</p>
          <p className="truncate text-xs text-slate-500">
            {[c.sumber_bisnis, c.cabang].filter(Boolean).join(' · ') || '—'}
          </p>
        </div>
      ),
    },
    {
      key: 'tipe',
      title: 'Tipe komite',
      width: '9rem',
      value: (c) => c.tipe_komite ?? '',
      render: (c) => <span className="text-slate-700">{c.tipe_komite || '—'}</span>,
    },
    {
      key: 'nilai',
      title: 'Nilai klaim',
      width: '11rem',
      alignRight: true,
      value: (c) => c.nilai_klaim,
      render: (c) => (
        <div className="min-w-0 text-right">
          <p className="truncate font-semibold tabular-nums text-slate-900">
            {formatRupiah(c.nilai_klaim)}
          </p>
          <p
            className="truncate text-xs tabular-nums text-slate-500"
            title="Nilai ASM share — di sistem lama tersimpan pada property bernama IBNR"
          >
            ASM {formatRupiah(c.nilai_asm_share)}
          </p>
        </div>
      ),
    },
    {
      key: 'aging',
      title: 'Aging',
      width: '7rem',
      value: (c) => String(c.aging_komite).padStart(6, '0'),
      render: (c) => <AgingBadge days={c.aging_komite} />,
    },
    {
      key: 'keadaan',
      title: 'Keadaan',
      width: '11rem',
      value: (c) => c.penjenjangan.kesimpulan,
      render: (c) => <StateCell item={c} />,
    },
    {
      key: 'tindakan',
      title: 'Tindakan',
      width: '8rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (c) => (
        <div className="flex justify-end">
          {c.penjenjangan.sudah_saya_putuskan ? (
            <span className="text-xs text-slate-400">sudah diputuskan</span>
          ) : (
            <Button
              tone="utama"
              onClick={() => setDeciding(c)}
              aria-label={`Beri keputusan komite untuk kasus ${c.nomor_case}`}
            >
              Putuskan
            </Button>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Proses Klaim</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Inbox Komite</li>
          </ol>
        </nav>
        <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight text-slate-900">
          <ScaleIcon className="h-6 w-6 text-slate-400" />
          Inbox Komite
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Kasus yang menunggu persetujuan Anda sebagai anggota komite, beserta riwayat
          keputusan yang pernah Anda berikan. Persetujuan komite adalah titik kewenangan
          tertinggi atas nilai klaim.
        </p>
      </header>

      <div className="mb-5">
        <InboxTabs
          summary={data?.ringkasan}
          active={kind}
          onSelect={changeKind}
          loading={list.isFetching}
        />
      </div>

      <div className="mb-5">
        <RangeFilter from={from} to={to} onChange={changeRange} />
      </div>

      {deciding && (
        <div className="mb-6">
          <DecisionPanel
            item={deciding}
            working={decide.isPending}
            error={decide.error}
            onClose={() => {
              setDeciding(null)
              decide.reset()
            }}
            onSubmit={(decision, note) =>
              decide.mutate(
                { caseID: deciding.nomor_case, decision, note },
                { onSuccess: () => setDeciding(null) },
              )
            }
          />
        </div>
      )}

      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(c) => c.nomor_case}
        title="Kasus Komite"
        description={
          data
            ? `${data.total} kasus pada kotak ini.`
            : 'Memuat daftar kasus komite…'
        }
        searchLabel="Cari nomor case komite atau nomor klaim"
        emptyMessage={emptyMessageFor(kind)}
        isLoading={list.isPending}
        error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
        serverSearch={{ value: search, onChange: changeSearch, matchCount: data?.total }}
        actions={
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
            {list.isFetching ? 'Memuat…' : 'Muat ulang'}
          </Button>
        }
      />

      {data && data.total > rows.length && (
        <Pagination
          offset={data.lewati}
          limit={data.batas}
          total={data.total}
          visible={rows.length}
          onMove={setOffset}
          loading={list.isFetching}
        />
      )}

      {/*
        Inbox yang kosong punya DUA sebab yang sangat berbeda, dan keduanya tidak dapat
        dibedakan dari tabel kosong: tidak ada pekerjaan, versus identitas sesi tidak
        cocok dengan satu pun OPERATOR_ID di data warisan.

        Yang kedua sangat mungkin terjadi selama pemetaan identitas HCC/HCQ ke OPERATOR_ID
        belum ada (`ADR-0024`), dan ia tidak muncul sebagai galat sama sekali. Karena itu
        operator yang dipakai menyaring ditampilkan apa adanya.
      */}
      {data && data.total === 0 && (
        <p className="mt-4 text-xs leading-relaxed text-slate-500">
          Inbox disaring untuk operator <span className="font-mono">{data.operator || '—'}</span>.
          Bila Anda yakin ada kasus yang menunggu, periksa apakah nama operator Anda di
          sistem lama sama dengan login yang Anda pakai masuk.
        </p>
      )}

      <p className="mt-6 max-w-3xl rounded-kartu border border-slate-200 bg-slate-50 p-3 text-xs leading-relaxed text-slate-600">
        <strong>Selama masa paralel:</strong> keputusan yang dicatat di sini belum
        menutup case yang sama di Pega. Kasusnya tetap terbuka di sana sampai modul
        penugasan dan penyelesaian nilai ikut berpindah.
      </p>
    </div>
  )
}

/**
 * Lencana umur: angka DAN warna, tidak pernah warna saja.
 *
 * Ambangnya — tiga hari — bukan angka karangan. Ia sama dengan `DefaultAutoAgeDays`, lama
 * menganggur yang di sistem lama membuat sebuah jenjang memenuhi syarat disetujui
 * OTOMATIS oleh job harian (`AutoAcceptKomite`, Work Owner menegaskan 72 jam).
 *
 * Menandainya di sini membuat kasus yang mendekati keadaan itu terlihat oleh manusia
 * lebih dulu — dan persetujuan otomatis melewati seluruh kontrol otorisasi, karena job
 * tidak punya pengguna sehingga tidak ada menu yang dapat diperiksa (`D-59`).
 */
function AgingBadge({ days }: { days: number }) {
  const tone =
    days >= 7
      ? 'bg-red-50 text-red-700 ring-red-100'
      : days >= 3
        ? 'bg-amber-50 text-amber-800 ring-amber-100'
        : 'bg-slate-100 text-slate-600 ring-slate-200'

  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium tabular-nums ring-1 ${tone}`}
      title={`Menunggu ${days} hari kalender`}
    >
      {days} hari
    </span>
  )
}

/**
 * Kolom keadaan menggabungkan DUA sumber yang selama masa paralel dapat berbeda:
 * keputusan yang tercatat di sistem ini, dan keputusan yang tercatat di Pega.
 *
 * Keduanya ditampilkan berdampingan alih-alih dipilih salah satu. Memilih salah satu
 * berarti memutuskan mana yang sah — pertanyaan yang belum dijawab siapa pun, dan yang
 * tidak boleh dijawab diam-diam oleh sebuah komponen tampilan.
 */
function StateCell({ item }: { item: KomiteCase }) {
  const progress = item.penjenjangan

  return (
    <div className="min-w-0">
      <OutcomeBadge outcome={progress.kesimpulan} />
      <p className="mt-1 truncate text-xs text-slate-500">
        {progress.jumlah_jenjang_belum_diketahui
          ? `Jenjang ke-${progress.jenjang_kini}`
          : `Jenjang ${progress.jenjang_kini || progress.jumlah_jenjang} dari ${progress.jumlah_jenjang}`}
      </p>
      {item.keputusan_pega && (
        <p className="truncate text-xs text-slate-400" title="Keputusan yang tercatat di Pega">
          Pega: {item.keputusan_pega}
        </p>
      )}
    </div>
  )
}

function OutcomeBadge({ outcome }: { outcome: KomiteCase['penjenjangan']['kesimpulan'] }) {
  const tone: Record<string, string> = {
    [KomiteOutcome.pending]: 'bg-blue-50 text-blue-700 ring-blue-100',
    [KomiteOutcome.approved]: 'bg-emerald-50 text-emerald-700 ring-emerald-100',
    [KomiteOutcome.rejected]: 'bg-red-50 text-red-700 ring-red-100',
    [KomiteOutcome.returned]: 'bg-amber-50 text-amber-800 ring-amber-100',
  }

  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ${
        tone[outcome] ?? 'bg-slate-100 text-slate-600 ring-slate-200'
      }`}
    >
      {outcome}
    </span>
  )
}

/**
 * Penyaring "Tgl Input Dari" dan "Tgl Input Sampai".
 *
 * Keduanya INKLUSIF di kedua ujung — mencari sampai tanggal 20 ikut menampilkan yang
 * masuk pukul 23:59 tanggal 20. Itu ditegakkan server; di sini ia hanya disebutkan,
 * supaya pengguna tidak perlu mencobanya untuk tahu.
 *
 * Rentang yang terbalik tidak diperbaiki diam-diam dengan menukar kedua ujungnya. Server
 * menolaknya, dan penolakannya terbaca sebagai pesan pada isian — menukarnya akan
 * menampilkan hasil yang benar untuk pertanyaan yang tidak diajukan.
 */
function RangeFilter({
  from,
  to,
  onChange,
}: {
  from: string
  to: string
  onChange: (from: string, to: string) => void
}) {
  const inverted = from !== '' && to !== '' && to < from

  return (
    <div className="rounded-kartu border border-slate-200 bg-white p-4">
      <div className="flex flex-wrap items-end gap-3">
        <div className="w-44">
          <Field
            id="tgl-input-dari"
            label="Tgl input dari"
            type="date"
            value={from}
            onChange={(e) => onChange(e.target.value, to)}
          />
        </div>
        <div className="w-44">
          <Field
            id="tgl-input-sampai"
            label="Tgl input sampai"
            type="date"
            value={to}
            onChange={(e) => onChange(from, e.target.value)}
          />
        </div>
        {(from !== '' || to !== '') && (
          <Button tone="halus" onClick={() => onChange('', '')}>
            Bersihkan
          </Button>
        )}
      </div>
      {inverted && (
        <p className="mt-2 text-sm text-red-700">
          Tanggal sampai tidak boleh lebih awal daripada tanggal dari.
        </p>
      )}
    </div>
  )
}

function Pagination({
  offset,
  limit,
  total,
  visible,
  onMove,
  loading,
}: {
  offset: number
  limit: number
  total: number
  visible: number
  onMove: (offset: number) => void
  loading: boolean
}) {
  const first = visible === 0 ? 0 : offset + 1
  const last = offset + visible

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-slate-600" role="status">
        Menampilkan {first}–{last} dari {total} kasus.
      </p>
      <div className="flex gap-2">
        <Button
          tone="kedua"
          onClick={() => onMove(Math.max(0, offset - limit))}
          disabled={offset === 0 || loading}
        >
          Sebelumnya
        </Button>
        <Button
          tone="kedua"
          onClick={() => onMove(offset + limit)}
          disabled={last >= total || loading}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}

/**
 * Pesan kosong dibedakan per kotak.
 *
 * "Tidak ada data" pada kotak Outstanding adalah kabar baik — tidak ada pekerjaan
 * menunggu. Pada kotak riwayat ia berarti belum pernah memutuskan apa pun. Satu kalimat
 * untuk keduanya akan membuat yang pertama terbaca seperti kegagalan.
 */
function emptyMessageFor(kind: Kind): string {
  switch (kind) {
    case KomiteInboxKind.outstanding:
      return 'Tidak ada kasus yang menunggu keputusan Anda.'
    case KomiteInboxKind.accepted:
      return 'Belum ada kasus yang Anda setujui.'
    default:
      return 'Belum ada kasus yang Anda tolak atau kembalikan.'
  }
}

/** Gagal memuat selalu bernada gangguan: pengguna baru membuka layarnya. */
function LoadErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Daftar kasus komite belum dapat dimuat. Periksa koneksi lalu tekan Muat ulang."
        tone="gangguan"
      />
    )
  }
  return (
    <ErrorMessage
      title="Daftar kasus komite gagal dimuat"
      description={
        error instanceof APIError ? error.message : 'Terjadi kesalahan pada sistem.'
      }
      tone="gangguan"
    />
  )
}
