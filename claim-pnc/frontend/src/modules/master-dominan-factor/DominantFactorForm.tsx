import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type DominantFactor } from '@/api/types'
import { Field } from '@/components/Field'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

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
export function DominantFactorForm({ factor, tutup }: Props) {
  const save = useSaveDominantFactor()
  const editing = factor !== null
  const firstField = useRef<HTMLInputElement | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { nama: factor?.nama ?? '' },
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan
  // ketik harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    firstField.current?.focus()
  }, [])

  const { ref: refNama, ...remainingNama } = register('nama')

  function send(values: FieldValues) {
    save.mutate(
      editing ? { id: factor.id, nama: values.nama } : { nama: values.nama },
      { onSuccess: tutup },
    )
  }

  return (
    /*
      Panel ini muncul di atas tabel, bukan sebagai dialog melayang.

      Alasannya praktis, dan di modul ini lebih kuat daripada di modul master lain:
      keterangan GANDA diterima, sehingga pengguna sering perlu melihat daftar yang ada
      untuk memastikan ia tidak sedang mengetik ulang faktor yang sudah terdaftar — dan
      dialog yang menutup layar justru menyembunyikan jawabannya.
    */
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={editing ? 'Memperbaharui Data faktor dominan' : 'Menambah Data faktor dominan'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {editing ? 'Memperbaharui Data' : 'Menambah Data'}
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          {editing
            ? 'Hanya keterangan yang dapat diubah. ID tetap, karena klaim lama menyimpannya.'
            : 'ID dibuat sistem setelah disimpan, melanjutkan nomor terakhir.'}
        </p>
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}

        {/* Peringatan ini menggantikan pemeriksaan yang sengaja tidak ada. Karena
            keterangan ganda diterima, pengguna tidak akan pernah ditolak sistem — jadi
            satu-satunya cara ia tahu adalah diberi tahu. */}
        {!editing && (
          <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2.5 text-xs leading-relaxed text-amber-900">
            Keterangan yang sama boleh terdaftar lebih dari satu kali — sistem tidak
            menolaknya. Periksa daftar di bawah lebih dulu agar tidak terdaftar ganda.
          </p>
        )}

        <div className="grid gap-5 sm:grid-cols-2">
          <div>
            <span className="block text-sm font-medium text-slate-700">ID</span>
            {/*
              ID digambar sebagai kotak mati, bukan input ber-`disabled`. Input yang
              dinonaktifkan tetap terlihat seperti isian dan mengundang pengguna
              mengkliknya; kotak ini jelas bukan tempat mengetik.
            */}
            <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 font-mono text-sm text-slate-500">
              {factor?.id ?? 'Dibuat sistem'}
            </p>
            <p className="mt-1.5 text-xs text-slate-500">
              ID tidak dapat disunting, sama seperti di sistem lama.
            </p>
          </div>

          <Field
            id="nama"
            label="Keterangan"
            placeholder="Contoh: Kelalaian pihak ketiga"
            maxLength={MAX_NAME_LENGTH}
            autoComplete="off"
            hint={`Paling panjang ${MAX_NAME_LENGTH} karakter.`}
            error={errors.nama?.message}
            disabled={save.isPending}
            {...remainingNama}
            ref={(element) => {
              refNama(element)
              firstField.current = element
            }}
          />
        </div>

        <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
          <Button type="submit" tone="utama" disabled={save.isPending}>
            {save.isPending && <Spinner />}
            {save.isPending ? 'Menyimpan…' : 'Simpan'}
          </Button>
          <Button tone="halus" onClick={tutup} disabled={save.isPending}>
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
 * Tiga jenis menuntut tindak lanjut berbeda: nomor yang bentrok cukup dicoba ulang,
 * validasi server menunjuk kolom tertentu, dan gangguan sistem tidak dapat ditolong
 * dengan mencoba ulang berkali-kali.
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

  const { title, description, tone } = parse(error)
  return <ErrorMessage title={title} description={description} tone={tone} />
}

function parse(error: APIError): { title: string; description: string; tone: ErrorTone } {
  switch (error.kode) {
    case ErrorCode.validationFailed:
      return {
        title: 'Isian belum benar',
        // Pesan dari server dipakai apa adanya: ia yang tahu aturan mana yang dilanggar,
        // dan menerjemahkannya ulang di sini akan membuat keduanya dapat berbeda.
        description: error.detail.map((d) => d.pesan).join(' ') || error.message,
        tone: 'penolakan',
      }

    case ErrorCode.dominantFactorNotFound:
      return {
        title: 'Faktor dominan tidak ditemukan',
        description: 'Baris ini mungkin sudah diubah orang lain. Muat ulang daftarnya.',
        tone: 'penolakan',
      }

    case ErrorCode.dominantFactorIDTaken:
      return {
        title: 'Nomor bentrok',
        description:
          'Nomor yang dibuat sistem sedang dipakai penyimpanan lain. Coba simpan sekali lagi.',
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
