import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type SurveyorType } from '@/api/types'
import { ReloadIcon, AddIcon, EditIcon } from '@/components/Icon'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import { useSurveyorTypeList } from './api'
import { SurveyorTypeForm } from './SurveyorTypeForm'

/**
 * Layar Master Tipe Surveyors.
 *
 * Menggantikan harness `SurveyorsInbox` beserta dua section-nya: `BrowseSuveryors` (grid)
 * dan `GridSurveyors` (kerangka dan tombolnya). Butir menunya `MENU_ID 14` pada
 * POOLDATA.M_MENU_APLIKASI_PNC.
 *
 * # Apa yang dikelola di sini
 *
 * GOLONGAN petugas survei — Internal Surveyor, Loss Adjuster, Expert, Survey Agent —
 * bukan daftar orangnya. Daftar orang ada satu tingkat di bawah, di butir menu "Master
 * Surveyors" (`MENU_ID 15`, harness `DetailSurveyorsInbox`), dan belum dibangun.
 *
 * # Yang ditiru dari layar lama
 *
 * Fungsinya sama persis: daftar bergrid dengan kolom kode dan deskripsi, tombol
 * **Tambah**, tombol **Refresh**, dan aksi **Ubah** per baris. **Tidak ada Hapus** —
 * layar Pega pun tidak punya, dan menghapus satu tipe akan membuat puluhan baris
 * D_SURVEYORS kehilangan golongannya.
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Judul kolom | nama kolom mentah (`M_SURVEY_ID`, `DESCRIPTION`) | nama yang dibaca manusia (`D-19`) |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Pencarian | filter per kolom | satu kotak cari yang menelusuri seluruh kolom |
 * | Isian kosong | diterima | ditolak |
 * | Nama ganda | diterima | ditolak |
 * | Entitas | disimpulkan dari nama server | dipilih pengguna dan disebut di layar |
 */
export function SurveyorTypePage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useSurveyorTypeList()

  // null = form tertutup; { kode: '' } = sedang menambah; berisi = sedang mengubah.
  const [beingEdited, setBeingEdited] = useState<SurveyorType | null>(null)
  const [formOpen, setFormOpen] = useState(false)

  function openAdd() {
    setBeingEdited(null)
    setFormOpen(true)
  }

  function openEdit(surveyorType: SurveyorType) {
    setBeingEdited(surveyorType)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setBeingEdited(null)
  }

  const rows = list.data?.tipe_surveyor ?? []

  // Kolom kode lama hanya ditampilkan bila ADA yang mengisinya.
  //
  // Pada portal ASM seluruh empat barisnya kosong, dan kolom yang selamanya berisi tanda
  // hubung hanya menambah lebar tabel tanpa memberi tahu apa pun. Ia tetap disiapkan
  // karena basis data entitas lain belum diperiksa — bila ternyata terisi di sana,
  // kolomnya muncul dengan sendirinya tanpa perubahan kode.
  const anyLegacyCode = rows.some((t) => t.kode_lama.trim() !== '')

  const columns: Column<SurveyorType>[] = [
    {
      key: 'kode',
      title: 'Kode',
      width: '8rem',
      value: (t) => t.kode,
      render: (t) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {t.kode}
        </span>
      ),
    },
    {
      key: 'deskripsi',
      title: 'Tipe Surveyor',
      value: (t) => t.deskripsi,
      render: (t) => <span className="font-medium text-slate-900">{t.deskripsi}</span>,
    },
    ...(anyLegacyCode
      ? [
          {
            key: 'kode_lama',
            title: 'Kode lama',
            width: '9rem',
            value: (t: SurveyorType) => t.kode_lama,
            render: (t: SurveyorType) =>
              t.kode_lama ? (
                <span className="inline-flex items-center rounded-md bg-blue-50 px-2 py-0.5 font-mono text-xs font-medium text-blue-700 ring-1 ring-blue-100">
                  {t.kode_lama}
                </span>
              ) : (
                <span className="text-slate-400" title="Tipe ini tidak punya penomoran lama">
                  —
                </span>
              ),
          },
        ]
      : []),
    {
      key: 'aksi',
      title: 'Aksi',
      width: '7rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (t) => (
        <Button
          tone="halus"
          onClick={() => openEdit(t)}
          aria-label={`Ubah tipe surveyor ${t.deskripsi}`}
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
            <li className="text-slate-700">Tipe Surveyors</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
          Master Tipe Surveyors
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Golongan petugas survei — Internal Surveyor, Loss Adjuster, Expert, dan
          seterusnya. Setiap surveyor digolongkan ke salah satu tipe di sini, dan nama
          yang diubah langsung terbaca layar pemilihan surveyor.
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

      {formOpen && (
        <div className="mb-6">
          <SurveyorTypeForm surveyorType={beingEdited} onClose={closeForm} />
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
          rows={rows}
          rowKey={(t) => t.kode}
          title="Daftar Tipe Surveyor"
          description={
            list.data
              ? `${list.data.total} tipe surveyor terdaftar pada entitas ini.`
              : 'Memuat daftar tipe surveyor…'
          }
          searchLabel="Cari kode atau nama tipe surveyor"
          emptyMessage="Belum ada tipe surveyor pada entitas ini."
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
              <Button tone="utama" onClick={openAdd} disabled={formOpen && !beingEdited}>
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
 * — ia baru membuka layarnya. Kecuali soal portal, yang justru dapat ia perbaiki sendiri.
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
        'Daftar tipe surveyor belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
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
          title: 'Daftar tipe surveyor gagal dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Daftar tipe surveyor gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
