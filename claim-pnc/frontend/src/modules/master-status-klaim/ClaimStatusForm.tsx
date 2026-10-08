import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, type ClaimStatus } from '@/api/types'
import {
  PanelCodeAndField,
  PanelFormActions,
  PanelFormFrame,
  PanelSaveErrorMessage,
  panelCodeTakenMessage,
  panelNotFoundMessage,
  panelPortalMessages,
  panelValidationMessage,
} from '@/components/masterform/PanelForm'
import type { CodeMessages } from '@/components/masterform/saveErrorMessage'

import { useSaveClaimStatus } from './api'

/** Batas panjang label; sama dengan masterstatus.MaxLabelLength di backend. */
const MAX_LABEL_LENGTH = 100

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Itu duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan
 * langsung ke API tidak melewati layar ini sama sekali. Menghapus salah satunya berarti
 * memilih antara layar yang lamban atau API yang tidak terjaga.
 *
 * Keunikan label TIDAK diperiksa di sini: ia menuntut mengetahui seluruh label yang ada,
 * dan jawabannya dapat berubah antara saat layar dimuat dan saat Simpan ditekan.
 * Penegakannya ada di indeks unik basis data, dan galatnya ditampilkan di bawah.
 */
const schema = z.object({
  label: z
    .string()
    .trim()
    .min(1, 'Status wajib diisi.')
    .max(MAX_LABEL_LENGTH, `Status paling panjang ${MAX_LABEL_LENGTH} karakter.`),
})

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  status: ClaimStatus | null
  tutup: () => void
}

/**
 * Form tambah dan ubah Master Status Klaim.
 *
 * Meniru fungsi form `ListStatusClaim` di Pega — kode read-only, satu isian Status, dan
 * tombol Simpan — dengan dua perbedaan yang disengaja: isiannya wajib diisi, dan nama
 * yang sudah dipakai ditolak (keputusan Work Owner 2026-09-17).
 */
export function ClaimStatusForm({ status, tutup }: Readonly<Props>) {
  const save = useSaveClaimStatus()
  const editing = status !== null

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { label: status?.label ?? '' },
  })

  function send(values: FieldValues) {
    save.mutate(
      editing ? { kode: status.kode, label: values.label } : { label: values.label },
      { onSuccess: tutup },
    )
  }

  // Panel muncul di atas tabel, bukan sebagai dialog melayang: pengguna sering perlu
  // melihat status lain yang sudah ada untuk memastikan nama yang diketiknya tidak
  // bertabrakan (lihat PanelFormFrame).
  return (
    <PanelFormFrame
      onSubmit={handleSubmit(send)}
      ariaLabel={editing ? 'Ubah status klaim' : 'Tambah status klaim'}
      title={editing ? 'Ubah Status Klaim' : 'Tambah Status Klaim'}
      subtitle={
        editing
          ? 'Hanya nama status yang dapat diubah. Kode tetap, karena klaim lama menyimpannya.'
          : 'Kode dibuat sistem setelah disimpan, melanjutkan nomor terakhir.'
      }
    >
      {save.isError && <PanelSaveErrorMessage error={save.error} messages={saveMessages} />}

      <PanelCodeAndField
        codeLabel="Kode"
        codeValue={status?.kode}
        id="label"
        label="Status"
        placeholder="Contoh: Reopen Claim"
        maxLength={MAX_LABEL_LENGTH}
        hint={`Paling panjang ${MAX_LABEL_LENGTH} karakter, dan belum dipakai status lain.`}
        error={errors.label?.message}
        disabled={save.isPending}
        registration={register('label')}
      />

      <PanelFormActions isPending={save.isPending} onCancel={tutup} />
    </PanelFormFrame>
  )
}

/**
 * Galat simpan dibedakan menurut KODE-nya, bukan teks pesannya: nama yang bentrok dapat
 * diperbaiki pengguna, validasi server menunjuk kolom tertentu, dan gangguan sistem tidak
 * dapat ditolong dengan mencoba ulang berkali-kali.
 */
const saveMessages: CodeMessages = {
  [ErrorCode.statusLabelTaken]: {
    title: 'Nama status sudah dipakai',
    description: 'Sudah ada status dengan nama itu. Pakai nama lain.',
    tone: 'penolakan',
  },
  [ErrorCode.validationFailed]: panelValidationMessage,
  [ErrorCode.claimStatusNotFound]: panelNotFoundMessage('Status tidak ditemukan'),
  [ErrorCode.statusCodeTaken]: panelCodeTakenMessage,
  ...panelPortalMessages,
}
