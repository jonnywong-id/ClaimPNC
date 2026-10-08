import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError } from '@/api/client'
import { ErrorCode, type SurveyorType } from '@/api/types'
import {
  PanelCodeAndField,
  PanelFormActions,
  PanelFormFrame,
  PanelSaveErrorMessage,
  panelCodeTakenMessage,
  panelPortalMessages,
} from '@/components/masterform/PanelForm'
import type { CodeMessages } from '@/components/masterform/saveErrorMessage'

import { useSaveSurveyorType } from './api'

/**
 * Batas panjang deskripsi; sama dengan mastertipesurveyors.MaxDescriptionLength di
 * backend.
 *
 * Ditetapkan Work Owner 2026-09-19. Ia keputusan, bukan bacaan: kolom DESCRIPTION
 * bertipe VARCHAR2(4000) dan layar Pega tidak membatasi apa pun, sehingga tidak ada
 * angka yang dapat dibaca dari mana pun. Seratus dipakai supaya batas yang dilihat
 * pengguna seragam dengan layar master lain.
 *
 * Bila angka ini berubah, KEDUA tempat wajib ikut berubah.
 */
const MAX_DESCRIPTION_LENGTH = 100

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Itu duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan langsung
 * ke API tidak melewati layar ini sama sekali. Menghapus salah satunya berarti memilih
 * antara layar yang lamban atau API yang tidak terjaga.
 *
 * Keunikan nama TIDAK diperiksa di sini: ia menuntut mengetahui seluruh nama yang ada, dan
 * jawabannya dapat berubah antara saat layar dimuat dan saat Simpan ditekan. Penegakannya
 * ada di indeks unik basis data, dan galatnya ditampilkan di bawah.
 */
const schema = z.object({
  deskripsi: z
    .string()
    .trim()
    .min(1, 'Tipe surveyor wajib diisi.')
    .max(
      MAX_DESCRIPTION_LENGTH,
      `Tipe surveyor paling panjang ${MAX_DESCRIPTION_LENGTH} karakter.`,
    ),
})

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  surveyorType: SurveyorType | null
  onClose: () => void
}

/**
 * Form tambah dan ubah Master Tipe Surveyors.
 *
 * Meniru fungsi form pada `Section/BrowseSuveryors-Section.xml` — kode read-only
 * (`pyEditOptions=Read-only`), satu isian deskripsi, dan tombol Simpan — dengan dua
 * perbedaan yang disengaja: isiannya wajib diisi, dan nama yang sudah dipakai ditolak
 * (keputusan Work Owner, sejalan dengan Master Status Klaim 2026-09-17).
 */
export function SurveyorTypeForm({ surveyorType, onClose }: Readonly<Props>) {
  const save = useSaveSurveyorType()
  const editing = surveyorType !== null

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { deskripsi: surveyorType?.deskripsi ?? '' },
  })

  // Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di
  // kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya
  // berguna bila layar menyorotnya di tempat isiannya.
  useEffect(() => {
    if (!(save.error instanceof APIError)) return
    const violation = save.error.violations()['deskripsi']
    if (violation) setError('deskripsi', { type: 'server', message: violation })
  }, [save.error, setError])

  function send(values: FieldValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal.
    save.mutate(
      editing ? { kode: surveyorType.kode, deskripsi: values.deskripsi } : values,
      { onSuccess: onClose },
    )
  }

  // Panel muncul di atas tabel, bukan sebagai dialog melayang: pengguna sering perlu
  // melihat tipe lain yang sudah ada untuk memastikan nama yang diketiknya tidak
  // bertabrakan (lihat PanelFormFrame).
  return (
    <PanelFormFrame
      onSubmit={handleSubmit(send)}
      ariaLabel={editing ? 'Ubah tipe surveyor' : 'Tambah tipe surveyor'}
      title={editing ? 'Ubah Tipe Surveyor' : 'Tambah Tipe Surveyor'}
      subtitle={
        editing
          ? 'Hanya nama tipe yang dapat diubah. Kode tetap, karena data surveyor menyimpannya.'
          : 'Kode dibuat sistem setelah disimpan, melanjutkan nomor terakhir.'
      }
    >
      {save.isError && <PanelSaveErrorMessage error={save.error} messages={saveMessages} />}

      <PanelCodeAndField
        codeLabel="Kode"
        codeValue={surveyorType?.kode}
        id="deskripsi"
        label="Tipe Surveyor"
        placeholder="Contoh: LOSS ADJUSTER"
        maxLength={MAX_DESCRIPTION_LENGTH}
        hint={`Paling panjang ${MAX_DESCRIPTION_LENGTH} karakter, dan belum dipakai tipe lain.`}
        error={errors.deskripsi?.message}
        disabled={save.isPending}
        registration={register('deskripsi')}
      />

      <PanelFormActions isPending={save.isPending} onCancel={onClose} />
    </PanelFormFrame>
  )
}

/**
 * Galat simpan dibedakan menurut KODE-nya, bukan teks pesannya: nama yang bentrok dapat
 * diperbaiki pengguna, portal yang belum dipilih diperbaiki di bilah atas, dan gangguan
 * sistem tidak dapat ditolong dengan mencoba ulang berkali-kali.
 */
const saveMessages: CodeMessages = {
  [ErrorCode.surveyorTypeTaken]: {
    title: 'Nama tipe surveyor sudah dipakai',
    description: 'Sudah ada tipe dengan nama itu di entitas ini. Pakai nama lain.',
    tone: 'penolakan',
  },
  // Bila detailnya ada, isiannya sudah disorot di tempatnya; kotak pesan hanya akan
  // mengulang hal yang sama.
  [ErrorCode.validationFailed]: (error) =>
    Object.keys(error.violations()).length > 0
      ? null
      : { title: 'Isian belum benar', description: error.message, tone: 'penolakan' },
  [ErrorCode.surveyorTypeNotFound]: {
    title: 'Tipe surveyor tidak ditemukan',
    description: 'Baris ini mungkin sudah diubah petugas lain. Muat ulang daftarnya.',
    tone: 'penolakan',
  },
  [ErrorCode.surveyorTypeCodeTaken]: panelCodeTakenMessage,
  [ErrorCode.surveyorTypeCodeUnavailable]: {
    title: 'Kode tidak dapat dibentuk',
    description:
      'Basis data entitas ini belum siap menerbitkan kode baru. Mengulang tidak akan menolong — hubungi administrator Claim PNC.',
    tone: 'gangguan',
  },
  ...panelPortalMessages,
}
