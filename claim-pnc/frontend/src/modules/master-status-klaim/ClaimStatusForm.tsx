import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type ClaimStatus } from '@/api/types'
import { Field } from '@/components/Field'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

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
export function ClaimStatusForm({ status, tutup }: Props) {
  const save = useSaveClaimStatus()
  const editing = status !== null
  const firstField = useRef<HTMLInputElement | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { label: status?.label ?? '' },
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan
  // ketik harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    firstField.current?.focus()
  }, [])

  const { ref: refLabel, ...remainingLabel } = register('label')

  function send(values: FieldValues) {
    save.mutate(
      editing ? { kode: status.kode, label: values.label } : { label: values.label },
      { onSuccess: tutup },
    )
  }

  return (
    /*
      Panel ini muncul di atas tabel, bukan sebagai dialog melayang.

      Alasannya praktis: pengguna sering perlu melihat status lain yang sudah ada untuk
      memastikan nama yang diketiknya tidak bertabrakan — dan dialog yang menutup layar
      justru menyembunyikan jawabannya. Garis aksen di tepi kiri menandai bahwa panel ini
      keadaan sementara, bukan bagian tetap halaman.
    */
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={editing ? 'Ubah status klaim' : 'Tambah status klaim'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {editing ? 'Ubah Status Klaim' : 'Tambah Status Klaim'}
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          {editing
            ? 'Hanya nama status yang dapat diubah. Kode tetap, karena klaim lama menyimpannya.'
            : 'Kode dibuat sistem setelah disimpan, melanjutkan nomor terakhir.'}
        </p>
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}

        <div className="grid gap-5 sm:grid-cols-2">
          <div>
            <span className="block text-sm font-medium text-slate-700">Kode</span>
            {/*
              Kode digambar sebagai kotak mati, bukan input ber-`disabled`. Input yang
              dinonaktifkan tetap terlihat seperti isian dan mengundang pengguna
              mengkliknya; kotak ini jelas bukan tempat mengetik.
            */}
            <p className="mt-1.5 flex items-center rounded-kontrol border border-dashed border-slate-300 bg-slate-50 px-3 py-2.5 font-mono text-sm text-slate-500">
              {status?.kode ?? 'Dibuat sistem'}
            </p>
            <p className="mt-1.5 text-xs text-slate-500">
              Kode tidak dapat disunting, sama seperti di sistem lama.
            </p>
          </div>

          <Field
            id="label"
            label="Status"
            placeholder="Contoh: Reopen Claim"
            maxLength={MAX_LABEL_LENGTH}
            autoComplete="off"
            petunjuk={`Paling panjang ${MAX_LABEL_LENGTH} karakter, dan belum dipakai status lain.`}
            error={errors.label?.message}
            disabled={save.isPending}
            {...remainingLabel}
            ref={(elemen) => {
              refLabel(elemen)
              firstField.current = elemen
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
 * Tiga jenis menuntut tindak lanjut berbeda: nama yang bentrok dapat diperbaiki
 * pengguna, validasi server menunjuk kolom tertentu, dan gangguan sistem tidak dapat
 * ditolong dengan mencoba ulang berkali-kali.
 */
function SaveErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        judul="Tidak dapat menghubungi server"
        keterangan="Perubahan belum tersimpan. Periksa koneksi lalu coba lagi."
        tone="gangguan"
      />
    )
  }

  if (!(error instanceof APIError)) {
    return (
      <ErrorMessage
        judul="Gagal menyimpan"
        keterangan="Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi."
        tone="gangguan"
      />
    )
  }

  const { judul, keterangan, tone } = parse(error)
  return <ErrorMessage judul={judul} keterangan={keterangan} tone={tone} />
}

function parse(error: APIError): { judul: string; keterangan: string; tone: ErrorTone } {
  switch (error.kode) {
    case ErrorCode.statusLabelTaken:
      return {
        judul: 'Nama status sudah dipakai',
        keterangan: 'Sudah ada status dengan nama itu. Pakai nama lain.',
        tone: 'penolakan',
      }

    case ErrorCode.validationFailed:
      return {
        judul: 'Isian belum benar',
        // Pesan dari server dipakai apa adanya: ia yang tahu aturan mana yang dilanggar,
        // dan menerjemahkannya ulang di sini akan membuat keduanya dapat berbeda.
        keterangan: error.detail.map((d) => d.pesan).join(' ') || error.message,
        tone: 'penolakan',
      }

    case ErrorCode.statusKlaimTidakDitemukan:
      return {
        judul: 'Status tidak ditemukan',
        keterangan: 'Baris ini mungkin sudah diubah orang lain. Muat ulang daftarnya.',
        tone: 'penolakan',
      }

    case ErrorCode.statusCodeTaken:
      return {
        judul: 'Kode bentrok',
        keterangan: 'Kode yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.',
        tone: 'gangguan',
      }

    default:
      return {
        judul: 'Gagal menyimpan',
        keterangan: error.message,
        tone: 'gangguan',
      }
  }
}
