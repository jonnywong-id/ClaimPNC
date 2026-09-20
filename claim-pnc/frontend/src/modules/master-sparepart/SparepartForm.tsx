import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useId, useMemo } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, SparepartErrorCode, type Sparepart, type SparepartInput } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useSparepartOptions } from './api'
import { ChoiceField } from './ChoiceField'

/**
 * Batas panjang harus sama dengan konstanta di `internal/mastersparepart/mastersparepart.go`.
 *
 * SELURUHNYA ASUMSI YANG DISADARI: DDL POOLDATA.SPAREPART_HE belum diterima (R-08), dan
 * sistem lama tidak memeriksa panjang satu pun isian — tidak ada satu pun `pyMaxLength` pada
 * layarnya. Batasnya dipasang supaya penolakan datang sebagai kalimat yang menuntun, bukan
 * sebagai ORA-12899.
 *
 * Bila salah satu berubah, KEDUA tempat harus ikut berubah. Uji
 * `TestLengthLimitsAreTheOnesTheFormRepeats` di backend menjaga duplikasinya tetap terlihat.
 */
const MAX = {
  nama: 100,
  nomor: 50,
  kode: 50,
  angka: 20,
  tanggal: 30,
  penanda: 30,
} as const

/** Batas atas harga jual; sama dengan mastersparepart.MaxPrice. */
const MAX_HARGA = 100_000_000_000

/**
 * angka membaca sebuah isian angka, meniru mastersparepart.ParseNumber.
 *
 * Koma DAN titik keduanya diterima sebagai pemisah desimal: petugas Indonesia mengetik
 * "1250000,5" sementara nilai yang tersimpan di basis data memakai titik.
 *
 * Titik sebagai pemisah RIBUAN tidak dikenali, dan itu disengaja: "1.250" dapat berarti
 * seribu dua ratus lima puluh ATAU satu koma dua lima nol, dan menebaknya berarti salah
 * membaca harga separuh waktu.
 */
function angka(text: string): number | null {
  const clean = text.trim()
  if (clean === '' || (clean.match(/,/g) ?? []).length > 1) return null

  const value = Number(clean.replace(',', '.'))
  return Number.isFinite(value) ? value : null
}

const wajib = (max: number, label: string) =>
  z
    .string()
    .trim()
    .min(1, `${label} wajib diisi.`)
    .max(max, `${label} paling panjang ${max} karakter.`)

/** Isian angka yang boleh kosong; bila diisi harus angka dan tidak negatif. */
const angkaOpsional = (label: string) =>
  z
    .string()
    .trim()
    .max(MAX.angka, `${label} paling panjang ${MAX.angka} karakter.`)
    .refine((v) => v === '' || angka(v) !== null, `${label} harus berupa angka.`)
    .refine((v) => v === '' || (angka(v) ?? 0) >= 0, `${label} tidak boleh negatif.`)

const schema = z
  .object({
    // Keempat isian wajib berasal dari `pyRequired=true` pada
    // `Section/BrowseMasterSparepartHEApproval-Section.xml`, dan HANYA keempat itu.
    nomor_sparepart: wajib(MAX.nomor, 'Nomor sparepart'),
    nama_sparepart: wajib(MAX.nama, 'Nama sparepart'),
    kode_sparepart: wajib(MAX.kode, 'Kode sparepart'),
    harga_jual: wajib(MAX.angka, 'Harga jual')
      .refine(
        (v) => angka(v) !== null,
        'Harga jual harus berupa angka, misalnya 1250000 atau 1250000,50.',
      )
      .refine((v) => (angka(v) ?? -1) >= 0, 'Harga jual tidak boleh negatif.')
      .refine(
        (v) => (angka(v) ?? 0) <= MAX_HARGA,
        'Harga jual terlalu besar. Periksa lagi jumlah nolnya.',
      ),

    kategori_sparepart: z.string().trim().max(MAX.kode),
    tipe_sparepart: z.string().trim().max(MAX.kode),

    berat: angkaOpsional('Berat sparepart'),
    panjang: angkaOpsional('Panjang sparepart'),
    lebar: angkaOpsional('Lebar sparepart'),
    tinggi: angkaOpsional('Tinggi sparepart'),
    stock_minimal: angkaOpsional('Stock minimal'),
    stock_maximal: angkaOpsional('Stock maximal'),
    kuantitas_pesanan: angkaOpsional('Kuantitas pesanan'),

    tanggal_produksi: z
      .string()
      .trim()
      .max(MAX.tanggal, `Tanggal produksi paling panjang ${MAX.tanggal} karakter.`),
    part_substitusi: z
      .string()
      .trim()
      .max(MAX.nama, `Part substitusi paling panjang ${MAX.nama} karakter.`),

    jenis_sparepart: z.string().trim().max(MAX.penanda),
    satuan: z.string().trim().max(MAX.penanda),
    status_aktif: z.string().trim().max(MAX.penanda),
    status_sparepart: z.string().trim().max(MAX.penanda),
  })
  // Stok minimal tidak boleh melampaui stok maksimal — SELISIH YANG DIRENCANAKAN terhadap
  // sistem lama, yang tidak memeriksanya sama sekali. Pasangan yang terbalik membuat setiap
  // pemeriksaan stok di modul hilir selalu benar atau selalu salah.
  .superRefine((values, ctx) => {
    const low = angka(values.stock_minimal)
    const high = angka(values.stock_maximal)
    if (low !== null && high !== null && low > high) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['stock_minimal'],
        message: 'Stock minimal tidak boleh melebihi stock maximal.',
      })
    }
  })

export type SparepartFields = z.infer<typeof schema>

/** Nilai yang dikirim ke pemanggil saat form disimpan. */
export type SparepartFormValues = SparepartInput

type Props = {
  /**
   * Baris yang sedang disunting, atau null bila menambah.
   *
   * Mode hanya menentukan judul dan teks tombolnya — seluruh isian dapat diubah pada kedua
   * mode, sama seperti layar lama yang memakai form yang sama untuk keduanya.
   */
  editing: Sparepart | null

  /**
   * Nilai yang SUDAH DIPAKAI baris lain, per isian penanda.
   *
   * Dipakai sebagai pilihan pada keempat penanda yang daftar pilihannya TIDAK ADA di export
   * Pega (R-16). Pilihannya berasal dari data, bukan dari tebakan.
   */
  knownValues: Record<string, string[]>

  isSaving: boolean
  error: unknown
  onSave: (values: SparepartFormValues) => void
  onCancel: () => void
}

type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot per isian (lihat violationsOf). Yang
 * ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian tertentu, karena
 * itulah yang tidak dapat diperbaiki pengguna dengan mengetik.
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
      case SparepartErrorCode.keyTaken:
        // Isiannya sudah disorot lewat `detail`; kotak pesan menjelaskan apa yang harus
        // dilakukan, bukan mengulang isian mana yang salah.
        return {
          title: 'Sparepart dengan kunci itu sudah ada',
          description:
            'Nomor, nama, dan kode sparepart harus unik. Buka baris yang sudah ada untuk mengubahnya, atau pakai nilai lain.',
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
  'nomor_sparepart',
  'nama_sparepart',
  'kode_sparepart',
  'harga_jual',
  'kategori_sparepart',
  'tipe_sparepart',
  'berat',
  'panjang',
  'lebar',
  'tinggi',
  'stock_minimal',
  'stock_maximal',
  'kuantitas_pesanan',
  'tanggal_produksi',
  'part_substitusi',
  'jenis_sparepart',
  'satuan',
  'status_aktif',
  'status_sparepart',
] as const

/**
 * SparepartForm adalah form tambah dan ubah Master Sparepart.
 *
 * Kedua puluh isiannya mengikuti caption
 * `Section/BrowseMasterSparepartHEApproval-Section.xml` apa adanya, pada urutan yang sama
 * (`D-13`).
 *
 * # Tiga isian layar lama yang TIDAK ada di sini
 *
 * **ID Sparepart.** Diterbitkan server dari kode situs dan sequence
 * (`PEGA_M_SPAREPART_HE.prc:21`), tidak pernah diketik. Pada mode ubah ia ditampilkan
 * sebagai keterangan.
 *
 * **User Update dan Tanggal Update.** Keduanya diisi penyimpanan
 * (`Activity/UpdateSparepartHE_act`), bukan oleh pengguna. Pada mode ubah keduanya
 * ditampilkan sebagai keterangan.
 *
 * **Unggah lampiran.** Jalur `PNCSaveAttachmentToDB` tidak dibawa modul ini; kolom DOKUMENID
 * baris yang sudah ada dipertahankan apa adanya.
 */
export function SparepartForm({
  editing,
  knownValues,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const isEditing = editing !== null
  const headingId = useId()

  const options = useSparepartOptions()

  const {
    register,
    handleSubmit,
    setError,
    setValue,
    control,
    formState: { errors },
  } = useForm<SparepartFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nomor_sparepart: editing?.nomor_sparepart ?? '',
      nama_sparepart: editing?.nama_sparepart ?? '',
      kode_sparepart: editing?.kode_sparepart ?? '',
      harga_jual: editing?.harga_jual ?? '',
      kategori_sparepart: editing?.kategori_sparepart ?? '',
      tipe_sparepart: editing?.tipe_sparepart ?? '',
      berat: editing?.berat ?? '',
      panjang: editing?.panjang ?? '',
      lebar: editing?.lebar ?? '',
      tinggi: editing?.tinggi ?? '',
      stock_minimal: editing?.stock_minimal ?? '',
      stock_maximal: editing?.stock_maximal ?? '',
      kuantitas_pesanan: editing?.kuantitas_pesanan ?? '',
      tanggal_produksi: editing?.tanggal_produksi ?? '',
      part_substitusi: editing?.part_substitusi ?? '',
      jenis_sparepart: editing?.jenis_sparepart ?? '',
      satuan: editing?.satuan ?? '',
      status_aktif: editing?.status_aktif ?? '',
      status_sparepart: editing?.status_sparepart ?? '',
    },
  })

  // Tipe BERCABANG dari Kategori: daftar Tipe dipersempit begitu Kategori dipilih.
  // `RDB List/BrowseSparepartTypeClaimHE_sql-SQL.xml` menggabungkan keduanya lewat
  // PART_CATEGORY_ID; tanpa penyempitan ini petugas dapat menautkan tipe milik kategori lain.
  const chosenCategory = useWatch({ control, name: 'kategori_sparepart' })

  const categoryOptions = useMemo(
    () => (options.data?.kategori ?? []).map((one) => ({ value: one.kode, label: one.nama })),
    [options.data],
  )

  const typeOptions = useMemo(() => {
    const all = options.data?.tipe ?? []
    const narrowed = chosenCategory
      ? all.filter((one) => one.kode_kategori === chosenCategory)
      : all
    return narrowed.map((one) => ({ value: one.kode, label: one.nama }))
  }, [options.data, chosenCategory])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan
  // itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if ((FIELD_NAMES as readonly string[]).includes(column)) {
        setError(column as (typeof FIELD_NAMES)[number], { type: 'server', message })
      }
    }
  }, [error, setError])

  const message = messageFor(error)

  return (
    <form
      onSubmit={handleSubmit(onSave)}
      noValidate
      aria-labelledby={headingId}
      className="space-y-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
    >
      <h2 id={headingId} className="text-base font-semibold text-slate-900">
        {isEditing ? `Ubah ${editing.nama_sparepart}` : 'Tambah Master Sparepart'}
      </h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {isEditing && (
        <div className="grid gap-3 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2.5 sm:grid-cols-3">
          <Keterangan
            label="ID sparepart"
            value={editing.id_sparepart}
            hint="Diterbitkan sistem. Tidak dapat diubah."
          />
          <Keterangan label="User update" value={editing.user_update || '—'} />
          <Keterangan
            label="Tanggal update harga"
            value={tanggalWIB(editing.tanggal_update_harga)}
          />
        </div>
      )}

      <Group title="Identitas sparepart">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="nomor_sparepart"
            label="Nomor Sparepart"
            type="text"
            maxLength={MAX.nomor}
            hint="Wajib, dan tidak boleh sama dengan sparepart lain."
            error={errors.nomor_sparepart?.message}
            disabled={isSaving}
            {...register('nomor_sparepart')}
          />
          <Field
            id="nama_sparepart"
            label="Nama Sparepart"
            type="text"
            maxLength={MAX.nama}
            hint="Wajib, dan tidak boleh sama dengan sparepart lain."
            error={errors.nama_sparepart?.message}
            disabled={isSaving}
            {...register('nama_sparepart')}
          />
          <Field
            id="kode_sparepart"
            label="Kode Sparepart"
            type="text"
            maxLength={MAX.kode}
            hint="Wajib, dan tidak boleh sama dengan sparepart lain."
            error={errors.kode_sparepart?.message}
            disabled={isSaving}
            {...register('kode_sparepart')}
          />
          {/*
            Harga jual dirender `pxCurrency` di Pega, dan di sini tetap isian teks biasa:
            nilainya dikirim sebagai TEKS supaya tidak pernah melewati IEEE-754 double
            (`I-12`, `D-51`). Pemformatan ribuan sengaja tidak dipasang — ia akan mengubah
            isi kotak saat pengguna sedang mengetik.
          */}
          <Field
            id="harga_jual"
            label="Harga Jual (Rp)"
            type="text"
            inputMode="decimal"
            maxLength={MAX.angka}
            hint="Wajib. Tulis angkanya saja, tanpa titik ribuan."
            error={errors.harga_jual?.message}
            disabled={isSaving}
            {...register('harga_jual')}
          />
        </div>
      </Group>

      <Group title="Penggolongan">
        {options.isError && (
          <ErrorMessage
            title="Daftar Kategori dan Tipe tidak dapat dimuat"
            description="Keduanya dibaca dari basis data entitas ini. Sparepart tetap dapat disimpan tanpa keduanya; muat ulang halaman untuk mencoba lagi."
            tone="gangguan"
          />
        )}

        <div className="grid gap-4 sm:grid-cols-2">
          {/*
            Pega merendernya `pxAutoComplete` atas daftar yang dimuat penuh sekali saat layar
            dibuka (`Activity/BrowseTipeKategoriPart`). Dengan daftar sependek ini, dropdown
            menyampaikan hal yang sama dan tidak menuntut pengguna menebak kata kunci.

            Yang ditawarkan HANYA kategori dan tipe yang sudah disetujui — persis penyaring
            APPROVAL='1' pada rule yang dipanggil layar itu.
          */}
          <Controller
            control={control}
            name="kategori_sparepart"
            render={({ field }) => (
              <SelectField
                id="kategori_sparepart"
                label="Kategori Sparepart"
                options={categoryOptions}
                error={errors.kategori_sparepart?.message}
                disabled={isSaving || options.isPending}
                value={field.value}
                onBlur={field.onBlur}
                onChange={(event) => {
                  field.onChange(event)
                  // Tipe yang sudah dipilih dibuang saat kategorinya berganti: tipe milik
                  // kategori lama tidak boleh tertinggal pada kategori yang baru — dan
                  // daftar pilihannya pun sudah tidak memuatnya lagi.
                  setValue('tipe_sparepart', '')
                }}
              />
            )}
          />

          <Controller
            control={control}
            name="tipe_sparepart"
            render={({ field }) => (
              <SelectField
                id="tipe_sparepart"
                label="Tipe Sparepart"
                options={typeOptions}
                emptyText={chosenCategory ? '— pilih —' : '— pilih kategori dulu —'}
                error={errors.tipe_sparepart?.message}
                disabled={isSaving || options.isPending}
                {...field}
              />
            )}
          />
        </div>
      </Group>

      <Group title="Dimensi dan stok">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Angka
            id="berat"
            label="Berat Sparepart (gram)"
            error={errors.berat?.message}
            disabled={isSaving}
            {...register('berat')}
          />
          <Angka
            id="panjang"
            label="Panjang Sparepart (cm)"
            error={errors.panjang?.message}
            disabled={isSaving}
            {...register('panjang')}
          />
          <Angka
            id="lebar"
            label="Lebar Sparepart (cm)"
            error={errors.lebar?.message}
            disabled={isSaving}
            {...register('lebar')}
          />
          <Angka
            id="tinggi"
            label="Tinggi Sparepart (cm)"
            error={errors.tinggi?.message}
            disabled={isSaving}
            {...register('tinggi')}
          />
          <Angka
            id="stock_minimal"
            label="Stock Minimal"
            error={errors.stock_minimal?.message}
            disabled={isSaving}
            {...register('stock_minimal')}
          />
          <Angka
            id="stock_maximal"
            label="Stock Maximal"
            error={errors.stock_maximal?.message}
            disabled={isSaving}
            {...register('stock_maximal')}
          />
          <Angka
            id="kuantitas_pesanan"
            label="Kuantitas Pesanan"
            error={errors.kuantitas_pesanan?.message}
            disabled={isSaving}
            {...register('kuantitas_pesanan')}
          />
        </div>
      </Group>

      <Group title="Keterangan lain">
        <div className="grid gap-4 sm:grid-cols-2">
          {/*
            Tanggal Produksi adalah isian TEKS, bukan pemilih tanggal — Pega merendernya
            `pxTextInput` tanpa `pyDateTimeFormat` sama sekali. Menggantinya dengan pemilih
            tanggal akan menolak nilai lama yang bentuknya berbeda, dan tipe kolomnya sendiri
            belum diketahui (R-08).
          */}
          <Field
            id="tanggal_produksi"
            label="Tanggal Produksi"
            type="text"
            maxLength={MAX.tanggal}
            hint="Isian bebas, mengikuti layar lama."
            error={errors.tanggal_produksi?.message}
            disabled={isSaving}
            {...register('tanggal_produksi')}
          />
          <Field
            id="part_substitusi"
            label="Part Substitusi"
            type="text"
            maxLength={MAX.nama}
            hint="Nama sparepart penggantinya. Tidak diperiksa keberadaannya."
            error={errors.part_substitusi?.message}
            disabled={isSaving}
            {...register('part_substitusi')}
          />
        </div>
      </Group>

      <Group title="Penanda">
        {/*
          Keempat penanda ini dirender pxRadioButtons atau pxDropdown di Pega, dan daftar
          pilihannya ada di rule Field Value yang TIDAK ikut di export (R-16).

          Bentuknya ditiru apa adanya — radio untuk Jenis, dropdown untuk tiga lainnya —
          tetapi isinya diambil dari NILAI YANG SUDAH DIPAKAI BARIS LAIN, bukan dari daftar
          yang dikarang. Pilihan "Lainnya…" selalu tersedia; lihat ChoiceField.
        */}
        <p className="text-xs text-slate-500">
          Daftar pilihan keempat penanda ini tidak ada di export Pega. Yang ditawarkan adalah
          nilai yang sudah dipakai sparepart lain pada entitas ini; pilih “Lainnya…” untuk
          mengetik nilai baru.
        </p>

        <div className="grid gap-4 sm:grid-cols-2">
          <Controller
            control={control}
            name="jenis_sparepart"
            render={({ field }) => (
              <ChoiceField
                label="Jenis Sparepart"
                variant="radio"
                value={field.value}
                known={knownValues.jenis_sparepart ?? []}
                maxLength={MAX.penanda}
                disabled={isSaving}
                error={errors.jenis_sparepart?.message}
                onChange={field.onChange}
              />
            )}
          />
          <Controller
            control={control}
            name="satuan"
            render={({ field }) => (
              <ChoiceField
                label="Satuan"
                variant="dropdown"
                value={field.value}
                known={knownValues.satuan ?? []}
                maxLength={MAX.penanda}
                disabled={isSaving}
                error={errors.satuan?.message}
                onChange={field.onChange}
              />
            )}
          />
          <Controller
            control={control}
            name="status_aktif"
            render={({ field }) => (
              <ChoiceField
                label="Status Aktif"
                variant="dropdown"
                value={field.value}
                known={knownValues.status_aktif ?? []}
                maxLength={MAX.penanda}
                disabled={isSaving}
                error={errors.status_aktif?.message}
                onChange={field.onChange}
              />
            )}
          />
          <Controller
            control={control}
            name="status_sparepart"
            render={({ field }) => (
              <ChoiceField
                label="Status Sparepart"
                variant="dropdown"
                value={field.value}
                known={knownValues.status_sparepart ?? []}
                maxLength={MAX.penanda}
                disabled={isSaving}
                error={errors.status_sparepart?.message}
                onChange={field.onChange}
              />
            )}
          />
        </div>
      </Group>

      <p className="text-xs text-slate-500">
        Baris yang disimpan selalu kembali ke <span className="font-medium">Waiting Approval</span>{' '}
        dan menunggu persetujuan — persetujuan sebelumnya tidak berlaku atas isi yang sudah
        berubah.
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
 * masuk ke salah satu isian di dalamnya.
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
 * Angka adalah Field dengan papan ketik angka pada perangkat sentuh.
 *
 * `inputMode="decimal"` dan bukan `type="number"`: yang kedua menolak koma sebagai pemisah
 * desimal pada sebagian peramban, dan membuang nilai yang tidak dapat diuraikannya tanpa
 * satu pun pesan — persis kelas kegagalan senyap yang sedang dihindari modul ini.
 */
const Angka = ({
  id,
  label,
  error,
  disabled,
  ...rest
}: React.ComponentProps<typeof Field>) => (
  <Field
    id={id}
    label={label}
    type="text"
    inputMode="decimal"
    maxLength={MAX.angka}
    error={error}
    disabled={disabled}
    {...rest}
  />
)

/** Keterangan baca-saja pada kepala form mode ubah. */
function Keterangan({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div>
      <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </span>
      <span className="mt-0.5 block text-sm text-slate-900">{value}</span>
      {hint && <span className="mt-1 block text-xs text-slate-500">{hint}</span>}
    </div>
  )
}

/**
 * tanggalWIB menampilkan stempel UTC dari server dalam waktu Jakarta.
 *
 * Konversi terjadi DI SINI, di satu tempat, lewat Intl — tidak ada satu pun penambahan 7 jam
 * manual (`F-5`, `08-TECHNICAL-STRATEGY.md` §4.4).
 */
function tanggalWIB(value: string | undefined): string {
  if (!value) return 'Belum pernah'

  const at = new Date(value)
  if (Number.isNaN(at.getTime())) return value

  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Asia/Jakarta',
  }).format(at)
}
