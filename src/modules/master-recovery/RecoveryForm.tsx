import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  type Recovery,
  type RecoveryClaimLine,
  type RecoveryPrincipal,
} from '@/api/types'
import { Button } from '@/components/Button'
import { ComboField } from '@/components/ComboField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { formatMoney, parseMoney, remainder } from '@/lib/money'

import { ClaimLineUpload } from './ClaimLineUpload'
import { VirtualAccountPanel } from './VirtualAccountPanel'
import { useSaveRecovery } from './api'

/** Batas panjang; sama dengan masterrecovery.Max* di backend. */
const MAX_PRINCIPAL_NAME = 200
const MAX_CLIENT_ID = 100
const MAX_VIRTUAL_ACCOUNT = 1000
const MAX_REMARK = 2000
const MAX_CASE_POSITION = 100
const MAX_POLICY_NO = 200

/**
 * money adalah pembaca isian nilai uang.
 *
 * Ia menerima TEKS, bukan angka, karena isiannya memang kotak teks: petugas mengetik
 * "1.000.000" dengan pemisah ribuan, dan `<input type="number">` menolaknya. Yang
 * ditegakkan sama dengan server — rupiah utuh, tidak boleh negatif.
 */
function money(label: string) {
  return z
    .string()
    .trim()
    .transform((value, ctx) => {
      if (value === '') return 0
      const parsed = parseMoney(value)
      if (parsed === null) {
        ctx.addIssue({ code: 'custom', message: `${label} harus berupa angka rupiah utuh.` })
        return z.NEVER
      }
      if (parsed < 0) {
        ctx.addIssue({ code: 'custom', message: `${label} tidak boleh negatif.` })
        return z.NEVER
      }
      return parsed
    })
}

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan langsung
 * ke API tidak melewati layar ini sama sekali.
 *
 * Keenam isian wajibnya diambil dari prasyarat penolakan di
 * `Activity/Insert_mst_recoveryKlaimASM-Act.xml:802`. Sisa TIDAK ada di sini karena ia
 * DIHITUNG — layar memperlihatkannya, server menetapkannya.
 */
const schema = z.object({
  nama_principal: z
    .string()
    .trim()
    .min(1, 'Nama principal wajib diisi.')
    .max(MAX_PRINCIPAL_NAME, `Nama principal paling panjang ${MAX_PRINCIPAL_NAME} karakter.`),
  client_id: z.string().trim().max(MAX_CLIENT_ID, `Client ID paling panjang ${MAX_CLIENT_ID} karakter.`),
  nomor_virtual_account: z
    .string()
    .trim()
    .max(MAX_VIRTUAL_ACCOUNT, `Nomor virtual account paling panjang ${MAX_VIRTUAL_ACCOUNT} karakter.`),
  tahun: z.string().trim().min(1, 'Tahun wajib dipilih.'),
  nilai_klaim: money('Nilai klaim'),
  pembayaran_sebelumnya: money('Nilai pembayaran sebelumnya'),
  pembayaran: money('Pembayaran'),
  keterangan: z
    .string()
    .trim()
    .min(1, 'Keterangan wajib diisi.')
    .max(MAX_REMARK, `Keterangan paling panjang ${MAX_REMARK} karakter.`),
  posisi_kasus: z
    .string()
    .trim()
    .min(1, 'Posisi kasus wajib diisi.')
    .max(MAX_CASE_POSITION, `Posisi kasus paling panjang ${MAX_CASE_POSITION} karakter.`),
  nomor_polis: z
    .string()
    .trim()
    .max(MAX_POLICY_NO, `Nomor polis paling panjang ${MAX_POLICY_NO} karakter.`),
})

type FieldValues = z.input<typeof schema>
type ParsedValues = z.output<typeof schema>

type Props = {
  /** Nomor batch PERKIRAAN, untuk ditampilkan saja. */
  nextBatch: number | undefined
  year: string[]
  principal: RecoveryPrincipal[]
  onSaved: (saved: Recovery, policyResolved: boolean) => void
}

/**
 * Form entri batch recovery.
 *
 * Menggantikan `Section/OutstandingMasterRecovery-Section.xml` — sembilan isian, tombol
 * unggah bukti bayar, panel data klaim, dan tombol **Transfer Recovery**.
 *
 * # Yang sengaja dibuat berbeda dari layar lama
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Isian kurang | satu pesan "Wajib ISI semua field" | tiap kolom ditandai di tempatnya |
 * | Sisa | isian biasa yang dapat disunting | dihitung, ditampilkan, tidak dapat diketik |
 * | Nomor polis salah | baru ketahuan setelah tersimpan | dicari saat ditekan Cari |
 * | Nilai negatif | diterima | ditolak |
 * | Judul kolom | nama properti Pega | nama yang dibaca manusia (`D-19`) |
 * | Layar sempit | digulir menyamping | tersusun satu kolom (`D-12`) |
 */
export function RecoveryForm({ year, principal, onSaved }: Props) {
  const save = useSaveRecovery()

  const [claimLine, setClaimLine] = useState<RecoveryClaimLine[]>([])

  const [vaIssued, setVAIssued] = useState(false)
  const [paymentProof, setPaymentProof] = useState<{ id: string; nama: string } | null>(null)

  const {
    register,
    handleSubmit,
    setValue,
    control,
    reset,
    setError,
    formState: { errors },
  } = useForm<FieldValues, unknown, ParsedValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama_principal: '',
      client_id: '',
      nomor_virtual_account: '',
      tahun: year[0] ?? '',
      nilai_klaim: '',
      pembayaran_sebelumnya: '',
      pembayaran: '',
      keterangan: '',
      posisi_kasus: '',
      nomor_polis: '',
    },
  })

  // Sisa dihitung ulang pada setiap ketikan, meniru aksi `HitungSisaKlaimRecovery` yang di
  // layar lama dijalankan pada perubahan ketiga isian nilai.
  const watched = useWatch({
    control,
    name: ['nilai_klaim', 'pembayaran_sebelumnya', 'pembayaran'],
  })
  const sisa = useMemo(() => {
    const claimAmount = parseMoney(watched[0] ?? '') ?? 0
    const previousPayment = parseMoney(watched[1] ?? '') ?? 0
    const payment = parseMoney(watched[2] ?? '') ?? 0
    return remainder(claimAmount, previousPayment, payment)
  }, [watched])

  const year0 = year[0]
  const chosenYear = useWatch({ control, name: 'tahun' })

  // Tahun terbaru dipilih lebih dulu — SETELAH daftarnya tiba.
  //
  // Ini perbaikan atas cacat yang nyata, bukan kerapian: `defaultValues` dibaca SEKALI saat
  // form pertama dirakit, dan pada saat itu daftar tahun masih kosong karena permintaannya
  // belum dijawab. Tanpa efek ini, isian Tahun tetap kosong meski daftarnya sudah terisi,
  // dan petugas harus memilihnya sendiri setiap kali membuka layar — atau menekan Simpan
  // lalu ditolak "Tahun wajib dipilih".
  //
  // Hanya mengisi yang MASIH KOSONG, sehingga pilihan yang sudah diubah petugas tidak
  // ditimpa ketika daftarnya dimuat ulang.
  useEffect(() => {
    if (year0 && !chosenYear) setValue('tahun', year0)
  }, [year0, chosenYear, setValue])

  // Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di
  // kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya
  // berguna bila layar menyorotnya di tempat isiannya.
  useEffect(() => {
    if (!(save.error instanceof APIError)) return
    const violation = save.error.violations()
    for (const name of [
      'nama_principal',
      'client_id',
      'nomor_virtual_account',
      'tahun',
      'nilai_klaim',
      'pembayaran_sebelumnya',
      'pembayaran',
      'keterangan',
      'posisi_kasus',
      'nomor_polis',
    ] as const) {
      const message = violation[name]
      if (message) setError(name, { type: 'server', message })
    }
  }, [save.error, setError])

  /**
   * Nomor yang baru terbit langsung mengisi ketiga isian principal.
   *
   * Itu yang membuat blok "Generated New VA" berguna di dalam alur Tambah: petugas tidak
   * perlu menyalin nomornya sendiri ke isian di bawah.
   */
  function onVAIssued(value: { client_id: string; nama_principal: string; nomor: string }) {
    // Padanan Property-Set `TempRecovery.IBNR := "1"` pada activity Get VA.
    setVAIssued(true)
    setValue('nama_principal', value.nama_principal, { shouldValidate: true })
    setValue('client_id', value.client_id, { shouldValidate: true })
    setValue('nomor_virtual_account', value.nomor, { shouldValidate: true })
  }

  function send(values: ParsedValues) {
    save.mutate(
      {
        ...values,
        id_dokumen: paymentProof?.id ?? '',
        // Kolom NOHPLL. Namanya menyiratkan nomor telepon; isinya di sistem lama adalah
        // nomor catatan log layanan, dan layar ini tidak punya nomor semacam itu —
        // dikirim kosong, bukan diisi nilai karangan.
        id_log_layanan: '',
        baris_klaim: claimLine,
      },
      {
        onSuccess: (answer) => {
          onSaved(answer.recovery, answer.identitas_polis_terisi)
          // Form dikosongkan HANYA setelah server menjawab berhasil. Mengosongkannya lebih
          // dulu akan membuang isian petugas saat penyimpanan gagal.
          reset()
          setClaimLine([])
          setPaymentProof(null)
        },
      },
    )
  }

  const busy = save.isPending

  return (
    <form onSubmit={handleSubmit(send)} noValidate className="space-y-6" aria-label="Catat batch recovery">
      {save.isError && <SaveErrorMessage error={save.error} />}

      {/*
        Upload Data Klaim berada PALING ATAS, bukan di bawah seperti sebelumnya.

        Itu susunan layar lama: menekan Tambah memunculkan kotak "Tambah Data" yang
        isinya tombol **Upload Data Klaim** beserta grid "No Polis | Nilai Klaims" —
        isian lainnya menyusul di bawahnya (`D-13`).
      */}
      <ClaimLineUpload claimLine={claimLine} onChange={setClaimLine} disabled={busy} />

      {/*
        SISA FORM BARU MUNCUL SETELAH DATA KLAIM DIUNGGAH.

        Itu alur layar lama: menekan Tambah hanya memunculkan kotak "Tambah Data"
        berisi tombol Upload Data Klaim dan grid "No Polis | Nilai Klaims" — tidak ada
        satu pun isian lain di layar sampai data klaimnya masuk (`D-13`).

        Gerbangnya BARIS KLAIM, bukan "tombol unggah pernah ditekan": berkas yang
        seluruh barisnya ditolak tidak membuka apa pun, dan petugas melihat daftar
        baris cacatnya lebih dulu — bukan form panjang yang menutupi pesan itu.
      */}
      {claimLine.length > 0 && (
        <>
          {/*
            "Generated New VA" duduk TEPAT SESUDAH grid data klaim, sama seperti layar
            lama — bukan sebagai tombol terpisah di kepala halaman.
          */}
          <VirtualAccountPanel onIssued={onVAIssued} />


          {/*
            KESEPULUH ISIAN DI BAWAH BARU MUNCUL SETELAH VA TERBIT.

            Bukan tafsiran: container yang memuatnya di layar lama bersyarat
            `TempRecovery.IBNR == 1`
            (`Section/OutstandingMasterRecovery-Section.xml:9173`), dan nilai itu di-set
            oleh activity tombol Get VA — `Activity/GeneratedVAClaimRecovery-Act.xml`
            menjalankan Property-Set `TempRecovery.IBNR := "1"`.

            Kolom IBNR dipakai di sana sebagai PENANDA langkah, bukan sebagai nilai IBNR
            dalam arti asuransi. Di sini penandanya adalah nomor VA yang sudah terbit,
            karena itulah yang sebenarnya dihasilkan langkah tersebut.

            Urutan dan nama labelnya disalin apa adanya dari urutan kemunculan
            `pyLabelFieldValue` pada section yang sama:

                :9809  Nama Principal              autocomplete
                :11154 Tahun                       dropdown
                :11717 Client ID
                :11893 Virtual Account Number
                :12023 Nilai Klaim
                :12902 Nilai Pembayaran Sebelumnya
                :13256 Pembayaran
                :13573 Sisa
                :13786 Keterangan
                :14055 Posisi Kasus

            Tidak ada pengelompokan "Principal" / "Nilai" / "Keterangan" di layar lama —
            isiannya satu runtun. Pengelompokan itu buatan saya dan dicabut (`D-13`).
          */}
          {vaIssued && (
            <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
              <div className="space-y-5 p-5">
                {/*
                  Autocomplete, bukan dropdown terpisah: section memuat lima
                  `pyUIElement: autocomplete` dan HANYA SATU `dropdown` — dan yang satu itu
                  Tahun. Isian "Pilih dari master" buatan saya karena itu dicabut; daftar
                  principal menjadi saran di sini, dan nilai di luar daftar tetap boleh
                  diketik.
                */}
                <ComboField
                  id="nama-principal"
                  label="Nama Principal"
                  options={principal.map((p) => p.nama_principal)}
                  maxLength={MAX_PRINCIPAL_NAME}
                  autoComplete="off"
                  error={errors.nama_principal?.message}
                  disabled={busy}
                  {...register('nama_principal')}
                />

                <SelectField
                  id="tahun"
                  label="Tahun"
                  options={year.map((y) => ({ value: y, label: y }))}
                  error={errors.tahun?.message}
                  disabled={busy}
                  {...register('tahun')}
                />

                <div className="grid gap-5 sm:grid-cols-2">
                  <Field
                    id="client-id"
                    label="Client ID"
                    maxLength={MAX_CLIENT_ID}
                    autoComplete="off"
                    error={errors.client_id?.message}
                    disabled={busy}
                    {...register('client_id')}
                  />
                  <Field
                    id="nomor-virtual-account"
                    label="Virtual Account Number"
                    maxLength={MAX_VIRTUAL_ACCOUNT}
                    autoComplete="off"
                    error={errors.nomor_virtual_account?.message}
                    disabled={busy}
                    {...register('nomor_virtual_account')}
                  />
                </div>

                <div className="grid gap-5 sm:grid-cols-2">
                  <Field
                    id="nilai-klaim"
                    label="Nilai Klaim"
                    inputMode="numeric"
                    autoComplete="off"
                    error={errors.nilai_klaim?.message}
                    disabled={busy}
                    {...register('nilai_klaim')}
                  />
                  <Field
                    id="pembayaran-sebelumnya"
                    label="Nilai Pembayaran Sebelumnya"
                    inputMode="numeric"
                    autoComplete="off"
                    error={errors.pembayaran_sebelumnya?.message}
                    disabled={busy}
                    {...register('pembayaran_sebelumnya')}
                  />
                </div>

                <div className="grid gap-5 sm:grid-cols-2">
                  <Field
                    id="pembayaran"
                    label="Pembayaran"
                    inputMode="numeric"
                    autoComplete="off"
                    error={errors.pembayaran?.message}
                    disabled={busy}
                    {...register('pembayaran')}
                  />

                  {/*
                    Sisa adalah ISIAN di layar lama (`:13573`), bukan keterangan di pojok —
                    tetapi ia hasil hitungan dan tidak diketik siapa pun. Digambar terkunci
                    supaya sejajar dengan isian di sebelahnya tanpa mengundang suntingan.
                  */}
                  <Field
                    id="sisa"
                    label="Sisa"
                    value={formatMoney(sisa)}
                    readOnly
                    disabled={busy}
                    hint="Dihitung sendiri; angka yang mengikat datang dari server."
                  />
                </div>

                <Field
                  id="keterangan"
                  label="Keterangan"
                  maxLength={MAX_REMARK}
                  autoComplete="off"
                  error={errors.keterangan?.message}
                  disabled={busy}
                  {...register('keterangan')}
                />

                <Field
                  id="posisi-kasus"
                  label="Posisi Kasus"
                  maxLength={MAX_CASE_POSITION}
                  autoComplete="off"
                  error={errors.posisi_kasus?.message}
                  disabled={busy}
                  {...register('posisi_kasus')}
                />
              </div>
            </section>
          )}


      <div className="flex flex-wrap gap-2 border-t border-slate-200 pt-5">
        <Button type="submit" tone="utama" disabled={busy}>
          {busy ? 'Menyimpan…' : 'Transfer Recovery'}
        </Button>
        <Button
          tone="halus"
          disabled={busy}
          onClick={() => {
            reset()
            setClaimLine([])
            setPaymentProof(null)
              save.reset()
          }}
        >
          Kosongkan isian
        </Button>
          </div>
        </>
      )}
    </form>
  )
}

// ── Pesan galat ───────────────────────────────────────────────────────────────────

function SaveErrorMessage({ error }: { error: unknown }) {
  if (error instanceof NetworkError) {
    return (
      <ErrorMessage
        title="Tidak dapat menghubungi server"
        description="Batch belum tersimpan. Isian Anda masih ada — periksa koneksi lalu simpan lagi."
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

  const message = parseSave(error)
  if (message === null) return null
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

function parseSave(error: APIError): MessageContent | null {
  switch (error.kode) {
    case ErrorCode.validationFailed:
      return Object.keys(error.violations()).length > 0
        ? null
        : { title: 'Isian belum benar', description: error.message, tone: 'penolakan' }

    case ErrorCode.recoveryBatchTaken:
      return {
        title: 'Nomor batch baru saja dipakai',
        description:
          'Petugas lain menyimpan lebih dulu. Tekan Transfer Recovery sekali lagi — isian Anda tidak hilang.',
        tone: 'penolakan',
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
      return { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
  }
}


