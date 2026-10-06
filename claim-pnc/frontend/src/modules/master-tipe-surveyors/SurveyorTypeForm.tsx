import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type SurveyorType } from '@/api/types'
import { Field } from '@/components/Field'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

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
 * Meniru fungsi form pada `Section/BrowseSuveryors-Section.xml` — satu isian deskripsi dan
 * tombol Simpan — dengan dua perbedaan yang disengaja: isiannya wajib diisi, dan nama yang
 * sudah dipakai ditolak (keputusan Work Owner, sejalan dengan Master Status Klaim
 * 2026-09-17).
 *
 * # Kode tidak lagi ditampilkan
 *
 * Layar Pega menampilkan kodenya sebagai isian read-only (`pyEditOptions=Read-only`). Itu
 * **tidak dibawa** (keputusan Work Owner 2026-10-03): nomornya diterbitkan sistem dan tidak
 * dapat disunting siapa pun, sehingga menampilkannya di form hanya menyita ruang tanpa
 * memberi pengguna satu pun hal yang dapat ia lakukan. Alasan lengkapnya ada di badan
 * komponen.
 */
export function SurveyorTypeForm({ surveyorType, onClose }: Props) {
  const save = useSaveSurveyorType()
  const editing = surveyorType !== null
  const firstField = useRef<HTMLInputElement | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { deskripsi: surveyorType?.deskripsi ?? '' },
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan ketik
  // harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    firstField.current?.focus()
  }, [])

  // Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di
  // kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya
  // berguna bila layar menyorotnya di tempat isiannya.
  useEffect(() => {
    if (!(save.error instanceof APIError)) return
    const violation = save.error.violations()['deskripsi']
    if (violation) setError('deskripsi', { type: 'server', message: violation })
  }, [save.error, setError])

  const { ref: refDescription, ...remainingDescription } = register('deskripsi')

  function send(values: FieldValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal.
    save.mutate(
      editing ? { kode: surveyorType.kode, deskripsi: values.deskripsi } : values,
      { onSuccess: onClose },
    )
  }

  return (
    /*
      Panel ini muncul di atas tabel, bukan sebagai dialog melayang.

      Alasannya praktis: pengguna sering perlu melihat tipe lain yang sudah ada untuk
      memastikan nama yang diketiknya tidak bertabrakan — dan dialog yang menutup layar
      justru menyembunyikan jawabannya. Garis aksen di tepi kiri menandai bahwa panel ini
      keadaan sementara, bukan bagian tetap halaman.
    */
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={editing ? 'Ubah tipe surveyor' : 'Tambah tipe surveyor'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {editing ? 'Ubah Tipe Surveyor' : 'Tambah Tipe Surveyor'}
        </h3>
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}

        {/*
          KODE TIDAK ditampilkan di form ini, baik saat menambah maupun saat mengubah —
          keputusan Work Owner 2026-10-03.

          Alasannya: nomornya diterbitkan sistem dan tidak dapat disunting siapa pun,
          sehingga menampilkannya di form hanya menyita ruang tanpa memberi pengguna satu
          pun hal yang dapat ia lakukan. Saat menambah ia belum ada sama sekali; saat
          mengubah ia tidak berubah.

          Kode tetap terlihat di KOLOM PERTAMA tabel, tempat ia memang berguna — untuk
          mengenali baris dan mencocokkannya dengan data surveyor yang menyimpannya.

          Bentuknya mengikuti Master Dominan Factor, Master Penyebab Kerugian, dan Master
          Status Klaim, yang dirapikan dengan keputusan yang sama.
        */}
        <Field
          id="deskripsi"
          label="Deskripsi"
          placeholder="Contoh: LOSS ADJUSTER"
          maxLength={MAX_DESCRIPTION_LENGTH}
          autoComplete="off"
          hint={`Paling panjang ${MAX_DESCRIPTION_LENGTH} karakter, dan belum dipakai tipe lain.`}
          error={errors.deskripsi?.message}
          disabled={save.isPending}
          {...remainingDescription}
          ref={(element) => {
            refDescription(element)
            firstField.current = element
          }}
        />

        <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
          <Button type="submit" tone="utama" disabled={save.isPending}>
            {save.isPending && <Spinner />}
            {save.isPending ? 'Menyimpan…' : 'Simpan'}
          </Button>
          <Button tone="halus" onClick={onClose} disabled={save.isPending}>
            Batal
          </Button>
        </div>
      </div>
    </form>
  )
}

/** Pemutar kecil pada tombol yang sedang bekerja. */
function Spinner() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" className="h-4 w-4 animate-spin">
      <circle cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" strokeOpacity="0.3" strokeWidth="2" />
      <path
        d="M8 1.5a6.5 6.5 0 0 1 6.5 6.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

/**
 * Galat simpan dibedakan menurut KODE-nya, bukan teks pesannya.
 *
 * Jenisnya menuntut tindak lanjut berbeda: nama yang bentrok dapat diperbaiki pengguna,
 * portal yang belum dipilih diperbaiki di bilah atas, dan gangguan sistem tidak dapat
 * ditolong dengan mencoba ulang berkali-kali.
 */
function SaveErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Perubahan belum tersimpan. Periksa koneksi lalu coba lagi."
        tone="gangguan"
      />
    )
  }

  if (!(error instanceof APIError)) {
    return (
      <ErrorMessage
        title="Gagal menyimpan"
        description="Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi."
        tone="gangguan"
      />
    )
  }

  const message = parse(error)
  if (message === null) return null
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

function parse(error: APIError): MessageContent | null {
  switch (error.kode) {
    case ErrorCode.surveyorTypeTaken:
      return {
        title: 'Nama tipe surveyor sudah dipakai',
        description: 'Sudah ada tipe dengan nama itu di entitas ini. Pakai nama lain.',
        tone: 'penolakan',
      }

    case ErrorCode.validationFailed:
      // Bila detailnya ada, isiannya sudah disorot di tempatnya; kotak pesan hanya akan
      // mengulang hal yang sama.
      return Object.keys(error.violations()).length > 0
        ? null
        : {
            title: 'Isian belum benar',
            description: error.message,
            tone: 'penolakan',
          }

    case ErrorCode.surveyorTypeNotFound:
      return {
        title: 'Tipe surveyor tidak ditemukan',
        description: 'Baris ini mungkin sudah diubah petugas lain. Muat ulang daftarnya.',
        tone: 'penolakan',
      }

    case ErrorCode.surveyorTypeCodeTaken:
      return {
        title: 'Kode bentrok',
        description: 'Kode yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.',
        tone: 'gangguan',
      }

    case ErrorCode.surveyorTypeCodeUnavailable:
      return {
        title: 'Kode tidak dapat dibentuk',
        description:
          'Basis data entitas ini belum siap menerbitkan kode baru. Mengulang tidak akan menolong — hubungi administrator Claim PNC.',
        tone: 'gangguan',
      }

    case ErrorCode.portalNotStated:
    case ErrorCode.portalUnknown:
      return {
        title: 'Portal entitas belum dipilih',
        description: 'Pilih portal entitas di bilah atas halaman, lalu simpan lagi.',
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
        title: 'Gagal menyimpan',
        description: error.message,
        tone: 'gangguan',
      }
  }
}
