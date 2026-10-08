import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type BusinessChoice,
  type BusinessDocumentRuleInput,
  type MasterChoice,
} from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import {
  DocumentRowsTable,
  emptyRow,
  labelFor,
  type RowDraft,
} from './DocumentRowsTable'

type Props = {
  businesses: BusinessChoice[]
  /** Tombol pemilihan massal ditampilkan atau tidak — meniru `pyVisible` layar lama. */
  mayBulkSelect: boolean
  /**
   * Baris awal, terisi saat form dibuka lewat Copy.
   *
   * `| undefined` ditulis eksplisit karena proyek ini menyalakan
   * `exactOptionalPropertyTypes` — tanpa itu, meneruskan `undefined` secara sengaja
   * ditolak kompilator.
   */
  initialRules?: BusinessDocumentRuleInput[] | undefined
  documentTypes: MasterChoice[]
  detailDocuments: MasterChoice[]
  objectDocuments: MasterChoice[]
  isSaving: boolean
  error: unknown
  onSave: (businessIDs: string[], rules: BusinessDocumentRuleInput[]) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa sambungan jaringan, lalu simpan sekali lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.businessDocumentRuleBusinessRequired:
        return {
          title: 'Nama Bisnis belum di isi',
          description: 'Isi sekurang-kurangnya satu lini bisnis sebelum menyimpan.',
          tone: 'penolakan',
        }
      // Tidak ada cabang untuk "baris dokumen kosong", dan itu disengaja: di Pega keadaan
      // itu berlalu diam-diam — perulangannya berputar nol kali lalu layar tertutup.
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih entitas di bagian atas layar, lalu simpan sekali lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description: 'Hubungi tim infrastruktur bila keadaan ini berlanjut.',
          tone: 'gangguan',
        }
      default:
        return { title: 'Penyimpanan gagal', description: error.message, tone: 'gangguan' }
    }
  }
  return null
}

/** BusinessDraft adalah satu baris isian Nama Bisnis: teks yang diketik beserta kodenya. */
type BusinessDraft = { teks: string; id: string }

/**
 * businessIDFromLabel mencari kode bisnis dari teks yang diketik.
 *
 * Pasangan idFromLabel untuk DocumentRowsTable, dipisahkan karena daftar bisnis memakai
 * `nama_bisnis` sementara ketiga master lain memakai `nama`. Perilakunya sama persis:
 * teks yang tidak cocok menghasilkan kode kosong, tidak ditolak.
 */
function businessIDFromLabel(list: BusinessChoice[], typed: string): string {
  const clean = typed.trim()
  if (clean === '') return ''
  const matched = list.find(
    (item) =>
      `${item.nama_bisnis} (${item.id})` === clean ||
      item.nama_bisnis === clean ||
      item.id === clean,
  )
  return matched?.id ?? ''
}

/** toInput membuang `id` dan teks bantu sebelum baris dikirim ke server. */
function toInput(row: RowDraft): BusinessDocumentRuleInput {
  return {
    id_tipe_dokumen: row.id_tipe_dokumen,
    id_object_dokumen: row.id_object_dokumen,
    id_detail_dokumen: row.id_detail_dokumen,
    detail_dokumen: row.detail_dokumen,
    status_wajib: row.status_wajib,
    minimum_dokumen: row.minimum_dokumen,
  }
}

/**
 * Form penambahan: beberapa lini bisnis dikali beberapa baris dokumen.
 *
 * # Kenapa bentuknya jamak kali jamak
 *
 * Karena begitulah layar lamanya bekerja, dan itu tertulis apa adanya sebagai komentar
 * langkah di `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml`:
 *
 *	"kalo tambah > bisa banyak bisnis & banyak dokumen"
 *
 * Aturan kelengkapan dokumen berulang nyaris sama di banyak lini bisnis, sehingga memaksa
 * petugas memasukkannya satu per satu akan mengubah pekerjaan sepuluh menit menjadi
 * sepuluh jam.
 *
 * # Nama Bisnis berupa BARIS AUTOCOMPLETE, bukan grid centang
 *
 * Versi pertama form ini menampilkan 206 lini bisnis sebagai kotak centang. Itu
 * penyimpangan yang saya buat sendiri: di layar lama Nama Bisnis adalah daftar baris
 * autocomplete yang ditambah satu per satu lewat "+ Tambah". Bedanya bukan selera —
 * dengan 206 kotak centang, menemukan satu lini bisnis berarti menggulung daftar, dan
 * petugas yang hafal namanya tidak dapat mengetikkannya.
 *
 * # Pengecualian pemilihan massal
 *
 * Tombol "Tamban semua bisnis NONMBU" mengecualikan lima kode bisnis yang di sistem lama
 * ditulis langsung di dalam rule (`Activity/SetAllBusiness-Act.xml:984`). Kelimanya tetap
 * dikecualikan di sini — tetapi daftarnya datang dari konfigurasi, bukan dari kode
 * (`D-15`). Bisnis yang sama tetap dapat diketik satu per satu, persis seperti di Pega.
 */
export function BusinessDocumentRuleCreateForm({
  businesses,
  mayBulkSelect,
  initialRules,
  documentTypes,
  detailDocuments,
  objectDocuments,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const [selected, setSelected] = useState<BusinessDraft[]>([{ teks: '', id: '' }])

  // Baris awal dibentuk SEKALI. Bila ia dihitung ulang setiap render, setiap ketikan
  // petugas akan tertimpa oleh baris asal salinannya.
  const [rows, setRows] = useState<RowDraft[]>(() => {
    if (initialRules === undefined || initialRules.length === 0) return [emptyRow()]
    return initialRules.map((rule) => ({
      ...rule,
      // Salinan SELALU kehilangan ID-nya, dan itulah seluruh inti tombol Copy:
      // `UpdateDetailTypeDocumentBusiness_act` memuat baris asal dengan pemetaan yang sama
      // seperti Ubah, tetapi cabang salinnya tidak pernah mengisi `.ID` — sehingga
      // penyimpanan berikutnya menerbitkan baris baru alih-alih menimpa baris asalnya.
      id: '',
      tipe_dokumen_teks: labelFor(documentTypes, rule.id_tipe_dokumen),
      object_dokumen_teks: labelFor(objectDocuments, rule.id_object_dokumen),
      detail_dokumen_teks: labelFor(detailDocuments, rule.id_detail_dokumen),
    }))
  })

  // Bisnis yang dilewati pemilihan massal — kelimanya lini MBU. Bisnis yang sama TETAP
  // dapat diketik satu per satu, persis seperti di Pega.
  const bulkSelectable = businesses.filter((item) => !item.dikecualikan_pilih_semua)

  const message = messageFor(error)

  // Kode yang benar-benar terkirim: baris kosong dan yang teksnya tidak cocok dengan
  // master dibuang di sini, bukan ditolak. Pega pun melewatinya diam-diam.
  const businessIDs = selected.map((item) => item.id).filter((id) => id !== '')

  function updateBusiness(index: number, teks: string) {
    setSelected((current) =>
      current.map((item, position) =>
        position === index ? { teks, id: businessIDFromLabel(businesses, teks) } : item,
      ),
    )
  }

  function updateRow(index: number, patch: Partial<RowDraft>) {
    setRows((current) =>
      current.map((row, position) => (position === index ? { ...row, ...patch } : row)),
    )
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        onSave(businessIDs, rows.map(toInput))
      }}
      noValidate
      aria-label="Tambah Data"
      className="space-y-5 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">Tambah Data</h2>

      {message && (
        <ErrorMessage
          title={message.title}
          description={message.description}
          tone={message.tone}
        />
      )}

      <section>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold text-slate-800">Nama Bisnis</h3>
          <div className="flex flex-wrap gap-2">
            {/*
              Tombolnya hanya tampil bagi operator ber-pyPosition NONMBU, meniru `pyVisible`
              layar lama. Ia penyembunyian TAMPILAN, bukan kewenangan: bisnis yang sama
              tetap dapat diketik satu per satu oleh siapa pun yang membuka layar ini,
              sehingga tidak ada kemampuan yang dijaga penyembunyian ini.
            */}
            {mayBulkSelect && (
              <Button
                tone="halus"
                onClick={() =>
                  setSelected(
                    bulkSelectable.map((item) => ({
                      teks: `${item.nama_bisnis} (${item.id})`,
                      id: item.id,
                    })),
                  )
                }
              >
                Tamban semua bisnis NONMBU
              </Button>
            )}
            <Button
              tone="kedua"
              onClick={() => setSelected((current) => [...current, { teks: '', id: '' }])}
            >
              Tambah
            </Button>
          </div>
        </div>

        <div className="mt-2 space-y-2">
          {selected.map((item, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                type="text"
                list="pilihan-bisnis"
                autoComplete="off"
                aria-label={`Nama Bisnis baris ${index + 1}`}
                // Bisnis yang teksnya terisi tetapi kodenya tidak ditemukan DIBUANG diam-diam
                // sebelum dikirim — sama seperti di Pega. Penandaan ini satu-satunya
                // kesempatan petugas menyadarinya sebelum menekan Simpan.
                className={
                  item.teks.trim() !== '' && item.id === ''
                    ? 'w-full max-w-md rounded border border-amber-400 bg-amber-50 px-3 py-2 text-sm'
                    : 'w-full max-w-md rounded border border-slate-300 bg-white px-3 py-2 text-sm'
                }
                value={item.teks}
                onChange={(event) => updateBusiness(index, event.target.value)}
              />
              {selected.length > 1 && (
                <Button
                  tone="halus"
                  aria-label={`Buang Nama Bisnis baris ${index + 1}`}
                  onClick={() =>
                    setSelected((current) =>
                      current.filter((_item, position) => position !== index),
                    )
                  }
                >
                  Buang
                </Button>
              )}
            </div>
          ))}
          <datalist id="pilihan-bisnis">
            {businesses.map((item) => (
              <option key={item.id} value={`${item.nama_bisnis} (${item.id})`} />
            ))}
          </datalist>
        </div>

        {selected.some((item) => item.teks.trim() !== '' && item.id === '') && (
          <p className="mt-2 rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-900">
            Bisnis bertanda kuning belum cocok dengan daftarnya dan{' '}
            <span className="font-medium">akan dilewati</span>. Pilih dari daftar yang muncul
            saat mengetik.
          </p>
        )}

        {businesses.length === 0 && (
          <p className="mt-2 text-sm text-slate-500">
            Daftar bisnis tidak dapat dimuat. Kodenya tetap dapat diketik langsung.
          </p>
        )}
      </section>

      <DocumentRowsTable
        rows={rows}
        documentTypes={documentTypes}
        detailDocuments={detailDocuments}
        objectDocuments={objectDocuments}
        onChange={updateRow}
        onAdd={() => setRows((current) => [...current, emptyRow()])}
        onRemove={(index) =>
          setRows((current) => current.filter((_row, position) => position !== index))
        }
      />

      {/*
        Jumlah baris yang akan lahir disebutkan terang-terangan. Penyimpanan ini dapat
        menghasilkan puluhan baris sekaligus, dan tidak ada jalur hapus untuk
        membatalkannya — angka ini satu-satunya kesempatan petugas menyadari bahwa ia
        memilih lebih banyak daripada yang dimaksud.
      */}
      <p className="text-sm text-slate-600">
        Akan tersimpan{' '}
        <span className="font-semibold text-slate-900">{businessIDs.length * rows.length}</span>{' '}
        baris aturan ({businessIDs.length} bisnis × {rows.length} dokumen). Baris yang tersimpan{' '}
        <span className="font-medium">tidak dapat dihapus</span> dari layar ini.
      </p>

      <div className="flex flex-wrap justify-end gap-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}
