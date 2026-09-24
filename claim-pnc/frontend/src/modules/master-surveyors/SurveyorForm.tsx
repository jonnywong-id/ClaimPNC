import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Surveyor, type SurveyorType } from '@/api/types'
import { Button } from '@/components/Button'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { useSaveSurveyor } from './api'

/**
 * Batas panjang nama; sama dengan mastersurveyors.MaxNameLength di backend.
 *
 * Ia ASUMSI, bukan bacaan: panjang kolom POOLDATA.D_SURVEYORS.NAME belum pernah diperiksa
 * ke katalog basis data, dan layar Pega tidak membatasi apa pun. Seratus dipakai supaya
 * batas yang dilihat pengguna seragam dengan layar master lain.
 *
 * Bila migrasi 0004 langkah 0c menunjukkan kolomnya lebih pendek, KEDUA tempat wajib ikut
 * berubah.
 */
const MAX_NAME_LENGTH = 100

/**
 * Kode tipe "Internal Surveyor"; sama dengan mastersurveyors.InternalTypeCode.
 *
 * Dipatok karena ia memang dipatok di sistem lama — `CNMInsertDetailSurveyors_act`
 * menyusun unit organisasi akun dengan `@If(M_SURVEY_ID=="1001","Internal","Eksternal")`.
 */
const INTERNAL_TYPE_CODE = '1001'

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Itu duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan langsung
 * ke API tidak melewati layar ini sama sekali.
 *
 * Keunikan nama dan keunikan login TIDAK diperiksa di sini: keduanya menuntut mengetahui
 * seluruh baris yang ada, dan jawabannya dapat berubah antara saat layar dimuat dan saat
 * Simpan ditekan. Penegakannya ada di indeks unik basis data, dan galatnya ditampilkan di
 * bawah.
 */
const schema = z
  .object({
    kode_tipe: z.string().trim().min(1, 'Tipe surveyor wajib dipilih.'),
    nama: z
      .string()
      .trim()
      .min(1, 'Nama surveyor wajib diisi.')
      .max(MAX_NAME_LENGTH, `Nama surveyor paling panjang ${MAX_NAME_LENGTH} karakter.`),
    alamat: z.string().trim(),
    kode_pos: z.string().trim(),
    provinsi: z.string().trim(),
    telepon: z.string().trim(),
    faksimile: z.string().trim(),
    // Email WAJIB. Pesannya diambil apa adanya dari sistem lama, yang menyiapkannya
    // sebagai `local.email := "Email harus diisi"` di langkah 4
    // `CNMInsertDetailSurveyors_act`, lalu memancarkannya di langkah 5 dengan prasyarat
    // `TempDetailSurveyors.EMAIL==""`.
    email: z
      .string()
      .trim()
      .min(1, 'Email harus diisi.')
      .email('Format email tidak benar.'),
    kontak_lain: z.string().trim(),
    kode_cabang: z.string().trim(),
    nama_cabang: z.string().trim(),
    login_aplikasi: z.string().trim(),
  })
  .superRefine((value, ctx) => {
    // Satu-satunya aturan wajib bersyarat, dan ia dibaca langsung dari langkah 8
    // `CNMInsertDetailSurveyors_act` yang deskripsinya berbunyi:
    //
    //     "error if \"internal surveyor\" & LOGIN_APLIKASI is null"
    //
    // Hanya surveyor internal yang memakai aplikasi ini untuk mengisi hasil survei.
    if (value.kode_tipe === INTERNAL_TYPE_CODE && value.login_aplikasi === '') {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['login_aplikasi'],
        message: 'Login aplikasi wajib diisi untuk Internal Surveyor.',
      })
    }
  })

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Null berarti mengajukan baru; terisi berarti mengubah baris itu. */
  surveyor: Surveyor | null

  /** Daftar tipe surveyor untuk dropdown. Dibaca dari modul Master Tipe Surveyors. */
  surveyorTypes: SurveyorType[]

  onClose: () => void
}

/**
 * Form pengajuan dan penyuntingan Master Surveyors.
 *
 * # Yang TIDAK ada di form ini, dan kenapa
 *
 * Status persetujuan, komite yang ditunjuk, tanggal keputusan, dan catatan komite.
 * Keempatnya dimiliki sistem. Server bahkan MENOLAK badan permintaan yang memuatnya,
 * sehingga tidak ada jalan bagi layar ini untuk menyetujui surveyornya sendiri.
 *
 * Menyimpan perubahan SELALU mengembalikan status ke menunggu — meniru tombol Simpan
 * sistem lama yang mengirim `approval = "0"` ke activity yang sama dengan Approve/Reject.
 * Itu kontrol yang nyata: tanpanya, nama login dapat diubah setelah komite menyetujui dan
 * komite tidak pernah melihat perubahannya.
 *
 * Unggah lampiran (`DOCID`) juga belum ada: di Pega ia hasil `Call PNCSaveAttachmentToDB`,
 * dan padanannya menunggu modul `S-1 Dokumen`. Nilai yang sudah ada dipertahankan apa
 * adanya saat menyunting — tidak dikosongkan diam-diam.
 */
export function SurveyorForm({ surveyor, surveyorTypes, onClose }: Props) {
  const save = useSaveSurveyor()
  const firstFieldRef = useRef<HTMLSelectElement | null>(null)

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      kode_tipe: surveyor?.kode_tipe ?? '',
      nama: surveyor?.nama ?? '',
      alamat: surveyor?.alamat ?? '',
      kode_pos: surveyor?.kode_pos ?? '',
      provinsi: surveyor?.provinsi ?? '',
      telepon: surveyor?.telepon ?? '',
      faksimile: surveyor?.faksimile ?? '',
      email: surveyor?.email ?? '',
      kontak_lain: surveyor?.kontak_lain ?? '',
      kode_cabang: surveyor?.kode_cabang ?? '',
      nama_cabang: surveyor?.nama_cabang ?? '',
      login_aplikasi: surveyor?.login_aplikasi ?? '',
    },
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka, supaya pengguna papan ketik
  // tidak perlu menelusuri seluruh halaman untuk sampai ke sini.
  useEffect(() => {
    firstFieldRef.current?.focus()
  }, [])

  const chosenType = watch('kode_tipe')
  const loginRequired = chosenType === INTERNAL_TYPE_CODE

  const typeOptions = surveyorTypes.map((t) => ({ value: t.kode, label: t.deskripsi }))

  async function onSubmit(value: FieldValues) {
    await save.mutateAsync({
      ...value,
      // Lampiran dipertahankan apa adanya. Form ini belum dapat mengunggah berkas, dan
      // mengirim nilai kosong akan MENGHAPUS tautan lampiran yang sudah ada.
      id_dokumen: surveyor?.id_dokumen ?? '',
      ...(surveyor ? { id: surveyor.id } : {}),
    })
    onClose()
  }

  return (
    <section
      aria-labelledby="judul-form-surveyor"
      className="rounded-kartu border border-slate-200 bg-white p-5 shadow-sm"
    >
      <h2 id="judul-form-surveyor" className="text-lg font-semibold text-slate-900">
        {surveyor ? 'Ubah Surveyor' : 'Tambah Surveyor'}
      </h2>
      <p className="mt-1 text-sm text-slate-600">
        {surveyor
          ? 'Menyimpan perubahan mengembalikan surveyor ini ke antrean komite — sama seperti sistem lama. Keputusan yang sudah tercatat digantikan keputusan baru.'
          : 'Surveyor baru masuk ke antrean komite dan belum dapat ditugaskan sebelum disetujui.'}
      </p>

      <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} className="mt-5 space-y-4" noValidate>
        <div className="grid gap-4 sm:grid-cols-2">
          <SelectField
            id="kode_tipe"
            label="Tipe surveyor"
            options={typeOptions}
            error={errors.kode_tipe?.message}
            {...register('kode_tipe')}
            ref={(element) => {
              register('kode_tipe').ref(element)
              firstFieldRef.current = element
            }}
          />
          <Field
            id="nama"
            label="Nama surveyor"
            error={errors.nama?.message}
            hint="Nama ganda ditolak; perbandingannya mengabaikan huruf besar-kecil dan spasi."
            {...register('nama')}
          />
        </div>

        <Field id="alamat" label="Alamat" error={errors.alamat?.message} {...register('alamat')} />

        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="kode_pos"
            label="Kode pos"
            error={errors.kode_pos?.message}
            {...register('kode_pos')}
          />
          <Field
            id="provinsi"
            label="Provinsi"
            error={errors.provinsi?.message}
            {...register('provinsi')}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="telepon"
            label="Telepon"
            error={errors.telepon?.message}
            {...register('telepon')}
          />
          <Field
            id="faksimile"
            label="Faksimile"
            error={errors.faksimile?.message}
            {...register('faksimile')}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="email"
            label="Email (wajib)"
            type="email"
            error={errors.email?.message}
            {...register('email')}
          />
          <Field
            id="kontak_lain"
            label="Kontak lain"
            error={errors.kontak_lain?.message}
            {...register('kontak_lain')}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="kode_cabang"
            label="Cabang"
            error={errors.kode_cabang?.message}
            hint="Isian teks — master cabang belum tersedia di aplikasi ini."
            {...register('kode_cabang')}
          />
          <Field
            id="nama_cabang"
            label="Nama cabang"
            error={errors.nama_cabang?.message}
            {...register('nama_cabang')}
          />
        </div>

        <Field
          id="login_aplikasi"
          label={loginRequired ? 'Login aplikasi (wajib)' : 'Login aplikasi'}
          error={errors.login_aplikasi?.message}
          hint={
            loginRequired
              ? 'Wajib untuk Internal Surveyor. Akun aplikasinya BELUM dibuat otomatis — pembuatan akun menunggu modul Identitas & Akses.'
              : 'Boleh dikosongkan. Surveyor eksternal tidak masuk ke aplikasi ini.'
          }
          {...register('login_aplikasi')}
        />

        {save.isError && <SaveErrorMessage error={save.error} />}

        <div className="flex flex-wrap items-center gap-2 pt-1">
          <Button type="submit" tone="utama" disabled={isSubmitting || save.isPending}>
            {save.isPending ? 'Menyimpan…' : 'Simpan'}
          </Button>
          <Button type="button" tone="kedua" onClick={onClose} disabled={save.isPending}>
            Batal
          </Button>
        </div>
      </form>
    </section>
  )
}

/**
 * Gagal menyimpan dibedakan dari gagal memuat.
 *
 * Yang di sini kerap dapat diperbaiki pengguna sendiri — nama yang bentrok, login yang
 * sudah dipakai — sehingga nadanya penolakan, bukan gangguan.
 */
function SaveErrorMessage({ error }: { error: unknown }) {
  const message = saveMessage(error)
  return (
    <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
  )
}

function saveMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Surveyor belum tersimpan. Periksa koneksi lalu tekan Simpan lagi.',
      tone: 'gangguan',
    }
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case 'nama_surveyor_sudah_ada':
        return {
          title: 'Nama surveyor sudah terdaftar',
          description: error.message,
          tone: 'penolakan',
        }
      case 'login_aplikasi_sudah_dipakai':
        return {
          title: 'Login aplikasi sudah dipakai',
          description: 'Pakai nama login lain, lalu tekan Simpan lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      default:
        return { title: 'Surveyor gagal disimpan', description: error.message, tone: 'gangguan' }
    }
  }

  return {
    title: 'Surveyor gagal disimpan',
    description: 'Terjadi kesalahan pada sistem. Coba simpan lagi.',
    tone: 'gangguan',
  }
}
