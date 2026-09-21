import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import type { DominantFactor } from '@/api/types'
import { ReloadIcon, AddIcon, EditIcon } from '@/components/Icon'
import { ErrorCode } from '@/api/types'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import { useDominantFactorList } from './api'
import { DominantFactorForm } from './DominantFactorForm'

/**
 * Layar Master Dominan Factor.
 *
 * Menggantikan harness `DetailDominanFactor` beserta section `DetailDominanFactor_Sec`.
 *
 * # Yang ditiru dari layar lama
 *
 * Fungsinya sama persis: daftar bergrid dengan kolom **ID** dan **Keterangan**, tombol
 * **Tambah**, tombol **Refresh**, dan aksi **Ubah** per baris. **Tidak ada Hapus** —
 * layar Pega pun tidak punya, `Database/PEGA_M_DOMINAN_FACTOR.prc` tidak punya cabangnya,
 * dan `ADR-0012` melarang master dihapus permanen karena klaim lama merujuknya.
 *
 * Judul form pun diambil apa adanya dari `pyTitle` layar lama: "Menambah Data" dan
 * "Memperbaharui Data" (`D-13`).
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Urutan baris | tanpa ORDER BY — ditentukan basis data | numerik menurut ID, tetap |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | tidak ada | satu kotak cari yang menelusuri ID dan keterangan |
 * | Keterangan kosong | diterima | **tetap diterima** (Work Owner 2026-09-20) |
 * | Keterangan ganda | diterima | **tetap diterima** (idem), dengan peringatan di form |
 *
 * Dua baris terakhir membedakan modul ini dari Master Status Klaim dan Master Tipe
 * Surveyors, yang justru menolak keduanya. Perbedaannya disengaja dan dicatat supaya
 * tidak "diseragamkan" tanpa keputusan baru.
 */
export function DominantFactorPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useDominantFactorList()

  // null = sedang menambah; berisi = sedang mengubah baris itu.
  const [sedangDisunting, setSedangDisunting] = useState<DominantFactor | null>(null)
  const [formTerbuka, setFormTerbuka] = useState(false)

  function openAdd() {
    setSedangDisunting(null)
    setFormTerbuka(true)
  }

  function openEdit(factor: DominantFactor) {
    setSedangDisunting(factor)
    setFormTerbuka(true)
  }

  function closeForm() {
    setFormTerbuka(false)
    setSedangDisunting(null)
  }

  const columns: Column<DominantFactor>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '7rem',
      value: (f) => f.id,
      render: (f) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {f.id}
        </span>
      ),
    },
    {
      key: 'nama',
      title: 'Keterangan',
      value: (f) => f.nama,
      render: (f) =>
        // Keterangan kosong DITERIMA modul ini, sehingga ia pasti akan muncul di daftar.
        // Ditandai, bukan dibiarkan sebagai sel kosong yang terlihat seperti tabel rusak.
        f.nama ? (
          <span className="font-medium text-slate-900">{f.nama}</span>
        ) : (
          <span className="italic text-slate-400" title="Baris ini tidak punya keterangan">
            (tanpa keterangan)
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
      render: (f) => (
        <Button
          tone="halus"
          onClick={() => openEdit(f)}
          aria-label={`Ubah faktor dominan ${f.nama || f.id}`}
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
            <li className="text-slate-700">Dominan Factor</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          Master Dominan Factor
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Daftar acuan faktor dominan penyebab kerugian sebuah klaim. Keterangan yang
          diubah di sini ikut terbaca laporan Outstanding per Cabang, yang merangkai
          seluruh faktor satu klaim menjadi satu baris.
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
          <DominantFactorForm factor={sedangDisunting} tutup={closeForm} />
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
          rows={list.data?.dominan_factor ?? []}
          rowKey={(f) => f.id}
          title="Daftar Faktor Dominan"
          description={
            list.data
              ? `${list.data.total} faktor dominan terdaftar pada entitas ini.`
              : 'Memuat daftar faktor dominan…'
          }
          searchLabel="Cari ID atau keterangan"
          emptyMessage="Belum ada faktor dominan yang terdaftar."
          isLoading={list.isPending}
          error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
          actions={
            <>
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
 * Yang di sini selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat
 * salah — ia baru membuka layarnya.
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
        'Daftar faktor dominan belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
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
          title: 'Daftar faktor dominan gagal dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Daftar faktor dominan gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
