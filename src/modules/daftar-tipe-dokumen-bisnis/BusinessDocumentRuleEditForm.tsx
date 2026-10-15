import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type BusinessDocumentRule,
  type BusinessDocumentRuleRowInput,
  type MasterChoice,
} from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { DocumentRowsTable, emptyRow, labelFor, type RowDraft } from './DocumentRowsTable'

type Props = {
  businessID: string
  businessName: string
  rules: BusinessDocumentRule[]
  documentTypes: MasterChoice[]
  detailDocuments: MasterChoice[]
  objectDocuments: MasterChoice[]
  isSaving: boolean
  error: unknown
  onSave: (rows: BusinessDocumentRuleRowInput[]) => void
  onCancel: () => void
  /** Tindakan Jenis Klaim per baris; hanya ada bagi baris yang sudah tersimpan. */
  onOpenCoverage: (ruleID: string) => void
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
      case ErrorCode.businessDocumentRuleNotFound:
        return {
          title: 'Baris aturan sudah tidak ada',
          description:
            'Salah satu baris hilang sejak layar ini dibuka, sehingga tidak ada yang tersimpan. Tutup lalu buka kembali.',
          tone: 'penolakan',
        }
      case ErrorCode.businessDocumentRuleBusinessRequired:
        return {
          title: 'Nama Bisnis belum di isi',
          description: 'Tutup layar ini lalu buka kembali dari daftar.',
          tone: 'penolakan',
        }
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

/** toRowInput membuang teks bantu sebelum baris dikirim ke server. */
function toRowInput(row: RowDraft): BusinessDocumentRuleRowInput {
  return {
    id: row.id,
    id_tipe_dokumen: row.id_tipe_dokumen,
    id_object_dokumen: row.id_object_dokumen,
    id_detail_dokumen: row.id_detail_dokumen,
    detail_dokumen: row.detail_dokumen,
    status_wajib: row.status_wajib,
    minimum_dokumen: row.minimum_dokumen,
  }
}

/**
 * Form Ubah: SELURUH baris satu lini bisnis sekaligus.
 *
 * # Kenapa bukan satu baris per layar
 *
 * Versi pertama modul ini membuka satu baris per penyuntingan, lewat grid tingkat kedua
 * dan tombol "Detail". Itu penyimpangan yang saya buat sendiri. Di layar lama, Ubah
 * memuat seluruh baris bisnis itu ke dalam satu form, dan Simpan menuliskannya sekaligus
 * — setiap baris dikirim dengan `@if(.ID!="",.ID,"UnknownID")`
 * (`Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:2399-2401`), sehingga satu
 * penekanan Simpan dapat memperbarui beberapa baris dan menambah beberapa lainnya.
 *
 * Bedanya bukan jumlah klik: aturan kelengkapan dokumen dibaca dan diubah sebagai satu
 * kesatuan per lini bisnis — "tahap Survey butuh tiga dokumen, dua di antaranya wajib" —
 * dan menyuntingnya satu per satu memaksa petugas mengingat keadaan yang seharusnya
 * terlihat di depannya.
 *
 * # Nama Bisnis tidak dapat diubah
 *
 * Ia ditampilkan, bukan diisi. `UPDATE` di sistem lama memang tidak menyentuh
 * `BUSINESSID` (`PEGA_LST_DET_TYPE_DOC_BUSINESS.prc`), dan memindahkan aturan ke lini lain
 * bukan pengubahan melainkan penghapusan dari satu lini sekaligus penambahan ke lini lain.
 * Untuk memakai ulang susunan yang sudah ada, jalurnya tombol Copy.
 */
export function BusinessDocumentRuleEditForm({
  businessID,
  businessName,
  rules,
  documentTypes,
  detailDocuments,
  objectDocuments,
  isSaving,
  error,
  onSave,
  onCancel,
  onOpenCoverage,
}: Props) {
  // Dibentuk SEKALI. Bila dihitung ulang setiap render, setiap ketikan petugas akan
  // tertimpa oleh baris yang dimuat dari server.
  const [rows, setRows] = useState<RowDraft[]>(() => {
    if (rules.length === 0) return [emptyRow()]
    return rules.map((rule) => ({
      id: rule.id,
      id_tipe_dokumen: rule.id_tipe_dokumen,
      id_object_dokumen: rule.id_object_dokumen,
      id_detail_dokumen: rule.id_detail_dokumen,
      detail_dokumen: rule.detail_dokumen,
      status_wajib: rule.status_wajib,
      minimum_dokumen: rule.minimum_dokumen,
      tipe_dokumen_teks: labelFor(documentTypes, rule.id_tipe_dokumen),
      object_dokumen_teks: labelFor(objectDocuments, rule.id_object_dokumen),
      detail_dokumen_teks: labelFor(detailDocuments, rule.id_detail_dokumen),
    }))
  })

  const message = messageFor(error)
  const added = rows.filter((row) => row.id === '').length

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        onSave(rows.map(toRowInput))
      }}
      noValidate
      aria-label="Update Data"
      className="space-y-5 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">Update Data</h2>

      {message && (
        <ErrorMessage
          title={message.title}
          description={message.description}
          tone={message.tone}
        />
      )}

      <div>
        <span className="block text-sm font-semibold text-slate-800">Nama Bisnis</span>
        <p className="mt-1 text-sm text-slate-700">
          {businessName || <span className="text-slate-400">(tanpa nama)</span>}{' '}
          <span className="font-mono text-xs text-slate-500">({businessID})</span>
        </p>
        <p className="mt-1 text-xs text-slate-500">
          Lini bisnis sebuah aturan tidak dapat diubah. Untuk memakai ulang susunan ini pada lini
          lain, gunakan Copy di daftar.
        </p>
      </div>

      <DocumentRowsTable
        rows={rows}
        documentTypes={documentTypes}
        detailDocuments={detailDocuments}
        objectDocuments={objectDocuments}
        onChange={(index, patch) =>
          setRows((current) =>
            current.map((row, position) => (position === index ? { ...row, ...patch } : row)),
          )
        }
        onAdd={() => setRows((current) => [...current, emptyRow()])}
        onRemove={(index) =>
          setRows((current) => current.filter((_row, position) => position !== index))
        }
        /*
          Jenis Klaim hanya ada bagi baris yang SUDAH tersimpan, dan itu bukan pembatasan
          yang dikarang: jaminan ditulis lewat pemanggilan tersendiri yang menunjuk ID
          baris, sehingga baris yang belum punya ID memang belum dapat menerimanya.
        */
        rowAction={(row) =>
          row.id === '' ? null : (
            <Button
              tone="kedua"
              onClick={() => onOpenCoverage(row.id)}
              aria-label={`Jenis Klaim baris ${row.id}`}
            >
              Jenis Klaim
            </Button>
          )
        }
      />

      {added > 0 && (
        <p className="text-sm text-slate-600">
          <span className="font-semibold text-slate-900">{added}</span> baris baru akan ditambahkan
          pada lini bisnis ini. Baris yang tersimpan{' '}
          <span className="font-medium">tidak dapat dihapus</span>.
        </p>
      )}

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
