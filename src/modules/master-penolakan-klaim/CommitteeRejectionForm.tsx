import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type CommitteeRejection } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'

/**
 * Batas panjang harus sama dengan masterpenolakan.MaxNoteLengthKomite di backend.
 *
 * ASUMSI yang disadari: DDL POOLDATA.MST_REJECTED_KOMITE belum ada (R-08), dan procedure
 * lama menerimanya sebagai `varchar2` tanpa panjang.
 */
const MAX_NOTE_LENGTH = 100

const schema = z.object({
  catatan: z
    .string()
    .trim()
    .min(1, 'Note Komite Reject wajib diisi.')
    .max(MAX_NOTE_LENGTH, `Note Komite Reject paling panjang ${MAX_NOTE_LENGTH} karakter.`),
})

export type CommitteeRejectionFields = z.infer<typeof schema>

type Props = {
  /** Baris yang sedang disunting; null berarti mode tambah. */
  edited: CommitteeRejection | null
  isSaving: boolean
  error: unknown
  onSave: (values: CommitteeRejectionFields) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

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
        return Object.keys(error.violations()).length > 0
          ? null
          : { title: 'Belum dapat disimpan', description: error.message, tone: 'penolakan' }
      case ErrorCode.notFound:
        return {
          title: 'Baris ini sudah tidak ada',
          description: 'Mungkin sudah diubah petugas lain. Muat ulang daftarnya.',
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

function violationsOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {}
}

/**
 * CommitteeRejectionForm adalah form tab Penolakan Komite — satu isian saja.
 *
 * Persis seperti layar lama: modal "Insert/Update Reject Komite" pada
 * `Section/BrowseNoteRejectClaim-Section.xml` hanya mengikat
 * `InsertUpdateMasterReject.NoteKasir`. ID tidak pernah diketik — pada penambahan ia
 * diturunkan server dari isi tabel, pada pengubahan ia kunci baris yang dipilih.
 *
 * Tidak ada peringatan "menyimpan akan membatalkan persetujuan" di sini seperti pada tab
 * sebelah, karena tabel ini memang tidak mengenal persetujuan sama sekali — hanya
 * IDMASTER dan NOTEMASTER.
 */
export function CommitteeRejectionForm({ edited, isSaving, error, onSave, onCancel }: Props) {
  const editMode = edited !== null

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CommitteeRejectionFields>({
    resolver: zodResolver(schema),
    defaultValues: { catatan: edited?.catatan ?? '' },
  })

  useEffect(() => {
    reset({ catatan: edited?.catatan ?? '' })
  }, [edited, reset])

  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if (column === 'catatan') setError('catatan', { type: 'server', message })
    }
  }, [error, setError])

  const message = messageFor(error)
  const title = editMode ? 'Ubah Penolakan Komite' : 'Tambah Penolakan Komite'

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

      {editMode && (
        <p className="text-sm text-slate-600">
          ID Master: <span className="font-medium text-slate-900">{edited.id}</span>
        </p>
      )}

      <Field
        id="catatan"
        label="Note Komite Reject"
        type="text"
        maxLength={MAX_NOTE_LENGTH}
        error={errors.catatan?.message}
        {...register('catatan')}
      />

      <div className="flex flex-wrap justify-end gap-2 pt-2">
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
