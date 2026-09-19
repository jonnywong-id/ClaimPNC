import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type ProgressStatus2 } from '@/api/types'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import {
  useCreateProgressStatus2,
  useProgressStatus2List,
  useProgressStatus2ParentList,
} from './api2'
import { ProgressStatus2Form, type ProgressStatus2Fields } from './ProgressStatus2Form'

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
  return {
    title: 'Daftar tidak dapat dimuat',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}

/**
 * Layar Master Status Progres 2.
 *
 * Pengganti `Harness/StatusProgress2-Harness.xml` atas tabel POOLDATA.GCNM_MST_PROGRESS.
 * Judul, susunan kolom, dan kedua tombolnya mengikuti layar lama (`D-13`: alur dan tata
 * letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *   - Judul "Master Status Progress 2"   — `Section/MasterStatusProgress2-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"      — section yang sama
 *   - Grid   — `Section/BrowseStatusProgress2-Section.xml`, ORDER BY ID_MST ASC
 *
 * # Yang SENGAJA tidak ada, dan itu bukan pekerjaan yang belum selesai
 *
 * **Tombol Ubah.** Layar lama memilikinya, tetapi tombol itu tidak mengubah apa pun pada
 * tabel master: `UpdateStatusProgress2_sql` menyentuh TABEL LAIN — GCNM_PROGRESS_CLAIM,
 * catatan progres milik satu klaim — dan menyaringnya dengan dua page klipboard yang tidak
 * pernah diisi activity maupun section-nya. Seluruh export tidak memuat satu pun UPDATE
 * atau DELETE terhadap POOLDATA.GCNM_MST_PROGRESS.
 *
 * Yang direplikasi adalah HASIL yang teramati — baris master tidak berubah — bukan jalur
 * yang menghasilkannya. Menyalin jalurnya berarti membawa pernyataan yang, bila kedua page
 * itu kebetulan terisi sisa nilai dari layar lain dalam sesi yang sama, MENIMPA catatan
 * progres sebuah klaim dengan isian layar master.
 *
 * **Tombol Hapus.** Tidak pernah ada di sistem lama, dan tabelnya pun tidak punya kolom
 * penanda terhapus yang dapat dipakai `D-66`.
 *
 * **Kolom TIPE.** Sempat ditampilkan di layar ini, lalu dicabut setelah header grid Pega
 * dibaca: `Section/BrowseStatusProgress2-Section.xml` hanya memuat tiga header —
 * `No`, `Status Progress 1`, `Status Progress 2` — dan `DistrictID` (TIPE) terikat ke page
 * `TempUpdateStatus2`, yaitu **modal penyuntingan**, bukan baris grid.
 *
 * Karena modal itu tidak dibawa (§ di atas), TIPE tidak punya tempat di layar ini. Ia tetap
 * dibaca dan tetap dikirim pada respons API, sehingga nilainya tidak hilang dan siap dipakai
 * bila kelak ada layar rincian. Yang dihapus hanyalah kolomnya — dan kolom itu memang tidak
 * pernah ada di Pega.
 */
export function ProgressStatus2Page() {
  const portal = useSelectedPortal((state) => state.alias)
  const [isFormOpen, setFormOpen] = useState(false)

  const list = useProgressStatus2List()
  const parents = useProgressStatus2ParentList()
  const create = useCreateProgressStatus2()

  function openCreate() {
    create.reset()
    setFormOpen(true)
  }

  function closeForm() {
    create.reset()
    setFormOpen(false)
  }

  function save(values: ProgressStatus2Fields) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
    // diketik, itu berarti mengetik ulang dari awal.
    create.mutate(values, { onSuccess: closeForm })
  }

  // Susunan kolom mengikuti grid Pega APA ADANYA, terbaca dari header
  // `Section/BrowseStatusProgress2-Section.xml`:
  //
  //	<b>No<b>                 -> .CaseID  (ID_MST)
  //	<b>Status Progress 1<b>  -> .City    (STS_PROGRESS1, nama induk)
  //	<b>Status Progress 2<b>  -> .CityID  (STS_PROGRESS2, nama baris ini)
  //	(tanpa judul)            -> tombol Update — tidak dibawa, lihat doc comment di atas
  //
  // Perhatikan urutannya: **induk mendahului nama barisnya sendiri**. Itu berlawanan
  // dengan dugaan yang wajar, dan sempat tertukar di layar ini sebelum header Pega dibaca.
  //
  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<ProgressStatus2>[] = [
    { key: 'id', title: 'ID', width: 'w-20', value: (row) => row.id },
    {
      key: 'induk',
      title: 'Status Progres 1',
      // ID induk ikut ke `value` supaya pencarian menemukan baris lewat kodenya maupun
      // lewat namanya.
      value: (row) => `${row.nama_induk} ${row.id_induk}`,
      render: (row) => (
        <span>
          {row.nama_induk}
          <span className="ml-2 text-xs text-slate-500">{row.id_induk}</span>
        </span>
      ),
    },
    { key: 'nama', title: 'Status Progres 2', value: (row) => row.nama },
  ]

  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Master Status Progres 2</h1>
          <p className="text-sm text-slate-600">
            Rincian status progres di bawah setiap Status Progres 1.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={openCreate} disabled={isFormOpen}>
            Tambah
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
      </p>

      {isFormOpen && (
        <section className="mt-5">
          <ProgressStatus2Form
            parents={parents.data?.induk ?? []}
            isSaving={create.isPending}
            error={create.error}
            onSave={save}
            onCancel={closeForm}
          />
        </section>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar status progres 2…</p>
        ) : list.isError ? (
          (() => {
            const message = loadMessage(list.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <DataTable
            columns={columns}
            rows={list.data.status_progres_2}
            rowKey={(row) => row.id}
            description="Sumber: POOLDATA.GCNM_MST_PROGRESS"
            emptyMessage="Belum ada status progres 2 pada entitas ini."
          />
        )}
      </section>
    </main>
  )
}
