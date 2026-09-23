import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, NON_PARTNER, WorkshopErrorCode, type Workshop, type WorkshopInput } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useWorkshopBankList, useWorkshopBranchList } from './api'
import { CityPicker } from './CityPicker'

/**
 * Batas panjang harus sama dengan konstanta di `internal/masterbengkel/masterbengkel.go`.
 *
 * SELURUHNYA ASUMSI YANG DISADARI: DDL POOLDATA.BENGKEL_HE belum diterima (R-08), dan
 * sistem lama tidak memeriksa panjang satu pun isian. Batasnya dipasang supaya penolakan
 * datang sebagai kalimat yang menuntun, bukan sebagai ORA-12899.
 *
 * Bila salah satu berubah, KEDUA tempat harus ikut berubah.
 */
const MAX = {
  nama: 100,
  alamat: 250,
  telepon: 30,
  email: 100,
  login: 32,
  bank: 100,
  rekening: 30,
  npwp: 30,
  kode: 20,
  pendek: 50,
  alasan: 250,
} as const

/**
 * Persentase diperiksa sebagai angka 0–100 bila diisi, dengan koma MAUPUN titik sebagai
 * pemisah desimal — petugas Indonesia mengetik "12,5" sementara basis data menyimpan
 * "12.5".
 *
 * Ini SELISIH YANG DIRENCANAKAN terhadap Pega, yang menerima teks apa pun. Server tetap
 * yang berwenang; pemeriksaan di layar hanya kenyamanan.
 */
const percent = z
  .string()
  .trim()
  .refine((value) => {
    if (value === '') return true
    const parsed = Number(value.replace(',', '.'))
    return Number.isFinite(parsed) && parsed >= 0 && parsed <= 100
  }, 'Harus berupa angka antara 0 dan 100, misalnya 11 atau 12,5.')

const optional = (max: number, label: string) =>
  z.string().trim().max(max, `${label} paling panjang ${max} karakter.`)

const schema = z
  .object({
    nama_bengkel: z
      .string()
      .trim()
      .min(1, 'Nama bengkel wajib diisi.')
      .max(MAX.nama, `Nama bengkel paling panjang ${MAX.nama} karakter.`),
    alamat_bengkel: optional(MAX.alamat, 'Alamat bengkel'),
    telp_bengkel: optional(MAX.telepon, 'Telepon bengkel'),
    nohp_bengkel: optional(MAX.telepon, 'No HP bengkel'),
    email: optional(MAX.email, 'Email'),
    email_wo: optional(MAX.email, 'Email WO'),

    id_cabang: optional(MAX.kode, 'Kode cabang'),
    status_rekanan: z.string().trim().min(1, 'Status rekanan wajib dipilih.'),
    status_bengkel: optional(MAX.kode, 'Status bengkel'),
    alasan_status_bengkel: optional(MAX.alasan, 'Alasan status bengkel'),
    tanggal_status: optional(MAX.pendek, 'Tanggal status'),

    login_aplikasi: optional(MAX.login, 'Login aplikasi'),

    id_bank: optional(MAX.kode, 'Kode bank'),
    no_rekening: optional(MAX.rekening, 'No rekening'),
    nama_rekening: optional(MAX.nama, 'Nama rekening'),
    id_rekening: optional(MAX.kode, 'ID rekening'),

    nama_npwp: optional(MAX.nama, 'Nama NPWP'),
    no_npwp: optional(MAX.npwp, 'No NPWP'),
    alamat_npwp: optional(MAX.alamat, 'Alamat NPWP'),
    jenis_pph: optional(MAX.pendek, 'Jenis PPh'),

    ppn: percent,
    diskon_jasa: percent,
    diskon_sparepart: percent,
    persen_material: percent,
    pct_selisih_pl: percent,
    sla: optional(MAX.pendek, 'SLA'),

    status_disupply_asm: optional(MAX.kode, 'Status disupply ASM'),
    supplier: optional(MAX.nama, 'Supplier'),
    status_eklaim: optional(MAX.kode, 'Status e-klaim'),
    status_auto_aksep: optional(MAX.kode, 'Status auto aksep'),
    status_payment: optional(MAX.kode, 'Status payment'),
    status_autopayment: optional(MAX.kode, 'Status autopayment'),
    status_tekno: optional(MAX.kode, 'Status Tekno'),
    status_order: optional(MAX.kode, 'Status order'),
  })
  .superRefine((values, ctx) => {
    /*
      Login aplikasi wajib HANYA bila bengkelnya rekanan.

      Syarat itu bukan karangan: `Activity/ValidationLoginBengkel_act` melompati seluruh
      urusan login pada prasyarat `Local.STS_REKANAN=='0'`. Mewajibkannya tanpa syarat
      akan menolak bengkel non-rekanan yang hari ini sah; tidak mewajibkannya sama sekali
      akan menyimpan bengkel rekanan yang tidak akan pernah dapat masuk.
    */
    if (values.status_rekanan !== NON_PARTNER && values.login_aplikasi.trim() === '') {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['login_aplikasi'],
        message: 'Login aplikasi wajib diisi untuk bengkel rekanan.',
      })
    }
  })

export type WorkshopFields = z.infer<typeof schema>

/** Nilai yang dikirim ke pemanggil saat form disimpan. */
export type WorkshopFormValues = WorkshopInput

type Props = {
  /**
   * Baris yang sedang disunting, atau null bila menambah.
   *
   * Mode hanya menentukan judul dan teks tombolnya — seluruh isian dapat diubah pada
   * kedua mode, sama seperti layar lama yang memakai form yang sama untuk keduanya.
   */
  editing: Workshop | null

  /**
   * Nilai yang SUDAH DIPAKAI baris lain, per isian.
   *
   * Dipakai sebagai saran pada ketujuh penanda status yang daftar pilihannya TIDAK ADA di
   * export Pega (R-16). Sarannya berasal dari data, bukan dari tebakan — lihat komentar
   * pada bagian "Penanda sistem" di bawah.
   */
  knownValues: Record<string, string[]>

  isSaving: boolean
  error: unknown
  onSave: (values: WorkshopFormValues) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot per isian (lihat violationsOf).
 * Yang ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian tertentu,
 * karena itulah yang tidak dapat diperbaiki pengguna dengan mengetik.
 */
function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case WorkshopErrorCode.nameTaken:
        return {
          title: 'Nama bengkel itu sudah dipakai',
          description:
            'Buka baris yang sudah ada untuk mengubahnya, atau pakai nama lain.',
          tone: 'penolakan',
        }
      case WorkshopErrorCode.loginTaken:
        return {
          title: 'Login aplikasi itu sudah dipakai bengkel lain',
          description: 'Ganti login aplikasinya, lalu simpan lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.validationFailed:
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya
        // akan mengulang hal yang sama.
        return Object.keys(error.violations()).length > 0
          ? null
          : { title: 'Belum dapat disimpan', description: error.message, tone: 'penolakan' }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: 'Pilih portal entitas di bagian atas halaman, lalu simpan lagi.',
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
          title: 'Terjadi kesalahan pada sistem',
          description: 'Isian Anda belum tersimpan. Coba beberapa saat lagi.',
          tone: 'gangguan',
        }
    }
  }
  return null
}

/** Mengambil pelanggaran per isian dari galat validasi server. */
function violationsOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {}
}

/** Nama isian yang dikenali form ini; dipakai menyorot pelanggaran dari server. */
const FIELD_NAMES = [
  'nama_bengkel',
  'alamat_bengkel',
  'telp_bengkel',
  'nohp_bengkel',
  'email',
  'email_wo',
  'id_cabang',
  'status_rekanan',
  'status_bengkel',
  'alasan_status_bengkel',
  'tanggal_status',
  'login_aplikasi',
  'id_bank',
  'no_rekening',
  'nama_rekening',
  'id_rekening',
  'nama_npwp',
  'no_npwp',
  'alamat_npwp',
  'jenis_pph',
  'ppn',
  'diskon_jasa',
  'diskon_sparepart',
  'persen_material',
  'pct_selisih_pl',
  'sla',
  'status_disupply_asm',
  'supplier',
  'status_eklaim',
  'status_auto_aksep',
  'status_payment',
  'status_autopayment',
  'status_tekno',
  'status_order',
] as const

/**
 * Status rekanan adalah satu-satunya isian status yang nilai sahnya DIKETAHUI.
 *
 * Artinya terbaca dari percabangan, bukan dari label: `ValidationLoginBengkel_act`
 * melompat keluar pada `Local.STS_REKANAN=='0'`, dan yang dilewatinya adalah pembuatan
 * login aplikasi. Karena isian itu menentukan wajib-tidaknya isian lain, ia dibuat
 * dropdown — bukan teks bebas.
 */
const PARTNER_OPTIONS = [
  { value: '1', label: 'Rekanan' },
  { value: NON_PARTNER, label: 'Non-rekanan' },
]

/**
 * WorkshopForm adalah form tambah dan ubah Master Bengkel.
 *
 * Ketiga puluh tiga isiannya mengikuti caption
 * `Section/BrowseMasterHEApprove-Section.xml` apa adanya (`D-13`: alur dan tata letak
 * ditiru supaya pengguna tidak perlu belajar ulang).
 *
 * # Yang berubah dari layar lama, dan hanya ini
 *
 * Isiannya DIKELOMPOKKAN menjadi enam bagian berjudul. Layar Pega menaruh ketiga puluh
 * tiga isian dalam satu aliran panjang; urutan dan katanya di sini sama, yang ditambahkan
 * hanyalah judul kelompok. Tanpa itu, form tiga puluh tiga isian menjadi dinding yang
 * tidak dapat dipindai mata — dan `D-13` menuntut alur yang sama, bukan tata letak yang
 * sama-sama sulit dibaca.
 *
 * # Tiga isian layar lama yang TIDAK ada di sini
 *
 * **ID BENGKEL.** Diterbitkan server dari kode situs dan sequence
 * (`PEGA_M_BENGKEL_HE.prc:19`), tidak pernah diketik. Pada mode ubah ia ditampilkan
 * sebagai keterangan.
 *
 * **MEMPERBAHARUI DATA.** Caption tombol pada layar lama, bukan isian.
 *
 * **Unggah lampiran.** Jalur `PNCSaveAttachmentToDB` tidak dibawa modul ini; kolom
 * DOKUMENID baris yang sudah ada dipertahankan apa adanya.
 */
export function WorkshopForm({
  editing,
  knownValues,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const isEditing = editing !== null
  const headingId = useId()

  const branchList = useWorkshopBranchList()
  const bankList = useWorkshopBankList()

  const [city, setCity] = useState(
    editing && editing.id_kota !== '' ? { id: editing.id_kota, nama: editing.nama_kota } : null,
  )

  const {
    register,
    handleSubmit,
    setError,
    watch,
    formState: { errors },
  } = useForm<WorkshopFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama_bengkel: editing?.nama_bengkel ?? '',
      alamat_bengkel: editing?.alamat_bengkel ?? '',
      telp_bengkel: editing?.telp_bengkel ?? '',
      nohp_bengkel: editing?.nohp_bengkel ?? '',
      email: editing?.email ?? '',
      email_wo: editing?.email_wo ?? '',
      id_cabang: editing?.id_cabang ?? '',
      status_rekanan: editing?.status_rekanan ?? '1',
      status_bengkel: editing?.status_bengkel ?? '',
      alasan_status_bengkel: editing?.alasan_status_bengkel ?? '',
      tanggal_status: editing?.tanggal_status ?? '',
      login_aplikasi: editing?.login_aplikasi ?? '',
      id_bank: editing?.id_bank ?? '',
      no_rekening: editing?.no_rekening ?? '',
      nama_rekening: editing?.nama_rekening ?? '',
      id_rekening: editing?.id_rekening ?? '',
      nama_npwp: editing?.nama_npwp ?? '',
      no_npwp: editing?.no_npwp ?? '',
      alamat_npwp: editing?.alamat_npwp ?? '',
      jenis_pph: editing?.jenis_pph ?? '',
      ppn: editing?.ppn ?? '',
      diskon_jasa: editing?.diskon_jasa ?? '',
      diskon_sparepart: editing?.diskon_sparepart ?? '',
      persen_material: editing?.persen_material ?? '',
      pct_selisih_pl: editing?.pct_selisih_pl ?? '',
      sla: editing?.sla ?? '',
      status_disupply_asm: editing?.status_disupply_asm ?? '',
      supplier: editing?.supplier ?? '',
      status_eklaim: editing?.status_eklaim ?? '',
      status_auto_aksep: editing?.status_auto_aksep ?? '',
      status_payment: editing?.status_payment ?? '',
      status_autopayment: editing?.status_autopayment ?? '',
      status_tekno: editing?.status_tekno ?? '',
      status_order: editing?.status_order ?? '',
    },
  })

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if ((FIELD_NAMES as readonly string[]).includes(column)) {
        setError(column as (typeof FIELD_NAMES)[number], { type: 'server', message })
      }
    }
  }, [error, setError])

  const violation = violationsOf(error)
  const message = messageFor(error)
  const isPartner = watch('status_rekanan') !== NON_PARTNER

  function submit(values: WorkshopFields) {
    // Nama cabang dan nama bank diambil dari daftar yang sama dengan pilihannya, bukan
    // diketik: keduanya disimpan BERPASANGAN dengan kodenya, dan pasangan yang tidak
    // menunjuk baris mana pun di masternya tidak dapat dikenali siapa pun.
    const branch = (branchList.data?.cabang ?? []).find((b) => b.id === values.id_cabang)
    const bank = (bankList.data?.bank ?? []).find((b) => b.kode === values.id_bank)

    onSave({
      ...values,
      nama_cabang: branch?.nama ?? editing?.nama_cabang ?? '',
      id_kota: city?.id ?? '',
      nama_kota: city?.nama ?? '',
      nama_bank: bank?.nama ?? editing?.nama_bank ?? '',
    })
  }

  return (
    <form
      onSubmit={handleSubmit(submit)}
      noValidate
      aria-labelledby={headingId}
      className="space-y-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
    >
      <h2 id={headingId} className="text-base font-semibold text-slate-900">
        {isEditing ? `Ubah ${editing.nama_bengkel}` : 'Tambah Master Bengkel'}
      </h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {isEditing && (
        <div className="rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2.5">
          <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">
            ID bengkel
          </span>
          <span className="mt-0.5 block text-sm text-slate-900">{editing.id_bengkel}</span>
          <span className="mt-1 block text-xs text-slate-500">
            Diterbitkan sistem dari kode situs dan nomor urut. Tidak dapat diubah.
          </span>
        </div>
      )}

      <Group title="Identitas bengkel">
        <Field
          id="nama_bengkel"
          label="Nama bengkel"
          type="text"
          maxLength={MAX.nama}
          hint="Tidak boleh sama dengan bengkel lain."
          error={errors.nama_bengkel?.message}
          disabled={isSaving}
          {...register('nama_bengkel')}
        />
        <Field
          id="alamat_bengkel"
          label="Alamat bengkel"
          type="text"
          maxLength={MAX.alamat}
          error={errors.alamat_bengkel?.message}
          disabled={isSaving}
          {...register('alamat_bengkel')}
        />
        <div className="grid gap-4 sm:grid-cols-2">
          <SelectField
            id="id_cabang"
            label="Nama cabang"
            options={(branchList.data?.cabang ?? []).map((b) => ({
              value: b.id,
              label: `${b.nama} (${b.id})`,
            }))}
            error={errors.id_cabang?.message ?? violation['nama_cabang']}
            disabled={isSaving}
            {...register('id_cabang')}
          />
          <CityPicker
            selected={city}
            onPick={setCity}
            onClear={() => setCity(null)}
            error={violation['id_kota'] ?? violation['nama_kota']}
            disabled={isSaving}
          />
        </div>
      </Group>

      <Group title="Kontak">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="telp_bengkel"
            label="Telp bengkel"
            type="text"
            inputMode="tel"
            maxLength={MAX.telepon}
            error={errors.telp_bengkel?.message}
            disabled={isSaving}
            {...register('telp_bengkel')}
          />
          <Field
            id="nohp_bengkel"
            label="No HP bengkel"
            type="text"
            inputMode="tel"
            maxLength={MAX.telepon}
            error={errors.nohp_bengkel?.message}
            disabled={isSaving}
            {...register('nohp_bengkel')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="email"
            label="Email"
            type="email"
            maxLength={MAX.email}
            error={errors.email?.message}
            disabled={isSaving}
            {...register('email')}
          />
          <Field
            id="email_wo"
            label="Email WO"
            type="email"
            maxLength={MAX.email}
            hint="Tujuan perintah kerja; terpisah dari email umum bengkel."
            error={errors.email_wo?.message}
            disabled={isSaving}
            {...register('email_wo')}
          />
        </div>
      </Group>

      <Group title="Status kerja sama">
        <div className="grid gap-4 sm:grid-cols-2">
          <SelectField
            id="status_rekanan"
            label="Status rekanan"
            options={PARTNER_OPTIONS}
            emptyText="— pilih —"
            error={errors.status_rekanan?.message}
            disabled={isSaving}
            {...register('status_rekanan')}
          />
          <Field
            id="login_aplikasi"
            label="Login aplikasi"
            type="text"
            maxLength={MAX.login}
            autoComplete="off"
            hint={
              isPartner
                ? 'Wajib untuk bengkel rekanan, dan tidak boleh sama dengan bengkel lain.'
                : 'Boleh dikosongkan: bengkel non-rekanan tidak diberi login.'
            }
            error={errors.login_aplikasi?.message}
            disabled={isSaving}
            {...register('login_aplikasi')}
          />
        </div>
        {/*
          Pemberitahuan yang JUJUR tentang apa yang tidak dikerjakan sistem baru.

          Di Pega, menyetujui bengkel rekanan MEMBUAT akun operator lewat
          `GCNMCreateOperator` — dengan kata sandi yang sama untuk setiap bengkel
          (`Local.password := "123456"`). Akun itu tidak dibawa: sistem baru tidak punya
          operator Pega, dan kontrak identitasnya belum ada (F-3, R-14).

          Tanpa kalimat ini, petugas akan mengira bengkelnya sudah bisa masuk hanya karena
          login-nya tersimpan.
        */}
        {isPartner && (
          <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
            Login aplikasi hanya <span className="font-medium">disimpan</span> di master ini.
            Akun untuk masuk aplikasi bengkel belum diterbitkan sistem baru — pembuatannya
            menunggu kontrak identitas (F-3).
          </p>
        )}
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="status_bengkel"
            label="Status bengkel"
            type="text"
            maxLength={MAX.kode}
            list="bengkel-status_bengkel"
            error={errors.status_bengkel?.message}
            disabled={isSaving}
            {...register('status_bengkel')}
          />
          <Field
            id="tanggal_status"
            label="Tanggal status"
            type="text"
            maxLength={MAX.pendek}
            hint="Ditulis apa adanya seperti di sistem lama, misalnya 01/02/2026."
            error={errors.tanggal_status?.message}
            disabled={isSaving}
            {...register('tanggal_status')}
          />
        </div>
        <Suggestions name="status_bengkel" values={knownValues['status_bengkel'] ?? []} />
        <Field
          id="alasan_status_bengkel"
          label="Alasan status bengkel"
          type="text"
          maxLength={MAX.alasan}
          error={errors.alasan_status_bengkel?.message}
          disabled={isSaving}
          {...register('alasan_status_bengkel')}
        />
      </Group>

      <Group title="Bank dan pajak">
        <div className="grid gap-4 sm:grid-cols-2">
          <SelectField
            id="id_bank"
            label="Nama bank"
            options={(bankList.data?.bank ?? []).map((b) => ({
              value: b.kode,
              label: `${b.nama} (${b.kode})`,
            }))}
            error={errors.id_bank?.message ?? violation['nama_bank']}
            disabled={isSaving}
            {...register('id_bank')}
          />
          <Field
            id="no_rekening"
            label="No rekening"
            type="text"
            inputMode="numeric"
            maxLength={MAX.rekening}
            error={errors.no_rekening?.message}
            disabled={isSaving}
            {...register('no_rekening')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="nama_rekening"
            label="Nama rekening"
            type="text"
            maxLength={MAX.nama}
            hint="Tidak ditampilkan layar Pega, tetapi tersimpan di tabelnya."
            error={errors.nama_rekening?.message}
            disabled={isSaving}
            {...register('nama_rekening')}
          />
          <Field
            id="id_rekening"
            label="ID rekening"
            type="text"
            maxLength={MAX.kode}
            error={errors.id_rekening?.message}
            disabled={isSaving}
            {...register('id_rekening')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="nama_npwp"
            label="Nama NPWP"
            type="text"
            maxLength={MAX.nama}
            error={errors.nama_npwp?.message}
            disabled={isSaving}
            {...register('nama_npwp')}
          />
          <Field
            id="no_npwp"
            label="No NPWP"
            type="text"
            maxLength={MAX.npwp}
            error={errors.no_npwp?.message}
            disabled={isSaving}
            {...register('no_npwp')}
          />
        </div>
        <Field
          id="alamat_npwp"
          label="Alamat NPWP"
          type="text"
          maxLength={MAX.alamat}
          error={errors.alamat_npwp?.message}
          disabled={isSaving}
          {...register('alamat_npwp')}
        />
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="jenis_pph"
            label="Jenis PPh"
            type="text"
            maxLength={MAX.pendek}
            list="bengkel-jenis_pph"
            error={errors.jenis_pph?.message}
            disabled={isSaving}
            {...register('jenis_pph')}
          />
          <Field
            id="ppn"
            label="PPN (%)"
            type="text"
            inputMode="decimal"
            error={errors.ppn?.message}
            disabled={isSaving}
            {...register('ppn')}
          />
        </div>
        <Suggestions name="jenis_pph" values={knownValues['jenis_pph'] ?? []} />
      </Group>

      <Group title="Syarat komersial">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="diskon_jasa"
            label="Discount jasa (%)"
            type="text"
            inputMode="decimal"
            error={errors.diskon_jasa?.message}
            disabled={isSaving}
            {...register('diskon_jasa')}
          />
          <Field
            id="diskon_sparepart"
            label="Discount sparepart (%)"
            type="text"
            inputMode="decimal"
            error={errors.diskon_sparepart?.message}
            disabled={isSaving}
            {...register('diskon_sparepart')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="persen_material"
            label="Persen material (%)"
            type="text"
            inputMode="decimal"
            error={errors.persen_material?.message}
            disabled={isSaving}
            {...register('persen_material')}
          />
          <Field
            id="pct_selisih_pl"
            label="PCT selisih price list (%)"
            type="text"
            inputMode="decimal"
            hint="Tidak ditampilkan layar Pega, tetapi tersimpan di tabelnya."
            error={errors.pct_selisih_pl?.message}
            disabled={isSaving}
            {...register('pct_selisih_pl')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="sla"
            label="SLA"
            type="text"
            maxLength={MAX.pendek}
            hint="Satuannya tidak disebut di mana pun pada sistem lama."
            error={errors.sla?.message}
            disabled={isSaving}
            {...register('sla')}
          />
          <Field
            id="supplier"
            label="Supplier"
            type="text"
            maxLength={MAX.nama}
            error={errors.supplier?.message}
            disabled={isSaving}
            {...register('supplier')}
          />
        </div>
      </Group>

      <Group title="Penanda sistem">
        {/*
          Ketujuh penanda ini dirender pxRadioButtons atau pxDropdown di Pega, dan daftar
          pilihannya ada di rule Field Value yang TIDAK ikut di export (R-16).

          Karena itu tidak satu pun dibuat dropdown: mengarang dua pilihan "Ya/Tidak"
          berarti menebak domain sebuah kolom yang menentukan kanal mana yang boleh
          dipakai bengkel. Yang ditawarkan sebagai gantinya adalah NILAI YANG SUDAH DIPAKAI
          BARIS LAIN — saran yang berasal dari data, bukan dari tebakan.
        */}
        <p className="text-xs text-slate-500">
          Daftar pilihan ketujuh penanda ini tidak ada di export Pega. Saran yang muncul saat
          mengetik adalah nilai yang sudah dipakai bengkel lain pada entitas ini.
        </p>
        <div className="grid gap-4 sm:grid-cols-2">
          {(
            [
              ['status_disupply_asm', 'Status disupply ASM'],
              ['status_eklaim', 'Status e-klaim'],
              ['status_auto_aksep', 'Status auto aksep'],
              ['status_payment', 'Status payment'],
              ['status_autopayment', 'Status autopayment'],
              ['status_tekno', 'Status Tekno'],
              ['status_order', 'Status order'],
            ] as const
          ).map(([name, label]) => (
            <div key={name}>
              <Field
                id={name}
                label={label}
                type="text"
                maxLength={MAX.kode}
                list={`bengkel-${name}`}
                error={errors[name]?.message}
                disabled={isSaving}
                {...register(name)}
              />
              <Suggestions name={name} values={knownValues[name] ?? []} />
            </div>
          ))}
        </div>
      </Group>

      <p className="text-xs text-slate-500">
        Baris yang disimpan selalu kembali ke{' '}
        <span className="font-medium">Waiting Approval</span> dan menunggu persetujuan —
        persetujuan sebelumnya tidak berlaku atas isi yang sudah berubah.
      </p>

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}

/**
 * Group membungkus sekumpulan isian di bawah satu judul.
 *
 * Ia `<fieldset>` dan bukan `<div>` supaya pembaca layar mengumumkan judulnya saat kursor
 * masuk ke salah satu isian di dalamnya — pada form tiga puluh tiga isian, "isian ke
 * berapa dari bagian apa" adalah satu-satunya cara menavigasinya tanpa melihat.
 */
function Group({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <fieldset className="space-y-4 rounded-kontrol border border-slate-200 p-4">
      <legend className="px-1 text-sm font-semibold text-slate-700">{title}</legend>
      {children}
    </fieldset>
  )
}

/**
 * Suggestions menggambar daftar saran untuk sebuah isian.
 *
 * Isinya berasal dari nilai yang SUDAH DIPAKAI baris lain pada entitas yang sedang
 * dibuka — bukan dari daftar yang dikarang. Ia menjawab pertanyaan yang tidak dapat
 * dijawab export Pega ("nilai apa yang sah di kolom ini?") dengan satu-satunya sumber
 * yang tersedia: data itu sendiri.
 *
 * Tidak menggambar apa pun bila belum ada nilai yang terpakai, supaya daftar kosong tidak
 * muncul sebagai kotak saran yang selalu hampa.
 */
function Suggestions({ name, values }: { name: string; values: string[] }) {
  if (values.length === 0) return null

  return (
    <datalist id={`bengkel-${name}`}>
      {values.map((value) => (
        <option key={value} value={value} />
      ))}
    </datalist>
  )
}
