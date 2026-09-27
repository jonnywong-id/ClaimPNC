import { useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { ReloadIcon } from '@/components/Icon'
import type { SelectOption } from '@/components/SelectField'

import {
  PAGE_SIZE,
  useCreateProtection,
  useProtectionDetail,
  useProtectionList,
  useProtectionTypes,
  useUpdateProtection,
} from './api'
import { ProtectionForm } from './ProtectionForm'
import {
  protectionTypeLabel,
  type Protection,
  type ProtectionFields,
} from './types'

/**
 * Layar Input Req Protection.
 *
 * Migrasi dari **`Harness/InputReqProtection_Harness-Harness.xml`** beserta section yang
 * dimuatnya, `Section/InboxReqProtection_Section-Section.xml`.
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Ketujuh kolom dan judulnya | `Section/InboxReqProtection_Section-Section.xml` |
 * | Baris yang tampil | `Report Definition/InboxReqOpenProtection_RD-RD.xml` |
 * | Tautan yang mati | `Section/…:8657` — `pyDisabledWhen` atas `.CaseID` |
 * | Tombol | `pyButtonLabel Input Open Protection` |
 * | Isi form | `Section/InputProtectionSection-Section.xml` |
 *
 * # Nama layarnya bersilang di Pega
 *
 * Butir menu yang membukanya bernama **"Input Req Protection"**, tetapi judul di dalam
 * harness-nya berbunyi "Inbox Open Protection" — dan nama itu juga dipakai butir menu LAIN
 * yang membuka layar akseptasi. Yang dipakai di sini adalah nama butir menunya (`D-81`).
 *
 * # Apa yang tampil
 *
 * Seluruh permintaan proteksi yang **belum diakseptasi** — `.AcceptStatus IS NULL` adalah
 * satu-satunya penyaring RD rujukan. Barisnya karena itu memuat dua keadaan sekaligus:
 * yang masih rancangan (belum tertaut klaim, masih dapat disunting) dan yang sudah lengkap
 * (menunggu giliran di meja akseptasi).
 *
 * # Yang sengaja dibuat berbeda dari Pega
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Pencarian | tidak ada | ada, dikerjakan server |
 * | Baris terkunci | tautan mati tanpa keterangan | mati **dan** dijelaskan sebabnya |
 * | Tipe proteksi | label dari rule Property | kode apa adanya bila labelnya tidak diketahui |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 *
 * Pencarian ditambahkan karena daftarnya hanya dibatasi "belum diakseptasi", sehingga ia
 * tumbuh tanpa batas seiring waktu.
 *
 * # Yang BELUM ada
 *
 * Pencarian klaim pada form — `.PNCCaseID` diisi manual, bukan lewat tombol cari. Ia
 * menembak modul klaim, yang rutenya belum tersedia.
 */
export function ProtectionListPage() {
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)

  // null berarti form tertutup; '' berarti form terbuka untuk membuat baru; berisi nomor
  // berarti form terbuka untuk menyunting.
  const [editing, setEditing] = useState<string | null>(null)

  const portal = useSelectedPortal((state) => state.alias)

  const list = useProtectionList({ search, offset })
  const detail = useProtectionDetail(editing !== null && editing !== '' ? editing : null)

  // Master tipe dibaca SEKALI di layar ini lalu diteruskan ke form sebagai props.
  // Memanggil hook-nya lagi di dalam form akan membuat dua komponen memegang keadaan
  // pemuatan yang sama — dan yang satu dapat menampilkan pilihan sementara yang lain masih
  // memuat.
  const types = useProtectionTypes()
  const create = useCreateProtection()
  const update = useUpdateProtection()

  const rows = list.data?.proteksi ?? []
  const total = list.data?.total ?? 0

  function changeSearch(next: string) {
    setSearch(next)
    // Halaman dikembalikan ke awal. Tanpa ini, mencari dari halaman empat akan menampilkan
    // tabel kosong yang tampak rusak.
    setOffset(0)
  }

  function closeForm() {
    setEditing(null)
    create.reset()
    update.reset()
  }

  async function submit(values: ProtectionFields) {
    // Galat DITELAN dengan sengaja, dan itu bukan kelalaian.
    //
    // Yang menampilkannya adalah form, lewat `create.error` / `update.error` — di sanalah
    // pelanggaran per kolom ditempelkan ke kolomnya masing-masing. Membiarkan promise ini
    // menolak akan menghasilkan **unhandled rejection** di peramban: galat yang sama
    // muncul dua kali, satu di layar dan satu di konsol, dan yang kedua tidak berguna bagi
    // siapa pun.
    //
    // Form TIDAK ditutup saat gagal — isian pengguna dipertahankan supaya ia dapat
    // memperbaikinya, bukan mengetiknya ulang.
    try {
      if (editing !== null && editing !== '') {
        await update.mutateAsync({ number: editing, values })
      } else {
        await create.mutateAsync(values)
      }
      closeForm()
    } catch {
      // Sudah tercatat di mutation.error; lihat komentar di atas.
    }
  }

  /**
   * Pilihan tipe proteksi, dari master `POOLDATA.M_CLAIM_PROTECTION_TYPE`.
   *
   * Sampai 2026-09-24 daftarnya HANYA TIGA — satu-satunya yang artinya terbukti dari export
   * (`R-16`) — dan kode lain ditampilkan apa adanya tanpa pernah ditawarkan untuk dipilih.
   * Master yang diterima memuat kesembilannya, sehingga pilihannya kini lengkap.
   *
   * Bila masternya gagal dibaca, daftarnya KOSONG — bukan jatuh ke daftar cadangan di kode.
   * Pilihan yang berasal dari kode akan membuat master yang bermasalah tampak beres, dan
   * proteksi tersimpan dengan tipe yang tidak dikenal basis datanya sendiri.
   */
  const typeOptions: SelectOption[] = (types.data?.tipe ?? []).map((t) => ({
    value: t.kode,
    label: protectionTypeLabel(t.kode, t.nama),
  }))

  /** Ketujuh kolom mengikuti `Section/InboxReqProtection_Section-Section.xml`. */
  const columns: Column<Protection>[] = [
    {
      key: 'nomor_proteksi',
      title: 'No Proteksi',
      width: '11rem',
      value: (p) => p.nomor_proteksi,
      render: (p) =>
        p.dapat_disunting ? (
          <button
            type="button"
            onClick={() => setEditing(p.nomor_proteksi)}
            className="truncate rounded font-mono text-xs font-medium text-blue-700 underline-offset-2 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          >
            {p.nomor_proteksi}
          </button>
        ) : (
          // Tautan mati, persis `pyDisabledWhen` layar lama — tetapi DENGAN keterangan.
          // Layar lama hanya mematikannya, dan pengguna tidak punya cara mengetahui
          // sebabnya.
          <span
            className="truncate font-mono text-xs text-slate-500"
            title="Sudah tertaut klaim, sehingga tidak dapat diubah lagi"
          >
            {p.nomor_proteksi}
          </span>
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
      render: (p) =>
        p.nomor_klaim ? (
          <span className="truncate font-mono text-xs text-slate-700">{p.nomor_klaim}</span>
        ) : (
          <span className="text-slate-400" title="Belum ditautkan ke klaim mana pun">
            belum tertaut
          </span>
        ),
    },
    {
      key: 'tipe_proteksi',
      title: 'Tipe Proteksi',
      width: '12rem',
      value: (p) => p.tipe_proteksi,
      render: (p) => (
        <span className="truncate">
          {protectionTypeLabel(p.tipe_proteksi, p.nama_tipe_proteksi)}
          {p.premi && (
            <span className="ml-2 rounded bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700 ring-1 ring-amber-100">
              PREMI
            </span>
          )}
        </span>
      ),
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

  const formOpen = editing !== null
  const submitting = create.isPending || update.isPending
  const failure = create.error ?? update.error

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Input Req Protection</h1>
        <p className="mt-1 text-sm text-slate-600">
          Permintaan pembukaan proteksi yang belum diakseptasi pada entitas yang sedang
          dibuka. Baris yang sudah tertaut klaim tidak dapat diubah lagi.
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

      {formOpen && (
        <section className="mt-6 rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <h2 className="mb-4 text-base font-semibold text-slate-900">
            {editing ? 'Ubah Permintaan Proteksi' : 'Input Open Protection'}
          </h2>

          {editing !== '' && detail.isPending ? (
            <p className="text-sm text-slate-500">Memuat isian…</p>
          ) : editing !== '' && detail.error ? (
            <ErrorMessage
              title="Permintaan tidak dapat dibuka"
              description={
                detail.error instanceof APIError
                  ? detail.error.message
                  : 'Terjadi kesalahan saat memuat isian.'
              }
              tone="gangguan"
            />
          ) : (
            <ProtectionForm
              {...(editing !== '' && detail.data ? { existing: detail.data } : {})}
              typeOptions={typeOptions}
              onSubmit={submit}
              onCancel={closeForm}
              isSubmitting={submitting}
              failure={failure}
            />
          )}
        </section>
      )}

      <div className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(p) => p.nomor_proteksi}
          title="Permintaan proteksi"
          // Prop tidak dikirim sama sekali saat kosong, bukan dikirim bernilai undefined:
          // tsconfig memakai exactOptionalPropertyTypes, yang membedakan keduanya.
          {...(total > 0 ? { description: `${total} permintaan menunggu akseptasi.` } : {})}
          isLoading={list.isPending && portal !== null}
          searchLabel="Cari No Proteksi / No Polis / No Klaim"
          emptyMessage="Tidak ada permintaan proteksi yang menunggu akseptasi."
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
            <>
              <Button
                tone="halus"
                onClick={() => void list.refetch()}
                disabled={list.isFetching || portal === null}
              >
                <ReloadIcon className="h-4 w-4" />
                {list.isFetching ? 'Memuat…' : 'Muat ulang'}
              </Button>
              <Button
                tone="utama"
                onClick={() => {
                  create.reset()
                  update.reset()
                  setEditing('')
                }}
                disabled={portal === null}
              >
                Input Open Protection
              </Button>
            </>
          }
        />
      </div>
    </div>
  )
}
