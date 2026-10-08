import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, type CommitteeRejection } from '@/api/types'
import { Field } from '@/components/Field'
import { MasterFormActions, MasterFormFrame } from '@/components/masterform/BoxForm'
import { type CodeMessages, portalMessages, saveErrorMessage, validationMessage, violationsOf } from '@/components/masterform/saveErrorMessage'

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

/** Pesan galat penyimpanan per kode; yang tidak dikenal jatuh ke pesan galat sistem. */
const saveMessages: CodeMessages = {
  [ErrorCode.validationFailed]: validationMessage,
  [ErrorCode.notFound]: {
    title: 'Baris ini sudah tidak ada',
    description: 'Mungkin sudah diubah petugas lain. Muat ulang daftarnya.',
    tone: 'penolakan',
  },
  ...portalMessages,
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
export function CommitteeRejectionForm({ edited, isSaving, error, onSave, onCancel }: Readonly<Props>) {
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

  const message = saveErrorMessage(error, saveMessages)
  const title = editMode ? 'Ubah Penolakan Komite' : 'Tambah Penolakan Komite'

  return (
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={message}>
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

      <MasterFormActions isSaving={isSaving} onCancel={onCancel} />
    </MasterFormFrame>
  )
}
