import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useId, useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import {
  ErrorCode,
  SupplierErrorCode,
  type Supplier,
  type SupplierCode,
  type SupplierInput,
} from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useSupplierBankList, useSupplierBranchList, useSupplierCountryList } from './api'
import { CityPicker } from './CityPicker'

/**
 * Batas panjang harus sama dengan konstanta di `internal/mastersupplier/mastersupplier.go`.
 *
 * SELURUHNYA ASUMSI YANG DISADARI: DDL `M_SUPPLIER` belum diterima (R-08), dan sistem lama
 * tidak memeriksa panjang satu pun isian.
 *
 * Di modul ini batasnya punya alasan tambahan yang tidak dimiliki modul lain: seluruh
 * nilainya masuk ke SATU kolom JSONDATA, sehingga isian yang sangat panjang tidak ditolak
 * basis data melainkan diterima diam-diam sampai dokumennya melampaui batas kolomnya
 * sendiri — dan yang gagal saat itu adalah SELURUH baris, bukan isian yang kepanjangan.
 *
 * Bila salah satu berubah, KEDUA tempat harus ikut berubah.
 */
const MAX = {
  nama: 100,
  alamat: 250,
  kota: 100,
  kodePos: 10,
  telepon: 30,
  email: 100,
  npwp: 30,
  kode: 20,
  pendek: 50,
  rekening: 30,
  keterangan: 500,
} as const

const wajib = (max: number, label: string) =>
  z
    .string()
    .trim()
    .min(1, `${label} wajib diisi.`)
    .max(max, `${label} paling panjang ${max} karakter.`)

const opsional = (max: number, label: string) =>
  z.string().trim().max(max, `${label} paling panjang ${max} karakter.`)

/**
 * Kelima belas isian wajib dibaca langsung dari `pyRequired=true` pada
 * `Section/CreateMasterSupplier_Sec-Section.xml`.
 *
 * Daftarnya dijaga persis: mewajibkan lebih banyak akan menolak penambahan yang hari ini
 * diterima, dan mewajibkan lebih sedikit akan meloloskan baris yang layar lamanya tolak.
 *
 * TOP dan TOD sengaja TIDAK diperiksa berbentuk angka. Satuannya tidak disebut di mana pun
 * dalam export, dan isiannya `pxTextInput` bebas — menolak "30 hari" pada isian yang hari
 * ini menerimanya adalah selisih yang tidak diminta siapa pun.
 */
const schema = z.object({
  nama: wajib(MAX.nama, 'Nama'),
  alamat: wajib(MAX.alamat, 'Alamat'),
  kode_pos: opsional(MAX.kodePos, 'Kode Pos'),
  negara: wajib(MAX.kota, 'Negara'),

  telepon: wajib(MAX.telepon, 'Telepon'),
  fax: opsional(MAX.telepon, 'Fax'),
  email: opsional(MAX.email, 'Email'),
  npwp: opsional(MAX.npwp, 'NPWP'),
  contact_person: wajib(MAX.nama, 'Contact Person'),

  status_rekanan: wajib(MAX.kode, 'Status Rekanan'),
  status_supply: wajib(MAX.kode, 'Status Supply'),

  term_of_payment: wajib(MAX.pendek, 'Term of Payment'),
  term_of_delivery: wajib(MAX.pendek, 'Term of Delivery'),
  keterangan: opsional(MAX.keterangan, 'Keterangan'),

  bank: wajib(MAX.nama, 'Bank'),
  no_account: wajib(MAX.rekening, 'No Account'),
  account_name: opsional(MAX.nama, 'Account Name'),
  bank_branch: opsional(MAX.nama, 'Bank Branch'),

  jenis_supplier: wajib(MAX.kode, 'Jenis Supplier'),
  status_aktif: wajib(MAX.kode, 'Status Aktif'),
  status_autopayment: opsional(MAX.kode, 'Status Autopayment'),

  /** Dipilih dari daftar cabang; nama cabangnyalah yang disimpan. */
  nama_cabang: wajib(MAX.nama, 'Cabang'),
})

type SupplierFields = z.infer<typeof schema>

/** Nilai yang dikirim ke server; `kota` ditambahkan dari CityPicker. */
export type SupplierFormValues = SupplierInput

type Props = {
  /**
   * Baris yang sedang disunting, atau null bila menambah.
   *
   * Mode menentukan judul, teks tombolnya, DAN satu hal yang menyentuh aturan bisnis:
   * isian Nama dikunci pada mode ubah — lihat catatan pada isian itu.
   */
  editing: Supplier | null

  /** Kelima daftar dropdown bersandi, dari endpoint `/sandi`. */
  codes: {
    status_rekanan: SupplierCode[]
    status_supply: SupplierCode[]
    jenis_supplier: SupplierCode[]
    status_aktif: SupplierCode[]
    status_autopayment: SupplierCode[]
  }

  isSaving: boolean
  error: unknown
  onSave: (values: SupplierFormValues) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot per isian (lihat violationsOf). Yang
 * ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian tertentu,
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
      case SupplierErrorCode.nameTaken:
        return {
          title: 'Nama supplier itu sudah dipakai',
          description: 'Buka baris yang sudah ada untuk mengubahnya, atau pakai nama lain.',
          tone: 'penolakan',
        }
      case SupplierErrorCode.nameLocked:
        return {
          title: 'Nama supplier tidak dapat diubah',
          description:
            'Nama terkunci sejak baris ini tersimpan. Batalkan, lalu muat ulang daftarnya.',
          tone: 'penolakan',
        }
      case ErrorCode.validationFailed:
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya akan
        // mengulang hal yang sama.
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
  'nama',
  'alamat',
  'kode_pos',
  'negara',
  'telepon',
  'fax',
  'email',
  'npwp',
  'contact_person',
  'status_rekanan',
  'status_supply',
  'term_of_payment',
  'term_of_delivery',
  'keterangan',
  'bank',
  'no_account',
  'account_name',
  'bank_branch',
  'jenis_supplier',
  'status_aktif',
  'status_autopayment',
  'nama_cabang',
] as const

/** Mengubah daftar sandi dari server menjadi pilihan dropdown. */
function toOptions(list: SupplierCode[]) {
  return list.map((code) => ({ value: code.nilai, label: code.label }))
}

/**
 * SupplierForm adalah form tambah dan ubah Master Supplier.
 *
 * Kedua puluh tiga isiannya mengikuti caption
 * `Section/CreateMasterSupplier_Sec-Section.xml` apa adanya (`D-13`: alur dan tata letak
 * ditiru supaya pengguna tidak perlu belajar ulang). Section itu dipakai KEDUA Flow Action
 * — `CreateMasterSupplier` dan `EditMasterSupplier` — sehingga satu form untuk dua jalur
 * memang bentuk aslinya.
 *
 * # Yang berubah dari layar lama, dan hanya ini
 *
 * Isiannya DIKELOMPOKKAN menjadi lima bagian berjudul. Layar Pega menaruh kedua puluh
 * empat isian dalam satu aliran panjang; urutan dan katanya di sini sama, yang ditambahkan
 * hanyalah judul kelompok.
 *
 * # Satu isian layar lama yang TIDAK ada di sini
 *
 * **ID.** Ia `Read-only` dan `pyVisible: NOTBLANK` di layar lama — tersembunyi saat
 * menambah, muncul saat menyunting. Di sini ia ditampilkan sebagai keterangan pada mode
 * ubah, bukan sebagai isian yang tidak dapat diketik.
 */
export function SupplierForm({ editing, codes, isSaving, error, onSave, onCancel }: Props) {
  const isEditing = editing !== null
  const headingId = useId()

  const branchList = useSupplierBranchList()
  const countryList = useSupplierCountryList()
  const bankList = useSupplierBankList()

  const [city, setCity] = useState(editing?.kota ?? '')

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<SupplierFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama: editing?.nama ?? '',
      alamat: editing?.alamat ?? '',
      nama_cabang: editing?.nama_cabang ?? '',
      kode_pos: editing?.kode_pos ?? '',
      negara: editing?.negara ?? '',
      telepon: editing?.telepon ?? '',
      fax: editing?.fax ?? '',
      email: editing?.email ?? '',
      npwp: editing?.npwp ?? '',
      contact_person: editing?.contact_person ?? '',
      status_rekanan: editing?.status_rekanan ?? '',
      status_supply: editing?.status_supply ?? '',
      term_of_payment: editing?.term_of_payment ?? '',
      term_of_delivery: editing?.term_of_delivery ?? '',
      keterangan: editing?.keterangan ?? '',
      bank: editing?.bank ?? '',
      no_account: editing?.no_account ?? '',
      account_name: editing?.account_name ?? '',
      bank_branch: editing?.bank_branch ?? '',
      jenis_supplier: editing?.jenis_supplier ?? '',
      status_aktif: editing?.status_aktif ?? '',
      status_autopayment: editing?.status_autopayment ?? '',
    },
  })

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu — pada form berisi lima
  // belas isian wajib, ringkasan saja tidak memberi tahu yang mana.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if ((FIELD_NAMES as readonly string[]).includes(column)) {
        setError(column as (typeof FIELD_NAMES)[number], { type: 'server', message })
      }
    }
  }, [error, setError])

  const violation = violationsOf(error)
  const message = messageFor(error)

  function submit(values: SupplierFields) {
    onSave({ ...values, kota: city })
  }

  return (
    <form
      onSubmit={handleSubmit(submit)}
      noValidate
      aria-labelledby={headingId}
      className="space-y-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
    >
      <h2 id={headingId} className="text-base font-semibold text-slate-900">
        {isEditing ? `Ubah ${editing.nama}` : 'Tambah Master Supplier'}
      </h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {isEditing && (
        <div className="rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2.5">
          <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">
            ID supplier
          </span>
          <span className="mt-0.5 block text-sm text-slate-900">{editing.id_supplier}</span>
          <span className="mt-1 block text-xs text-slate-500">
            Diterbitkan sistem dari kode situs dan nomor urut. Tidak dapat diubah.
            {editing.id_lama !== '' && <> Nomor lama: {editing.id_lama}.</>}
          </span>
        </div>
      )}

      <Group title="Identitas supplier">
        {/*
          Nama DIKUNCI pada mode ubah.

          Itu bukan pilihan tampilan melainkan aturan yang dibaca dari layar lamanya:
          `Section/CreateMasterSupplier_Sec-Section.xml` memasang
          `pyReadOnlyCondition: MasterSupplier.ID != ''` pada isian ini — satu-satunya
          isian di form itu yang punya syarat semacam itu.

          Server ikut menolaknya, dan itu yang sebenarnya menegakkan: penguncian di sini
          adalah kenyamanan tampilan, dan permintaan yang tidak datang dari layar ini tidak
          tersentuh olehnya sama sekali.
        */}
        <Field
          id="nama"
          label="Nama"
          type="text"
          maxLength={MAX.nama}
          readOnly={isEditing}
          hint={
            isEditing
              ? 'Nama terkunci sejak baris ini tersimpan.'
              : 'Tidak boleh sama dengan supplier lain, dan tidak dapat diubah setelah disimpan.'
          }
          error={errors.nama?.message}
          disabled={isSaving}
          {...register('nama')}
        />
        <Field
          id="alamat"
          label="Alamat"
          type="text"
          maxLength={MAX.alamat}
          error={errors.alamat?.message}
          disabled={isSaving}
          {...register('alamat')}
        />
        <div className="grid gap-4 sm:grid-cols-2">
          <CityPicker
            value={city}
            onPick={(picked) => setCity(picked.nama)}
            onClear={() => setCity('')}
            error={violation['kota']}
            disabled={isSaving}
          />
          <SelectField
            id="nama_cabang"
            label="Cabang"
            options={(branchList.data?.cabang ?? []).map((b) => ({
              value: b.nama,
              label: `${b.nama} (${b.id})`,
            }))}
            error={errors.nama_cabang?.message}
            disabled={isSaving}
            {...register('nama_cabang')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="kode_pos"
            label="Kode Pos"
            type="text"
            inputMode="numeric"
            maxLength={MAX.kodePos}
            error={errors.kode_pos?.message}
            disabled={isSaving}
            {...register('kode_pos')}
          />
          <SelectField
            id="negara"
            label="Negara"
            options={(countryList.data?.negara ?? []).map((c) => ({
              value: c.nama,
              label: c.nama,
            }))}
            error={errors.negara?.message}
            disabled={isSaving}
            {...register('negara')}
          />
        </div>
      </Group>

      <Group title="Kontak">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="telepon"
            label="Telepon"
            type="text"
            inputMode="tel"
            maxLength={MAX.telepon}
            error={errors.telepon?.message}
            disabled={isSaving}
            {...register('telepon')}
          />
          <Field
            id="fax"
            label="Fax"
            type="text"
            inputMode="tel"
            maxLength={MAX.telepon}
            error={errors.fax?.message}
            disabled={isSaving}
            {...register('fax')}
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
            id="npwp"
            label="NPWP"
            type="text"
            maxLength={MAX.npwp}
            error={errors.npwp?.message}
            disabled={isSaving}
            {...register('npwp')}
          />
        </div>
        <Field
          id="contact_person"
          label="Contact Person"
          type="text"
          maxLength={MAX.nama}
          error={errors.contact_person?.message}
          disabled={isSaving}
          {...register('contact_person')}
        />
      </Group>

      <Group title="Syarat dagang">
        <div className="grid gap-4 sm:grid-cols-2">
          {/*
            Kedua isian ini TIDAK diperiksa berbentuk angka.

            Satuannya tidak disebut di mana pun dalam export, dan isiannya `pxTextInput`
            bebas di layar lama — menolak "30 hari" pada isian yang hari ini menerimanya
            adalah selisih yang tidak diminta siapa pun.
          */}
          <Field
            id="term_of_payment"
            label="Term of Payment"
            type="text"
            maxLength={MAX.pendek}
            hint="Satuannya mengikuti kesepakatan; sistem lama tidak menentukannya."
            error={errors.term_of_payment?.message}
            disabled={isSaving}
            {...register('term_of_payment')}
          />
          <Field
            id="term_of_delivery"
            label="Term of Delivery"
            type="text"
            maxLength={MAX.pendek}
            error={errors.term_of_delivery?.message}
            disabled={isSaving}
            {...register('term_of_delivery')}
          />
        </div>
        <div>
          <label htmlFor="keterangan" className="block text-sm font-medium text-slate-700">
            Keterangan
          </label>
          <textarea
            id="keterangan"
            rows={3}
            maxLength={MAX.keterangan}
            disabled={isSaving}
            aria-invalid={errors.keterangan ? 'true' : 'false'}
            className="mt-1.5 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2.5 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15 disabled:bg-slate-50"
            {...register('keterangan')}
          />
          {/* Keterangan bukan sekadar catatan: ia ikut tersalin ke kolom ALASAN_REQ pada
              baris permintaan persetujuan, sehingga isinya dibaca pihak yang memutuskan. */}
          <p className="mt-1.5 text-xs text-slate-500">
            Ikut terbaca oleh yang menyetujui pengajuan ini.
          </p>
          {errors.keterangan && (
            <p className="mt-1.5 text-sm text-red-700">{errors.keterangan.message}</p>
          )}
        </div>
      </Group>

      <Group title="Rekening pembayaran">
        <div className="grid gap-4 sm:grid-cols-2">
          {/*
            Yang disimpan adalah NAMA banknya, bukan kodenya — dokumen supplier tidak punya
            kunci BANK_ID. Kodenya tetap ditampilkan pada label sebagai pembeda bila ada dua
            bank bernama mirip.
          */}
          <SelectField
            id="bank"
            label="Bank"
            options={(bankList.data?.bank ?? []).map((b) => ({
              value: b.nama,
              label: `${b.nama} (${b.kode})`,
            }))}
            error={errors.bank?.message}
            disabled={isSaving}
            {...register('bank')}
          />
          <Field
            id="no_account"
            label="No Account"
            type="text"
            inputMode="numeric"
            maxLength={MAX.rekening}
            error={errors.no_account?.message}
            disabled={isSaving}
            {...register('no_account')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="account_name"
            label="Account Name"
            type="text"
            maxLength={MAX.nama}
            error={errors.account_name?.message}
            disabled={isSaving}
            {...register('account_name')}
          />
          <Field
            id="bank_branch"
            label="Bank Branch"
            type="text"
            maxLength={MAX.nama}
            error={errors.bank_branch?.message}
            disabled={isSaving}
            {...register('bank_branch')}
          />
        </div>
      </Group>

      {/*
        Kelima isian di bawah bersandi, dan daftar pilihannya TIDAK ADA di export:
        keduanya `pxDropdown` bersumber `associated` di Pega, artinya daftarnya hidup di
        rule Field Value yang tidak ikut diekspor (R-16).

        Yang ditawarkan di sini adalah gabungan sandi yang artinya TERBUKTI di activity dan
        sandi yang BENAR-BENAR dipakai baris yang ada — tidak ada satu pun yang dikarang.
        Sandi yang hanya ditemukan di data ditampilkan apa adanya, tanpa tebakan artinya.
      */}
      <Group title="Penggolongan dan status">
        <div className="grid gap-4 sm:grid-cols-2">
          <SelectField
            id="status_rekanan"
            label="Status Rekanan"
            options={toOptions(codes.status_rekanan)}
            error={errors.status_rekanan?.message}
            disabled={isSaving}
            {...register('status_rekanan')}
          />
          <SelectField
            id="status_supply"
            label="Status Supply"
            options={toOptions(codes.status_supply)}
            error={errors.status_supply?.message}
            disabled={isSaving}
            {...register('status_supply')}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <SelectField
            id="jenis_supplier"
            label="Jenis Supplier"
            options={toOptions(codes.jenis_supplier)}
            error={errors.jenis_supplier?.message}
            disabled={isSaving}
            {...register('jenis_supplier')}
          />
          <SelectField
            id="status_autopayment"
            label="Status Autopayment"
            options={toOptions(codes.status_autopayment)}
            error={errors.status_autopayment?.message}
            disabled={isSaving}
            {...register('status_autopayment')}
          />
        </div>
        <SelectField
          id="status_aktif"
          label="Status Aktif"
          options={toOptions(codes.status_aktif)}
          error={errors.status_aktif?.message}
          disabled={isSaving}
          {...register('status_aktif')}
        />
      </Group>

      {/*
        Kedua jalur berperilaku BERBEDA, dan bedanya menyentuh apa yang boleh dipakai
        sesudahnya — sehingga harus disebut sebelum tombol Simpan ditekan, bukan ditemukan
        sesudahnya.
      */}
      <p className="text-xs text-slate-500">
        {isEditing ? (
          <>
            Mengaktifkan supplier menunggu persetujuan lebih dulu. Menonaktifkannya berlaku{' '}
            <span className="font-medium">seketika</span>, tanpa persetujuan.
          </>
        ) : (
          <>
            Supplier baru selalu tersimpan sebagai{' '}
            <span className="font-medium">belum aktif</span> dan menunggu persetujuan —
            berapa pun Status Aktif yang dipilih di atas.
          </>
        )}
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
 * masuk ke salah satu isian di dalamnya — pada form dua puluh tiga isian, "isian ke berapa
 * dari bagian apa" adalah satu-satunya cara menavigasinya tanpa melihat.
 */
function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <fieldset className="space-y-4 rounded-kontrol border border-slate-200 p-4">
      <legend className="px-1 text-sm font-semibold text-slate-700">{title}</legend>
      {children}
    </fieldset>
  )
}
