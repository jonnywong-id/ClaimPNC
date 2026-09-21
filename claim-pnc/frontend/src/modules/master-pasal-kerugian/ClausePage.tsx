import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Clause, type ClauseInput } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import {
  useClause,
  useClauseCategoryList,
  useClauseList,
  useCreateClause,
  useDeleteClause,
  useUpdateClause,
} from './api'
import { ClauseForm } from './ClauseForm'

type MessageContent = { title: string; description: string; tone: ErrorTone }

function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan Anda, lalu muat ulang.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return { title: 'Daftar tidak dapat dimuat', description: 'Coba beberapa saat lagi.', tone: 'gangguan' }
}

/** Nada lencana kategori, mengikuti arti ketiga kodenya. */
function categoryClass(code: string): string {
  switch (code) {
    case '1':
      return 'bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200'
    case '2':
      return 'bg-red-50 text-red-800 ring-1 ring-red-200'
    default:
      return 'bg-slate-100 text-slate-700 ring-1 ring-slate-200'
  }
}

/**
 * Layar Master Pasal Kerugian.
 *
 * Pengganti `Harness/DetailMasterPasalRejected-Harness.xml` (MENU_ID 27) — judulnya di
 * Pega "Detail Pasal Kerugian", dan yang dikelolanya adalah daftar baku butir ketentuan
 * polis yang dirujuk saat klaim dinilai.
 *
 * Susunan kolom, urutan isian, dan keempat tombolnya mengikuti layar lama (`D-13`: alur
 * dan tata letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *	Kolom grid  — No Pasal · ISI PASAL · Deskripsi · Kategori · Action
 *	              (`Section/BrowsePasalDeatailMaster-Section.xml`)
 *	Tombol      — Tambah dan Refresh di pembungkusnya
 *	              (`Section/GridDetailMasterPasalRejected-Section.xml`),
 *	              Ubah per baris, Simpan dan Delete pada form
 *
 * # Tombol Hapus MENGHAPUS PERMANEN
 *
 * `D-66` menetapkan soft delete menyeluruh, tetapi POOLDATA.V_M_DATA_PASAL hanya punya
 * tiga kolom dan tidak punya penanda terhapus; menambah kolom menempuh `D-63`. Work Owner
 * memilih "jalankan as is" pada 2026-09-19 — yaitu `DELETE` fisik seperti Pega.
 *
 * Yang ditambahkan layar ini adalah KONFIRMASI sebelum menghapus, yang di Pega tidak ada.
 * Ia bukan perubahan aturan bisnis: barisnya tetap terhapus permanen, tetapi tidak lagi
 * hilang hanya karena satu klik yang tidak disengaja pada tabel yang padat. Tabel ini juga
 * tidak punya kolom pencatat siapa dan kapan, sehingga tidak ada satu pun jejak yang dapat
 * dipakai menelusuri penghapusan sesudahnya.
 */
export function ClausePage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [editedID, setEditedID] = useState<string | null>(null)
  const [isFormOpen, setFormOpen] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<Clause | null>(null)

  const list = useClauseList()
  const category = useClauseCategoryList()
  const detail = useClause(editedID ?? '')
  const create = useCreateClause()
  const update = useUpdateClause()
  const remove = useDeleteClause()

  const saving = editedID === null ? create : update
  const edited = editedID === null ? null : (detail.data?.pasal_kerugian ?? null)

  function openCreate() {
    create.reset()
    update.reset()
    remove.reset()
    setEditedID(null)
    setPendingDelete(null)
    setFormOpen(true)
  }

  function openEdit(row: Clause) {
    create.reset()
    update.reset()
    remove.reset()
    setEditedID(row.id)
    setPendingDelete(null)
    setFormOpen(true)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setEditedID(null)
    setFormOpen(false)
  }

  // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
  // membuang isian pengguna saat penyimpanan gagal — dan pada Isi Pasal yang panjang, itu
  // berarti mengetik ulang dari awal.
  function save(input: ClauseInput) {
    if (editedID === null) {
      create.mutate(input, { onSuccess: closeForm })
      return
    }
    update.mutate({ id: editedID, input }, { onSuccess: closeForm })
  }

  function confirmDelete() {
    if (pendingDelete === null) return

    const id = pendingDelete.id
    remove.mutate(id, {
      onSuccess: () => {
        setPendingDelete(null)
        // Form ditutup bila yang terhapus justru baris yang sedang disunting; kalau tidak,
        // form akan tetap terbuka atas baris yang sudah tidak ada.
        if (editedID === id) closeForm()
      },
    })
  }

  // Susunan kolom mengikuti grid Pega apa adanya, terbaca dari label pada
  // `Section/BrowsePasalDeatailMaster-Section.xml`:
  //
  //	No Pasal   -> .M_COL_ID
  //	ISI PASAL  -> .DESCRIPTION
  //	Deskripsi  -> .OLD_D_COL_ID
  //	Kategori   -> .LOSS_CODE
  //	Action     -> tombol Ubah
  //
  // Perhatikan "ISI PASAL" dan "Deskripsi" memang menunjuk properti yang terbalik dari
  // dugaan yang wajar; itu bentuk aslinya.
  const columns: Column<Clause>[] = [
    { key: 'no_pasal', title: 'No Pasal', width: 'w-32', value: (row) => row.no_pasal },
    {
      key: 'isi_pasal',
      title: 'Isi Pasal',
      value: (row) => row.isi_pasal,
      render: (row) => (
        <span className="block max-w-xl truncate" title={row.isi_pasal}>
          {row.isi_pasal === '' ? <span className="text-slate-400">—</span> : row.isi_pasal}
        </span>
      ),
    },
    {
      key: 'deskripsi',
      title: 'Deskripsi',
      value: (row) => row.deskripsi,
      render: (row) =>
        row.deskripsi === '' ? <span className="text-slate-400">—</span> : <span>{row.deskripsi}</span>,
    },
    {
      key: 'kategori',
      title: 'Kategori',
      width: 'w-40',
      value: (row) => row.kategori_label,
      render: (row) => (
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${categoryClass(row.kategori)}`}
        >
          {row.kategori_label}
        </span>
      ),
    },
    {
      key: 'aksi',
      title: '',
      width: 'w-40',
      // Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya
      // tombol, bukan data.
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <span className="inline-flex gap-2">
          <Button tone="kedua" onClick={() => openEdit(row)} aria-label={`Ubah ${row.no_pasal}`}>
            Ubah
          </Button>
          <Button
            tone="halus"
            onClick={() => setPendingDelete(row)}
            aria-label={`Hapus ${row.no_pasal}`}
          >
            Hapus
          </Button>
        </span>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Master Pasal Kerugian</h1>
          <p className="text-sm text-slate-600">
            Daftar baku butir ketentuan polis yang dirujuk saat klaim dinilai.
          </p>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        Portal entitas: <span className="font-medium text-slate-700">{portal ?? '—'}</span>
      </p>

      {portal === null ? (
        <section className="mt-6">
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        </section>
      ) : (
        <>
          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
              {list.isFetching ? 'Memuat…' : 'Refresh'}
            </Button>
            <Button tone="utama" onClick={openCreate} disabled={isFormOpen && editedID === null}>
              Tambah
            </Button>
          </div>

          {/* Konfirmasi penghapusan. Ia bukan modal: modal menuntut perangkap fokus dan
              penanganan Escape sendiri, sedangkan yang dibutuhkan di sini hanyalah agar
              akibatnya terbaca SEBELUM tombolnya ditekan — dan panel di tempat justru tidak
              menutupi baris yang sedang dibicarakan. */}
          {pendingDelete && (
            <section className="mt-5 rounded-lg border border-red-200 bg-red-50 p-5" role="alertdialog" aria-labelledby="hapus-judul">
              <h2 id="hapus-judul" className="text-base font-semibold text-red-900">
                Hapus pasal {pendingDelete.no_pasal}?
              </h2>
              <p className="mt-1.5 text-sm text-red-800">
                Baris ini akan dihapus <strong>permanen</strong> dan tidak dapat dipulihkan. Tabelnya
                tidak mencatat siapa yang menghapus maupun kapan, sehingga tidak ada jejak yang dapat
                ditelusuri sesudahnya.
              </p>
              {remove.isError && (
                <p className="mt-2 text-sm text-red-900" role="alert">
                  Penghapusan gagal. Baris mungkin sudah dihapus petugas lain — muat ulang daftarnya.
                </p>
              )}
              <div className="mt-4 flex flex-wrap justify-end gap-2">
                <Button tone="halus" onClick={() => setPendingDelete(null)} disabled={remove.isPending}>
                  Batal
                </Button>
                <Button tone="utama" onClick={confirmDelete} disabled={remove.isPending}>
                  {remove.isPending ? 'Menghapus…' : 'Hapus permanen'}
                </Button>
              </div>
            </section>
          )}

          {isFormOpen && (
            <section className="mt-5">
              <ClauseForm
                edited={edited}
                isLoading={editedID !== null && detail.isPending}
                category={category.data?.kategori ?? []}
                isSaving={saving.isPending}
                error={saving.error}
                onSave={save}
                onCancel={closeForm}
              />
            </section>
          )}

          <section className="mt-6">
            {list.isPending ? (
              <p className="text-sm text-slate-500">Memuat daftar pasal kerugian…</p>
            ) : list.isError ? (
              <LoadError error={list.error} />
            ) : (
              <DataTable
                columns={columns}
                rows={list.data.pasal_kerugian}
                rowKey={(row) => row.id}
                description="Sumber: POOLDATA.V_M_DATA_PASAL · lini bisnis terlihat saat pasal dibuka"
                emptyMessage="Belum ada pasal kerugian pada entitas ini."
              />
            )}
          </section>
        </>
      )}
    </main>
  )
}

function LoadError({ error }: { error: unknown }) {
  const message = loadMessage(error)
  return (
    <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
  )
}
