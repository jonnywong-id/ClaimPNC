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
  useUpdateProgressStatus2,
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
 * # Tombol Ubah — ADA di sini, dan itu perilaku BARU
 *
 * Layar lama memilikinya, tetapi tombol itu **tidak mengubah apa pun** pada tabel master:
 * `UpdateStatusProgress2_sql` menyentuh TABEL LAIN — `GCNM_PROGRESS_CLAIM`, catatan progres
 * milik satu klaim — dan menyaringnya dengan dua page klipboard yang tidak pernah diisi.
 * `TempInputStatusProgress2` bahkan hanya muncul di berkas SQL itu sendiri. Seluruh export
 * tidak memuat satu pun `UPDATE` terhadap `POOLDATA.GCNM_MST_PROGRESS`.
 *
 * Versi pertama layar ini karena itu mereplikasi **hasil yang teramati** — baris master
 * tidak berubah — dan tidak menyediakan penyuntingan sama sekali.
 *
 * **Work Owner memutuskan sebaliknya pada 2026-09-20**, setelah ditunjukkan bahwa tanpa
 * penyuntingan, salah ketik nama tidak dapat diperbaiki dengan cara apa pun — tombol hapus
 * pun tidak ada, sehingga baris yang keliru akan menetap selamanya.
 *
 * Yang ditambahkan **bukan** jalur Pega yang rusak itu, melainkan `UPDATE` yang benar ke
 * tabel master. Dua batas dijaga: `ID_MST` tidak pernah ikut di-`SET` karena ia dirujuk
 * data klaim berjalan, dan salinan nama induk (`STS_PROGRESS1`) ditulis ulang server bila
 * induknya berpindah.
 *
 * Konsekuensinya disadari: ia **selisih** pada uji kesetaraan gerbang 1, dan dinyatakan di
 * muka sebagai perbaikan terencana — bukan ditemukan sebagai kejutan (`D-54`).
 *
 * # Yang SENGAJA tidak ada
 *
 * **Tombol Hapus.** Tidak pernah ada di sistem lama, tabelnya tidak punya kolom penanda
 * terhapus yang dapat dipakai `D-66`, dan barisnya dirujuk data klaim yang sudah berjalan.
 *
 * **Kolom TIPE.** Sempat ditampilkan di layar ini, lalu dicabut setelah header grid Pega
 * dibaca: `Section/BrowseStatusProgress2-Section.xml` hanya memuat tiga header —
 * `No`, `Status Progress 1`, `Status Progress 2` — dan `DistrictID` (TIPE) terikat ke page
 * `TempUpdateStatus2`, bukan baris grid. Ia tetap dibaca dan tetap dikirim pada respons
 * API; yang dihapus hanyalah kolomnya.
 */
/** Tidak ada form yang terbuka. */
const CLOSED = 'closed'
/** Form terbuka dalam mode tambah. */
const CREATE = 'create'

type FormState = typeof CLOSED | typeof CREATE | ProgressStatus2

export function ProgressStatus2Page() {
  const portal = useSelectedPortal((state) => state.alias)
  const [form, setForm] = useState<FormState>(CLOSED)

  const list = useProgressStatus2List()
  const parents = useProgressStatus2ParentList()
  const create = useCreateProgressStatus2()
  const update = useUpdateProgressStatus2()

  // Keadaan form memikul tiga hal sekaligus — tertutup, tambah, atau baris yang sedang
  // disunting — mengikuti pola tingkat 1. Menyimpannya sebagai dua state terpisah
  // (`isOpen` + `edited`) membuka keadaan yang tidak masuk akal: terbuka tanpa mode.
  const edited = typeof form === 'string' ? null : form
  const isSaving = create.isPending || update.isPending
  const saveError = edited ? update.error : create.error

  function openCreate() {
    create.reset()
    update.reset()
    setForm(CREATE)
  }

  function openEdit(row: ProgressStatus2) {
    create.reset()
    update.reset()
    setForm(row)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setForm(CLOSED)
  }

  function save(values: ProgressStatus2Fields) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
    // diketik, itu berarti mengetik ulang dari awal.
    if (edited) {
      update.mutate({ id: edited.id, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  // Susunan kolom mengikuti grid Pega APA ADANYA, terbaca dari header
  // `Section/BrowseStatusProgress2-Section.xml`:
  //
  //	<b>No<b>                 -> .CaseID  (ID_MST)
  //	<b>Status Progress 1<b>  -> .City    (STS_PROGRESS1, nama induk)
  //	<b>Status Progress 2<b>  -> .CityID  (STS_PROGRESS2, nama baris ini)
  //	(tanpa judul)            -> tombol Update
  //
  // Perhatikan urutannya: **induk mendahului nama barisnya sendiri**. Itu berlawanan
  // dengan dugaan yang wajar, dan sempat tertukar di layar ini sebelum header Pega dibaca.
  //
  // `value` dipisah dari `render` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup.
  const columns: Column<ProgressStatus2>[] = [
    // Judulnya **"No"**, bukan "ID" — itu header Pega apa adanya
    // (`<b>No<b>` terikat ke `.CaseID`, yaitu `ID_MST`). Layar ini sempat menuliskannya
    // "ID"; Work Owner menanyakannya pada 2026-09-20. `D-13` menetapkan teks yang dilihat
    // pengguna mengikuti layar lama.
    //
    // Isinya TETAP `ID_MST` — bukan nomor urut baris. Menomori ulang per halaman akan
    // menyesatkan: yang dirujuk `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` adalah `ID_MST`,
    // dan itulah nomor yang dicari petugas.
    { key: 'id', title: 'No', width: 'w-20', value: (row) => row.id },
    {
      key: 'induk',
      title: 'Status Progres 1',
      // Yang DITAMPILKAN hanya namanya, persis seperti Pega: kolom `Status Progress 1`
      // terikat ke `.City` saja, yaitu `STS_PROGRESS1`. `ID_PROGRESS` memang ikut
      // di-SELECT `BrowseStatusProgress2-SQL`, tetapi tidak pernah digambar di grid — ia
      // dipakai modal penyuntingan, yang tidak dibawa.
      //
      // Versi sebelumnya menempelkan ID induk sebagai teks abu-abu di sebelah namanya
      // (`LAPORAN AWAL 001`). Itu tambahan saya, bukan perilaku sistem lama, dan Work Owner
      // menanyakannya pada 2026-09-20.
      //
      // ID-nya TETAP ikut ke `value`, bukan ke `render`: `value` adalah yang dipakai
      // pencarian dan pengurutan, sehingga petugas yang hafal kode induk tetap dapat
      // menemukan barisnya — tanpa satu karakter pun bertambah di layar.
      value: (row) => `${row.nama_induk} ${row.id_induk}`,
      render: (row) => <span>{row.nama_induk}</span>,
    },
    { key: 'nama', title: 'Status Progres 2', value: (row) => row.nama },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-24',
      // Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya
      // tombol, bukan data. Posisinya paling kanan, sama seperti kolom tanpa judul yang
      // memuat tombol Update pada grid Pega.
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button tone="kedua" onClick={() => openEdit(row)} aria-label={`Ubah ${row.nama}`}>
          Ubah
        </Button>
      ),
    },
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
          <Button tone="utama" onClick={openCreate} disabled={form !== CLOSED}>
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

      {form !== CLOSED && (
        <section className="mt-5">
          <ProgressStatus2Form
            edited={edited}
            parents={parents.data?.induk ?? []}
            isSaving={isSaving}
            error={saveError}
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
            // 15 baris per halaman, sama seperti layar lama. Angkanya dibaca dari section
            // MILIK LAYAR INI, bukan disalin dari tingkat 1:
            // `Section/BrowseStatusProgress2-Section.xml` menyisipkan `pyGridPaginator`
            // dengan `pyPageSize = Other` dan `pyPageSizeOther = 15`. Kebetulan sama
            // dengan tingkat 1 — dan kebetulan itu diperiksa, bukan diandaikan.
            pageSize={15}
          />
        )}
      </section>
    </main>
  )
}
