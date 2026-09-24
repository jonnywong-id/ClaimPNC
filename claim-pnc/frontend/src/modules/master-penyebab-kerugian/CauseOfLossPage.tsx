import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { CauseOfLoss } from '@/api/types'
import { ReloadIcon, AddIcon, EditIcon } from '@/components/Icon'
import { ErrorCode } from '@/api/types'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import { useCauseOfLossList } from './api'
import { CauseOfLossForm } from './CauseOfLossForm'

/**
 * Layar Master Penyebab Kerugian.
 *
 * Menggantikan harness `CauseOfLossInbox` beserta dua section-nya: `BrowseCauseOfLoss`
 * (grid dan formnya) dan `GridCauseOfLoss` (kerangkanya).
 *
 * # Yang ditiru dari layar lama
 *
 * Fungsinya sama persis, dan seluruh teksnya diambil apa adanya dari export (`D-13`):
 *
 * | Yang tampil | Sumbernya di Pega |
 * |---|---|
 * | Judul **Master Penyebab Kerugian** | `pyValue` pada harness |
 * | Kolom **ID** | `pyValue` pada sel ber-`pyCellHeader=true` |
 * | Kolom **Deskripsi Kerugian** | idem |
 * | Tombol **Tambah** dan **Refresh** | `pyButtonLabel` pada harness |
 * | Aksi **Ubah** per baris | `pxLink` berlabel `Ubah` pada section |
 * | Judul form **Memperbaharui Data** | `pyTitle` pada section |
 *
 * **Tidak ada Hapus** — layar Pega pun tidak punya,
 * `Database/PEGA_M_CAUSE_OF_LOSS.prc` tidak punya cabangnya, dan `ADR-0012` melarang
 * master dihapus permanen.
 *
 * Grid Pega hanya memuat **dua kolom**, dan layar ini mengikutinya. `OLD_M_COL_ID` ada di
 * Report Definition dan ikut dikirim API, tetapi **tidak ditampilkan** — karena di layar
 * lama pun tidak.
 *
 * # Ini tingkat GOLONGAN saja
 *
 * Rincian penyebab kerugian — `D_CAUSE_OF_LOSS`, harness `DetailCauseOfLoss`, MENU_ID 38 —
 * butir menu tersendiri dan belum dibangun. Layar ini tidak menautkan ke sana, karena
 * tautannya pun tidak ada di layar Pega.
 *
 * # Yang sengaja dibuat berbeda
 *
 * Yang tersisa hanyalah hal yang tidak punya padanan langsung, dan seluruhnya soal
 * mekanisme — bukan soal isi layar:
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Urutan baris | ditentukan basis data | menurut ID, tetap |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | filter dropdown per kolom | satu kotak cari yang menelusuri kedua kolom |
 * | Deskripsi kosong | diterima | **tetap diterima** (Work Owner 2026-09-20) |
 * | Deskripsi ganda | diterima | **tetap diterima** (idem), dengan peringatan di form |
 * | Pesan hasil simpan | isian **Catatan** berisi kalimat dari procedure | galat berkode di form; berhasil menutup form |
 *
 * Baris terakhir perlu penjelasan: layar lama menampilkan `TempCauseOfLoss.pyNote`, yang
 * diisi `ErrMsg` procedure — dan pada jalur BERHASIL pun ia berisi kalimat, sehingga
 * pengguna harus membaca teksnya untuk tahu apakah simpanannya jadi. Kontrak galat itu
 * tidak dibawa (`D-68`), sehingga penanda berhasil-atau-gagal tidak lagi berupa teks.
 */
export function CauseOfLossPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useCauseOfLossList()

  // null = sedang menambah; berisi = sedang mengubah baris itu.
  const [sedangDisunting, setSedangDisunting] = useState<CauseOfLoss | null>(null)
  const [formTerbuka, setFormTerbuka] = useState(false)

  function openAdd() {
    setSedangDisunting(null)
    setFormTerbuka(true)
  }

  function openEdit(cause: CauseOfLoss) {
    setSedangDisunting(cause)
    setFormTerbuka(true)
  }

  function closeForm() {
    setFormTerbuka(false)
    setSedangDisunting(null)
  }

  const columns: Column<CauseOfLoss>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '7rem',
      value: (c) => c.id,
      render: (c) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {c.id}
        </span>
      ),
    },
    {
      key: 'deskripsi',
      // Judul kolom diambil apa adanya dari layar Pega (`D-13`): `pyValue` pada sel
      // ber-`pyCellHeader=true`, berdampingan dengan `pyLabelFor .COL_DESC`.
      title: 'Deskripsi Kerugian',
      value: (c) => c.deskripsi,
      render: (c) =>
        // Deskripsi kosong DITERIMA modul ini, sehingga ia pasti akan muncul di daftar.
        // Ditandai, bukan dibiarkan sebagai sel kosong yang terlihat seperti tabel rusak.
        c.deskripsi ? (
          <span className="font-medium text-slate-900">{c.deskripsi}</span>
        ) : (
          <span className="italic text-slate-400" title="Baris ini tidak punya deskripsi">
            (tanpa deskripsi)
          </span>
        ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '7rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (c) => (
        <Button
          tone="halus"
          onClick={() => openEdit(c)}
          // Label aksesibilitas jatuh ke ID bila deskripsinya kosong, supaya tombolnya
          // tetap dapat disebut pengguna papan ketik dan pembaca layar.
          aria-label={`Ubah penyebab kerugian ${c.deskripsi || c.id}`}
        >
          <EditIcon className="h-3.5 w-3.5" />
          Ubah
        </Button>
      ),
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
            <li className="text-slate-700">Penyebab Kerugian</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          Master Penyebab Kerugian
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Daftar acuan golongan sebab terjadinya kerugian sebuah klaim. Deskripsi yang
          diubah di sini dipakai mengelompokkan laporan klaim per penyebab kerugian, jadi
          akibatnya terbaca di luar layar ini.
        </p>

        {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
            empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
            hanya diandaikan pengguna (ADR-0030, R-20). */}
        <p className="mt-3 text-xs text-slate-500">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">
            {list.data?.portal ?? portal ?? '—'}
          </span>
        </p>
      </header>

      {formTerbuka && (
        <div className="mb-6">
          <CauseOfLossForm cause={sedangDisunting} tutup={closeForm} />
        </div>
      )}

      {portal === null ? (
        <ErrorMessage
          title="Portal entitas belum dipilih"
          description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu."
          tone="penolakan"
        />
      ) : (
        <DataTable
          columns={columns}
          rows={list.data?.penyebab_kerugian ?? []}
          rowKey={(c) => c.id}
          title="Daftar Penyebab Kerugian"
          description={
            list.data
              ? `${list.data.total} penyebab kerugian terdaftar pada entitas ini.`
              : 'Memuat daftar penyebab kerugian…'
          }
          searchLabel="Cari ID atau Deskripsi Kerugian"
          emptyMessage="Belum ada penyebab kerugian yang terdaftar."
          isLoading={list.isPending}
          error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
          actions={
            <>
              {/* Labelnya **Refresh**: itulah `pyButtonLabel` yang terpasang di harness
                  Pega, dan `D-13` menetapkan teks yang dilihat pengguna mengikuti layar
                  lama.

                  Modul master terdahulu sempat memakai "Muat ulang" meski ke-10 harness
                  master di Pega seluruhnya berbunyi "Refresh". Work Owner memutuskan
                  2026-09-20 agar semuanya mengikuti Pega, dan label itu sudah
                  diseragamkan di seluruh modul master. */}
              <Button
                tone="kedua"
                onClick={() => void list.refetch()}
                disabled={list.isFetching}
              >
                <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
                {list.isFetching ? 'Memuat…' : 'Refresh'}
              </Button>
              <Button tone="utama" onClick={openAdd} disabled={formTerbuka && !sedangDisunting}>
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
      description:
        'Daftar penyebab kerugian belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
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
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu.',
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
          title: 'Daftar penyebab kerugian gagal dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Daftar penyebab kerugian gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
