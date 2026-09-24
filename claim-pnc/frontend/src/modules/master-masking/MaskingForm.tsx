import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Masking } from '@/api/types'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Button } from '@/components/Button'

import { useBranchOptions, useSaveMasking } from './api'

/**
 * Batas panjang isian; sama dengan konstanta di domain Go.
 *
 * Seluruh kolom teks tabel ini bertipe VARCHAR2(1000), dan angka itu tidak membatasi apa
 * pun yang berguna — login sepanjang 1000 karakter akan merusak setiap grid yang
 * menampilkannya. Angka di bawah dipilih terhadap isi yang benar-benar ada.
 *
 * Bila angka ini berubah, KEDUA tempat wajib ikut berubah.
 */
const MAX_LOGIN = 100
const MAX_MODULE = 100
const MAX_SUB_MODULE = 400
const MAX_BRANCH_ID = 6
const MAX_QUOTA = 1_000_000

/**
 * Aturan yang sama dinyatakan dua kali: di sini dan di domain Go.
 *
 * Itu duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini menjawab pengguna tanpa
 * perjalanan jaringan; yang di sana adalah yang menegakkan — karena pemanggilan langsung
 * ke API tidak melewati layar ini sama sekali. Menghapus salah satunya berarti memilih
 * antara layar yang lamban atau API yang tidak terjaga.
 *
 * Dua hal TIDAK diperiksa di sini karena keduanya menuntut membaca penyimpanan, dan
 * jawabannya dapat berubah antara saat layar dimuat dan saat Simpan ditekan: apakah
 * cabangnya benar-benar ada, dan apakah pasangan cabang+pengguna sudah punya baris.
 * Keduanya ditegakkan server, dan galatnya ditampilkan di bawah.
 */
const schema = z.object({
  cabang: z
    .string()
    .trim()
    .min(1, 'Cabang wajib dipilih.')
    .max(MAX_BRANCH_ID, `Kode cabang paling panjang ${MAX_BRANCH_ID} karakter.`),
  login: z
    .string()
    .trim()
    .min(1, 'Nama pengguna wajib diisi.')
    .max(MAX_LOGIN, `Nama pengguna paling panjang ${MAX_LOGIN} karakter.`),
  modul: z
    .string()
    .trim()
    .min(1, 'Modul wajib diisi.')
    .max(MAX_MODULE, `Modul paling panjang ${MAX_MODULE} karakter.`),
  // Sub modul BOLEH kosong: satu baris produksi memang tidak punya sub modul, dan
  // memaksanya terisi akan menolak keadaan yang sah dan sudah berjalan.
  sub_modul: z
    .string()
    .trim()
    .max(MAX_SUB_MODULE, `Sub modul paling panjang ${MAX_SUB_MODULE} karakter.`),
  // Kuota NOL diterima — ia berarti "tidak boleh mencari sama sekali", dan itu pernyataan
  // kewenangan yang sah, bukan isian yang belum diisi.
  // `error`, bukan `invalid_type_error`: zod 4 menyatukan pesan tipe dan pesan wajib ke
  // satu opsi. Isian yang dikosongkan menghasilkan NaN karena `valueAsNumber`, dan pesan
  // inilah yang menjelaskannya kepada pengguna.
  maks_cari: z
    .number({ error: 'Maks. cari data harus berupa angka.' })
    .int('Maks. cari data harus bilangan bulat.')
    .min(0, 'Maks. cari data tidak boleh kurang dari 0.')
    .max(MAX_QUOTA, `Maks. cari data paling besar ${MAX_QUOTA.toLocaleString('id-ID')}.`),
  maks_lihat: z
    .number({ error: 'Maks. lihat data harus berupa angka.' })
    .int('Maks. lihat data harus bilangan bulat.')
    .min(0, 'Maks. lihat data tidak boleh kurang dari 0.')
    .max(MAX_QUOTA, `Maks. lihat data paling besar ${MAX_QUOTA.toLocaleString('id-ID')}.`),
  lihat_ktp: z.boolean(),
  lihat_email: z.boolean(),
  lihat_notelp: z.boolean(),
  // Status ikut di dalam form, meniru isian berlabel "STATUS" pada
  // `Section/MasterProteksi_Sec-Section.xml:22449` yang terikat `InputData.BranchID` dan
  // dipetakan ke parameter `T_STSAKTF`.
  aktif: z.boolean(),
})

type FieldValues = z.infer<typeof schema>

type Props = {
  /** Null berarti menambah; terisi berarti mengubah baris itu. */
  masking: Masking | null
  onClose: () => void
}

/**
 * Form tambah dan ubah Master Masking.
 *
 * Isian dan labelnya mengikuti `Section/MasterProteksi_Sec-Section.xml`: CABANG, NAMA
 * USER, MODUL, SUB MODUL, STATUS, KTP, EMAIL, NOTELP, MAX CARI DATA, MAX LIHAT DATA.
 *
 * **STATUS ada di form**, sama seperti layar lama — isian di `:22449` terikat
 * `InputData.BranchID`, yang dipetakan ke parameter `T_STSAKTF`. Yang menjaganya tidak
 * berubah tanpa sengaja adalah LAYAR DAFTAR: tombol Ubah hanya muncul pada baris aktif,
 * meniru `ActionMaskingData_Sec` yang bersyarat `.STS_AKTF=='AKTIF'`. Form ini karena itu
 * tidak pernah terbuka untuk baris yang sudah nonaktif.
 *
 * Pada penambahan, isian STATUS disembunyikan: baris baru selalu aktif, dan server
 * menimpanya demikian.
 */
export function MaskingForm({ masking, onClose }: Props) {
  const save = useSaveMasking()
  const editing = masking !== null
  const firstField = useRef<HTMLInputElement | null>(null)

  // Kata kunci pencarian cabang. Terpisah dari nilai form: yang disimpan adalah KODE
  // cabang, yang diketik adalah namanya.
  const [branchKeyword, setBranchKeyword] = useState('')
  const branches = useBranchOptions(branchKeyword)

  const {
    register,
    handleSubmit,
    setError,
    setValue,
    watch,
    formState: { errors },
  } = useForm<FieldValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      cabang: masking?.cabang ?? '',
      login: masking?.login ?? '',
      // Modul diisi awal dengan satu-satunya nilai yang ada di data produksi.
      //
      // Ia SARAN, bukan batasan: isiannya tetap dapat diketik bebas (keputusan Work Owner
      // 2026-09-20, "seperti aplikasi Pega saja"). Daftar pilihannya di Pega dipasok rule
      // `MODULKLAIMMASKING` yang HILANG dari export (`R-16`), sehingga daftarnya tidak
      // dapat direproduksi tanpa mengarangnya.
      modul: masking?.modul ?? 'PNCSearchKlaim',
      sub_modul: masking?.sub_modul ?? '',
      maks_cari: masking?.maks_cari ?? 0,
      maks_lihat: masking?.maks_lihat ?? 0,
      lihat_ktp: masking?.lihat_ktp ?? false,
      lihat_email: masking?.lihat_email ?? false,
      lihat_notelp: masking?.lihat_notelp ?? false,
      // Baris baru selalu aktif — server pun menimpanya demikian.
      aktif: masking?.aktif ?? true,
    },
  })

  // Fokus dipindahkan ke isian pertama saat form terbuka. Tanpa ini, pengguna papan ketik
  // harus menekan Tab berkali-kali dari awal halaman untuk mencapainya.
  useEffect(() => {
    firstField.current?.focus()
  }, [])

  // Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di
  // kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya
  // berguna bila layar menyorotnya di tempat isiannya.
  useEffect(() => {
    if (!(save.error instanceof APIError)) return
    const violation = save.error.violations()
    for (const name of [
      'cabang',
      'login',
      'modul',
      'sub_modul',
      'maks_cari',
      'maks_lihat',
    ] as const) {
      const message = violation[name]
      if (message) setError(name, { type: 'server', message })
    }
  }, [save.error, setError])

  const selectedBranch = watch('cabang')
  const { ref: refLogin, ...remainingLogin } = register('login')

  function send(values: FieldValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal.
    save.mutate(editing ? { id: masking.id, ...values } : values, { onSuccess: onClose })
  }

  return (
    /*
      Panel ini muncul di atas tabel, bukan sebagai dialog melayang.

      Alasannya praktis: pengguna sering perlu melihat baris lain untuk memastikan orang
      yang diketiknya belum punya kewenangan di cabang itu — dan dialog yang menutup layar
      justru menyembunyikan jawabannya.
    */
    <form
      onSubmit={handleSubmit(send)}
      noValidate
      className="overflow-hidden rounded-kartu border border-slate-200 border-l-4 border-l-blue-500 bg-white shadow-angkat"
      aria-label={editing ? 'Ubah data masking' : 'Tambah data masking'}
    >
      <div className="border-b border-slate-100 bg-slate-50/70 px-5 py-4">
        <h3 className="text-base font-semibold text-slate-900">
          {editing ? 'Ubah Data Masking' : 'Tambah Data Masking'}
        </h3>
        <p className="mt-1 text-sm text-slate-600">
          {editing
            ? 'Status aktif tidak diubah dari sini — gunakan tombol pada barisnya.'
            : 'Satu pengguna hanya boleh punya satu data masking per cabang.'}
        </p>
      </div>

      <div className="space-y-5 p-5">
        {save.isError && <SaveErrorMessage error={save.error} />}

        <div className="grid gap-5 sm:grid-cols-2">
          <div>
            <Field
              id="cariCabang"
              label="Cabang"
              placeholder="Ketik nama atau kode cabang"
              value={branchKeyword}
              onChange={(event) => setBranchKeyword(event.target.value)}
              error={errors.cabang?.message}
              hint={
                selectedBranch
                  ? `Terpilih: ${selectedBranch}`
                  : 'Pilih satu cabang dari daftar di bawah.'
              }
            />
            {/*
              Daftar cabang disaring di server, bukan dimuat seluruhnya. POOLDATA.BRANCH
              memuat 803 baris — mengirim semuanya ke peramban membebani jaringan untuk
              daftar yang hampir seluruhnya tidak akan dilihat.
            */}
            <div className="mt-2 max-h-40 overflow-y-auto rounded-kontrol border border-slate-200">
              {branches.isPending && (
                <p className="px-3 py-2 text-xs text-slate-500">Memuat cabang…</p>
              )}
              {branches.isError && (
                <p className="px-3 py-2 text-xs text-red-600">Daftar cabang gagal dimuat.</p>
              )}
              {branches.data?.cabang.length === 0 && (
                <p className="px-3 py-2 text-xs text-slate-500">Tidak ada cabang yang cocok.</p>
              )}
              {branches.data?.cabang.map((branch) => (
                <button
                  key={branch.kode}
                  type="button"
                  onClick={() => setValue('cabang', branch.kode, { shouldValidate: true })}
                  className={`flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm hover:bg-slate-50 ${
                    selectedBranch === branch.kode ? 'bg-blue-50 font-medium text-blue-800' : 'text-slate-700'
                  }`}
                >
                  <span>{branch.nama}</span>
                  <span className="font-mono text-xs text-slate-500">{branch.kode}</span>
                </button>
              ))}
            </div>
            <input type="hidden" {...register('cabang')} />
          </div>

          <Field
            id="login"
            label="Nama User"
            placeholder="Login pengguna"
            error={errors.login?.message}
            hint="Login yang dipakai pengguna untuk masuk aplikasi."
            ref={(element) => {
              refLogin(element)
              firstField.current = element
            }}
            {...remainingLogin}
          />

          <Field
            id="modul"
            label="Modul"
            error={errors.modul?.message}
            hint="Layar tempat kewenangan ini berlaku."
            {...register('modul')}
          />

          <Field
            id="sub_modul"
            label="Sub modul"
            placeholder="Registrasi,Dokumen,"
            error={errors.sub_modul?.message}
            hint="Boleh lebih dari satu, dipisah koma. Boleh dikosongkan."
            {...register('sub_modul')}
          />

          <Field
            id="maks_cari"
            label="Max Cari Data"
            type="number"
            min={0}
            error={errors.maks_cari?.message}
            hint="Berapa kali pengguna boleh mencari. 0 berarti tidak boleh."
            {...register('maks_cari', { valueAsNumber: true })}
          />

          <Field
            id="maks_lihat"
            label="Max Lihat Data"
            type="number"
            min={0}
            error={errors.maks_lihat?.message}
            hint="Berapa banyak data yang boleh dilihat. 0 berarti tidak boleh."
            {...register('maks_lihat', { valueAsNumber: true })}
          />

          {/*
            Isian STATUS hanya muncul saat MENGUBAH.

            Pada penambahan ia tidak ada gunanya: baris baru selalu aktif, dan server
            menimpanya demikian. Menampilkannya akan menawarkan pilihan yang tidak
            berpengaruh — bentuk kebohongan kecil yang membuat pengguna berhenti memercayai
            isian lain di layar yang sama.
          */}
          {editing && (
            <SelectField
              id="aktif"
              label="Status"
              options={[
                { value: 'true', label: 'AKTIF' },
                { value: 'false', label: 'TIDAK AKTIF' },
              ]}
              error={errors.aktif?.message}
              {...register('aktif', {
                // Dropdown mengirim teks; skema menuntut boolean.
                setValueAs: (value: string | boolean) => value === true || value === 'true',
              })}
            />
          )}
        </div>

        {/*
          Ketiga izin dikelompokkan dan diberi keterangan tegas.

          Ia bagian paling menentukan di form ini: mencentangnya berarti seseorang dapat
          membaca nomor KTP dan nomor telepon nasabah utuh. Mengelompokkannya membuat
          pengguna melihat ketiganya sebagai satu keputusan, bukan tiga kotak centang biasa
          yang tercampur di antara isian lain.
        */}
        <fieldset className="rounded-kartu border border-slate-200 bg-slate-50/50 p-4">
          <legend className="px-1 text-sm font-medium text-slate-800">
            Boleh melihat data pribadi tanpa disamarkan
          </legend>
          <p className="mb-3 text-xs text-slate-600">
            Yang tidak dicentang tetap ditampilkan tersamar bagi pengguna ini.
          </p>
          <div className="grid gap-2 sm:grid-cols-3">
            <Checkbox id="lihat_ktp" label="Nomor KTP" {...register('lihat_ktp')} />
            <Checkbox id="lihat_email" label="Alamat surel" {...register('lihat_email')} />
            <Checkbox id="lihat_notelp" label="Nomor telepon" {...register('lihat_notelp')} />
          </div>
        </fieldset>
      </div>

      <div className="flex justify-end gap-2 border-t border-slate-100 bg-slate-50/70 px-5 py-4">
        <Button tone="kedua" onClick={onClose} disabled={save.isPending}>
          Batal
        </Button>
        <Button tone="utama" type="submit" disabled={save.isPending}>
          {save.isPending ? 'Menyimpan…' : 'Simpan'}
        </Button>
      </div>
    </form>
  )
}

/**
 * Kotak centang dengan label yang dapat diklik.
 *
 * Ditulis di sini, bukan di `components/`, karena ia satu-satunya pemakainya sekarang.
 * Bila modul berikutnya membutuhkannya, ia naik ke komponen bersama — bukan disalin.
 */
const Checkbox = function Checkbox({
  id,
  label,
  ...rest
}: { id: string; label: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label
      htmlFor={id}
      className="flex cursor-pointer items-center gap-2 rounded-kontrol border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 hover:border-slate-300"
    >
      <input
        id={id}
        type="checkbox"
        className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500/30"
        {...rest}
      />
      {label}
    </label>
  )
}

/** Gagal menyimpan dibedakan dari gagal memuat: di sini pengguna baru saja melakukan sesuatu. */
function SaveErrorMessage({ error }: { error: unknown }) {
  const message = saveMessage(error)
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

function saveMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Data masking belum tersimpan. Periksa koneksi lalu tekan Simpan lagi.',
      tone: 'gangguan',
    }
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.maskingPairTaken:
        return {
          title: 'Pengguna itu sudah punya data masking di cabang ini',
          description:
            'Ubah data yang sudah ada, jangan menambah baris baru. Dua baris untuk orang yang sama berarti dua kewenangan yang keduanya berlaku.',
          tone: 'penolakan',
        }
      case ErrorCode.maskingBranchUnknown:
        return {
          title: 'Cabang tidak dikenal',
          description: 'Pilih cabang dari daftar, jangan mengetik kodenya sendiri.',
          tone: 'penolakan',
        }
      case ErrorCode.maskingIDTaken:
        return {
          title: 'Nomor data sedang dipakai permintaan lain',
          description: 'Tekan Simpan sekali lagi; hampir pasti berhasil.',
          tone: 'gangguan',
        }
      case ErrorCode.maskingNotFound:
        return {
          title: 'Data masking tidak ditemukan',
          description: 'Mungkin baru saja diubah petugas lain. Tutup form ini lalu muat ulang daftarnya.',
          tone: 'penolakan',
        }
      case ErrorCode.validationFailed:
        return {
          title: 'Isian belum benar',
          description: 'Perbaiki isian yang ditandai lalu simpan lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
      case ErrorCode.portalNotReady:
        return {
          title: 'Portal entitas bermasalah',
          description: 'Pilih ulang portal entitas di bilah atas halaman ini.',
          tone: 'penolakan',
        }
      default:
        return { title: 'Gagal menyimpan', description: error.message, tone: 'gangguan' }
    }
  }

  return {
    title: 'Gagal menyimpan',
    description: 'Terjadi kesalahan pada sistem. Coba simpan lagi.',
    tone: 'gangguan',
  }
}
