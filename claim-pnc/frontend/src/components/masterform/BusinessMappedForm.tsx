import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo } from 'react'
import {
  useFieldArray,
  useForm,
  type ArrayPath,
  type DefaultValues,
  type FieldArray,
  type Path,
} from 'react-hook-form'
import type { z } from 'zod'

import { Field } from '@/components/Field'

import { MasterFormActions, MasterFormFrame, ReadOnlyIdRow, useServerViolations } from './BoxForm'
import { BusinessRowsField, businessRowsOf } from './BusinessRowsField'
import { saveErrorMessage } from './saveErrorMessage'

/** Nilai form: satu isian teks bernama bebas, ditambah grid `bisnis`. */
type BusinessMappedValues = { bisnis: { nama: string }[] }

type Props<T, V extends BusinessMappedValues> = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: T | null
  /** Skema form; ia yang memeriksa panjang isian teks dan nama bisnis. */
  schema: z.ZodType<V, V>
  /** Nama isian teks — sekaligus `id` input dan kolom pelanggaran server. */
  textName: Exclude<keyof V, 'bisnis'> & string
  textLabel: string
  textMaxLength: number
  /** `autoComplete="off"` pada isian teks, bila layar lama memakainya. */
  textAutoCompleteOff?: boolean | undefined
  /** Membaca isian teks, ID, dan pemetaan bisnis dari baris yang disunting. */
  textOf: (row: T) => string | undefined
  idOf: (row: T) => string | number
  businessesOf: (row: T) => readonly { nama: string }[] | undefined
  /** Label baris ID read-only; bawaannya "ID". */
  idLabel?: string | undefined
  title: string
  legend: string
  emptyText: string
  /** Saran nama bisnis milik entitas yang aktif. */
  businessNames: string[]
  businessMaxLength: number
  /** Pemetaan bisnis baris yang disunting masih dimuat. */
  isLoadingBusinessMapping: boolean
  isSaving: boolean
  error: unknown
  onSave: (values: V) => void
  onCancel: () => void
}

/**
 * BusinessMappedForm adalah form master "satu isian teks + grid bisnis" untuk DUA mode —
 * tambah dan ubah (Daftar Objek Dokumen, Master COL Simas Online).
 *
 * Satu form, bukan dua, mengikuti sistem lama yang memakai satu halaman untuk keduanya dan
 * menandai modusnya lewat isian ID yang hanya tampil bila terisi. Judulnya pun sama untuk
 * kedua modus; label tombol simpan "Simpan" atau "Ubah" mengikuti layar Pega.
 */
export function BusinessMappedForm<T, V extends BusinessMappedValues>({
  edited,
  schema,
  textName,
  textLabel,
  textMaxLength,
  textAutoCompleteOff = false,
  textOf,
  idOf,
  businessesOf,
  idLabel,
  title,
  legend,
  emptyText,
  businessNames,
  businessMaxLength,
  isLoadingBusinessMapping,
  isSaving,
  error,
  onSave,
  onCancel,
}: Readonly<Props<T, V>>) {
  const editMode = edited !== null

  function valuesOf(row: T | null): DefaultValues<V> {
    return {
      [textName]: (row === null ? undefined : textOf(row)) ?? '',
      bisnis: businessRowsOf(row === null ? undefined : businessesOf(row)),
    } as unknown as DefaultValues<V>
  }

  const {
    control,
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<V>({
    resolver: zodResolver(schema),
    defaultValues: valuesOf(edited),
  })

  // `bisnis` adalah grid, bukan satu isian. useFieldArray yang memegang barisnya supaya
  // setiap baris punya kunci stabil — tanpa itu, menghapus baris tengah akan membuat React
  // memakai ulang elemen input dan nilai baris berikutnya ikut bergeser.
  const { fields, append, remove } = useFieldArray({
    control,
    name: 'bisnis' as ArrayPath<V>,
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu —
  // misalnya pengguna menekan "Ubah" pada baris lain, atau saat pemetaan bisnisnya baru
  // selesai dimuat dari server.
  useEffect(() => {
    reset(valuesOf(edited))
    // valuesOf hanya membaca props yang tetap; yang berganti adalah `edited`.
  }, [edited, reset])

  const violationFields = useMemo(() => [textName, 'bisnis'] as Path<V>[], [textName])
  useServerViolations(error, setError, violationFields)

  const fieldErrors = errors as unknown as {
    [name: string]: { message?: string | undefined } | undefined
  } & { bisnis?: ({ nama?: { message?: string | undefined } } | undefined)[] & { message?: string | undefined } }

  return (
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={saveErrorMessage(error)}>
      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — nomornya diterbitkan server dari urutan basis data. Di Pega pun
          isiannya `pyReadOnly=true`. */}
      {editMode && <ReadOnlyIdRow label={idLabel} value={idOf(edited)} />}

      <Field
        id={textName}
        label={textLabel}
        type="text"
        autoFocus
        {...(textAutoCompleteOff ? { autoComplete: 'off' } : {})}
        maxLength={textMaxLength}
        error={fieldErrors[textName]?.message}
        {...register(textName as unknown as Path<V>)}
      />

      <BusinessRowsField
        legend={legend}
        emptyText={emptyText}
        isLoading={isLoadingBusinessMapping}
        rows={fields}
        suggestions={businessNames}
        maxLength={businessMaxLength}
        rowError={(index) => fieldErrors.bisnis?.[index]?.nama?.message}
        rowRegistration={(index) => register(`bisnis.${index}.nama` as Path<V>)}
        gridError={fieldErrors.bisnis?.message}
        onAdd={() => append({ nama: '' } as FieldArray<V, ArrayPath<V>>)}
        onRemove={remove}
      />

      <MasterFormActions
        isSaving={isSaving}
        onCancel={onCancel}
        submitLabel={editMode ? 'Ubah' : 'Simpan'}
        submitDisabled={isLoadingBusinessMapping}
      />
    </MasterFormFrame>
  )
}
