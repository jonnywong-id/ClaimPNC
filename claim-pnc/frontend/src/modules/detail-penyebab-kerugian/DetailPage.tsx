import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  CauseOfLossDetailErrorCode,
  ErrorCode,
  type CauseOfLossDetail,
  type CauseOfLossDetailInput,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import {
  useCauseOfLossDetail,
  useCauseOfLossDetailList,
  useCauseOfLossOptions,
  useCreateCauseOfLossDetail,
  useSaveCauseOfLossDetail,
} from './api'
import { DetailForm, type DetailFormValues } from './DetailForm'

/**
 * Ukuran halaman diambil dari `pyPageSize` pada grid utama
 * `Section/BrowseDetailCauseOfLoss-Section.xml:8070`.
 *
 * **50**, dan itu jauh lebih besar daripada layar master lain di aplikasi ini — 15 pada
 * Master Rekening dan kerabatnya, 20 pada Supplier, Bengkel, serta Panel. Angkanya milik
 * layar, bukan milik komponen tabel, dan dipertahankan apa adanya (`D-13`).
 *
 * Mode paginasinya `Numeric` (`:8068`), sama dengan `DataTable` — sehingga di layar ini
 * tidak ada selisih mode seperti pada layar yang memakai `Next Previous`.
 */
const PAGE_SIZE = 50

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
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
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian ' +
            'atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar detail penyebab kerugian tidak dapat dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
    tone: 'gangguan',
  }
}

/** Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti. */
function saveMessage(error: unknown): MessageContent {
  if (error instanceof APIError) {
    if (error.kode === CauseOfLossDetailErrorCode.idTaken) {
      return {
        title: 'Penomoran ID bentrok',
        description:
          'Isian Anda sudah benar — yang bermasalah adalah nomor yang diterbitkan sistem. ' +
          'Coba simpan sekali lagi; bila tetap ditolak, hubungi administrator Claim PNC.',
        tone: 'gangguan',
      }
    }
    if (error.kode === ErrorCode.notFound) {
      return {
        title: 'Baris ini sudah tidak ada',
        description:
          'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
        tone: 'penolakan',
      }
    }
    // Portal yang belum dipilih ditangani DI SINI, bukan dengan mengunci tombol Tambah.
    // Pengguna tetap dapat membuka form; yang ditolak adalah penyimpanannya, dengan pesan
    // yang menyebut apa yang harus dilakukan.
    if (
      error.kode === ErrorCode.portalNotStated ||
      error.kode === ErrorCode.portalUnknown
    ) {
      return {
        title: 'Portal entitas belum dipilih',
        description:
          'Isian Anda belum tersimpan. Pilih portal entitas di bagian atas halaman ini, ' +
          'lalu tekan Simpan lagi — isian yang sudah diketik tidak hilang.',
        tone: 'penolakan',
      }
    }
    if (error.kode === ErrorCode.portalNotReady) {
      return {
        title: 'Basis data entitas ini belum tersedia',
        description:
          'Isian Anda belum tersimpan, dan mengulang tidak akan menolong. Hubungi ' +
          'administrator Claim PNC untuk melengkapi kredensial basis datanya.',
        tone: 'gangguan',
      }
    }
    return { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
  }
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Perubahan belum tersimpan. Periksa koneksi jaringan, lalu coba lagi.',
      tone: 'gangguan',
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Perubahan belum tersimpan. Coba lagi beberapa saat kemudian.',
    tone: 'gangguan',
  }
}

/**
 * Mengambil pesan galat per isian dari jawaban 422.
 *
 * Memakai `APIError.violations()` — helper bersama yang menyatukan kedua bentuk detail
 * galat yang hidup berdampingan di aplikasi ini (`detail[].field` dan `detail[].kolom`).
 * Menguraikannya sendiri di sini berarti layar ini harus tahu bentuk mana yang dipakai
 * modulnya, dan akan ikut berubah saat TKT-F1-004 menyeragamkannya.
 */
function fieldErrorOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {}
}

/**
 * Layar Detail Penyebab Kerugian.
 *
 * Pengganti `Harness/DetailCauseOfLoss-Harness.xml` atas POOLDATA.D_CAUSE_OF_LOSS
 * (MENU_ID 38).
 *
 * # Apa yang dikelola layar ini
 *
 * **Rincian di bawah Master Penyebab Kerugian** — butir-butir sebab kerugian yang
 * benar-benar dipilih petugas saat klaim diregistrasi. Master menyebut sebabnya secara umum
 * (kebakaran, kecelakaan, kehilangan); layar inilah yang memecahnya.
 *
 * # Induknya BELUM dibangun, dan layar ini hanya membacanya
 *
 * Master Penyebab Kerugian adalah MENU_ID 20 (`CauseOfLossInbox`), dan modulnya belum ada.
 * Isian "ID Master Kerugian" karena itu hanya MEMBACA `POOLDATA.V_M_CAUSE_OF_LOSS` sebagai
 * daftar pilihan — tidak ada satu pun jalur di layar ini yang menulis ke sana.
 *
 * Akibatnya yang harus disadari: **induk baru tidak dapat dibuat dari sini.** Bila sebuah
 * sebab kerugian belum ada di master, ia harus ditambahkan lewat Pega sampai modul induknya
 * dibangun.
 *
 * # TANPA tombol Hapus, dan itu mengikuti layar lama
 *
 * Tombol yang ada di Pega hanya `Simpan`, `Ubah`, `Cari Data`, dan `Clear Pencarian`.
 * `pyDeleteSQL` pada rule simpannya kosong, dan tidak ada satu pun activity penghapus di
 * export. `D-66` pun melarang penghapusan fisik data bernilai bisnis.
 *
 * Baris yang tidak lagi dipakai dinyatakan lewat **Status Aktif** — itulah gunanya kolom
 * itu, dan itulah satu-satunya cara menonaktifkan sebuah detail.
 *
 * # Daftarnya TIDAK menyaring yang tidak aktif
 *
 * Report Definition yang mengisi grid Pega — `BrowseVDCauseOfLoss_RD` — **tidak ada di
 * export** (`R-16`), sehingga tidak ada yang dapat memastikan apakah ia menyaring sesuatu.
 * Yang dipilih di sini adalah TIDAK menyaring, mengikuti
 * `RDB List/QueryGetAllDataCauseOfLoss-SQL.xml` yang membaca view yang sama tanpa satu pun
 * penyaring.
 *
 * Alasannya: baris tidak aktif yang disembunyikan akan tampak hilang bagi petugas, dan
 * tidak ada tombol mana pun di layar ini untuk memunculkannya kembali.
 *
 * # Isinya belum dapat dipakai terhadap Oracle
 *
 * Sampai hak akses `POOLDATA.D_CAUSE_OF_LOSS` diberikan DBA, layar ini berjalan atas
 * penyimpanan memori yang isinya **dikarang** — dan isi itu lenyap saat aplikasi berhenti.
 * Keadaan itu dinyatakan di layar alih-alih ditutupi.
 */
export function DetailPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const list = useCauseOfLossDetailList()
  const options = useCauseOfLossOptions()

  // null berarti form tertutup; '' berarti form terbuka untuk baris BARU.
  const [editingID, setEditingID] = useState<string | null>(null)
  const loaded = useCauseOfLossDetail(editingID === '' ? null : editingID)

  const create = useCreateCauseOfLossDetail()
  const save = useSaveCauseOfLossDetail()

  const busy = create.isPending || save.isPending
  const saveError = create.error ?? save.error
  const fieldError = fieldErrorOf(saveError)

  function closeForm() {
    setEditingID(null)
    create.reset()
    save.reset()
  }

  function submit(values: DetailFormValues) {
    const input: CauseOfLossDetailInput = {
      id_lama: values.id_lama,
      id_master: values.id_master,
      deskripsi_kerugian: values.deskripsi_kerugian,
      kode_kehilangan: values.kode_kehilangan,
      status_aktif: values.status_aktif,
      bisnis: values.bisnis,
    }

    if (editingID === '' || editingID === null) {
      create.mutate(input, { onSuccess: closeForm })
      return
    }
    save.mutate({ id: editingID, input }, { onSuccess: closeForm })
  }

  const columns: Column<CauseOfLossDetail>[] = [
    {
      key: 'id',
      title: 'ID',
      value: (row) => row.id,
      width: '9rem',
    },
    {
      key: 'deskripsi_kerugian',
      title: 'Deskripsi Kerugian',
      value: (row) => row.deskripsi_kerugian,
    },
    {
      key: 'nama_master',
      title: 'Master Kerugian',
      // Kolom ini TIDAK ada di grid Pega. Ia ditambahkan karena tanpa sebutan induknya,
      // sebuah detail tidak dapat dibedakan dari detail lain yang deskripsinya mirip —
      // dan di Pega petugas harus membuka barisnya untuk mengetahuinya.
      value: (row) => row.nama_master,
      render: (row) =>
        row.nama_master === '' ? (
          <span className="text-slate-400">
            {row.id_master === '' ? '—' : `${row.id_master} (induk tidak ada)`}
          </span>
        ) : (
          row.nama_master
        ),
    },
    {
      key: 'status_aktif',
      title: 'Status Aktif',
      value: (row) => row.label_status_aktif,
      width: '9rem',
      render: (row) => (
        <span
          className={[
            'rounded-full px-2 py-0.5 text-xs ring-1',
            row.status_aktif === '1'
              ? 'bg-emerald-50 text-emerald-800 ring-emerald-200'
              : row.status_aktif === ''
                ? 'bg-slate-50 text-slate-600 ring-slate-200'
                : 'bg-amber-50 text-amber-800 ring-amber-200',
          ].join(' ')}
        >
          {row.label_status_aktif}
        </span>
      ),
    },
    {
      key: 'kode_kehilangan',
      title: 'Kode Kehilangan',
      value: (row) => row.kode_kehilangan,
      width: '10rem',
    },
    {
      key: 'aksi',
      title: '',
      value: () => '',
      noSort: true,
      alignRight: true,
      width: '6rem',
      render: (row) => (
        <Button tone="halus" onClick={() => setEditingID(row.id)}>
          Ubah
        </Button>
      ),
    },
  ]

  const loadError = list.error === null ? null : loadMessage(list.error)
  const formMessage = saveError == null ? null : saveMessage(saveError)

  return (
    <div className="space-y-4">
      <DataTable
        title="Detail Penyebab Kerugian"
        description={
          'Rincian di bawah Master Penyebab Kerugian — butir yang dipilih petugas saat ' +
          'klaim diregistrasi. Baris yang tidak lagi dipakai ditandai lewat Status Aktif, ' +
          'bukan dihapus.'
        }
        columns={columns}
        rows={list.data?.detail ?? []}
        rowKey={(row) => row.id}
        isLoading={list.isLoading}
        pageSize={PAGE_SIZE}
        searchLabel="Cari detail penyebab kerugian"
        emptyMessage={
          portal === null
            ? 'Pilih portal entitas lebih dulu.'
            : 'Belum ada detail penyebab kerugian pada entitas ini.'
        }
        error={
          loadError === null ? undefined : (
            <ErrorMessage
              title={loadError.title}
              description={loadError.description}
              tone={loadError.tone}
            />
          )
        }
        actions={
          <Button
            tone="utama"
            // Digerbangi HANYA oleh form yang sedang terbuka, sama seperti seluruh layar
            // master lain.
            //
            // Portal yang belum dipilih SENGAJA tidak ikut menggerbanginya. Sempat
            // demikian, dan itu membuat layar ini satu-satunya yang tombol Tambah-nya
            // kelabu pada sesi baru — perilaku yang tidak dapat dijelaskan pengguna,
            // karena 17 layar tetangganya tidak begitu.
            //
            // Portal tetap ditegakkan, tetapi di tempat yang benar: server menolak
            // permintaan tanpa portal (TKT-F6-002), dan penolakannya diterjemahkan
            // saveMessage menjadi pesan yang menyebut apa yang harus dilakukan.
            disabled={editingID !== null}
            onClick={() => {
              create.reset()
              save.reset()
              setEditingID('')
            }}
          >
            Tambah
          </Button>
        }
      />

      {editingID !== null && (
        <section className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
          <h2 className="text-base font-semibold text-slate-900">
            {editingID === '' ? 'Tambah Detail Penyebab Kerugian' : 'Memperbaharui Data'}
          </h2>

          {formMessage !== null && (
            <div className="mt-3">
              <ErrorMessage
                title={formMessage.title}
                description={formMessage.description}
                tone={formMessage.tone}
              />
            </div>
          )}

          <div className="mt-4">
            {editingID !== '' && loaded.isLoading ? (
              <p className="text-sm text-slate-500">Memuat baris…</p>
            ) : editingID !== '' && loaded.error !== null ? (
              <ErrorMessage
                title="Baris ini tidak dapat dimuat"
                description="Tutup form ini dan muat ulang daftarnya."
                tone="gangguan"
              />
            ) : (
              <DetailForm
                // key memaksa form lahir ulang saat berpindah baris. Tanpa itu, nilai
                // awalnya tidak ikut berganti — `useState` hanya membaca nilai awal sekali.
                key={editingID === '' ? 'baru' : editingID}
                existing={editingID === '' ? null : (loaded.data?.detail ?? null)}
                options={options.data?.status_aktif ?? []}
                fieldError={fieldError}
                busy={busy}
                onSubmit={submit}
                onCancel={closeForm}
              />
            )}
          </div>
        </section>
      )}
    </div>
  )
}
