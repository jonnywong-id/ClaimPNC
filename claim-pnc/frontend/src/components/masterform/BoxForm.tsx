import { useEffect, type FormEventHandler, type ReactNode } from 'react'
import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'

import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { violationsOf, type MessageContent } from './saveErrorMessage'

/**
 * Bagian bersama form master bergaya "kotak" — satu form untuk DUA mode, tambah dan ubah,
 * yang menerima status penyimpanan (`isSaving`, `error`) dari halamannya.
 */

type FrameProps = {
  onSubmit: FormEventHandler<HTMLFormElement>
  /** Judul form; sekaligus nama aksesibelnya. */
  title: string
  /** Pesan galat penyimpanan yang tidak menunjuk isian tertentu; null bila tidak ada. */
  message: MessageContent | null
  children: ReactNode
}

/** MasterFormFrame menggambar bingkai form: judul, kotak pesan galat, lalu isiannya. */
export function MasterFormFrame({ onSubmit, title, message, children }: Readonly<FrameProps>) {
  return (
    <form
      onSubmit={onSubmit}
      noValidate
      aria-label={title}
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 className="text-base font-semibold text-slate-900">{title}</h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {children}
    </form>
  )
}

type ReadOnlyIdProps = {
  /** Label baris, mis. "ID" atau "ID Kerugian". */
  label?: string | undefined
  value: ReactNode
}

/**
 * ReadOnlyIdRow menampilkan ID baris yang disunting, yang tidak dapat diubah.
 *
 * Hanya dipakai saat menyunting: pada penambahan ID belum ada — nomornya diterbitkan server.
 */
export function ReadOnlyIdRow({ label = 'ID', value }: Readonly<ReadOnlyIdProps>) {
  return (
    <div>
      <span className="block text-sm font-medium text-slate-700">{label}</span>
      <p className="mt-1 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-slate-600">
        {value}
        <span className="ml-2 text-xs text-slate-500">(tidak dapat diubah)</span>
      </p>
    </div>
  )
}

type ActionsProps = {
  isSaving: boolean
  onCancel: () => void
  /** Label tombol simpan saat tidak sedang menyimpan; bawaannya "Simpan". */
  submitLabel?: string | undefined
  /** Penonaktif tambahan tombol simpan, di luar `isSaving`. */
  submitDisabled?: boolean | undefined
}

/** MasterFormActions menggambar tombol Batal dan Simpan di kaki form. */
export function MasterFormActions({
  isSaving,
  onCancel,
  submitLabel = 'Simpan',
  submitDisabled = false,
}: Readonly<ActionsProps>) {
  return (
    <div className="flex flex-wrap justify-end gap-2 pt-2">
      <Button tone="halus" onClick={onCancel} disabled={isSaving}>
        Batal
      </Button>
      <Button type="submit" tone="utama" disabled={isSaving || submitDisabled}>
        {isSaving ? 'Menyimpan…' : submitLabel}
      </Button>
    </div>
  )
}

/**
 * useServerViolations menyorot pelanggaran yang dilaporkan server pada isiannya
 * masing-masing, bukan hanya meringkasnya di satu kotak pesan. Server mengirim SELURUH
 * pelanggaran sekaligus (P-5), dan itu hanya berguna bila layar menyorotnya satu per satu.
 *
 * `fields` adalah isian yang dikenal form ini; pelanggaran atas kolom lain diabaikan.
 * Senarainya harus tetap (konstanta modul): ia dependensi efek, dan senarai baru di setiap
 * render akan menyorot ulang tanpa henti.
 */
export function useServerViolations<T extends FieldValues>(
  error: unknown,
  setError: UseFormSetError<T>,
  fields: readonly Path<T>[],
) {
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      const field = fields.find((name) => name === column)
      if (field !== undefined) setError(field, { type: 'server', message })
    }
  }, [error, setError, fields])
}
