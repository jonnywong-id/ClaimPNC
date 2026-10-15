import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { XOL } from '@/api/types'
import { ErrorCode } from '@/api/types'
import { AddIcon, EditIcon, ReloadIcon } from '@/components/Icon'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import { useXOLList } from './api'
import { XOLForm } from './XOLForm'
import { formatMoney } from './format'

/**
 * Layar Master XOL.
 *
 * Menggantikan harness `DetailMasterXOL` beserta section `DetailXOL` dan `DetailXOL_sec`.
 *
 * # Kolomnya SAMA PERSIS dengan layar Pega
 *
 * Grid induk di Pega memuat **empat kolom dan satu tombol**, tidak lebih:
 *
 *	ID · Tahun · Kurs · Remark Komite · [Update]
 *
 * Keempatnya terbaca dari `Section/DetailXOL_sec-Section.xml` sebagai header
 * `<b>ID</b>`, `<b>Tahun</b>`, `<b>Kurs</b>`, `<b>Remark Komite</b>`, dan dikonfirmasi
 * Work Owner terhadap layar produksi pada 2026-10-05.
 *
 * Versi pertama layar ini menambahkan **Nama**, **Type XOL**, dan **Status Komite** —
 * ketiganya memang ada di tabel, tetapi TIDAK ada di grid Pega. Ketiganya dibuang: kolom
 * yang tidak ada di layar lama adalah kolom yang dikarang, dan `D-13` menetapkan tata
 * letak mengikuti Pega apa adanya supaya pengguna tidak perlu belajar ulang.
 *
 * **Tidak ada tombol Hapus di grid ini**, juga mengikuti Pega. Penghapusan baris hanya ada
 * di dalam form, pada ketiga grid anaknya — sama seperti tombol Hapus yang dideklarasikan
 * `Section/DetailXOL_sec-Section.xml`. Endpoint hapus induk tetap ada di server dan dapat
 * dipasang kembali bila Work Owner memintanya.
 *
 * # Yang tetap berbeda, dan alasannya
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Urutan baris | ditentukan basis data | numerik menurut ID, tetap |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | penyaring per kolom | satu kotak cari yang menelusuri seluruh kolom |
 * | Portal entitas | tidak ada | disebut terang-terangan (`R-20`) |
 */
export function XOLPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useXOLList()

  // null = sedang menambah; berisi = sedang mengubah induk itu.
  const [editing, setEditing] = useState<XOL | null>(null)
  const [formOpen, setFormOpen] = useState(false)

  /**
   * Kabar penyimpanan berhasil, ditampilkan di atas daftar.
   *
   * Ia tinggal di sini, bukan di dalam form, karena penyimpanan yang bersih MENUTUP form —
   * kabar yang dipasang di dalamnya akan ikut hilang sebelum sempat dibaca.
   */
  const [kabar, setKabar] = useState('')

  function openAdd() {
    setKabar('')
    setEditing(null)
    setFormOpen(true)
  }

  function openEdit(master: XOL) {
    setKabar('')
    setEditing(master)
    setFormOpen(true)
  }

  function closeForm(pesan?: string) {
    setKabar(pesan ?? '')
    setFormOpen(false)
    setEditing(null)
  }

  const columns: Column<XOL>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '7rem',
      value: (m) => m.id,
      render: (m) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {m.id}
        </span>
      ),
    },
    { key: 'tahun', title: 'Tahun', width: '7rem', value: (m) => m.tahun },
    {
      key: 'kurs',
      title: 'Kurs',
      width: '9rem',
      alignRight: true,
      value: (m) => String(m.kurs),
      render: (m) => <span className="font-mono text-xs">{formatMoney(m.kurs)}</span>,
    },
    {
      key: 'remark_komite',
      title: 'Remark Komite',
      value: (m) => m.remark_komite,
      render: (m) =>
        m.remark_komite ? (
          <span className="text-slate-700">{m.remark_komite}</span>
        ) : (
          <span className="text-slate-300">—</span>
        ),
    },
    {
      key: 'aksi',
      title: '',
      width: '7rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (m) => (
        // Labelnya "Update", bukan "Ubah" — itu teks tombol di layar Pega, dan `D-13`
        // menetapkan teks yang dilihat pengguna mengikuti layar lama apa adanya.
        <Button tone="halus" onClick={() => openEdit(m)} aria-label={`Update master XOL ${m.id}`}>
          <EditIcon className="h-3.5 w-3.5" />
          Update
        </Button>
      ),
    },
  ]

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div>
          <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
            <ol className="flex items-center gap-1.5">
              <li>Master Data</li>
              <li aria-hidden="true" className="text-slate-300">
                /
              </li>
              <li className="text-slate-700">XOL</li>
            </ol>
          </nav>
          <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
            Detail Master XOL
          </h1>

          {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
              empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak
              boleh hanya diandaikan pengguna (ADR-0030, R-20). */}
          <p className="mt-2 text-xs text-slate-500">
            Portal entitas:{' '}
            <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
          </p>
        </div>

        {/* Kedua tombol berada di kanan atas halaman, bukan di atas grid — sama seperti
            layar Pega, dan dengan teks yang sama. */}
        <div className="flex gap-2">
          <Button tone="utama" onClick={openAdd} disabled={formOpen && editing === null}>
            <AddIcon className="h-4 w-4" />
            Tambah
          </Button>
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {kabar !== '' && (
        <div
          role="status"
          className="mb-6 flex items-start justify-between gap-3 rounded-md border border-emerald-300 bg-emerald-50 px-4 py-3 text-sm font-medium text-emerald-800"
        >
          <span>{kabar}</span>
          <button
            type="button"
            onClick={() => setKabar('')}
            aria-label="Tutup pesan"
            className="shrink-0 text-emerald-700 hover:text-emerald-900"
          >
            ×
          </button>
        </div>
      )}

      {formOpen && (
        <div className="mb-6">
          {/* `key` memaksa form dirakit ulang saat berpindah baris. Tanpa itu, state di
              dalamnya — termasuk nomor induk yang sedang disunting — terbawa dari baris
              sebelumnya, dan menyimpan akan menimpa induk yang salah. */}
          <XOLForm key={editing?.id ?? 'baru'} master={editing} tutup={closeForm} />
        </div>
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
          searchLabel="Cari ID, tahun, kurs, atau remark komite"
          emptyMessage="Data Tidak Ada"
          isLoading={list.isPending}
          error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
        />
      )}
    </div>
  )
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
      description: 'Daftar master XOL belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
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
