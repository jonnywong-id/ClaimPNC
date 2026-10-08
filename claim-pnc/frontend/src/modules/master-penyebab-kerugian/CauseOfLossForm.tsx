import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { ErrorCode, type CauseOfLoss } from '@/api/types'
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

import { useSaveCauseOfLoss } from './api'

/**
 * Batas panjang keterangan; sama dengan masterpenyebabkerugian.MaxDescriptionLength di
 * backend.
 *
 * Lebar kolom `COL_DESC` yang sebenarnya BELUM DIKETAHUI (`R-08`). Bila DBA kelak
 * melaporkan kolomnya lebih sempit, yang berubah cukup konstanta ini dan pasangannya di Go.
 */
const MAX_DESCRIPTION_LENGTH = 100

/**
 * Satu aturan saja, dan itu bukan kelalaian.
 *
 * Layar Pega tidak memvalidasi apa pun di sini — `Section/BrowseCauseOfLoss-Section.xml`
 * tidak memasang `pyRequired` maupun `pyMaxLength`, dan
 * `Database/PEGA_M_CAUSE_OF_LOSS.prc` tidak memeriksa apa pun sebelum menyisipkan. Work
 * Owner memutuskan perilaku itu DIPERTAHANKAN pada 2026-09-20 sesuai `P-5`, sehingga
 * keterangan KOSONG dan keterangan GANDA sama-sama diterima.
 *
 * Modul ini karena itu sejalan dengan Master Dominan Factor, dan berbeda dari Master
 * Status Klaim serta Master Tipe Surveyors yang justru menolak keduanya. Perbedaan itu
 * disengaja.
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
  deskripsi: z
    .string()
    .trim()
    .max(
      MAX_DESCRIPTION_LENGTH,
      `Deskripsi Kerugian paling panjang ${MAX_DESCRIPTION_LENGTH} karakter.`,
    ),
})

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  cause: CauseOfLoss | null
  tutup: () => void
}

/**
 * Form tambah dan ubah Master Penyebab Kerugian.
 *
 * Meniru fungsi form pada section `BrowseCauseOfLoss` — ID read-only, satu isian
 * **Deskripsi Kerugian**, dan tombol Simpan. Judulnya pun diambil apa adanya dari `pyTitle`
 * layar lama, "Memperbaharui Data" (`D-13`). Padanannya untuk penambahan tidak ada di XML —
 * layar lama memakai judul yang sama untuk keduanya — sehingga "Menambah Data" dipakai,
 * mengikuti pasangan yang sudah ada di Master Dominan Factor.
 *
 * # Satu isian Pega yang sengaja TIDAK dibawa
 *
 * Form Pega sebenarnya memuat DUA isian berlabel "Deskripsi Kerugian": satu terikat
 * `TempCauseOfLoss.COL_DESC`, satu lagi terikat `TempCauseOfLoss.Description`. Keduanya
 * berbagi `pyAutomationID` yang sama — tanda salin-tempel.
 *
 * Yang kedua **mati**: `Activity/SetCauseOfLossValue_act-Act.xml` mengisi form dengan
 * `M_COL_ID`, `COL_DESC`, dan `pyNote` saja, sehingga isian itu tidak pernah terisi saat
 * mengubah; dan view `V_M_CAUSE_OF_LOSS` tidak punya kolom untuk membacanya kembali.
 * Menyalinnya berarti menghadirkan isian yang sejak semula tidak berfungsi.
 */
export function CauseOfLossForm({ cause, tutup }: Readonly<Props>) {
  const save = useSaveCauseOfLoss()
  const editing = cause !== null

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { deskripsi: cause?.deskripsi ?? '' },
  })

  function send(values: FieldValues) {
    save.mutate(
      editing ? { id: cause.id, deskripsi: values.deskripsi } : { deskripsi: values.deskripsi },
      { onSuccess: tutup },
    )
  }

  // Panel muncul di atas tabel, bukan sebagai dialog melayang — dan di modul ini alasannya
  // lebih kuat daripada di modul master lain: keterangan GANDA diterima, sehingga pengguna
  // sering perlu melihat daftar yang ada untuk memastikan ia tidak sedang mengetik ulang
  // golongan yang sudah terdaftar.
  return (
    <PanelFormFrame
      onSubmit={handleSubmit(send)}
      ariaLabel={
        editing ? 'Memperbaharui Data penyebab kerugian' : 'Menambah Data penyebab kerugian'
      }
      title={editing ? 'Memperbaharui Data' : 'Menambah Data'}
      subtitle={
        editing
          ? 'Hanya Deskripsi Kerugian yang dapat diubah. ID tetap, karena rincian penyebab kerugian bernaung di bawahnya.'
          : 'ID dibuat sistem setelah disimpan, melanjutkan nomor terakhir.'
      }
    >
      {save.isError && <PanelSaveErrorMessage error={save.error} messages={saveMessages} />}

      {/* Satu peringatan saja, dan ia menggantikan pemeriksaan yang sengaja tidak ada:
          karena deskripsi ganda diterima, pengguna tidak akan pernah ditolak sistem —
          jadi satu-satunya cara ia tahu adalah diberi tahu. Bentuknya sama dengan
          Master Dominan Factor, yang keputusannya sama.

          Pemberitahuan kedua — bahwa deskripsi dipakai mengelompokkan laporan — sempat
          ada di sini lalu DICABUT: layar Pega tidak memuatnya, dan Work Owner meminta
          layar mengikuti Pega saja. Akibat itu tetap nyata dan tetap tercatat di
          docs/keputusan-implementasi.md, hanya tidak lagi ditampilkan di layar. */}
      {!editing && <DuplicateAllowedNote noun="Deskripsi" />}

      <PanelCodeAndField
        codeLabel="ID"
        codeValue={cause?.id}
        id="deskripsi"
        label="Deskripsi Kerugian"
        placeholder="Contoh: Kebakaran"
        maxLength={MAX_DESCRIPTION_LENGTH}
        hint={`Paling panjang ${MAX_DESCRIPTION_LENGTH} karakter.`}
        error={errors.deskripsi?.message}
        disabled={save.isPending}
        registration={register('deskripsi')}
      />

      <PanelFormActions isPending={save.isPending} onCancel={tutup} />
    </PanelFormFrame>
  )
}

/**
 * Galat simpan dibedakan menurut KODE-nya, bukan teks pesannya.
 *
 * Empat jenis menuntut tindak lanjut berbeda: nomor yang bentrok cukup dicoba ulang,
 * validasi server menunjuk kolom tertentu, tabel situs yang kosong tidak dapat ditolong
 * pengguna sama sekali, dan gangguan sistem tidak dapat ditolong dengan mencoba ulang
 * berkali-kali.
 */
const saveMessages: CodeMessages = {
  [ErrorCode.validationFailed]: panelValidationMessage,
  [ErrorCode.causeOfLossNotFound]: panelNotFoundMessage('Penyebab kerugian tidak ditemukan'),
  [ErrorCode.causeOfLossIDTaken]: panelIDTakenMessage,
  [ErrorCode.causeOfLossIDUnavailable]: {
    title: 'Nomor tidak dapat dibentuk',
    description:
      'Mengulang tidak akan menolong: tabel situs pada basis data entitas ini belum berisi baris aktif. Hubungi administrator Claim PNC.',
    tone: 'gangguan',
  },
  ...panelPortalMessages,
}
