import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useId, useMemo, useState } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, GroupingErrorCode, type Grouping, type GroupingInput } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'

import { useGroupingOptions, useGroupingPart, useGroupingSides } from './api'

/**
 * Batas panjang harus sama dengan konstanta di
 * `internal/mastergroupingsparepart/mastergroupingsparepart.go`.
 *
 * SELURUHNYA ASUMSI YANG DISADARI: DDL kedua tabelnya belum diterima (R-08), dan sistem lama
 * tidak memeriksa panjang satu pun isian — tidak ada satu pun `pyMaxLength` pada layarnya.
 * Batasnya dipasang supaya penolakan datang sebagai kalimat yang menuntun, bukan sebagai
 * ORA-12899.
 *
 * Bila salah satu berubah, KEDUA tempat harus ikut berubah. Uji
 * `TestLengthLimitsAreTheOnesTheFormRepeats` di backend menjaga duplikasinya tetap terlihat.
 */
const MAX = {
  nomor: 50,
  panel: 100,
  rangka: 50,
  sisi: 30,
  kendaraan: 100,
  catatan: 500,
} as const

const wajib = (max: number, label: string) =>
  z
    .string()
    .trim()
    .min(1, `${label} wajib diisi.`)
    .max(max, `${label} paling panjang ${max} karakter.`)

const schema = z
  .object({
    // Keempat isian wajib adalah persis anggota kunci alami. Bukan karena layar lama
    // menandainya `pyRequired` — ia tidak menandai satu pun — melainkan karena kunci yang
    // salah satu anggotanya kosong tidak dapat membedakan dua baris.
    nomor_sparepart: wajib(MAX.nomor, 'Nomor sparepart'),
    nama_panel: wajib(MAX.panel, 'Nama panel'),
    no_rangka: wajib(MAX.rangka, 'No rangka'),
    sisi: wajib(MAX.sisi, 'Sisi'),

    id_panel: z.string().trim().max(MAX.panel),
    tipe_kendaraan: z
      .string()
      .trim()
      .max(MAX.kendaraan, `Tipe kendaraan paling panjang ${MAX.kendaraan} karakter.`),
    grouping_dengan_no_rangka: z
      .string()
      .trim()
      .max(MAX.rangka, `Grouping dengan no rangka paling panjang ${MAX.rangka} karakter.`),
    catatan: z
      .string()
      .trim()
      .max(MAX.catatan, `Catatan paling panjang ${MAX.catatan} karakter.`),
  })
  // Menggabungkan baris dengan nomor rangkanya sendiri ditolak — SELISIH YANG DIRENCANAKAN
  // terhadap sistem lama, yang tidak memeriksanya sama sekali.
  .superRefine((values, ctx) => {
    const target = values.grouping_dengan_no_rangka.trim().toUpperCase()
    const own = values.no_rangka.trim().toUpperCase()
    if (target !== '' && target === own) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['grouping_dengan_no_rangka'],
        message:
          'Nomor rangka ini sama dengan No Rangka baris ini sendiri. ' +
          'Kosongkan bila ingin membuka grup baru.',
      })
    }
  })

export type GroupingFields = z.infer<typeof schema>

/** Nilai yang dikirim ke pemanggil saat form disimpan. */
export type GroupingFormValues = GroupingInput

type Props = {
  /**
   * Baris yang sedang disunting, atau null bila menambah.
   *
   * Mode hanya menentukan judul dan teks tombolnya — seluruh isian dapat diubah pada kedua
   * mode, sama seperti layar lama yang memakai form yang sama untuk keduanya.
   */
  editing: Grouping | null

  isSaving: boolean
  error: unknown
  onSave: (values: GroupingFormValues) => void
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
      case GroupingErrorCode.duplicate:
        return {
          title: 'Data sudah ada',
          description:
            'Kombinasi nomor sparepart, nama panel, no rangka, dan sisi ini sudah dipakai grouping lain. ' +
            'Ubah salah satunya, atau buka grouping yang sudah ada.',
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
  'id_panel',
  'nama_panel',
  'sisi',
  'no_rangka',
  'tipe_kendaraan',
  'grouping_dengan_no_rangka',
  'catatan',
] as const

/**
 * GroupingForm adalah form tambah dan ubah Master Grouping Sparepart.
 *
 * Kesebelas isiannya mengikuti caption
 * `Section/MasterGroupingSparepartHEApproval-Section.xml` apa adanya, pada urutan yang sama
 * (`D-13`) — termasuk ejaan "Kategory Sparepart" dengan y.
 *
 * # Lima isian yang TIDAK diketik pengguna
 *
 * Nama Sparepart, Kategory Sparepart, Type Sparepart, Kode Sparepart, dan Tanggal Produksi
 * seluruhnya DIISI OTOMATIS begitu Nomor Sparepart selesai diketik — meniru
 * `Activity/SetDataSparepart-Act.xml`, yang menyalin kelimanya dari Master Sparepart dan
 * menolak dengan "Data Sparepart tidak ditemukan" bila nomornya tidak ada.
 *
 * Kelimanya digambar BACA-SAJA, bukan disembunyikan: itulah yang memberitahu pengguna bahwa
 * nomor yang diketiknya sudah benar sebelum ia menekan Simpan.
 *
 * # Tiga isian layar lama yang TIDAK ada di sini
 *
 * **ID.** Diterbitkan server (`PEGA_M_GROUPING_SPAREPART_HE.prc:11`), tidak pernah diketik.
 * Pada mode ubah ia ditampilkan sebagai keterangan.
 *
 * **Nomor grup.** Diterbitkan atau diwarisi penyimpanan. Pada mode ubah ia ditampilkan
 * sebagai keterangan — di layar lama ia tidak digambar sama sekali, dan tanpa itu pengguna
 * tidak punya cara melihat bahwa penggabungannya berhasil.
 *
 * **ID Panel.** Isian tersembunyi yang diisi pilihan Nama Panel, persis seperti
 * `pySetValueOnSelect` pada autocomplete Pega.
 */
export function GroupingForm({ editing, isSaving, error, onSave, onCancel }: Props) {
  const isEditing = editing !== null
  const headingId = useId()

  const options = useGroupingOptions()

  const {
    register,
    handleSubmit,
    setError,
    setValue,
    control,
    formState: { errors },
  } = useForm<GroupingFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nomor_sparepart: editing?.nomor_sparepart ?? '',
      id_panel: editing?.id_panel ?? '',
      nama_panel: editing?.nama_panel ?? '',
      sisi: editing?.sisi ?? '',
      no_rangka: editing?.no_rangka ?? '',
      tipe_kendaraan: editing?.tipe_kendaraan ?? '',
      grouping_dengan_no_rangka: editing?.grouping_dengan_no_rangka ?? '',
      catatan: editing?.catatan ?? '',
    },
  })

  const chosenPanelID = useWatch({ control, name: 'id_panel' })
  const chosenPanelName = useWatch({ control, name: 'nama_panel' })

  // Daftar Sisi BARU dibaca setelah panelnya dipilih — persis seperti
  // `Activity/GetSisiPanel-Act.xml` yang dipanggil belakangan, bukan saat layar dibuka.
  const sides = useGroupingSides(chosenPanelID ?? '', chosenPanelName ?? '')

  /*
    Nomor sparepart yang SUDAH selesai diketik.

    Ia terpisah dari nilai isiannya, dan itu yang membuat pencariannya berjalan saat isian
    kehilangan fokus alih-alih pada setiap ketikan — meniru `SetDataSparepart` yang dipicu
    `onBlur`. Menjalankannya per ketikan berarti satu permintaan basis data untuk setiap
    huruf, dan pesan "tidak ditemukan" yang berkedip sepanjang pengguna mengetik.
  */
  const [settledNumber, setSettledNumber] = useState(editing?.nomor_sparepart ?? '')
  const part = useGroupingPart(settledNumber)

  const panelOptions = useMemo(
    () => (options.data?.panel ?? []).map((one) => ({ value: one.kode, label: one.nama })),
    [options.data],
  )

  const vehicleOptions = useMemo(
    // Yang DISIMPAN adalah namanya, bukan kodenya: autocomplete Pega menulis ke kolom yang
    // sama dengan yang ditampilkan kembali oleh daftar.
    () =>
      (options.data?.tipe_kendaraan ?? []).map((one) => ({ value: one.nama, label: one.nama })),
    [options.data],
  )

  const sideOptions = useMemo(
    () => (sides.data?.sisi ?? []).map((one) => ({ value: one.kode, label: one.nama })),
    [sides.data],
  )

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

  // Kelima isian turunan: hasil pencarian bila ada, jika tidak nilai baris yang sedang
  // disunting. Urutan itu penting — begitu nomornya diganti, yang tampil harus nilai BARU,
  // bukan sisa nilai lama yang menyesatkan.
  const derived = part.data ?? {
    nama_sparepart: editing?.nama_sparepart ?? '',
    kategori_sparepart: editing?.kategori_sparepart ?? '',
    tipe_sparepart: editing?.tipe_sparepart ?? '',
    kode_sparepart: editing?.kode_sparepart ?? '',
    tanggal_produksi: editing?.tanggal_produksi ?? '',
  }

  const partNotFound =
    part.isError &&
    part.error instanceof APIError &&
    part.error.kode === GroupingErrorCode.partNotFound

  return (
    <form
      onSubmit={handleSubmit(onSave)}
      noValidate
      aria-labelledby={headingId}
      className="space-y-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
    >
      <h2 id={headingId} className="text-base font-semibold text-slate-900">
        {isEditing
          ? `Ubah grouping ${editing.nomor_sparepart} — ${editing.nama_panel}`
          : 'Tambah Master Grouping Sparepart'}
      </h2>

      {message && (
        <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
      )}

      {isEditing && (
        <div className="grid gap-3 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2.5 sm:grid-cols-2">
          <Keterangan
            label="ID grouping"
            value={editing.id_grouping}
            hint="Diterbitkan sistem. Tidak dapat diubah."
          />
          <Keterangan
            label="Nomor grup kendaraan"
            value={editing.nomor_grup || '—'}
            hint="Baris dengan nomor grup yang sama terpasang di kendaraan yang sama."
          />
        </div>
      )}

      <Group title="Sparepart">
        <div className="grid gap-4 sm:grid-cols-2">
          {/*
            Nomor Sparepart adalah SATU-SATUNYA identitas sparepart yang diketik. Pencariannya
            berjalan saat isian kehilangan fokus, meniru `SetDataSparepart` yang dipicu onBlur.
          */}
          <Field
            id="nomor_sparepart"
            label="Nomor Sparepart"
            type="text"
            maxLength={MAX.nomor}
            hint="Wajib. Nama, kategori, dan tipenya terisi sendiri dari Master Sparepart."
            error={errors.nomor_sparepart?.message}
            disabled={isSaving}
            {...register('nomor_sparepart', {
              onBlur: (event) => setSettledNumber(event.target.value.trim()),
            })}
          />

          <Turunan label="Nama Sparepart" value={derived.nama_sparepart} />
          <Turunan label="Kategory Sparepart" value={derived.kategori_sparepart} />
          <Turunan label="Type Sparepart" value={derived.tipe_sparepart} />
        </div>

        {partNotFound && (
          <ErrorMessage
            title="Data Sparepart tidak ditemukan"
            description="Periksa nomornya di Master Sparepart. Grouping tidak dapat disimpan sebelum nomornya benar."
            tone="penolakan"
          />
        )}

        {part.isFetching && (
          <p className="text-xs text-slate-500">Mencari data sparepart…</p>
        )}
      </Group>

      <Group title="Panel">
        {options.isError && (
          <ErrorMessage
            title="Daftar Panel dan Tipe Kendaraan tidak dapat dimuat"
            description="Keduanya dibaca dari basis data entitas ini. Muat ulang halaman untuk mencoba lagi."
            tone="gangguan"
          />
        )}

        <div className="grid gap-4 sm:grid-cols-2">
          {/*
            Pega merendernya `pxAutoComplete` atas `BrowseMasterPanel_HE_RD` dan menyalin
            `.ID_PANEL` ke isian tersembunyi lewat `pySetValueOnSelect`. Di sini dropdown yang
            dipakai: daftarnya pendek, dan ia tidak menuntut pengguna menebak kata kunci.

            Yang ditawarkan HANYA panel yang sudah disetujui — persis penyaring APPROVAL='1'
            pada parameter report definition-nya.
          */}
          <Controller
            control={control}
            name="id_panel"
            render={({ field }) => (
              <SelectField
                id="id_panel"
                label="Nama Panel"
                options={panelOptions}
                error={errors.nama_panel?.message ?? errors.id_panel?.message}
                disabled={isSaving || options.isPending}
                value={field.value}
                onBlur={field.onBlur}
                onChange={(event) => {
                  field.onChange(event)
                  // Nama panelnya ikut disimpan, bukan hanya kodenya: NAMA_PANEL yang menjadi
                  // anggota kunci alami, dan ia pula yang dipakai mencari daftar Sisi.
                  const chosen = options.data?.panel.find((one) => one.kode === event.target.value)
                  setValue('nama_panel', chosen?.nama ?? '', { shouldValidate: true })
                  // Sisi yang sudah dipilih dibuang saat panelnya berganti: sisi milik panel
                  // lama tidak boleh tertinggal pada panel yang baru — dan daftar pilihannya
                  // pun sudah tidak memuatnya lagi.
                  setValue('sisi', '', { shouldValidate: false })
                }}
              />
            )}
          />

          <Controller
            control={control}
            name="sisi"
            render={({ field }) => (
              <SelectField
                id="sisi"
                label="Sisi"
                options={sideOptions}
                emptyText={chosenPanelID ? '— pilih —' : '— pilih panel dulu —'}
                error={errors.sisi?.message}
                disabled={isSaving || !chosenPanelID || sides.isPending}
                {...field}
              />
            )}
          />
        </div>

        {chosenPanelID && !sides.isPending && sideOptions.length === 0 && (
          <p className="text-xs text-amber-700">
            Panel ini belum punya sisi yang terdaftar. Lengkapi lokasinya di Master Panel lebih
            dulu — Sisi wajib diisi.
          </p>
        )}
      </Group>

      <Group title="Kendaraan">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="no_rangka"
            label="No Rangka"
            type="text"
            maxLength={MAX.rangka}
            hint="Wajib. Nomor rangka kendaraan tempat sparepart ini terpasang."
            error={errors.no_rangka?.message}
            disabled={isSaving}
            {...register('no_rangka')}
          />

          <Controller
            control={control}
            name="tipe_kendaraan"
            render={({ field }) => (
              <SelectField
                id="tipe_kendaraan"
                label="Tipe Kendaraan"
                options={vehicleOptions}
                error={errors.tipe_kendaraan?.message}
                disabled={isSaving || options.isPending}
                {...field}
              />
            )}
          />

          {/*
            Isinya sebuah NOMOR RANGKA, bukan nomor grup — dan itu yang membuat labelnya
            terbaca benar. Kosong berarti baris ini membuka grup sendiri.
          */}
          <Field
            id="grouping_dengan_no_rangka"
            label="Grouping Dengan No Rangka"
            type="text"
            maxLength={MAX.rangka}
            hint="Isi nomor rangka yang sudah terdaftar untuk menggabungkan ke grupnya. Kosongkan untuk membuka grup baru."
            error={errors.grouping_dengan_no_rangka?.message}
            disabled={isSaving}
            {...register('grouping_dengan_no_rangka')}
          />
        </div>
      </Group>

      <Group title="Keterangan">
        <TextAreaField
          id="catatan"
          label="Catatan"
          rows={3}
          maxLength={MAX.catatan}
          error={errors.catatan?.message}
          disabled={isSaving}
          {...register('catatan')}
        />
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
 * Turunan adalah isian BACA-SAJA yang nilainya datang dari Master Sparepart.
 *
 * Ia digambar sebagai teks, bukan sebagai kotak isian yang dimatikan: kotak yang dimatikan
 * mengundang pengguna mencari cara menyalakannya, sedangkan teks menyatakan bahwa nilainya
 * memang bukan miliknya untuk diisi.
 */
function Turunan({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="block text-sm font-medium text-slate-700">{label}</span>
      <p className="mt-1 rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-900">
        {value || <span className="text-slate-400">— terisi dari Master Sparepart —</span>}
      </p>
    </div>
  )
}

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
