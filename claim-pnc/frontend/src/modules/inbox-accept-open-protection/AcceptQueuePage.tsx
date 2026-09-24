import { useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { ReloadIcon } from '@/components/Icon'

import { PAGE_SIZE, useAcceptQueue, useDecideProtection, useProtectionDetail } from './api'
import { protectionTypeLabel, type Protection, type Queue } from './types'

/**
 * Layar Inbox Accept Open Protection.
 *
 * Migrasi dari **`Harness/InputProtection_Harness-Harness.xml`** beserta
 * `Section/InputProtection_Section-Section.xml` dan form
 * `Section/AcceptProtectionSection-Section.xml`.
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Ketujuh kolom | `Section/InputProtection_Section-Section.xml` |
 * | Antrean NON PREMI | `Report Definition/InboxOpenProtection2_RD-RD.xml` |
 * | Antrean PREMI | `Report Definition/InboxOpenProtection2_RD_collection-RD.xml` |
 * | Setuju / Tolak | `Flow/CreateProtection_Flow.xml` · `When/IsAcceptProtection-When.xml` |
 *
 * # Dua antrean, dan asal pemisahannya
 *
 * Layar lama memuat tiga grid berisi kolom yang sama persis, dibedakan syarat tampilnya.
 * Dua di antaranya nyata: PREMI (`TypeProtection = "2"`, hanya bagi peran penagihan premi)
 * dan NON PREMI (`!= "2"`).
 *
 * # Grid ketiga TIDAK dibawa
 *
 * Grid ketiga hanya muncul untuk **satu alamat Gmail pribadi**
 * (`Section/InputProtection_Section-Section.xml:25104`), dan penyaringnya lebih longgar —
 * tanpa syarat nomor klaim. `D-15` melarang nilai bisnis di-hardcode dan `D-67` melarang
 * akun pribadi dibawa ke sistem baru.
 *
 * Ini SELISIH TERENCANA (`P-5`): orang tersebut akan melihat antrean yang berbeda dari hari
 * ini. Bila keleluasaan itu memang dibutuhkan bisnis, ia harus kembali sebagai PERAN di
 * master data.
 *
 * # Kewenangan belum ditegakkan
 *
 * Pemilihan antrean di sini adalah TAB, sedangkan di Pega ia ditentukan access group.
 * Sampai `TKT-F3-004` dapat diisi, tidak ada yang mencegah pengguna membuka antrean yang
 * bukan haknya. Dinyatakan di layar, bukan disembunyikan.
 */
export function AcceptQueuePage() {
  const [queue, setQueue] = useState<Queue>('non-premi')
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)
  const [opened, setOpened] = useState<string | null>(null)

  const portal = useSelectedPortal((state) => state.alias)

  const list = useAcceptQueue({ queue, search, offset })
  const detail = useProtectionDetail(opened)
  const decide = useDecideProtection()

  const rows = list.data?.proteksi ?? []
  const total = list.data?.total ?? 0

  function changeQueue(next: Queue) {
    setQueue(next)
    // Halaman dan pencarian dikembalikan: keduanya milik antrean sebelumnya, dan
    // mempertahankannya akan menampilkan halaman empat dari daftar yang hanya punya satu.
    setOffset(0)
    setSearch('')
    setOpened(null)
    decide.reset()
  }

  function changeSearch(next: string) {
    setSearch(next)
    setOffset(0)
  }

  async function submit(decision: 'setuju' | 'tolak') {
    if (opened === null) return

    // Galat DITELAN dengan sengaja; ia sudah ditampilkan lewat `decide.error`. Membiarkan
    // promise ini menolak akan menghasilkan unhandled rejection di peramban.
    //
    // Panel TIDAK ditutup saat gagal, dan justru itu yang penting di sini: kegagalan yang
    // paling sering terjadi adalah `409` — proteksi sudah diputuskan petugas lain. Menutup
    // panelnya akan menyembunyikan pesan yang menjelaskannya.
    try {
      await decide.mutateAsync({ number: opened, decision })
      setOpened(null)
    } catch {
      // Sudah tercatat di decide.error; lihat komentar di atas.
    }
  }

  const columns: Column<Protection>[] = [
    {
      key: 'nomor_proteksi',
      title: 'No Proteksi',
      width: '11rem',
      value: (p) => p.nomor_proteksi,
      render: (p) => (
        <button
          type="button"
          onClick={() => {
            decide.reset()
            setOpened(p.nomor_proteksi)
          }}
          className="truncate rounded font-mono text-xs font-medium text-blue-700 underline-offset-2 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
        >
          {p.nomor_proteksi}
        </button>
      ),
    },
    {
      key: 'nomor_polis',
      title: 'No Polis',
      width: '12rem',
      value: (p) => p.nomor_polis,
      render: (p) => (
        <span className="truncate font-mono text-xs text-slate-700">{p.nomor_polis || '—'}</span>
      ),
    },
    {
      key: 'nomor_klaim',
      title: 'No Klaim',
      width: '11rem',
      value: (p) => p.nomor_klaim,
      render: (p) => (
        <span className="truncate font-mono text-xs text-slate-700">{p.nomor_klaim || '—'}</span>
      ),
    },
    {
      key: 'tipe_proteksi',
      title: 'Tipe Proteksi',
      width: '12rem',
      value: (p) => p.tipe_proteksi,
      render: (p) => <span className="truncate">{protectionTypeLabel(p.tipe_proteksi, p.nama_tipe_proteksi)}</span>,
    },
    {
      key: 'tanggal_proteksi',
      title: 'Tanggal Proteksi Dibuat',
      width: '10rem',
      value: (p) => p.tanggal_proteksi,
      render: (p) => (
        <span className="tabular-nums">
          {p.tanggal_proteksi ? formatDate(p.tanggal_proteksi) : '—'}
        </span>
      ),
    },
    {
      key: 'keterangan',
      title: 'Keterangan',
      value: (p) => p.keterangan,
      render: (p) => (
        <span className="truncate" title={p.keterangan}>
          {p.keterangan || '—'}
        </span>
      ),
    },
    {
      key: 'user_create',
      title: 'User Create',
      width: '10rem',
      value: (p) => p.user_create,
      render: (p) => <span className="truncate">{p.user_create || '—'}</span>,
    },
  ]

  const decisionFailure = decide.error instanceof APIError ? decide.error.message : null

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Accept Open Protection</h1>
        <p className="mt-1 text-sm text-slate-600">
          Permintaan pembukaan proteksi yang sudah lengkap dan menunggu keputusan. Baris
          hilang begitu diputuskan — juga bila yang memutuskan petugas lain.
        </p>
      </header>

      {portal === null && (
        <div className="mt-6">
          <ErrorMessage
            title="Pilih entitas lebih dulu"
            description="Layar ini membaca proteksi milik satu badan hukum, sehingga entitasnya harus dipilih di bilah atas."
            tone="gangguan"
          />
        </div>
      )}

      <div className="mt-6 flex flex-wrap items-center gap-2" role="tablist" aria-label="Antrean akseptasi">
        <QueueTab current={queue} value="non-premi" label="Proteksi Klaim NON PREMI" onPick={changeQueue} />
        <QueueTab current={queue} value="premi" label="Proteksi Klaim PREMI" onPick={changeQueue} />
      </div>

      {opened !== null && (
        <section className="mt-6 rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <h2 className="mb-4 text-base font-semibold text-slate-900">Akseptasi Proteksi</h2>

          {decisionFailure && (
            <div className="mb-4">
              <ErrorMessage
                title="Keputusan tidak dapat disimpan"
                description={decisionFailure}
                tone="penolakan"
              />
            </div>
          )}

          {detail.isPending ? (
            <p className="text-sm text-slate-500">Memuat rincian…</p>
          ) : detail.error ? (
            <ErrorMessage
              title="Rincian tidak dapat dibuka"
              description={
                detail.error instanceof APIError
                  ? detail.error.message
                  : 'Terjadi kesalahan saat memuat rincian.'
              }
              tone="gangguan"
            />
          ) : detail.data ? (
            <>
              <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
                <Detail label="No Proteksi" value={detail.data.nomor_proteksi} mono />
                <Detail label="No Polis" value={detail.data.nomor_polis} mono />
                <Detail label="No Klaim" value={detail.data.nomor_klaim} mono />
                <Detail label="Nama Tertanggung" value={detail.data.nama_tertanggung} />
                <Detail
                  label="Start Date Time"
                  value={detail.data.polis_mulai ? formatDate(detail.data.polis_mulai) : ''}
                />
                <Detail
                  label="End Date Time"
                  value={detail.data.polis_akhir ? formatDate(detail.data.polis_akhir) : ''}
                />
                <Detail
                  label="Tipe Proteksi"
                  value={protectionTypeLabel(detail.data.tipe_proteksi, detail.data.nama_tipe_proteksi)}
                />
                <Detail
                  label="Tanggal Input"
                  value={
                    detail.data.tanggal_proteksi ? formatDate(detail.data.tanggal_proteksi) : ''
                  }
                />
                <Detail label="User Create" value={detail.data.user_create} />
                <div className="sm:col-span-2 lg:col-span-3">
                  <Detail label="Keterangan" value={detail.data.keterangan} />
                </div>
              </dl>

              <div className="mt-5 flex flex-wrap items-center gap-3">
                {detail.data.menunggu_keputusan ? (
                  <>
                    <Button
                      tone="utama"
                      onClick={() => void submit('setuju')}
                      disabled={decide.isPending}
                    >
                      {decide.isPending ? 'Menyimpan…' : 'Setujui'}
                    </Button>
                    <Button
                      tone="kedua"
                      onClick={() => void submit('tolak')}
                      disabled={decide.isPending}
                    >
                      Tolak
                    </Button>
                  </>
                ) : (
                  // Sudah diputuskan: tombolnya tidak ditampilkan sama sekali, dan yang
                  // ditampilkan adalah keputusannya beserta pelakunya. Menampilkan tombol
                  // yang pasti ditolak hanya membuat pengguna mencobanya.
                  <p className="text-sm text-slate-600">
                    Sudah{' '}
                    <span className="font-medium">
                      {detail.data.status_akseptasi === '1' ? 'disetujui' : 'ditolak'}
                    </span>
                    {detail.data.diaksep_oleh && <> oleh {detail.data.diaksep_oleh}</>}
                    {detail.data.tanggal_akseptasi && (
                      <> pada {formatDate(detail.data.tanggal_akseptasi)}</>
                    )}
                    .
                  </p>
                )}

                <Button tone="halus" onClick={() => setOpened(null)} disabled={decide.isPending}>
                  Tutup
                </Button>
              </div>
            </>
          ) : null}
        </section>
      )}

      <div className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(p) => p.nomor_proteksi}
          title={queue === 'premi' ? 'Proteksi Klaim PREMI' : 'Proteksi Klaim NON PREMI'}
          label={`Antrean akseptasi ${queue}`}
          // Prop tidak dikirim sama sekali saat kosong: tsconfig memakai
          // exactOptionalPropertyTypes, yang membedakan "tidak ada" dari "undefined".
          {...(total > 0 ? { description: `${total} permintaan menunggu keputusan.` } : {})}
          isLoading={list.isPending && portal !== null}
          searchLabel="Cari No Proteksi / No Polis / No Klaim"
          emptyMessage="Tidak ada permintaan proteksi yang menunggu keputusan pada antrean ini."
          serverSearch={{ value: search, onChange: changeSearch, matchCount: total }}
          pagination={{
            page: Math.floor(offset / PAGE_SIZE) + 1,
            size: PAGE_SIZE,
            total,
            totalPage: Math.max(1, Math.ceil(total / PAGE_SIZE)),
            onPageChange: (page) => setOffset((page - 1) * PAGE_SIZE),
            isLoading: list.isFetching,
          }}
          actions={
            <Button
              tone="halus"
              onClick={() => void list.refetch()}
              disabled={list.isFetching || portal === null}
            >
              <ReloadIcon className="h-4 w-4" />
              {list.isFetching ? 'Memuat…' : 'Muat ulang'}
            </Button>
          }
        />
      </div>

      <p className="mt-4 text-xs text-slate-500">
        Pemisahan antrean di sistem lama mengikuti peran pengguna, bukan pilihan. Selama
        tabel peran belum terisi, kedua antrean dapat dibuka siapa pun yang berhak masuk
        layar ini.
      </p>
    </div>
  )
}

/** Satu tab antrean. */
function QueueTab({
  current,
  value,
  label,
  onPick,
}: {
  current: Queue
  value: Queue
  label: string
  onPick: (queue: Queue) => void
}) {
  const active = current === value

  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={() => onPick(value)}
      className={
        active
          ? 'rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white'
          : 'rounded-lg bg-slate-100 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-200'
      }
    >
      {label}
    </button>
  )
}

/** Satu pasang label dan nilai pada rincian proteksi. */
function Detail({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className={mono ? 'mt-0.5 font-mono text-sm text-slate-800' : 'mt-0.5 text-sm text-slate-800'}>
        {value || '—'}
      </dd>
    </div>
  )
}
