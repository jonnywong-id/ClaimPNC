import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'

import { useIssueVirtualAccount } from './api'

/** Batas panjang; sama dengan masterrecovery.Max* di backend. */
const MAX_CLIENT_ID = 100
const MAX_PRINCIPAL_NAME = 200
const MAX_EMAIL = 100

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Duplikasi yang DISENGAJA. Yang di sini menjawab pengguna tanpa perjalanan jaringan; yang
 * di sana adalah yang menegakkan — karena pemanggilan langsung ke API tidak melewati layar
 * ini sama sekali.
 *
 * Ketiganya wajib. Di layar Pega hanya Email yang ditandai wajib, tetapi Client ID dan Nama
 * Principal keduanya dipakai sebagai KUNCI pencarian VA yang sudah ada — salah satu yang
 * kosong membuat pencocokan itu mencocokkan hal yang salah, dan akibatnya VA ganda untuk
 * principal yang sama.
 */
const schema = z.object({
  client_id: z
    .string()
    .trim()
    .min(1, 'Client ID wajib diisi.')
    .max(MAX_CLIENT_ID, `Client ID paling panjang ${MAX_CLIENT_ID} karakter.`),
  nama_principal: z
    .string()
    .trim()
    .min(1, 'Nama principal wajib diisi.')
    .max(MAX_PRINCIPAL_NAME, `Nama principal paling panjang ${MAX_PRINCIPAL_NAME} karakter.`),
  email_inputor_va: z
    .string()
    .trim()
    .min(1, 'Email inputor VA wajib diisi.')
    .max(MAX_EMAIL, `Email inputor VA paling panjang ${MAX_EMAIL} karakter.`)
    // Pemeriksaan sengaja dangkal, sama dengan server. Satu-satunya cara membuktikan
    // sebuah alamat surel benar adalah mengirim surat ke sana; pola yang rumit hanya
    // menolak alamat sah yang tidak umum.
    .refine((value) => /^[^@\s]+@[^@\s]+$/.test(value), 'Email inputor VA belum berupa alamat surel.'),
})

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Dipanggil setelah nomor terbit, supaya form utama dapat memakainya langsung. */
  onIssued: (value: { client_id: string; nama_principal: string; nomor: string }) => void
  onClose: () => void
}

/**
 * Panel penerbitan Virtual Account.
 *
 * Menggantikan `Section/DetailMasterRecovery-Section.xml` beserta panel "Generated New VA"
 * pada harness — tiga isian, satu tombol, dan dua kotak hasil yang tidak dapat disunting.
 *
 * # Yang sengaja dibuat berbeda dari layar lama
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | "Sudah punya VA" | disiratkan lewat kalimat di dalam pesan | penanda tersendiri, warnanya berbeda |
 * | Client ID kosong | diterima | ditolak — ia kunci pencarian VA |
 * | Galat layanan | satu pesan untuk semua sebab | dibedakan: belum terdaftar · tidak terhubung · ditolak |
 */
export function VirtualAccountPanel({ onIssued, onClose }: Props) {
  const issue = useIssueVirtualAccount()

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: { client_id: '', nama_principal: '', email_inputor_va: '' },
  })

  // Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di
  // kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya
  // berguna bila layar menyorotnya di tempat isiannya.
  useEffect(() => {
    if (!(issue.error instanceof APIError)) return
    const violation = issue.error.violations()
    for (const name of ['client_id', 'nama_principal', 'email_inputor_va'] as const) {
      const message = violation[name]
      if (message) setError(name, { type: 'server', message })
    }
  }, [issue.error, setError])

  function send(values: FieldValues) {
    issue.mutate(values, {
      onSuccess: (answer) => {
        onIssued({
          client_id: values.client_id,
          nama_principal: values.nama_principal,
          nomor: answer.nomor_virtual_account,
        })
      },
    })
  }

  const issued = issue.data

  return (
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-emerald-500 bg-white shadow-angkat"
      aria-label="Terbitkan virtual account"
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">Terbitkan Virtual Account</h3>
        <p className="mt-1 text-sm text-slate-600">
          Nomor rekening virtual tempat principal mengembalikan dana. Bila principal ini
          sudah punya, nomor yang lama yang dipakai — bukan diterbitkan yang baru.
        </p>
      </div>

      <div className="space-y-5 p-5">
        {issue.isError && <IssueErrorMessage error={issue.error} />}

        {issued && (
          /*
            Hasil ditampilkan dengan warna yang BERBEDA menurut asalnya, bukan hanya
            dengan kalimat. Petugas perlu tahu seketika apakah ia baru saja menerbitkan
            rekening baru atau memakai yang sudah ada — keduanya menuntut tindak lanjut
            yang berbeda terhadap principal.
          */
          <div
            className={
              'rounded-kartu border p-4 ' +
              (issued.dipakai_ulang
                ? 'border-blue-200 bg-blue-50'
                : 'border-emerald-200 bg-emerald-50')
            }
          >
            <p
              className={
                'text-sm font-medium ' +
                (issued.dipakai_ulang ? 'text-blue-900' : 'text-emerald-900')
              }
            >
              {issued.dipakai_ulang
                ? 'Principal ini sudah punya virtual account'
                : 'Virtual account berhasil diterbitkan'}
            </p>
            <p className="mt-2 font-mono text-lg font-semibold tracking-wide text-slate-900">
              {issued.nomor_virtual_account}
            </p>
            {issued.pesan && <p className="mt-1 text-xs text-slate-600">{issued.pesan}</p>}
          </div>
        )}

        <div className="grid gap-5 sm:grid-cols-2">
          <Field
            id="va-client-id"
            label="Client ID"
            placeholder="Contoh: ASM-PRINCIPAL-001"
            maxLength={MAX_CLIENT_ID}
            autoComplete="off"
            hint="Dipakai sebagai kunci pencarian VA yang sudah ada."
            error={errors.client_id?.message}
            disabled={issue.isPending}
            {...register('client_id')}
          />
          <Field
            id="va-nama-principal"
            label="Nama Principal"
            placeholder="Contoh: PT CONTOH PENJAMINAN"
            maxLength={MAX_PRINCIPAL_NAME}
            autoComplete="off"
            hint="Huruf besar-kecil tidak dibedakan saat dicocokkan."
            error={errors.nama_principal?.message}
            disabled={issue.isPending}
            {...register('nama_principal')}
          />
        </div>

        <Field
          id="va-email"
          label="Email Inputor VA"
          type="email"
          placeholder="nama.petugas@sinarmas.co.id"
          maxLength={MAX_EMAIL}
          autoComplete="off"
          hint="Surel petugas yang menerbitkan; tercatat bersama nomor VA."
          error={errors.email_inputor_va?.message}
          disabled={issue.isPending}
          {...register('email_inputor_va')}
        />

        <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
          <Button type="submit" tone="utama" disabled={issue.isPending}>
            {issue.isPending ? 'Memproses…' : 'Terbitkan VA'}
          </Button>
          <Button tone="halus" onClick={onClose} disabled={issue.isPending}>
            Tutup
          </Button>
        </div>
      </div>
    </form>
  )
}

/**
 * Galat penerbitan dibedakan menurut KODE-nya, bukan teks pesannya.
 *
 * Ketiganya menuntut tindak lanjut yang benar-benar berbeda, dan menyatukannya menjadi satu
 * pesan — seperti sistem lama — membuat petugas mengulang permintaan yang tidak akan pernah
 * berhasil.
 */
function IssueErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Virtual account belum diterbitkan. Periksa koneksi lalu coba lagi."
        tone="gangguan"
      />
    )
  }
  if (!(error instanceof APIError)) {
    return (
      <ErrorMessage
        title="Gagal menerbitkan virtual account"
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
    case ErrorCode.validationFailed:
      // Bila detailnya ada, isiannya sudah disorot di tempatnya; kotak pesan hanya akan
      // mengulang hal yang sama.
      return Object.keys(error.violations()).length > 0
        ? null
        : { title: 'Isian belum benar', description: error.message, tone: 'penolakan' }

    case ErrorCode.vaIssuerUnconfigured:
      return {
        title: 'Layanan penerbit VA belum terdaftar',
        description:
          'Mengulang tidak akan menolong — alamat layanannya belum diisi untuk entitas ini. Hubungi administrator Claim PNC.',
        tone: 'gangguan',
      }

    case ErrorCode.vaIssuerUnreachable:
      return {
        title: 'Layanan penerbit VA sedang tidak dapat dihubungi',
        description: 'Gangguan sementara di sistem penerbit. Coba lagi beberapa saat.',
        tone: 'gangguan',
      }

    case ErrorCode.vaIssuerRejected:
      return {
        title: 'Permintaan ditolak layanan penerbit',
        description:
          'Layanan menjawab tetapi menolak menerbitkan nomor. Periksa Client ID dan nama principal, lalu coba lagi.',
        tone: 'penolakan',
      }

    case ErrorCode.portalNotStated:
    case ErrorCode.portalUnknown:
      return {
        title: 'Portal entitas belum dipilih',
        description: 'Pilih portal entitas di bilah atas halaman, lalu coba lagi.',
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
        title: 'Gagal menerbitkan virtual account',
        description: error.message,
        tone: 'gangguan',
      }
  }
}
