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

import { useBranchList, useCountryList, useEmployeeList, useSaveSurveyor } from './api'

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
/**
 * Pilihan dropdown "Apakah perlu ke direksi?" (TRFKOMITE).
 *
 * Daftar nilainya TIDAK ADA di export — dropdown-nya tidak merujuk Report Definition
 * maupun data page mana pun, sehingga pilihannya tersimpan di dalam rule yang tidak
 * ikut diekspor. Yang terlihat di layar lama hanyalah nilai awalnya: "Tidak".
 *
 * Dua pilihan Ya/Tidak dipakai sebagai bentuk yang paling sederhana dan paling mungkin,
 * dengan nilai "1"/"0" mengikuti kolom penanda biner lain di basis data yang sama.
 * Bila Tim Pega mengirim daftar sebenarnya, yang berubah hanya konstanta ini.
 */
const DIRECTOR_OPTIONS = [
  { value: '0', label: 'Tidak' },
  { value: '1', label: 'Ya' },
]

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
    // Tidak wajib: layar lama menampilkannya dengan nilai awal "Tidak".
    perlu_direksi: z.string().trim(),
    nama: z
      .string()
      .trim()
      .min(1, 'Nama surveyor wajib diisi.')
      .max(MAX_NAME_LENGTH, `Nama surveyor paling panjang ${MAX_NAME_LENGTH} karakter.`),
    alamat: z.string().trim(),
    kode_pos: z.string().trim(),
    negara: z.string().trim(),
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
    nama_pic: z.string().trim(),
    kode_cabang: z.string().trim(),
    nama_cabang: z.string().trim(),
    // Tidak pernah tampil sebagai isian, tetapi ADA di formulir: ia diisi oleh pemilihan
    // pegawai, persis `pyAdditionalFields` pada autocomplete layar lama.
    login_aplikasi: z.string().trim(),
  })
  .superRefine((value, ctx) => {
    // Cabang WAJIB untuk Internal Surveyor, dan hanya untuk itu.
    //
    // Dibaca dari blok kontrolnya sendiri pada
    // `Section/BrowseDetailSuveryorsApprove-Section.xml`:
    //
    //     BRANCHNAME  pyVisible=OTHER  pyCondition=PNCIsInternalSurveyors  pyRequired=true
    //
    // Pada tipe lain, kolomnya tidak ditampilkan sama sekali sehingga tidak dapat wajib.
    if (value.kode_tipe === INTERNAL_TYPE_CODE && value.nama_cabang === '') {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['nama_cabang'],
        message: 'Cabang wajib diisi untuk Internal Surveyor.',
      })
    }

    // Login aplikasi wajib untuk Internal Surveyor — aturan langkah 8
    // `CNMInsertDetailSurveyors_act`, yang juga ditegakkan server.
    //
    // Pengguna tidak mengetiknya; ia terisi saat pegawai dipilih. Jadi pesan ini tidak
    // menunjuk isian login, melainkan menunjuk isian Nama — di situlah pengguna dapat
    // bertindak.
    if (value.kode_tipe === INTERNAL_TYPE_CODE && value.login_aplikasi === '') {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['nama'],
        message: 'Pilih nama pegawai dari daftar — login aplikasinya ikut dari pilihan itu.',
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
  const countries = useCountryList()
  const firstFieldRef = useRef<HTMLSelectElement | null>(null)

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      kode_tipe: surveyor?.kode_tipe ?? '',
      perlu_direksi: surveyor?.perlu_direksi ?? '',
      nama: surveyor?.nama ?? '',
      alamat: surveyor?.alamat ?? '',
      kode_pos: surveyor?.kode_pos ?? '',
      negara: surveyor?.negara ?? '',
      telepon: surveyor?.telepon ?? '',
      faksimile: surveyor?.faksimile ?? '',
      email: surveyor?.email ?? '',
      nama_pic: surveyor?.nama_pic ?? '',
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

  // Isian yang tampil BERBEDA menurut tipe surveyor — persis seperti layar lama, yang
  // mengatur tiap kontrol lewat `pyCondition` terhadap When rule `PNCIsInternalSurveyors`
  // (isinya: `M_SURVEY_ID = "1001"`).
  //
  //     selalu          M_SURVEY_ID, NAME, EMAIL (wajib), TELEPHONE, ADDRESS
  //     non-internal    TRFKOMITE, FAKSIMILE, STATE, KDPOS, OTHER_CONTACT
  //     internal saja   BRANCHNAME (wajib)
  //     TIDAK PERNAH    LOGIN_APLIKASI — kondisinya `1=2`, yang tidak pernah benar
  //
  // Baris terakhir itu yang membuat layar lama TIDAK punya isian Login Aplikasi sama
  // sekali. Untuk surveyor internal, nilainya datang dari daftar pilihan pegawai pada
  // isian Nama — bukan diketik.
  //
  // Dibandingkan setelah dipangkas: M_SURVEY_ID bertipe CHAR(4) di Oracle, dan Oracle
  // memadatkan nilai CHAR dengan spasi tanpa memberi tanda apa pun. Satu spasi di ujung
  // membuat perbandingan ini gagal diam-diam — seluruh isian muncul untuk setiap tipe,
  // dan tidak ada pesan galat yang menandainya.
  const cleanType = chosenType.trim()

  /** Tipe sudah dipilih. Sebelum itu, tidak ada isian lain yang ditampilkan. */
  const typeChosen = cleanType !== ''

  const isInternal = cleanType === INTERNAL_TYPE_CODE

  // Kedua daftar ini HANYA ditembak saat Internal Surveyor dipilih — untuk tipe lain,
  // Nama memang kotak teks dan Cabang memang tidak ada.
  const employees = useEmployeeList(isInternal)
  const branches = useBranchList(isInternal)

  const typeOptions = surveyorTypes.map((t) => ({ value: t.kode, label: t.deskripsi }))

  const chosenName = watch('nama')
  const chosenBranchName = watch('nama_cabang')

  /**
   * Nilai yang sedang tersimpan ditambahkan ke daftar bila ia tidak ada di sana.
   *
   * Ini bukan kerapian melainkan keharusan: kueri pegawai membuang siapa pun yang SUDAH
   * terdaftar sebagai surveyor (`login_aplikasi not in (select ... from v_d_surveyors)`).
   * Artinya saat MENYUNTING surveyor internal, namanya sendiri tidak ada di daftar — dan
   * tanpa baris ini, membuka lalu menyimpan ulang akan diam-diam mengosongkan namanya.
   *
   * Hal yang sama berlaku untuk cabang yang sudah tidak aktif lagi.
   */
  function withCurrent(options: { value: string; label: string }[], current: string) {
    const clean = current.trim()
    if (clean === '' || options.some((o) => o.value === clean)) return options
    return [{ value: clean, label: clean }, ...options]
  }

  const employeeOptions = withCurrent(
    (employees.data?.pegawai ?? []).map((p) => ({ value: p.nama, label: p.nama })),
    chosenName,
  )

  const branchOptions = withCurrent(
    (branches.data?.cabang ?? []).map((b) => ({ value: b.nama, label: b.nama })),
    chosenBranchName,
  )

  /**
   * Memilih pegawai mengisi TIGA nilai sekaligus — bukan hanya nama.
   *
   * Cerminan `pyAdditionalFields` pada autocomplete layar lama:
   *
   *	.NAME            -> TempDetailSurveyors.NAME
   *	.LOGIN_APLIKASI  -> TempDetailSurveyors.LOGIN_APLIKASI
   *	.EMAIL           -> TempDetailSurveyors.EMAIL
   *
   * Email yang kosong TIDAK menimpa email yang sudah diisi: sebagian pegawai memang tidak
   * punya email di HRD, dan kuerinya hanya menyaring login — bukan email.
   */
  function pickEmployee(name: string) {
    setValue('nama', name, { shouldValidate: true })
    const match = (employees.data?.pegawai ?? []).find((p) => p.nama === name)
    if (!match) return
    setValue('login_aplikasi', match.login_aplikasi, { shouldValidate: true })
    if (match.email.trim() !== '') setValue('email', match.email, { shouldValidate: true })
  }

  /** Memilih cabang mengisi nama DAN kodenya — `BRANCHNAME` dan `BRANCH`. */
  function pickBranch(name: string) {
    setValue('nama_cabang', name, { shouldValidate: true })
    const match = (branches.data?.cabang ?? []).find((b) => b.nama === name)
    setValue('kode_cabang', match?.kode ?? '', { shouldValidate: true })
  }

  // Dropdown Negara menyimpan NAMA negaranya, bukan kodenya: kolom STATE di basis data
  // berisi teks negara, dan itulah yang dibaca grid maupun rule lama.
  const countryOptions = (countries.data?.negara ?? []).map((c) => ({ value: c.nama, label: c.nama }))

  async function onSubmit(value: FieldValues) {
    await save.mutateAsync({
      ...value,

      // `login_aplikasi` dan `kode_cabang` SUDAH ada di `value`: keduanya isian formulir
      // yang diisi oleh pemilihan pegawai dan pemilihan cabang, bukan diketik. Keduanya
      // tidak perlu diambil dari baris lama.
      //
      // `id_dokumen` lain halnya — ia hasil unggahan lampiran dan menunggu modul
      // `S-1 Dokumen`. Mengirimnya kosong akan MENGHAPUS lampiran yang sudah ada, dan itu
      // kelas cacat yang tidak menimbulkan pesan galat apa pun.
      id_dokumen: surveyor?.id_dokumen ?? '',

      // Isian yang DISEMBUNYIKAN tipe ini juga dipertahankan, bukan dikosongkan: berpindah
      // tipe pada baris yang sudah ada tidak boleh diam-diam membuang alamat atau nomor
      // faks yang pernah diisi.
      ...(isInternal
        ? {
            perlu_direksi: surveyor?.perlu_direksi ?? '',
            faksimile: surveyor?.faksimile ?? '',
            negara: surveyor?.negara ?? '',
            kode_pos: surveyor?.kode_pos ?? '',
            nama_pic: surveyor?.nama_pic ?? '',
          }
        : {
            nama_cabang: surveyor?.nama_cabang ?? '',
            kode_cabang: surveyor?.kode_cabang ?? '',
            login_aplikasi: surveyor?.login_aplikasi ?? '',
          }),

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

      {/*
        URUTAN DAN LABEL mengikuti layar lama persis, dibaca dari
        `Section/BrowseDetailSuveryorsApprove-Section.xml` (pasangan `pyValue` dan
        `pyLabelPreview`):

            M_SURVEY_ID     Tipe Surveyor              pxDropdown
            TRFKOMITE       Apakah perlu ke direksi?   pxDropdown
            NAME            Input Nama                 pxTextInput
            EMAIL           Email                      pxTextInput  WAJIB
            TELEPHONE       Telp                       pxTextInput
            FAKSIMILE       Fax                        pxTextInput
            STATE           Negara                     pxDropdown   (BrowseCountry_RD)
            KDPOS           Kode Pos                   pxTextInput
            OTHER_CONTACT   Nama PIC                   pxTextInput
            ADDRESS         Alamat                     pxTextArea
            BRANCHNAME      Cabang                     pxAutoComplete
            LOGIN_APLIKASI  Login Aplikasi             pxTextInput

        Satu kolom, bukan dua, karena layar lama pun satu kolom.
      */}
      <form onSubmit={(e) => { handleSubmit(onSubmit)(e) }} className="mt-5 space-y-4" noValidate>
        <SelectField
          id="kode_tipe"
          label="Tipe Surveyor"
          options={typeOptions}
          error={errors.kode_tipe?.message}
          {...register('kode_tipe')}
          ref={(element) => {
            register('kode_tipe').ref(element)
            firstFieldRef.current = element
          }}
        />

        {/*
          SELURUH isian lain baru muncul SETELAH tipe dipilih.

          Di layar lama, dropdown Tipe Surveyor memicu refresh section-nya sendiri
          (`pyEvent=change` → `pyAction=refresh`, `pyTarget=thisSection`), dan refresh itulah
          yang menjalankan ulang seluruh `pyCondition`. Sebelum ada tipe, tidak ada kondisi
          yang dapat dinilai.

          Ini juga menghapus satu keadaan yang menyesatkan: tanpa tipe, `isInternal` bernilai
          false, sehingga isian NON-INTERNAL akan terlanjur muncul seolah tipenya sudah
          dipilih.
        */}
        {!typeChosen ? (
          <p className="rounded-kontrol bg-slate-50 px-3 py-6 text-center text-sm text-slate-500">
            Pilih tipe surveyor lebih dulu. Isian berikutnya menyesuaikan tipe yang dipilih.
          </p>
        ) : (
          <>
        {!isInternal && (
          <SelectField
            id="perlu_direksi"
            label="Apakah perlu ke direksi?"
            options={DIRECTOR_OPTIONS}
            emptyText="Tidak"
            error={errors.perlu_direksi?.message}
            {...register('perlu_direksi')}
          />
        )}

        {/*
          DUA kontrol untuk satu kolom, dan itu persis seperti layar lama. Section-nya
          memuat keduanya, masing-masing dengan syaratnya sendiri:

              pyUIElement=text          pyCondition=!PNCIsInternalSurveyors
              pyUIElement=autocomplete  pyCondition=PNCIsInternalSurveyors

          Yang kedua bukan kotak teks: ia daftar pegawai, dan pemilihannya mengisi login
          aplikasi serta email sekaligus. Itulah satu-satunya jalan LOGIN_APLIKASI terisi.
        */}
        {isInternal ? (
          <SelectField
            id="nama"
            label="Nama"
            options={employeeOptions}
            error={errors.nama?.message}
            value={chosenName}
            onChange={(e) => { pickEmployee(e.target.value) }}
          />
        ) : (
          <Field
            id="nama"
            label="Nama"
            error={errors.nama?.message}
            hint="Nama ganda ditolak; perbandingannya mengabaikan huruf besar-kecil dan spasi."
            {...register('nama')}
          />
        )}

        <Field
          id="email"
          label="Email"
          type="email"
          error={errors.email?.message}
          {...register('email')}
        />

        {isInternal ? (
          <Field id="telepon" label="Telp" error={errors.telepon?.message} {...register('telepon')} />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              id="telepon"
              label="Telp"
              error={errors.telepon?.message}
              {...register('telepon')}
            />
            <Field
              id="faksimile"
              label="Fax"
              error={errors.faksimile?.message}
              {...register('faksimile')}
            />
          </div>
        )}

        {!isInternal && (
          <>
            <div className="grid gap-4 sm:grid-cols-2">
              <SelectField
                id="negara"
                label="Negara"
                options={countryOptions}
                error={errors.negara?.message}
                {...register('negara')}
              />
              <Field
                id="kode_pos"
                label="Kode Pos"
                error={errors.kode_pos?.message}
                {...register('kode_pos')}
              />
            </div>

            <Field
              id="nama_pic"
              label="Nama PIC"
              error={errors.nama_pic?.message}
              {...register('nama_pic')}
            />
          </>
        )}

        <div>
          <label htmlFor="alamat" className="block text-sm font-medium text-slate-700">
            Alamat
          </label>
          <textarea
            id="alamat"
            rows={3}
            className="mt-1.5 w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
            {...register('alamat')}
          />
          {errors.alamat?.message && (
            <p className="mt-1 text-xs text-rose-600">{errors.alamat.message}</p>
          )}
        </div>

        {/*
          Cabang pun daftar pilihan di layar lama, bukan kotak teks:
          `pyUIElement=autocomplete` bersumber `BrowseBranch_RD`, dan pemilihannya mengisi
          DUA kolom — `.ID` ke BRANCH dan namanya ke BRANCHNAME.

          Mengetiknya bebas membuat kedua kolom itu berbeda tanpa satu pun pesan galat.
        */}
        {isInternal && (
          <SelectField
            id="nama_cabang"
            label="Cabang"
            options={branchOptions}
            error={errors.nama_cabang?.message}
            value={chosenBranchName}
            onChange={(e) => { pickBranch(e.target.value) }}
          />
        )}

        {/*
          Daftar pegawai KOSONG dinyatakan, bukan dibiarkan tampak seperti dropdown rusak.

          Separuh daftarnya datang dari HRD lewat DB Link (`D-25`, `R-03`), dan separuh
          lagi dari tabel lokal. Bila keduanya tidak menghasilkan apa pun, pengguna perlu
          tahu bahwa yang salah bukan pilihannya.
        */}
        {isInternal && !employees.isPending && employeeOptions.length === 0 && (
          <ErrorMessage
            title="Daftar pegawai kosong"
            description="Tidak ada pegawai yang dapat dijadikan surveyor internal di portal ini. Daftar itu berasal dari data HRD dan Master User Teknis; sampaikan ke administrator bila seharusnya ada isinya."
            tone="penolakan"
          />
        )}
          </>
        )}

        {save.isError && <SaveErrorMessage error={save.error} />}

        <div className="flex flex-wrap items-center gap-2 pt-1">
          {/* Simpan hanya berguna setelah ada tipe; tanpa itu ia pasti ditolak validasi. */}
          <Button
            type="submit"
            tone="utama"
            disabled={!typeChosen || isSubmitting || save.isPending}
          >
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
