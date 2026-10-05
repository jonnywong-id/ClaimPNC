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
 * Meniru fungsi form `ListStatusClaim` di Pega — satu isian Status dan tombol Simpan —
 * dengan dua perbedaan yang disengaja: isiannya wajib diisi, dan nama yang sudah dipakai
 * ditolak (keputusan Work Owner 2026-09-17).
 *
 * # Kode tidak lagi ditampilkan
 *
 * Layar Pega menampilkan `LSC_ID` sebagai isian read-only. Itu **tidak dibawa** (keputusan
 * Work Owner 2026-10-03): nomornya diterbitkan sistem dan tidak dapat disunting siapa pun,
 * sehingga menampilkannya di form hanya menyita ruang tanpa memberi pengguna satu pun hal
 * yang dapat ia lakukan. Alasan lengkapnya ada di badan komponen.
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
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}

        {/*
          KODE TIDAK ditampilkan di form ini, baik saat menambah maupun saat mengubah —
          keputusan Work Owner 2026-10-03.

          Alasannya: nomornya diterbitkan sistem — kode situs ditambah urutan
          `M_STS_CLAIM_SEQ` — dan tidak dapat disunting siapa pun, sehingga menampilkannya
          di form hanya menyita ruang tanpa memberi pengguna satu pun hal yang dapat ia
          lakukan. Saat menambah ia belum ada sama sekali; saat mengubah ia tidak berubah.

          Kode tetap terlihat di KOLOM PERTAMA tabel, tempat ia memang berguna — untuk
          mengenali baris dan mencocokkannya dengan klaim yang menyimpan kode itu.

          Bentuknya mengikuti Master Dominan Factor dan Master Penyebab Kerugian, yang
          dirapikan dengan keputusan yang sama.
        */}
        <Field
          id="label"
          label="Status"
          placeholder="Contoh: Reopen Claim"
          maxLength={MAX_LABEL_LENGTH}
          autoComplete="off"
          hint={`Paling panjang ${MAX_LABEL_LENGTH} karakter, dan belum dipakai status lain.`}
          error={errors.label?.message}
          disabled={save.isPending}
          {...remainingLabel}
          ref={(elemen) => {
            refLabel(elemen)
            firstField.current = elemen
          }}
        />

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
    case ErrorCode.statusLabelTaken:
      return {
        title: 'Nama status sudah dipakai',
        description: 'Sudah ada status dengan nama itu. Pakai nama lain.',
        tone: 'penolakan',
      }

    case ErrorCode.validationFailed:
      return {
        title: 'Isian belum benar',
        // Pesan dari server dipakai apa adanya: ia yang tahu aturan mana yang dilanggar,
        // dan menerjemahkannya ulang di sini akan membuat keduanya dapat berbeda.
        description: error.detail.map((d) => d.pesan).join(' ') || error.message,
        tone: 'penolakan',
      }

    case ErrorCode.claimStatusNotFound:
      return {
        title: 'Status tidak ditemukan',
        description: 'Baris ini mungkin sudah diubah orang lain. Muat ulang daftarnya.',
        tone: 'penolakan',
      }

    case ErrorCode.statusCodeTaken:
      return {
        title: 'Kode bentrok',
        description: 'Kode yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.',
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
