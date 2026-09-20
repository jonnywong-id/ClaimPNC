import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { XOL } from '@/api/types'
import { ErrorCode } from '@/api/types'
import { AddIcon, EditIcon, ReloadIcon, TrashIcon } from '@/components/Icon'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import { useDeleteXOL, useXOLList } from './api'
import { XOLForm } from './XOLForm'
import { formatMoney, committeeLabel, committeeTone } from './format'

/**
 * Layar Master XOL.
 *
 * Menggantikan harness `DetailMasterXOL` beserta section `DetailXOL` dan `DetailXOL_sec`.
 *
 * # Yang ditiru dari layar lama
 *
 * Judulnya, kedua tombolnya (**Tambah** dan **Refresh**), dan bentuk dasarnya: satu grid
 * induk di bawah, satu form bertingkat di atasnya. Label isian pun diambil apa adanya —
 * "Tahun", "Kurs IDR", "Type XOL", "Remark PIC", "Limit (USD)", "Excess (USD)",
 * "Share (%)" (`D-13`).
 *
 * **Menyimpan sekaligus mengajukan ke komite**, tanpa syarat — itulah yang dilakukan
 * `InsertUpdateMasterXOL` pada dua langkah terakhirnya. Layar menyatakannya terang-
 * terangan supaya pengguna tidak terkejut menemukan induk yang sudah disetujui kembali
 * berstatus menunggu.
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Hapus induk | satu tabel saja; anaknya tertinggal | **berkaskade** — Work Owner 2026-09-20 |
 * | Total share ≠ 100% | pesan muncul setelah tersimpan | **sama** — peringatan, bukan penolakan |
 * | Urutan baris | ditentukan basis data | numerik menurut ID, tetap |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | tidak ada | satu kotak cari yang menelusuri seluruh kolom |
 */
export function XOLPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useXOLList()
  const remove = useDeleteXOL()

  // null = sedang menambah; berisi = sedang mengubah induk itu.
  const [editing, setEditing] = useState<XOL | null>(null)
  const [formOpen, setFormOpen] = useState(false)

  // Induk yang menunggu konfirmasi hapus. Konfirmasi dua langkah dipakai karena hapus di
  // sini BERKASKADE — satu klik membuang seluruh layer dan reas-nya sekaligus.
  const [confirming, setConfirming] = useState<XOL | null>(null)

  function openAdd() {
    setEditing(null)
    setFormOpen(true)
  }

  function openEdit(master: XOL) {
    setEditing(master)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setEditing(null)
  }

  const columns: Column<XOL>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '6rem',
      value: (m) => m.id,
      render: (m) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {m.id}
        </span>
      ),
    },
    {
      key: 'nama',
      title: 'Nama',
      value: (m) => m.nama,
      render: (m) =>
        m.nama ? (
          <span className="font-medium text-slate-900">{m.nama}</span>
        ) : (
          // Nama kosong diterima — layar lama pun tidak mewajibkannya. Ditandai, bukan
          // dibiarkan sebagai sel kosong yang terlihat seperti tabel rusak.
          <span className="italic text-slate-400" title="Induk ini tidak punya nama">
            (tanpa nama)
          </span>
        ),
    },
    { key: 'tahun', title: 'Tahun', width: '6rem', value: (m) => m.tahun },
    {
      key: 'kurs',
      title: 'Kurs IDR',
      width: '8rem',
      alignRight: true,
      value: (m) => String(m.kurs),
      render: (m) => <span className="font-mono text-xs">{formatMoney(m.kurs)}</span>,
    },
    {
      key: 'tipe',
      title: 'Type XOL',
      value: (m) => m.tipe_label || m.tipe,
      render: (m) =>
        m.tipe_label ? (
          <span className="text-slate-700">{m.tipe_label}</span>
        ) : (
          // Dua dari delapan induk produksi menyimpan TYPEXOL kosong. Ditampilkan apa
          // adanya, bukan dipaksa menjadi salah satu kode.
          <span className="italic text-slate-400">(belum dipilih)</span>
        ),
    },
    {
      key: 'status_komite',
      title: 'Status Komite',
      width: '9rem',
      value: (m) => committeeLabel(m.status_komite),
      render: (m) => {
        const tone = committeeTone(m.status_komite)
        return (
          <span
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ${tone}`}
          >
            {committeeLabel(m.status_komite)}
          </span>
        )
      },
    },
    {
      key: 'remark_komite',
      title: 'Remark Komite',
      value: (m) => m.remark_komite,
      render: (m) =>
        m.remark_komite ? (
          <span className="text-slate-600">{m.remark_komite}</span>
        ) : (
          <span className="text-slate-300">—</span>
        ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '11rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (m) => (
        <div className="flex justify-end gap-1.5">
          <Button tone="halus" onClick={() => openEdit(m)} aria-label={`Ubah master XOL ${m.id}`}>
            <EditIcon className="h-3.5 w-3.5" />
            Ubah
          </Button>
          <Button
            tone="halus"
            onClick={() => setConfirming(m)}
            aria-label={`Hapus master XOL ${m.id}`}
          >
            <TrashIcon className="h-3.5 w-3.5" />
            Hapus
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Master Data</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">XOL</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Master XOL</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Struktur treaty <em>Excess of Loss</em> per tahun: grup bisnis yang dicakup,
          lapisan beserta limit dan excess-nya, dan pembagian share ke para reasuradur.
          Angka di sini menentukan pembagian klaim pada perhitungan PLA dan DLA.
        </p>

        {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
            empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
            hanya diandaikan pengguna (ADR-0030, R-20). */}
        <p className="mt-3 text-xs text-slate-500">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </p>
      </header>

      {formOpen && (
        <div className="mb-6">
          <XOLForm master={editing} tutup={closeForm} />
        </div>
      )}

      {confirming && (
        <ConfirmDelete
          master={confirming}
          sedangHapus={remove.isPending}
          galat={remove.isError ? deleteMessage(remove.error) : null}
          batal={() => {
            remove.reset()
            setConfirming(null)
          }}
          lanjut={() => {
            remove.mutate(confirming.id, {
              onSuccess: () => {
                if (editing?.id === confirming.id) closeForm()
                setConfirming(null)
              },
            })
          }}
        />
      )}

      {portal === null ? (
        <ErrorMessage
          title="Portal entitas belum dipilih"
          description="Struktur treaty dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu."
          tone="penolakan"
        />
      ) : (
        <DataTable
          columns={columns}
          rows={list.data?.xol ?? []}
          rowKey={(m) => m.id}
          title="Daftar Master XOL"
          description={
            list.data
              ? `${list.data.total} master XOL terdaftar pada entitas ini.`
              : 'Memuat daftar master XOL…'
          }
          searchLabel="Cari ID, nama, tahun, atau status"
          emptyMessage="Belum ada master XOL yang terdaftar."
          isLoading={list.isPending}
          error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
          actions={
            <>
              <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
                <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
                {list.isFetching ? 'Memuat…' : 'Muat ulang'}
              </Button>
              <Button tone="utama" onClick={openAdd} disabled={formOpen && editing === null}>
                <AddIcon className="h-4 w-4" />
                Tambah
              </Button>
            </>
          }
        />
      )}
    </div>
  )
}

/**
 * Konfirmasi hapus.
 *
 * Ia ada karena hapus di layar ini BERKASKADE: satu klik membuang induk beserta seluruh
 * grup bisnis, lapisan, dan baris reas-nya. Sistem lama tidak berkaskade — dan justru itu
 * yang meninggalkan baris yatim di produksi — sehingga pengguna yang terbiasa dengan
 * layar lama tidak akan menduga akibatnya seluas ini.
 */
function ConfirmDelete({
  master,
  sedangHapus,
  galat,
  batal,
  lanjut,
}: {
  master: XOL
  sedangHapus: boolean
  galat: string | null
  batal: () => void
  lanjut: () => void
}) {
  const jumlahLayer = master.layer?.length ?? 0

  return (
    <div
      role="alertdialog"
      aria-labelledby="judul-hapus-xol"
      className="mb-6 rounded-lg border border-amber-300 bg-amber-50 p-4"
    >
      <h2 id="judul-hapus-xol" className="text-sm font-semibold text-amber-900">
        Hapus master XOL {master.id}
        {master.nama ? ` — ${master.nama}` : ''}?
      </h2>
      <p className="mt-1.5 text-sm leading-relaxed text-amber-800">
        Seluruh grup bisnis, layer
        {jumlahLayer > 0 ? ` (${jumlahLayer})` : ''}, dan baris reas di bawahnya ikut
        terhapus. Tindakan ini tidak dapat dibatalkan.
      </p>

      {galat && <p className="mt-2 text-sm font-medium text-rose-700">{galat}</p>}

      <div className="mt-3 flex gap-2">
        <Button tone="utama" onClick={lanjut} disabled={sedangHapus}>
          {sedangHapus ? 'Menghapus…' : 'Ya, hapus'}
        </Button>
        <Button tone="kedua" onClick={batal} disabled={sedangHapus}>
          Batal
        </Button>
      </div>
    </div>
  )
}

function deleteMessage(error: unknown): string {
  if (error instanceof NetworkError) {
    return 'Tidak dapat menghubungi server. Periksa koneksi lalu coba lagi.'
  }
  if (error instanceof APIError) {
    if (error.kode === ErrorCode.xolNotFound) {
      return 'Master XOL ini sudah tidak ada. Muat ulang daftarnya.'
    }
    return error.message
  }
  return 'Terjadi kesalahan pada sistem. Coba lagi.'
}

/**
 * Gagal memuat dibedakan dari gagal menyimpan.
 *
 * Yang di sini selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat salah
 * — ia baru membuka layarnya.
 */
function LoadErrorMessage({ error }: { error: unknown }) {
  const message = loadMessage(error)
  return (
    <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
  )
}

function loadMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Daftar master XOL belum dapat dimuat. Periksa koneksi lalu tekan Muat ulang.',
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
            'Struktur treaty dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu.',
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
          title: 'Daftar master XOL gagal dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Daftar master XOL gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
