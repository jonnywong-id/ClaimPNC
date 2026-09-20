import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type ProgressStatus } from '@/api/types'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import {
  useClaimPositionList,
  useCreateProgressStatus,
  useProgressStatusList,
  useUpdateProgressStatus,
} from './api'
import { ProgressStatusForm, type ProgressStatusFields } from './ProgressStatusForm'

/** Tidak ada form yang terbuka. */
const CLOSED = 'closed'
/** Form terbuka dalam mode tambah. */
const CREATE = 'create'

type FormState = typeof CLOSED | typeof CREATE | ProgressStatus

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
 * Layar Master Status Progres 1.
 *
 * Pengganti `Harness/StatusProgress-Harness.xml` atas tabel
 * POOLDATA.GCNM_MST_PROGRESS_KLAIM. Judul, susunan kolom, dan kedua tombolnya mengikuti
 * layar lama (`D-13`: alur dan tata letak ditiru supaya pengguna tidak perlu belajar
 * ulang):
 *
 *   - Judul "Master Status Progres 1" — `Section/MasterStatusProgress-Section.xml`
 *   - Tombol "Tambah" dan "Refresh"  — section yang sama
 *   - Grid tiga kolom: ID, Status Progres, Posisi — `BrowseStatusProgress-Section.xml`
 *
 * Yang SENGAJA tidak ada: tombol hapus. Sistem lama tidak punya satu pun pernyataan
 * DELETE terhadap tabel ini — sudah diperiksa ke seluruh export — dan tabelnya pun tidak
 * punya kolom penanda terhapus yang dapat dipakai `D-66`. Menambahkannya berarti
 * mengarang perilaku yang tidak pernah ada, sekaligus berisiko: baris ini dirujuk
 * `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1` pada data klaim yang sudah berjalan.
 */
export function ProgressStatusPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [form, setForm] = useState<FormState>(CLOSED)

  const list = useProgressStatusList()
  const positions = useClaimPositionList()
  const create = useCreateProgressStatus()
  const update = useUpdateProgressStatus()

  const edited = typeof form === 'string' ? null : form
  const isSaving = create.isPending || update.isPending
  const saveError = edited ? update.error : create.error

  function openCreate() {
    create.reset()
    update.reset()
    setForm(CREATE)
  }

  function openEdit(row: ProgressStatus) {
    create.reset()
    update.reset()
    setForm(row)
  }

  function closeForm() {
    create.reset()
    update.reset()
    setForm(CLOSED)
  }

  function save(values: ProgressStatusFields) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form yang isinya baru
    // diketik, itu berarti mengetik ulang dari awal.
    if (edited) {
      update.mutate({ id: edited.id, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  // `nilai` dipisah dari `tampil` mengikuti kontrak Column: yang dicari dan diurutkan
  // adalah teks polos, yang dilihat pengguna boleh berisi markup. Menyatukannya akan
  // membuat pencarian ikut menelusuri kelas CSS.
  const columns: Column<ProgressStatus>[] = [
    { key: 'id', title: 'ID', width: 'w-20', value: (row) => row.id },
    { key: 'nama', title: 'Status Progres', value: (row) => row.nama },
    {
      key: 'posisi',
      title: 'Posisi',
      width: 'w-40',
      // Hanya SATU nilai yang digambar, bukan label plus kode di sebelahnya.
      //
      // Versi sebelumnya menyandingkan keduanya karena saya mengira yang tersimpan
      // adalah kode angka ("002") dan labelnya terpisah. Koreksi 2026-09-20 membuktikan
      // sebaliknya: dropdown-nya di Pega mengikat nilai simpanan dan label ke properti
      // yang sama, sehingga keduanya identik — menyandingkannya hanya menulis
      // "REGISTER REGISTER".
      value: (row) => row.nama_posisi,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: 'w-24',
      // Kolom aksi tidak layak diurutkan dan tidak punya teks untuk dicari — isinya
      // tombol, bukan data.
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
      {/* Tidak ada tautan "kembali ke beranda" di sini: menu utama di kerangka sudah
          menyediakannya, dan dua jalan ke tempat yang sama pada satu layar membuat
          pengguna menebak mana yang dimaksud. */}
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Master Status Progres 1</h1>
          <p className="text-sm text-slate-600">
            Daftar status progres yang dapat dicatat petugas pada setiap posisi klaim.
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

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
          empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
          hanya diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
      </p>

      {form !== CLOSED && (
        <section className="mt-5">
          <ProgressStatusForm
            edited={edited}
            positions={positions.data?.posisi ?? []}
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
          <p className="text-sm text-slate-500">Memuat daftar status progres…</p>
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
            rows={list.data.status_progres}
            rowKey={(row) => row.id}
            description="Sumber: POOLDATA.GCNM_MST_PROGRESS_KLAIM"
            emptyMessage="Belum ada status progres pada entitas ini."
            // 15 baris per halaman, sama seperti layar lama. Angkanya BUKAN dikarang:
            // `Section/BrowseStatusProgress-Section.xml` menyisipkan `pyGridPaginator`
            // dengan `pyPageSize = Other` dan `pyPageSizeOther = 15`.
            //
            // Ditambahkan 2026-09-20 setelah Work Owner menemukan grid ini menggambar
            // seluruh baris sekaligus, padahal layar lamanya berhalaman.
            pageSize={15}
          />
        )}
      </section>
    </main>
  )
}
