import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DetailDocumentType, type DetailDocumentTypeChoice } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

/**
 * Batas panjang isian, disamakan dengan konstanta di lapisan domain.
 *
 * Ini BUKAN aturan bisnis melainkan bentuk kolom — layar Pega tidak memuat satu pun
 * `pyMaxLength` pada isian mana pun. Angkanya ada di sini supaya pesannya muncul sebelum
 * permintaan dikirim, bukan sesudah server menolaknya; server tetap memeriksanya sendiri.
 */
const MAX_DETAIL = 200
const MAX_REFERENCE = 50

/**
 * Yang diperiksa di layar hanyalah panjang isian.
 *
 * Layar Pega tidak memuat satu pun `pyRequired` bernilai true maupun Rule-Obj-Validate
 * pada `Section/BrowseListDetailTypeDocument-Section.xml`, sehingga Detail Dokumen kosong
 * dan tipe dokumen kosong sama-sama sah (`P-5`).
 */
const schema = z.object({
  id_tipe_dokumen: z.string().trim().max(MAX_REFERENCE, `Paling panjang ${MAX_REFERENCE} karakter.`),
  detail_dokumen: z.string().trim().max(MAX_DETAIL, `Paling panjang ${MAX_DETAIL} karakter.`),
})

export type DetailDocumentTypeFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: DetailDocumentType | null
  /** Pilihan ID Tipe Dokumen, dibaca dari POOLDATA.V_LST_DOC_TYPE. */
  documentTypes: DetailDocumentTypeChoice[]
  isSaving: boolean
  error: unknown
  onSave: (values: DetailDocumentTypeFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti. */
function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.validationFailed:
        return {
          title: 'Ada isian yang belum benar',
          description:
            'Periksa keterangan di bawah setiap isian, perbaiki, lalu simpan lagi. Isian Anda belum tersimpan.',
          tone: 'penolakan',
        }
      case ErrorCode.notFound:
        return {
          title: 'Baris ini sudah tidak ada',
          description:
            'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih portal entitas di bagian atas halaman, lalu simpan lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Terjadi kesalahan pada sistem',
          description: 'Isian Anda belum tersimpan. Coba beberapa saat lagi.',
          tone: 'gangguan',
        }
    }
  }
  return null
}

/** Menyusun nilai awal form dari baris yang disunting. */
function valuesOf(edited: DetailDocumentType | null): DetailDocumentTypeFields {
  return {
    id_tipe_dokumen: edited?.id_tipe_dokumen ?? '',
    detail_dokumen: edited?.detail_dokumen ?? '',
  }
}

/**
 * DetailDocumentTypeForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama:
 * `Section/BrowseListDetailTypeDocument-Section.xml` memakai satu halaman `TempDTDoc`
 * untuk keduanya dan membedakan modusnya hanya lewat `TempDTDoc.pyLabel` yang berisi
 * `"tambah"` atau `"update"`.
 *
 * # Dua isian, mengikuti layar yang benar-benar berjalan
 *
 *	ID                 TempDTDoc.ID               hanya ditampilkan, saat menyunting
 *	ID Tipe Dokumen    TempDTDoc.DOC_TYPE_ID      dropdown V_LST_DOC_TYPE, isinya KODE
 *	Detail Dokumen     TempDTDoc.DETAIL_DOCUMENT  teks
 *
 * # Enam kolom yang TIDAK disunting dari sini
 *
 * `STS_INSURED`, `DOC_COL_ID`, `DOC_COL_INFO`, `OBJ_DOC`, `OBJ_DOC_DESC`, dan `RISK` tetap
 * ada di tabel, tetap ikut terbaca, dan tetap DIBAWA APA ADANYA saat menyimpan — tetapi
 * tidak punya isian di layar ini. Begitu pula aturan per lini bisnis, yang tetap hidup di
 * `LST_DET_TYPE_DOC.JSON_DATA` milik Pega dan tidak disentuh modul ini.
 *
 * Buktinya bertentangan, dan pertentangannya dicatat di sini supaya tidak dibaca sebagai
 * kelalaian: section itu memuat keenamnya dengan caption lengkap dan `pyVisible = ALWAYS`,
 * sementara layar "Tambah Data" yang BENAR-BENAR berjalan berhenti pada Detail Dokumen dan
 * langsung tombol Simpan (koreksi Work Owner 2026-10-03, dengan tangkapan layarnya). Yang
 * diikuti adalah layar yang berjalan.
 *
 * Konsekuensinya satu dan harus disadari: keenam kolom itu beserta aturan lini bisnisnya
 * kini hanya dapat diisi dari layar Pega selama masa paralel.
 *
 * # Kenapa SelectField untuk ID Tipe Dokumen
 *
 * `pyFormat = pxDropdown` dengan `pyDisplayAsComboBox = false` pada TempDTDoc.DOC_TYPE_ID,
 * dan layar Pega memang menampilkannya sebagai dropdown. Kode di luar daftar tidak dapat
 * diketik di sana, sehingga tidak dapat diketik di sini.
 */
export function DetailDocumentTypeForm({
  edited,
  documentTypes,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const editMode = edited !== null

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<DetailDocumentTypeFields>({
    resolver: zodResolver(schema),
    defaultValues: valuesOf(edited),
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu
  // — misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset(valuesOf(edited))
  }, [edited, reset])

  const message = messageFor(error)

  // Judul yang sama untuk kedua modus, mengikuti layar lama. Beda modus tetap terlihat
  // dari ada-tidaknya baris "ID" di bawahnya dan dari label tombol simpannya.
  const title = 'Detail Tipe Dokumen'

  return (
    <form
      onSubmit={handleSubmit(onSave)}
      noValidate
      aria-label={title}
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">{title}</h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — kodenya diterbitkan server dari urutan basis data. Di Pega pun
          isiannya `pyEditOptions=Read-only`. */}
      {editMode && (
        <div>
          <span className="block text-sm font-medium text-slate-700">ID</span>
          <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
            {edited.id}
            <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
          </p>
        </div>
      )}

      {/* Pilihan kosong di paling atas memang disediakan: tidak ada `pyRequired` di layar
          lama, sehingga tipe dokumen kosong adalah keadaan yang sah. */}
      <SelectField
        id="id_tipe_dokumen"
        label="ID Tipe Dokumen"
        options={[
          { value: '', label: '' },
          ...documentTypes.map((row) => ({
            value: row.id,
            label: row.nama ? `${row.id} — ${row.nama}` : row.id,
          })),
        ]}
        autoFocus
        error={errors.id_tipe_dokumen?.message}
        {...register('id_tipe_dokumen')}
      />

      <Field
        id="detail_dokumen"
        label="Detail Dokumen"
        type="text"
        error={errors.detail_dokumen?.message}
        {...register('detail_dokumen')}
      />

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        {/* Label tombolnya mengikuti layar Pega, yang memakai "Simpan" dan "Ubah" untuk
            kedua modusnya. */}
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : editMode ? 'Ubah' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}
