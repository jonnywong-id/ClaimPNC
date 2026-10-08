import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, type DominantFactor } from '@/api/types'
import {
  DuplicateAllowedNote,
  PanelCodeAndField,
  PanelFormActions,
  PanelFormFrame,
  PanelSaveErrorMessage,
  panelIDTakenMessage,
  panelNotFoundMessage,
  panelPortalMessages,
  panelValidationMessage,
} from '@/components/masterform/PanelForm'
import type { CodeMessages } from '@/components/masterform/saveErrorMessage'

import { useSaveDominantFactor } from './api'

/** Batas panjang keterangan; sama dengan masterdominanfactor.MaxNameLength di backend. */
const MAX_NAME_LENGTH = 100

/**
 * Satu aturan saja, dan itu bukan kelalaian.
 *
 * Layar Pega menerima keterangan KOSONG dan keterangan GANDA
 * (`Section/DetailDominanFactor_Sec-Section.xml` — `pyRequired=false`, tanpa constraint
 * keunikan), dan Work Owner memutuskan keduanya DIPERTAHANKAN pada 2026-09-20 sesuai
 * `P-5`. Modul ini karena itu berbeda dari Master Status Klaim dan Master Tipe Surveyors,
 * yang justru menolak keduanya — dan perbedaan itu disengaja.
 *
 * Yang tersisa hanyalah batas panjang, dan ia bukan aturan bisnis melainkan penjaga
 * terhadap penolakan basis data: tanpa ini, isian yang melebihi lebar kolom sampai ke
 * pengguna sebagai galat 500 beserta nomor galat Oracle.
 *
 * Aturan yang sama dinyatakan dua kali — di sini dan di domain Go. Itu duplikasi yang
 * DISENGAJA: yang di sini menjawab pengguna tanpa perjalanan jaringan; yang di sana yang
 * menegakkan, karena pemanggilan langsung ke API tidak melewati layar ini sama sekali.
 */
const schema = z.object({
  nama: z
    .string()
    .trim()
    .max(MAX_NAME_LENGTH, `Keterangan paling panjang ${MAX_NAME_LENGTH} karakter.`),
})

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  factor: DominantFactor | null
  tutup: () => void
}

/**
 * Form tambah dan ubah Master Dominan Factor.
 *
 * Meniru fungsi form pada harness `DetailDominanFactor` — ID read-only, satu isian
 * Keterangan, dan tombol Simpan. Judulnya pun mengikuti yang di sana: layar lama memakai
 * "Menambah Data" dan "Memperbaharui Data" (`pyTitle`), dan `D-13` menetapkan teks yang
 * dilihat pengguna mengikuti layar Pega apa adanya.
 */
export function DominantFactorForm({ factor, tutup }: Readonly<Props>) {
  const save = useSaveDominantFactor()
  const editing = factor !== null

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { nama: factor?.nama ?? '' },
  })

  function send(values: FieldValues) {
    save.mutate(
      editing ? { id: factor.id, nama: values.nama } : { nama: values.nama },
      { onSuccess: tutup },
    )
  }

  // Panel muncul di atas tabel, bukan sebagai dialog melayang — dan di modul ini alasannya
  // lebih kuat daripada di modul master lain: keterangan GANDA diterima, sehingga pengguna
  // sering perlu melihat daftar yang ada untuk memastikan ia tidak sedang mengetik ulang
  // faktor yang sudah terdaftar.
  return (
    <PanelFormFrame
      onSubmit={handleSubmit(send)}
      ariaLabel={editing ? 'Memperbaharui Data faktor dominan' : 'Menambah Data faktor dominan'}
      title={editing ? 'Memperbaharui Data' : 'Menambah Data'}
      subtitle={
        editing
          ? 'Hanya keterangan yang dapat diubah. ID tetap, karena klaim lama menyimpannya.'
          : 'ID dibuat sistem setelah disimpan, melanjutkan nomor terakhir.'
      }
    >
      {save.isError && <PanelSaveErrorMessage error={save.error} messages={saveMessages} />}

      {/* Peringatan ini menggantikan pemeriksaan yang sengaja tidak ada. Karena
          keterangan ganda diterima, pengguna tidak akan pernah ditolak sistem — jadi
          satu-satunya cara ia tahu adalah diberi tahu. */}
      {!editing && <DuplicateAllowedNote noun="Keterangan" />}

      <PanelCodeAndField
        codeLabel="ID"
        codeValue={factor?.id}
        id="nama"
        label="Keterangan"
        placeholder="Contoh: Kelalaian pihak ketiga"
        maxLength={MAX_NAME_LENGTH}
        hint={`Paling panjang ${MAX_NAME_LENGTH} karakter.`}
        error={errors.nama?.message}
        disabled={save.isPending}
        registration={register('nama')}
      />

      <PanelFormActions isPending={save.isPending} onCancel={tutup} />
    </PanelFormFrame>
  )
}

/**
 * Galat simpan dibedakan menurut KODE-nya, bukan teks pesannya: nomor yang bentrok cukup
 * dicoba ulang, validasi server menunjuk kolom tertentu, dan gangguan sistem tidak dapat
 * ditolong dengan mencoba ulang berkali-kali.
 */
const saveMessages: CodeMessages = {
  [ErrorCode.validationFailed]: panelValidationMessage,
  [ErrorCode.dominantFactorNotFound]: panelNotFoundMessage('Faktor dominan tidak ditemukan'),
  [ErrorCode.dominantFactorIDTaken]: panelIDTakenMessage,
  ...panelPortalMessages,
}
