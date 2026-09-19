import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { GalatAPI, GalatJaringan } from '@/api/klien'
import { KodeGalat, ClaimReportErrorCode, type ClaimReport } from '@/api/tipe'
import { KolomIsian } from '@/components/KolomIsian'
import { TextAreaField } from '@/components/TextAreaField'
import { PesanGalat, type NadaGalat } from '@/components/PesanGalat'
import { Tombol } from '@/components/Tombol'

import { useSaveReport, type ReportFormValues } from './api'

/** Batas panjang; sama dengan konstanta di paket pelaporanklaim pada backend. */
const MAX_LENGTH = {
  name: 100,
  email: 100,
  phone: 30,
  policyNumber: 50,
  code: 20,
  longText: 4000,
} as const

/** Bentuk tanggal yang diterima API: `YYYY-MM-DD`. */
const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/

/**
 * Bentuk nilai uang yang diterima: angka, paling banyak dua angka desimal.
 *
 * Pemisah ribuan sengaja DITOLAK, bukan dibuang diam-diam: "1.500" berarti seribu lima
 * ratus bagi sebagian orang dan satu koma lima bagi yang lain. Menebaknya berarti salah
 * pada separuh kasus, dan salahnya seratus kali lipat. Ekspresi ini sama artinya dengan
 * `pelaporanklaim.LooksLikeMoney` di Go dan dengan CHECK pada migrasi 0003.
 */
const MONEY_PATTERN = /^\d{1,16}(\.\d{1,2})?$/

/**
 * Bentuk alamat surel yang diterima — sengaja LONGGAR.
 *
 * Satu-satunya cara membuktikan sebuah alamat sah adalah mengirim surel ke sana;
 * pemeriksaan yang ketat hanya menolak alamat sah yang bentuknya tidak biasa. Yang
 * ditangkap hanya salah ketik yang jelas, sama persis dengan `pelaporanklaim.LooksLikeEmail`
 * di Go — bila keduanya berbeda ketat, layar akan menerima alamat yang kemudian ditolak
 * server, atau sebaliknya.
 */
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** Jumlah dokumen: bilangan bulat, paling banyak lima digit — sesuai kolom NUMBER(5). */
const COUNT_PATTERN = /^\d{1,5}$/

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Duplikasi yang DISENGAJA. Yang di sini menjawab pengguna tanpa perjalanan jaringan; yang
 * di sana adalah yang menegakkan — karena pemanggilan langsung ke API tidak melewati layar
 * ini sama sekali. Menghapus salah satunya berarti memilih antara layar yang lamban atau
 * API yang tidak terjaga.
 *
 * # Kenapa hanya nama pelapor yang wajib
 *
 * Bukan kelonggaran, melainkan sifat pekerjaannya. Laporan kerugian datang lewat telepon
 * dan surel dengan kelengkapan yang berbeda-beda, dan petugas harus dapat mencatatnya
 * SEKARANG lalu melengkapinya kemudian. Layar Pega pun tidak mewajibkan satu field pun.
 * Kelengkapan yang sesungguhnya ditegakkan saat REGISTRASI.
 */
const schema = z.object({
  nama_pelapor: z
    .string()
    .trim()
    .min(1, 'Nama pelapor wajib diisi.')
    .max(MAX_LENGTH.name, `Nama pelapor paling panjang ${MAX_LENGTH.name} karakter.`),
  email_pengirim: z
    .string()
    .trim()
    .max(MAX_LENGTH.email, `Email pengirim paling panjang ${MAX_LENGTH.email} karakter.`)
    .refine((v) => v === '' || EMAIL_PATTERN.test(v), {
      message: 'Email pengirim tidak berbentuk alamat surel.',
    }),
  telepon_pengirim: z
    .string()
    .trim()
    .max(MAX_LENGTH.phone, `Paling panjang ${MAX_LENGTH.phone} karakter.`),
  nama_kurir: z.string().trim().max(MAX_LENGTH.name, `Paling panjang ${MAX_LENGTH.name} karakter.`),
  subjek_email: z
    .string()
    .trim()
    .max(MAX_LENGTH.name, `Paling panjang ${MAX_LENGTH.name} karakter.`),

  nomor_polis: z
    .string()
    .trim()
    .max(MAX_LENGTH.policyNumber, `Paling panjang ${MAX_LENGTH.policyNumber} karakter.`),
  nama_tertanggung: z
    .string()
    .trim()
    .max(MAX_LENGTH.name, `Paling panjang ${MAX_LENGTH.name} karakter.`),
  email_tertanggung: z
    .string()
    .trim()
    .max(MAX_LENGTH.email, `Paling panjang ${MAX_LENGTH.email} karakter.`)
    .refine((v) => v === '' || EMAIL_PATTERN.test(v), {
      message: 'Email tertanggung tidak berbentuk alamat surel.',
    }),
  kode_bisnis: z.string().trim().max(MAX_LENGTH.code, `Paling panjang ${MAX_LENGTH.code} karakter.`),
  group_panel: z.string().trim().max(MAX_LENGTH.code, `Paling panjang ${MAX_LENGTH.code} karakter.`),
  nomor_referensi: z
    .string()
    .trim()
    .max(MAX_LENGTH.policyNumber, `Paling panjang ${MAX_LENGTH.policyNumber} karakter.`),

  tanggal_kejadian: z
    .string()
    .trim()
    .refine((v) => v === '' || DATE_PATTERN.test(v), { message: 'Isi tanggal yang sah.' }),
  lokasi_kejadian: z.string().trim().max(MAX_LENGTH.longText, 'Terlalu panjang.'),
  kronologi: z.string().trim().max(MAX_LENGTH.longText, 'Terlalu panjang.'),
  rincian_kerusakan: z.string().trim().max(MAX_LENGTH.longText, 'Terlalu panjang.'),
  sim_pengendara: z
    .string()
    .trim()
    .max(MAX_LENGTH.policyNumber, `Paling panjang ${MAX_LENGTH.policyNumber} karakter.`),
  nilai_estimasi: z
    .string()
    .trim()
    .refine((v) => v === '' || MONEY_PATTERN.test(v), {
      message: 'Isi angka saja, tanpa titik ribuan. Contoh: 450000000 atau 12.50',
    }),
  tipe_klaim: z.string().trim().max(MAX_LENGTH.code, `Paling panjang ${MAX_LENGTH.code} karakter.`),

  // Disimpan sebagai TEKS di skema, bukan angka.
  //
  // `<input type="number">` mengembalikan string, dan memakai `z.coerce.number()` membuat
  // tipe masukan dan tipe keluaran skema berbeda — yang berarti React Hook Form dan Zod
  // bertengkar soal tipe `defaultValues`. Mengubahnya menjadi angka satu kali saat
  // mengirim jauh lebih sederhana daripada memaksa keduanya sepakat.
  jumlah_dokumen: z
    .string()
    .trim()
    .refine((v) => v === '' || COUNT_PATTERN.test(v), {
      message: 'Isi bilangan bulat 0 atau lebih.',
    }),
  tanggal_terima_dokumen: z
    .string()
    .trim()
    .refine((v) => v === '' || DATE_PATTERN.test(v), { message: 'Isi tanggal yang sah.' }),

  alasan_belum_transfer: z.string().trim().max(MAX_LENGTH.longText, 'Terlalu panjang.'),
  catatan_belum_registrasi: z.string().trim().max(MAX_LENGTH.longText, 'Terlalu panjang.'),
})

type FormValues = z.infer<typeof schema>

type Props = {
  /** Null berarti mencatat laporan baru; terisi berarti mengubah laporan itu. */
  report: ClaimReport | null
  onClose: () => void
}

function initialValues(report: ClaimReport | null): FormValues {
  return {
    nama_pelapor: report?.nama_pelapor ?? '',
    email_pengirim: report?.email_pengirim ?? '',
    telepon_pengirim: report?.telepon_pengirim ?? '',
    nama_kurir: report?.nama_kurir ?? '',
    subjek_email: report?.subjek_email ?? '',
    nomor_polis: report?.nomor_polis ?? '',
    nama_tertanggung: report?.nama_tertanggung ?? '',
    email_tertanggung: report?.email_tertanggung ?? '',
    kode_bisnis: report?.kode_bisnis ?? '',
    group_panel: report?.group_panel ?? '',
    nomor_referensi: report?.nomor_referensi ?? '',
    tanggal_kejadian: report?.tanggal_kejadian ?? '',
    lokasi_kejadian: report?.lokasi_kejadian ?? '',
    kronologi: report?.kronologi ?? '',
    rincian_kerusakan: report?.rincian_kerusakan ?? '',
    sim_pengendara: report?.sim_pengendara ?? '',
    nilai_estimasi: report?.nilai_estimasi ?? '',
    tipe_klaim: report?.tipe_klaim ?? '',
    jumlah_dokumen: String(report?.jumlah_dokumen ?? 0),
    tanggal_terima_dokumen: report?.tanggal_terima_dokumen ?? '',
    alasan_belum_transfer: report?.alasan_belum_transfer ?? '',
    catatan_belum_registrasi: report?.catatan_belum_registrasi ?? '',
  }
}

/**
 * Form catat dan ubah Laporan Klaim.
 *
 * Menggantikan flow action `InputReceiveDocument` beserta section
 * `ViewInputReceiveDocument_sec`.
 *
 * # Isinya dikelompokkan, bukan dijejer
 *
 * Tujuh belas isian dalam satu kolom panjang adalah bentuk yang membuat orang berhenti
 * membaca. Kelompoknya mengikuti urutan bertanya yang sebenarnya dipakai petugas saat
 * menerima telepon: siapa yang melapor, polis apa, apa yang terjadi, dokumennya
 * bagaimana.
 *
 * # Satu perbedaan yang disengaja dari layar lama
 *
 * Layar Pega mengunci seluruh isian begitu laporan ditransfer (`StatusLock`), TETAPI
 * kuncinya dapat dilewati operator berjabatan "PA" atau bercabang kantor pusat
 * (`Activity/Pre_ActReceiveDocument-Act.xml` step 3-4). Pengecualian itu tidak dibawa: ia
 * pagar kewenangan yang ditulis di dalam kode, persis yang `D-15` larang. Di sini yang
 * mengunci adalah REGISTRASI — sejak klaim terbit, yang berlaku adalah data klaimnya.
 */
export function ClaimReportForm({ report, onClose }: Props) {
  const save = useSaveReport()
  const isEditing = report !== null
  const firstField = useRef<HTMLInputElement | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: initialValues(report),
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan ketik
  // harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    firstField.current?.focus()
  }, [])

  const { ref: nameRef, ...nameRest } = register('nama_pelapor')

  function submit(values: FormValues) {
    // Satu-satunya tempat teks diubah menjadi angka. Isian kosong menjadi nol, bukan
    // NaN — `Number('')` memang nol, tetapi menuliskannya eksplisit membuat maksudnya
    // terbaca tanpa harus mengingat kekhasan itu.
    const payload: ReportFormValues = {
      ...values,
      jumlah_dokumen: values.jumlah_dokumen === '' ? 0 : Number(values.jumlah_dokumen),
    }
    save.mutate(isEditing ? { number: report.nomor, values: payload } : { values: payload }, {
      onSuccess: onClose,
    })
  }

  return (
    /*
      Panel muncul di atas tabel, bukan sebagai dialog melayang. Petugas sering perlu
      melihat laporan lain yang sudah ada untuk memastikan ia tidak mencatat laporan yang
      sama dua kali — dan dialog yang menutup layar justru menyembunyikan jawabannya.
    */
    <form
      onSubmit={handleSubmit(submit)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={isEditing ? 'Ubah laporan klaim' : 'Catat laporan klaim'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {isEditing ? `Ubah Laporan ${report.nomor}` : 'Catat Laporan Klaim'}
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          {isEditing
            ? 'Nomor laporan tetap. Perpindahan tahap punya tombolnya sendiri di daftar.'
            : 'Hanya nama pelapor yang wajib — laporan yang belum lengkap tetap dapat dicatat sekarang dan dilengkapi kemudian.'}
        </p>
      </div>

      <div className="space-y-6 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}

        <FieldGroup
          title="Pelapor"
          description="Siapa yang melaporkan, dan bagaimana menghubunginya."
        >
          <KolomIsian
            id="nama_pelapor"
            label="Nama pelapor"
            placeholder="Contoh: Bagas Prasetya"
            maxLength={MAX_LENGTH.name}
            autoComplete="off"
            galat={errors.nama_pelapor?.message}
            disabled={save.isPending}
            {...nameRest}
            ref={(element) => {
              nameRef(element)
              firstField.current = element
            }}
          />
          <KolomIsian
            id="email_pengirim"
            label="Email pengirim"
            type="email"
            placeholder="nama@perusahaan.co.id"
            maxLength={MAX_LENGTH.email}
            autoComplete="off"
            galat={errors.email_pengirim?.message}
            disabled={save.isPending}
            {...register('email_pengirim')}
          />
          <KolomIsian
            id="telepon_pengirim"
            label="Telepon pengirim"
            placeholder="021-5550101"
            maxLength={MAX_LENGTH.phone}
            autoComplete="off"
            galat={errors.telepon_pengirim?.message}
            disabled={save.isPending}
            {...register('telepon_pengirim')}
          />
          <KolomIsian
            id="nama_kurir"
            label="Kurir ASM"
            placeholder="Contoh: JNE"
            maxLength={MAX_LENGTH.name}
            autoComplete="off"
            petunjuk="Nama ekspedisi atau kurir yang membawa dokumennya."
            galat={errors.nama_kurir?.message}
            disabled={save.isPending}
            {...register('nama_kurir')}
          />
          <KolomIsian
            id="subjek_email"
            label="Subjek email"
            placeholder="Judul surel laporan yang masuk"
            maxLength={MAX_LENGTH.name}
            autoComplete="off"
            galat={errors.subjek_email?.message}
            disabled={save.isPending}
            {...register('subjek_email')}
          />
        </FieldGroup>

        <FieldGroup
          title="Polis dan tertanggung"
          description="Sebagaimana disebut pelapor. Data polis yang sah dibaca saat registrasi, bukan di sini — jadi nomor yang belum pasti tetap boleh dicatat."
        >
          <KolomIsian
            id="nomor_polis"
            label="Nomor polis"
            maxLength={MAX_LENGTH.policyNumber}
            autoComplete="off"
            galat={errors.nomor_polis?.message}
            disabled={save.isPending}
            {...register('nomor_polis')}
          />
          <KolomIsian
            id="nama_tertanggung"
            label="Nama tertanggung"
            maxLength={MAX_LENGTH.name}
            autoComplete="off"
            galat={errors.nama_tertanggung?.message}
            disabled={save.isPending}
            {...register('nama_tertanggung')}
          />
          <KolomIsian
            id="email_tertanggung"
            label="Email tertanggung"
            type="email"
            maxLength={MAX_LENGTH.email}
            autoComplete="off"
            galat={errors.email_tertanggung?.message}
            disabled={save.isPending}
            {...register('email_tertanggung')}
          />
          <KolomIsian
            id="nomor_referensi"
            label="Nomor referensi"
            maxLength={MAX_LENGTH.policyNumber}
            autoComplete="off"
            galat={errors.nomor_referensi?.message}
            disabled={save.isPending}
            {...register('nomor_referensi')}
          />
          <KolomIsian
            id="kode_bisnis"
            label="Kode bisnis"
            maxLength={MAX_LENGTH.code}
            autoComplete="off"
            galat={errors.kode_bisnis?.message}
            disabled={save.isPending}
            {...register('kode_bisnis')}
          />
          <KolomIsian
            id="group_panel"
            label="Group panel"
            maxLength={MAX_LENGTH.code}
            autoComplete="off"
            petunjuk="002 PA · 003/009 Aneka · 004 Marine · 005 Travel · 006 Fire."
            galat={errors.group_panel?.message}
            disabled={save.isPending}
            {...register('group_panel')}
          />
        </FieldGroup>

        <FieldGroup
          title="Kerugian yang dilaporkan"
          description="Apa yang terjadi, kapan, dan di mana."
        >
          <KolomIsian
            id="tanggal_kejadian"
            label="Tanggal kejadian"
            type="date"
            galat={errors.tanggal_kejadian?.message}
            disabled={save.isPending}
            {...register('tanggal_kejadian')}
          />
          <KolomIsian
            id="tipe_klaim"
            label="Tipe klaim"
            maxLength={MAX_LENGTH.code}
            autoComplete="off"
            galat={errors.tipe_klaim?.message}
            disabled={save.isPending}
            {...register('tipe_klaim')}
          />
          <KolomIsian
            id="nilai_estimasi"
            label="Estimasi kerugian"
            inputMode="decimal"
            placeholder="450000000"
            autoComplete="off"
            petunjuk="Angka saja, tanpa titik ribuan dan tanpa Rp."
            galat={errors.nilai_estimasi?.message}
            disabled={save.isPending}
            {...register('nilai_estimasi')}
          />
          <KolomIsian
            id="sim_pengendara"
            label="SIM pengendara"
            maxLength={MAX_LENGTH.policyNumber}
            autoComplete="off"
            galat={errors.sim_pengendara?.message}
            disabled={save.isPending}
            {...register('sim_pengendara')}
          />
          <div className="sm:col-span-2">
            <TextAreaField
              id="lokasi_kejadian"
              label="Lokasi kejadian"
              rows={2}
              maxLength={MAX_LENGTH.longText}
              error={errors.lokasi_kejadian?.message}
              disabled={save.isPending}
              {...register('lokasi_kejadian')}
            />
          </div>
          <div className="sm:col-span-2">
            <TextAreaField
              id="kronologi"
              label="Kronologi kejadian"
              rows={4}
              maxLength={MAX_LENGTH.longText}
              hint="Urutan kejadian menurut pelapor. Inilah yang dibaca ulang saat klaim diregistrasi."
              error={errors.kronologi?.message}
              disabled={save.isPending}
              {...register('kronologi')}
            />
          </div>
          <div className="sm:col-span-2">
            <TextAreaField
              id="rincian_kerusakan"
              label="Rincian kerusakan"
              rows={3}
              maxLength={MAX_LENGTH.longText}
              error={errors.rincian_kerusakan?.message}
              disabled={save.isPending}
              {...register('rincian_kerusakan')}
            />
          </div>
        </FieldGroup>

        <FieldGroup
          title="Dokumen dan catatan"
          description="Kelengkapan dokumen dan alasan bila laporan tertahan."
        >
          <KolomIsian
            id="jumlah_dokumen"
            label="Jumlah dokumen"
            type="number"
            min={0}
            galat={errors.jumlah_dokumen?.message}
            disabled={save.isPending}
            {...register('jumlah_dokumen')}
          />
          <KolomIsian
            id="tanggal_terima_dokumen"
            label="Tanggal terima dokumen"
            type="date"
            petunjuk="Kapan dokumennya benar-benar diterima."
            galat={errors.tanggal_terima_dokumen?.message}
            disabled={save.isPending}
            {...register('tanggal_terima_dokumen')}
          />
          <div className="sm:col-span-2">
            <TextAreaField
              id="alasan_belum_transfer"
              label="Alasan belum ditransfer"
              rows={2}
              maxLength={MAX_LENGTH.longText}
              hint="Diisi bila laporan tertahan di cabang."
              error={errors.alasan_belum_transfer?.message}
              disabled={save.isPending}
              {...register('alasan_belum_transfer')}
            />
          </div>
          <div className="sm:col-span-2">
            <TextAreaField
              id="catatan_belum_registrasi"
              label="Catatan belum diregistrasi"
              rows={2}
              maxLength={MAX_LENGTH.longText}
              hint="Diisi bila laporan sudah sampai pusat tetapi belum dapat menjadi klaim."
              error={errors.catatan_belum_registrasi?.message}
              disabled={save.isPending}
              {...register('catatan_belum_registrasi')}
            />
          </div>
        </FieldGroup>

        <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-5">
          <Tombol type="submit" nada="utama" disabled={save.isPending}>
            {save.isPending && <Spinner />}
            {save.isPending ? 'Menyimpan…' : 'Simpan'}
          </Tombol>
          <Tombol nada="halus" onClick={onClose} disabled={save.isPending}>
            Batal
          </Tombol>
        </div>
      </div>
    </form>
  )
}

/**
 * Satu kelompok isian beserta judulnya.
 *
 * `<fieldset>` dan `<legend>`, bukan `<div>` dan `<h4>`: pembaca layar menyebutkan legend
 * setiap kali kursor masuk ke isian di dalamnya, sehingga "Nama" pada kelompok Pelapor
 * terdengar berbeda dari "Nama" pada kelompok Tertanggung — tanpa perlu memanjangkan
 * labelnya bagi pengguna yang melihat.
 */
function FieldGroup({
  title,
  description,
  children,
}: {
  title: string
  description: string
  children: ReactNode
}) {
  return (
    <fieldset className="border-0 p-0">
      <legend className="mb-1 text-sm font-semibold text-slate-900">{title}</legend>
      <p className="mb-4 max-w-2xl text-xs leading-relaxed text-slate-500">{description}</p>
      <div className="grid gap-5 sm:grid-cols-2">{children}</div>
    </fieldset>
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
 * Teks bisa berubah kapan saja tanpa mengubah artinya; kode tidak.
 */
function SaveErrorMessage({ error }: { error: unknown }) {
  if (error instanceof GalatJaringan) {
    return (
      <PesanGalat
        judul="Tidak dapat menghubungi server"
        keterangan="Laporan belum tersimpan. Periksa koneksi lalu coba lagi."
        nada="gangguan"
      />
    )
  }

  if (!(error instanceof GalatAPI)) {
    return (
      <PesanGalat
        judul="Gagal menyimpan"
        keterangan="Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi."
        nada="gangguan"
      />
    )
  }

  const { judul, keterangan, nada } = describeError(error)
  return <PesanGalat judul={judul} keterangan={keterangan} nada={nada} />
}

function describeError(error: GalatAPI): {
  judul: string
  keterangan: string
  nada: NadaGalat
} {
  switch (error.kode) {
    case KodeGalat.validasiGagal:
      return {
        judul: 'Isian belum benar',
        // Pesan dari server dipakai apa adanya: ia yang tahu aturan mana yang dilanggar,
        // dan menerjemahkannya ulang di sini akan membuat keduanya dapat berbeda.
        keterangan: error.detail.map((d) => d.pesan).join(' ') || error.message,
        nada: 'penolakan',
      }

    case ClaimReportErrorCode.alreadyRegistered:
      return {
        judul: 'Laporan sudah menjadi klaim',
        keterangan:
          'Sejak klaimnya terbit, yang berlaku adalah data klaim — laporannya tidak dapat diubah lagi. Muat ulang daftarnya.',
        nada: 'penolakan',
      }

    case ClaimReportErrorCode.notFound:
      return {
        judul: 'Laporan tidak ditemukan',
        keterangan: 'Laporan ini mungkin sudah diubah orang lain. Muat ulang daftarnya.',
        nada: 'penolakan',
      }

    case ClaimReportErrorCode.numberTaken:
      return {
        judul: 'Nomor bentrok',
        keterangan: 'Nomor yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.',
        nada: 'gangguan',
      }

    default:
      return { judul: 'Gagal menyimpan', keterangan: error.message, nada: 'gangguan' }
  }
}
