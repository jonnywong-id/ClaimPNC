import { useRef, useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ComboField } from '@/components/ComboField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'

import { RiwayatSalvage } from './RiwayatSalvage'
import { UploadDocumentModal } from './UploadDocumentModal'
import {
  useCreateSalvage,
  useSalvageClaimLookup,
  useUploadSalvageDetail,
} from './api'
import type {
  CoveragePilihan,
  CreateRequest,
  DetailItem,
  HistoryRow,
  ObjekPilihan,
  StatusOption,
} from './types'

/**
 * Isian yang sudah diketahui saat form dibuka dari sebuah baris klaim.
 *
 * Nomor klaimnya berasal dari baris yang diklik, bukan diketik â€” dan hanya ITU yang
 * dikunci. Nama object dan nama coverage ikut dibawa bila klaim itu pernah diajukan
 * salvage sebelumnya, tetapi keduanya tetap dapat dipilih ulang.
 *
 * # Kenapa keduanya TIDAK lagi dikunci
 *
 * Karena form ini terbuka justru ketika klaimnya BELUM punya pengajuan salvage â€” itulah
 * syaratnya di `SalvageInboxPage`. Pada keadaan itu `nama_object` dan `nama_coverage`
 * datang kosong, sementara keduanya wajib diisi di server. Mengunci keduanya dalam
 * keadaan kosong membuat form ini tidak pernah dapat disimpan sama sekali.
 */
export type Prefill = {
  nomor_klaim: string
  nama_object: string
  nama_coverage: string

  /** Penunjuk objek dan coverage, bila pengajuan sebelumnya pernah memilihnya. */
  id_object: string
  id_coverage: string
}

/**
 * Isi kedua autocomplete, sebagaimana dibaca dari klaimnya.
 *
 * Diberikan dari luar pada jalur prefill â€” halaman sudah menembak rincian klaimnya, dan
 * menembaknya lagi dari dalam form berarti dua permintaan untuk satu jawaban yang sama.
 * Pada jalur tombol "Tambah" form yang mencarinya sendiri, karena di sana nomor klaimnya
 * baru diketahui setelah diketik.
 */
export type Pilihan = {
  objek: ObjekPilihan[]
  coverage: CoveragePilihan[]
}

type Props = {
  statusOptions: StatusOption[]

  /**
   * Pilihan **Mata Uang**, dibaca server dari `POOLDATA.CURRENCY`.
   *
   * Kosong berarti tabelnya tidak terbaca. Kolomnya tetap digambar â€” hanya tanpa isi dan
   * tanpa tanda wajib, supaya kegagalan membaca satu master tidak berubah menjadi layar
   * yang tidak dapat dipakai.
   */
  currencyOptions: StatusOption[]
  uploadColumns: string[]
  onClose: () => void
  onSaved: (message: string) => void

  /**
   * Isian yang sudah diketahui. Tanpa ini form dibuka kosong â€” jalur tombol "Tambah".
   *
   * Dengan ini, form dibuka dari sebuah baris klaim: ketiga isiannya terisi dan
   * dikunci, supaya pengajuan tidak dapat tersimpan atas klaim yang berbeda dari baris
   * yang diklik.
   */
  prefill?: Prefill

  /**
   * Pengajuan yang sedang DISUNTING. Tanpa ini, form membuat pengajuan baru.
   *
   * # Kapan ia terisi
   *
   * Saat baris daftar "Rejected Checker" dibuka — pengajuan yang checker kembalikan
   * kepada PIC untuk diperbaiki.
   *
   * # Daftar barangnya DIGANTI, bukan ditambahi
   *
   * `item` di bawah mengisi grid Detail Item Salvage, dan apa yang ada di grid itulah
   * yang tersimpan: server membuang seluruh barang lama lebih dulu. Itu mekanisme layar
   * lama (`pyDeleteSQL` pada `PNCSalvageGetChekerDataKlaimAllData`), dan tanpanya
   * menyimpan ulang akan menggandakan barisnya.
   */
  editing?: {
    id_salvage: string
    isian: Partial<FormState>
    item: DetailItem[]
  }

  /**
   * Isi kedua autocomplete. Dipakai bersama `prefill` maupun `editing`.
   *
   * Tanpa keduanya, form mencarinya sendiri setelah nomor klaim diketik.
   */
  pilihan?: Pilihan

  /**
   * Isi grid "Detail History Salvage" â€” seluruh pengajuan milik klaim ini.
   *
   * Grid ini ADA di layar lama, dan ia menjawab pertanyaan yang tidak dapat dijawab
   * daftar mana pun: klaim ini sudah pernah diajukan berapa kali, dan masing-masing
   * berakhir di mana. Tanpa itu, petugas dapat mengajukan salvage kedua atas klaim yang
   * pengajuan pertamanya masih berjalan tanpa pernah melihatnya.
   */
  history?: HistoryRow[]
}

/** Isian form dalam bentuk yang dipegang komponen ini. */
type FormState = Omit<CreateRequest, 'mode' | 'id_salvage' | 'detail_item_salvage'>

const emptyForm: FormState = {
  nomor_klaim: '',
  id_object: '',
  nama_object: '',
  id_coverage: '',
  nama_coverage: '',
  tanggal_input: today(),
  jenis_salvage: '',
  status_salvage: '',
  lokasi_salvage: '',
  lokasi_salvage_di_jabodetabek: false,
  mata_uang: '',
  minimum_salvage: '',
  quantity_salvage: '',
  nilai_penawaran: '',
  share_tertanggung: '',
  remark: '',
  email: '',
  nama_pic_survey: '',
  no_telp_pic_survey: '',
  email_pic_survey: '',
}

/**
 * Form **"Menambahkan Data Salvage"** â€” di balik tombol Tambah.
 *
 * # Apa yang digantikan
 *
 * `Section/TambahData_Salvage-Section.xml`, yang dibuka setelah
 * `Data Transform/CNMShowInsertSalvage_dt-DT.xml` membersihkan halaman formnya.
 *
 * Data transform itu ternyata TIDAK menyimpan apa pun â€” kedelapan langkahnya hanya
 * membuang halaman klipboard lama dan menandai mode `"Insert"`. Ia pembersih form, bukan
 * penyimpan. Padanannya di sini adalah keadaan awal `emptyForm`.
 *
 * # Tombol "Upload File" TIDAK menyimpan apa pun
 *
 * Itu yang paling mudah disalahpahami, dan penelusuran ke
 * `Activity/UploadDetailSalvage-Act.xml` membuktikannya: ketiga langkahnya hanya menyalin
 * isi berkas CSV ke grid DI DALAM form. Penyimpanan baru terjadi saat Submit ditekan.
 *
 * Karena itu keterangan di bawah tombolnya menyebutkan hal itu apa adanya â€” pengguna yang
 * mengunggah lalu menutup layar akan kehilangan isinya, sama seperti di Pega.
 *
 * # Isian yang TIDAK digambar, dan itu disengaja
 *
 * Procedure `INSERT_SALVAGE` menerima enam parameter yang form Tambah tidak pernah isi:
 * tanggal transfer ke bagian umum, tanggal dan nomor akseptasi, nama pemenang lelang,
 * tanggal lelang, dan tanggal terima. Keenamnya berasal dari isian yang hanya tergambar
 * pada mode UBAH â€” bukan saat pengajuan dibuat â€” dan menggambarnya di sini akan meminta
 * pengguna mengisi hal yang belum terjadi.
 */
export function TambahSalvageForm({
  statusOptions,
  currencyOptions,
  uploadColumns,
  onClose,
  onSaved,
  prefill,
  editing,
  pilihan,
  history = [],
}: Props) {
  // Prefill dipakai sebagai keadaan AWAL, bukan disalin ulang setiap render.
  //
  // Komponen ini dibongkar dan dipasang kembali setiap kali form dibuka â€” sehingga nilai
  // awalnya selalu segar, dan isian yang sudah diketik pengguna tidak pernah tertimpa
  // oleh jawaban server yang datang belakangan.
  const [form, setForm] = useState<FormState>(() => ({
    ...emptyForm,
    ...(prefill ?? {}),
    ...(editing?.isian ?? {}),
  }))
  const [items, setItems] = useState<DetailItem[]>(() => editing?.item ?? [])
  const [uploadNote, setUploadNote] = useState('')

  // Kabar dari unggahan DOKUMEN, dipisahkan dari kabar unggahan CSV di atas.
  //
  // Keduanya tampak serupa tetapi menjawab tindakan yang berbeda: yang satu mengisi grid
  // Detail Item, yang lain melampirkan berkas ke klaim. Menyatukannya membuat "3 dokumen
  // tersimpan" muncul berdampingan dengan galat pembacaan CSV, seolah keduanya satu hal.
  const [docNote, setDocNote] = useState('')

  // Modal "UploadDocument_Salvage" sedang terbuka.
  //
  // Ia state komponen, bukan bagian alamat: modal dibuka untuk satu tindakan lalu
  // ditutup, dan menaruhnya di alamat membuat tombol "kembali" peramban menutup modal
  // alih-alih meninggalkan layar.
  const [uploading, setUploading] = useState(false)

  // Nomor klaim yang SUDAH selesai diketik â€” bukan yang sedang diketik.
  //
  // Keduanya dipisah dengan sengaja. `form.nomor_klaim` berubah pada setiap ketukan
  // papan ketik; isian ini baru berubah ketika kolomnya ditinggalkan, dan itulah yang
  // memicu pencarian. Memakai yang pertama berarti satu permintaan per huruf.
  //
  // Pemicunya meniru layar lama apa adanya: di Pega kedua aksi kolom Nomor Klaim â€”
  // `postValue` lalu `refresh` â€” terpasang pada event `change`, yang pada kotak teks
  // HTML berarti "selesai diubah", bukan "sedang diubah".
  const [lookupClaim, setLookupClaim] = useState('')

  const fileInput = useRef<HTMLInputElement>(null)

  const create = useCreateSalvage()
  const upload = useUploadSalvageDetail()

  // Pencarian hanya hidup pada jalur tombol "Tambah".
  //
  // Pada jalur prefill, halaman sudah menembak rincian klaimnya dan meneruskan hasilnya
  // lewat `pilihan` â€” menembaknya lagi dari sini berarti dua permintaan untuk satu
  // jawaban yang sama.
  // Klaimnya SUDAH diketahui pada dua jalur: form yang dibuka dari baris klaim, dan form
  // sunting yang dibuka dari baris pengajuan. Pada keduanya nomor klaim tidak diketik dan
  // tidak dicari — halaman sudah membawanya beserta pilihan objek dan coverage-nya.
  const terikatKlaim = prefill !== undefined || editing !== undefined

  const lookup = useSalvageClaimLookup(terikatKlaim ? '' : lookupClaim)

  const objectChoices: ObjekPilihan[] = terikatKlaim
    ? (pilihan?.objek ?? [])
    : (lookup.data?.pilihan_objek ?? [])

  const coverageChoices: CoveragePilihan[] = terikatKlaim
    ? (pilihan?.coverage ?? [])
    : (lookup.data?.pilihan_coverage ?? [])

  // Nama yang benar-benar ditawarkan â€” tanpa pengulangan. Dihitung sekali, lalu dipakai
  // daftar sarannya DAN keterangan jumlahnya, supaya keduanya tidak pernah berselisih.
  //
  // Daftar coverage TIDAK dipersempit menurut objek yang dipilih, dan itu mengikuti layar
  // lama: kueri pemasoknya tidak mengambil `OBJECTID` sama sekali, sehingga seluruh
  // coverage milik klaim ditawarkan siapa pun objeknya (`P-5`).
  const objectNames = namesOf(objectChoices)
  const coverageNames = namesOf(coverageChoices)

  // Riwayat pada jalur Tambah datang dari pencarian; pada jalur prefill dari halaman.
  const riwayat = terikatKlaim ? history : (lookup.data?.riwayat ?? [])

  // Pelanggaran per isian datang dari server sebagai SENARAI dan dipakai menandai
  // isiannya di tempatnya â€” bukan sebagai satu pesan di atas form. Pada form berisi tujuh
  // belas isian, satu pesan umum memaksa pengguna menebak isian mana yang dimaksud.
  const violations =
    create.error instanceof APIError ? create.error.violations() : {}

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((current) => ({ ...current, [key]: value }))

  /**
   * chooseObject mencatat nama objek yang diketik BESERTA penunjuknya.
   *
   * Inilah yang di Pega dikerjakan `pyAdditionalFields` ber-`pyShow=false`: saat sebuah
   * baris dipilih, `.CaseID` miliknya disalin diam-diam ke `TempInsert.NewNoKTP`.
   *
   * Nama yang TIDAK ada di daftar tetap diterima â€” `pyAllowFreeFormInput=true` di layar
   * lama â€” tetapi penunjuknya dikosongkan. Membiarkan penunjuk lama melekat pada nama
   * baru akan menyimpan pasangan yang tidak cocok satu sama lain.
   */
  function chooseObject(name: string) {
    const match = objectChoices.find((choice) => choice.nama === name)
    setForm((current) => ({
      ...current,
      nama_object: name,
      id_object: match?.id ?? '',
    }))
  }

  /**
   * chooseCoverage bekerja sama seperti chooseObject.
   *
   * Coverage yang sudah dipilih TIDAK dikosongkan saat objeknya berganti: di layar lama
   * kedua kolom itu memang tidak saling terikat â€” daftar coverage ditarik per klaim,
   * bukan per objek. Mengosongkannya di sini akan menghapus ketikan orang atas dasar
   * keterikatan yang tidak pernah ada.
   */
  function chooseCoverage(name: string) {
    const match = coverageChoices.find((choice) => choice.nama === name)
    setForm((current) => ({
      ...current,
      nama_coverage: name,
      id_coverage: match?.id ?? '',
    }))
  }

  /**
   * lookupByClaim menjalankan pencarian klaim, lalu mengosongkan pilihan yang sudah lewat.
   *
   * Pengosongan itu yang paling mudah terlewat: objek dan coverage milik klaim SEBELUMNYA
   * tidak ada hubungannya dengan klaim yang baru diketik, dan membiarkannya akan menyimpan
   * pengajuan yang menunjuk objek milik klaim lain.
   */
  function lookupByClaim(typed: string) {
    const clean = typed.trim()
    if (clean === lookupClaim) return

    setLookupClaim(clean)
    setForm((current) => ({
      ...current,
      nama_object: '',
      id_object: '',
      nama_coverage: '',
      id_coverage: '',
    }))
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()

    create.mutate(
      {
        ...form,

        // Mode ditentukan ada-tidaknya pengajuan yang sedang disunting — sama seperti di
        // Pega, tempat kedua jalur menempuh activity yang SAMA dan yang membedakannya
        // hanyalah terisi-tidaknya ID salvage.
        mode: editing === undefined ? 'insert' : 'ubah',
        id_salvage: editing?.id_salvage ?? '',
        detail_item_salvage: items,
      },
      {
        onSuccess: (result) => {
          onSaved(result.pesan)
          onClose()
        },
      },
    )
  }

  function chooseFile(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return

    upload.mutate(file, {
      onSuccess: (result) => {
        // Baris yang diunggah DITAMBAHKAN ke yang sudah ada, bukan menggantikannya.
        // `UploadDetailSalvage` memakai `<APPEND>` pada setiap barisnya, sehingga dua
        // berkas yang diunggah berturut-turut menghasilkan gabungan keduanya.
        setItems((current) => [...current, ...result.detail_item_salvage])
        setUploadNote(result.pesan)
      },
    })

    // Nilai input dikosongkan supaya berkas yang SAMA dapat diunggah lagi. Tanpa ini,
    // memilih berkas yang sama dua kali tidak memicu perubahan apa pun.
    event.target.value = ''
  }

  return (
    <form
      onSubmit={submit}
      className="space-y-6 rounded-kartu border border-slate-200 bg-white p-5"
      aria-label="Menambahkan Data Salvage"
    >
      <div>
        <h2 className="text-base font-semibold text-slate-900">
          Menambahkan Data Salvage
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          {form.nomor_klaim === '' ? null : (
            <>
              Untuk klaim <strong>{form.nomor_klaim}</strong>.{' '}
            </>
          )}
          {editing === undefined ? (
            <>
              Pengajuan yang disimpan langsung masuk antrean <strong>Checker</strong>.
            </>
          ) : (
            <>
              Pengajuan <strong>{editing.id_salvage}</strong> dikembalikan checker untuk
              diperbaiki. Menyimpan akan <strong>mengganti</strong> daftar barangnya
              dengan isi tabel di bawah.
            </>
          )}
        </p>

        {/* Peringatan ini ada di layar lama, kata demi kata (`:884`). */}
        <p className="mt-2 text-sm text-slate-700">
          Perhatian Lokasi Salvage Di Jabodatabek diInput Jika Lokasi Salvage DI
          Jabodatabek
        </p>
      </div>

      {create.error != null && (
        <ErrorMessage
          title="Pengajuan tidak tersimpan"
          description={messageOf(create.error)}
          tone="penolakan"
        />
      )}

      {/*
        Blok pertama, TIGA kolom, dan urutannya mengikuti layar lama baris demi baris:

          Tanggal Input  | Nomor Klaim     | â˜‘ Lokasi Salvage Di Jabodatabek
          Email          | Nama Object     | Lokasi Salvage
          Jenis Salvage  | Nama Coverage âœ± | Remark

        Kolom TENGAH-lah yang bekerja sebagai satu rangkaian: mengetik Nomor Klaim lalu
        meninggalkan kolomnya memuat objek dan coverage milik klaim itu, dan kedua kolom
        di bawahnya memilih dari hasilnya. Di Pega rangkaian itu terpasang sebagai
        `postValue` + `refresh` yang memanggil `SetDataDetailSalvage_act`, dan kedua kolom
        bawah adalah `pxAutoComplete` yang membaca page yang diisinya.
      */}
      <fieldset className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <legend className="sr-only">Data Salvage</legend>

        {/*
          Tanggal Input TIDAK dapat diubah, sama seperti di layar lama â€” di sana ia teks
          biasa, bukan kotak isian. Nilainya tetap dikirim saat menyimpan.
        */}
        <div>
          <span className="block text-sm font-medium text-slate-700">Tanggal Input</span>
          <p className="mt-1 py-2 text-slate-900">{formatDateTime(form.tanggal_input)}</p>
        </div>

        <FormField
          id="salvage-nomor-klaim"
          label="Nomor Klaim"
          value={form.nomor_klaim}
          onChange={(event) => set('nomor_klaim', event.target.value)}
          onBlur={(event) => {
            if (!terikatKlaim) lookupByClaim(event.target.value)
          }}
          onKeyDown={(event) => {
            // Enter mencari, bukan menyimpan.
            //
            // Tanpa ini, menekan Enter di kolom pertama akan mengirim form yang kedua
            // kolom di bawahnya belum terisi â€” dan server menolaknya dengan dua pesan
            // yang penyebabnya justru belum dikerjakan pengguna.
            if (event.key !== 'Enter' || terikatKlaim) return
            event.preventDefault()
            lookupByClaim(form.nomor_klaim)
          }}
          failure={violations['nomor_klaim']}
          readOnly={terikatKlaim}
          required
        />

        <label className="flex items-end gap-2 pb-2 text-sm text-slate-700">
          <input
            type="checkbox"
            checked={form.lokasi_salvage_di_jabodetabek}
            onChange={(event) =>
              set('lokasi_salvage_di_jabodetabek', event.target.checked)
            }
            className="size-4 rounded border-slate-300"
          />
          Lokasi Salvage Di Jabodatabek
        </label>

        <FormField
          id="salvage-email"
          label="Email"
          type="email"
          value={form.email}
          onChange={(event) => set('email', event.target.value)}
          failure={violations['email']}
        />

        <ComboField
          id="salvage-nama-object"
          label="Nama Object"
          options={objectNames}
          value={form.nama_object}
          onChange={(event) => chooseObject(event.target.value)}
          error={violations['nama_object']}
          hint={hintFor(objectNames.length, 'objek')}
          required
        />

        <FormField
          id="salvage-lokasi"
          label="Lokasi Salvage"
          value={form.lokasi_salvage}
          onChange={(event) => set('lokasi_salvage', event.target.value)}
          failure={violations['lokasi_salvage']}
        />

        <FormField
          id="salvage-jenis"
          label="Jenis Salvage"
          value={form.jenis_salvage}
          onChange={(event) => set('jenis_salvage', event.target.value)}
          failure={violations['jenis_salvage']}
          required
        />

        <ComboField
          id="salvage-nama-coverage"
          label="Nama Coverage"
          options={coverageNames}
          value={form.nama_coverage}
          onChange={(event) => chooseCoverage(event.target.value)}
          error={violations['nama_coverage']}
          hint={hintFor(coverageNames.length, 'coverage')}
          required
        />

        <TextAreaField
          id="salvage-remark"
          label="Remark"
          rows={2}
          value={form.remark}
          onChange={(event) => set('remark', event.target.value)}
          error={violations['remark']}
        />

        {/*
          Keadaan pencarian digambar SEKALI, melebar penuh di bawah blok ini â€” bukan di
          dalam salah satu kolomnya. Yang dilaporkannya menyangkut ketiganya sekaligus.
        */}
        {!terikatKlaim && (
          <div className="sm:col-span-2 lg:col-span-3">
            {lookupClaim === '' ? (
              <p className="text-sm text-slate-600">
                Isi <strong>Nomor Klaim</strong> lebih dulu, lalu pindah ke kolom
                berikutnya â€” daftar objek dan coverage klaim itu dimuat saat kolomnya
                ditinggalkan.
              </p>
            ) : (
              <ClaimLookupState
                claimNo={lookupClaim}
                isPending={lookup.isPending}
                isError={lookup.isError}
                error={lookup.error}
                objects={objectChoices.length}
                coverages={coverageChoices.length}
              />
            )}
          </div>
        )}
      </fieldset>

      {/*
        Blok kedua, EMPAT kolom, dua baris:

          Mata Uang âœ±    | Minimum Salvage | Nama PIC Survey  | No Telp PIC Survey
          Status Salvage âœ± | Nilai Penawaran | Email PIC Survey | Share Tertanggung
      */}
      <fieldset className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <legend className="sr-only">Nilai dan PIC Survey</legend>

        {/*
          Mata Uang adalah DAFTAR PILIHAN, bukan kotak ketik.

          Di layar lama ia dropdown bersumber Report Definition `SelectCurrency_RD`, yang
          membaca kelas `ASM-FW-GISFW-Int-CURRENCY` â€” tabel `POOLDATA.CURRENCY`. Isinya
          karena itu datang dari server, bukan ditulis di sini: mengarang kode mata uang
          berarti menawarkan kode yang mungkin tidak dikenali sistem hilir.

          `required` menyala HANYA bila daftarnya benar-benar terisi. Bila tabel mata
          uangnya tidak terbaca, kolom ini tidak boleh menahan pengajuan â€” kegagalan
          membaca satu master tidak boleh berubah menjadi layar yang tidak dapat dipakai.
        */}
        <SelectField
          id="salvage-mata-uang"
          label="Mata Uang"
          emptyText="--Pilih--"
          options={currencyOptions.map((option) => ({
            value: option.kode,
            label: option.label,
          }))}
          value={form.mata_uang}
          onChange={(event) => set('mata_uang', event.target.value)}
          error={violations['mata_uang']}
          required={currencyOptions.length > 0}
        />

        {/*
          Nilai uang diketik sebagai TEKS, bukan `type="number"`.

          `type="number"` menyerahkan pembacaannya ke peramban, dan peramban membulatkannya
          menjadi bilangan pecahan biner â€” persis yang `D-51` larang. Pemeriksaan bentuknya
          dikerjakan server, yang menolak isian yang bukan angka beserta nama isiannya.
        */}
        <FormField
          id="salvage-minimum"
          label="Minimum Salvage"
          inputMode="decimal"
          value={form.minimum_salvage}
          onChange={(event) => set('minimum_salvage', event.target.value)}
          failure={violations['minimum_salvage']}
        />

        <FormField
          id="salvage-pic-survey"
          label="Nama PIC Survey"
          value={form.nama_pic_survey}
          onChange={(event) => set('nama_pic_survey', event.target.value)}
        />
        <FormField
          id="salvage-telp-survey"
          label="No Telp PIC Survey"
          value={form.no_telp_pic_survey}
          onChange={(event) => set('no_telp_pic_survey', event.target.value)}
        />

        {/* "--Pilih--" ada di layar lama apa adanya (`pyCaption --Pilih--`). */}
        <SelectField
          id="salvage-status"
          label="Status Salvage"
          emptyText="--Pilih--"
          options={statusOptions.map((option) => ({
            value: option.kode,
            label: option.label,
          }))}
          value={form.status_salvage}
          onChange={(event) => set('status_salvage', event.target.value)}
          error={violations['status_salvage']}
          required={statusOptions.length > 0}
        />

        <FormField
          id="salvage-penawaran"
          label="Nilai Penawaran"
          inputMode="decimal"
          value={form.nilai_penawaran}
          onChange={(event) => set('nilai_penawaran', event.target.value)}
          failure={violations['nilai_penawaran']}
        />

        <FormField
          id="salvage-email-survey"
          label="Email PIC Survey"
          type="email"
          value={form.email_pic_survey}
          onChange={(event) => set('email_pic_survey', event.target.value)}
        />
        <FormField
          id="salvage-share"
          label="Share Tertanggung"
          inputMode="decimal"
          value={form.share_tertanggung}
          onChange={(event) => set('share_tertanggung', event.target.value)}
          failure={violations['share_tertanggung']}
        />
      </fieldset>

      <DetailItemTable
        items={items}
        uploadColumns={uploadColumns}
        uploadNote={uploadNote}
        uploadError={upload.error}
        isUploading={upload.isPending}
        fileInput={fileInput}
        onChooseFile={chooseFile}
        onAdd={() => setItems((current) => [...current, emptyItem()])}
        onChange={(index, patch) =>
          setItems((current) =>
            current.map((item, position) =>
              position === index ? { ...item, ...patch } : item,
            ),
          )
        }
        onRemove={(index) =>
          setItems((current) => current.filter((_, position) => position !== index))
        }
        failure={violations['detail_item_salvage']}
        actions={
          <>
            {/*
              Tombol ini MEMBUKA MODAL, tidak langsung memilih berkas.

              Begitulah layar lama: ia local action `UploadDocument_Salvage`
              (`Section/TambahData_Salvage-Section.xml:13259`) yang membuka jendela
              tersendiri berisi kedua batas unggahan sebelum berkasnya dipilih.
            */}
            <Button type="button" tone="kedua" onClick={() => setUploading(true)}>
              Upload file
            </Button>

            {/*
              Kabarnya digambar DI BAWAH tombolnya, bukan di dalam modal â€” modalnya sudah
              tertutup saat kabar ini datang, dan tanpa jejak di layar pengguna tidak
              punya cara memastikan dokumennya benar-benar tersimpan.
            */}
            {docNote !== '' && (
              <p className="rounded-kontrol bg-emerald-50 px-3 py-2 text-sm text-emerald-900">
                {docNote}
              </p>
            )}

            {/*
              "Batal" TIDAK ada di layar lama â€” di sana form ini modal dengan tanda
              silang di sudutnya. Di sini ia menggantikan tanda silang itu: tanpanya
              tidak ada jalan keluar dari form selain menyimpan.
            */}
            <Button type="button" tone="halus" onClick={onClose}>
              Batal
            </Button>

            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? 'Menyimpanâ€¦' : 'Submit'}
            </Button>
          </>
        }
      />

      {/*
        Grid riwayat digambar HANYA bila klaimnya pernah diajukan salvage.

        Begitulah layar lama: pada pengajuan baru ia tidak muncul sama sekali, dan baru
        tampak pada klaim yang sudah punya pengajuan sebelumnya.
      */}
      {riwayat.length > 0 && <RiwayatSalvage rows={riwayat} />}

      {uploading && (
        <UploadDocumentModal
          claimNo={form.nomor_klaim}
          salvageID={editing?.id_salvage ?? ''}
          onClose={() => setUploading(false)}
          onSaved={setDocNote}
        />
      )}
    </form>
  )
}

/**
 * namesOf menyusun daftar saran dari pilihan, TANPA nama yang berulang.
 *
 * # Kenapa pengulangan nyata terjadi
 *
 * Karena coverage dikenali per objek, bukan per klaim: satu klaim yang punya dua objek
 * dapat punya "Kebakaran" pada keduanya. Menampilkannya dua kali tidak menambah satu pun
 * keterangan â€” kedua barisnya terbaca sama persis â€” dan `<datalist>` yang memuat dua
 * pilihan bernilai sama membuat React menandai kunci ganda.
 *
 * Nama yang sama tetap dapat dipilih; yang hilang hanyalah barisnya yang kedua. Begitu
 * objeknya dipilih, daftarnya menyempit dan penunjuk yang terpasang menjadi pasti.
 */
function namesOf(choices: { nama: string }[]): string[] {
  return [...new Set(choices.map((choice) => choice.nama))]
}

/**
 * hintFor menyusun keterangan di bawah kedua autocomplete.
 *
 * Isinya menjawab satu pertanyaan yang pasti muncul: "daftarnya kosong, rusak atau memang
 * begitu?" Kedua kolom menerima ketikan bebas, sehingga daftar kosong BUKAN jalan buntu â€”
 * dan mengatakannya di tempat lebih murah daripada membiarkan orang menebaknya.
 */
function hintFor(count: number, what: string): string {
  if (count === 0) {
    return `Belum ada ${what} yang dapat dipilih. Nama boleh diketik langsung.`
  }
  return `${count} ${what} tersedia. Nama di luar daftar boleh diketik langsung.`
}

type LookupStateProps = {
  claimNo: string
  isPending: boolean
  isError: boolean
  error: unknown
  objects: number
  coverages: number
}

/**
 * ClaimLookupState melaporkan hasil pencarian klaim pada form Tambah.
 *
 * # Kenapa `404` TIDAK digambar sebagai kegagalan
 *
 * Karena ia jawaban, bukan gangguan: nomor klaim yang salah ketik adalah hal yang lazim,
 * dan memberinya rupa pesan galat merah membuat kekeliruan sehari-hari terbaca seperti
 * kerusakan sistem. Yang digambar merah hanyalah kegagalan yang benar-benar di luar
 * kendali pengguna.
 *
 * Nomor klaim yang DITAMPILKAN kembali bukan hiasan: ia yang membedakan "klaim ini tidak
 * ada" dari "saya salah mengetik satu huruf", dan tanpa itu keduanya terbaca sama.
 */
function ClaimLookupState({
  claimNo,
  isPending,
  isError,
  error,
  objects,
  coverages,
}: LookupStateProps) {
  if (isPending) {
    return (
      <p className="text-sm text-slate-600" role="status">
        Memuat objek dan coverage klaim <strong>{claimNo}</strong>â€¦
      </p>
    )
  }

  if (isError) {
    const notFound = error instanceof APIError && error.status === 404
    if (notFound) {
      return (
        <p
          className="rounded-kontrol bg-amber-50 px-3 py-2 text-sm text-amber-900"
          role="status"
        >
          Klaim <strong>{claimNo}</strong> tidak ditemukan di portal ini. Periksa
          nomornya, atau pastikan entitas yang dipilih di bilah atas sudah benar.
        </p>
      )
    }

    return (
      <ErrorMessage
        title="Objek dan coverage klaim tidak dapat dimuat"
        description={messageOf(error)}
        tone="gangguan"
      />
    )
  }

  if (objects === 0 && coverages === 0) {
    return (
      <p
        className="rounded-kontrol bg-amber-50 px-3 py-2 text-sm text-amber-900"
        role="status"
      >
        Klaim <strong>{claimNo}</strong> ditemukan, tetapi objek dan coverage-nya belum
        terisi. Kedua nama di atas dapat diketik langsung.
      </p>
    )
  }

  return (
    <p className="text-sm text-emerald-800" role="status">
      Klaim <strong>{claimNo}</strong> ditemukan.
    </p>
  )
}

type DetailProps = {
  items: DetailItem[]
  uploadColumns: string[]
  uploadNote: string
  uploadError: unknown
  isUploading: boolean
  fileInput: React.RefObject<HTMLInputElement | null>
  onChooseFile: (event: React.ChangeEvent<HTMLInputElement>) => void
  onAdd: () => void
  onChange: (index: number, patch: Partial<DetailItem>) => void
  onRemove: (index: number) => void
  failure: string | undefined

  /** Tombol yang digambar DI SEBELAH KANAN tabel, sebagaimana di layar lama. */
  actions: ReactNode
}

/** Kelas isian di dalam sel grid â€” dibuat sekali supaya kelima kolom sama persis. */
const SEL_INPUT =
  'w-full rounded-kontrol border border-slate-300 bg-white px-2 py-1 text-sm ' +
  'text-slate-900 focus:outline-none focus-visible:border-blue-500 ' +
  'focus-visible:ring-2 focus-visible:ring-blue-500/25'

/**
 * Grid **"Detail Item Salvage"**.
 *
 * # Barisnya DIISI LANGSUNG di dalam tabel
 *
 * Itu bentuk aslinya: `Section/TambahData_Salvage-Section.xml:8691` menggambar page list
 * `DetailSalvage.Data` sebagai grid yang ketiga kolomnya `pxTextInput` â€” `.Item`,
 * `.Quantity`, dan `.REMARKS` â€” dengan tautan **Tambah** dan **Hapus** di atasnya.
 *
 * Sebelumnya grid ini hanya dapat diisi lewat CSV, dan itu meleset: unggahan CSV adalah
 * jalur KEDUA di layar lama (tombol "Upload Detail Salvage"), bukan satu-satunya.
 * Keduanya kini ada, dan keduanya mengisi tabel yang sama.
 *
 * # Dua kolom yang digambar tetapi BELUM dapat disimpan
 *
 * **Harga Total** dan **Upload file** ada di layar yang berjalan sekarang, tetapi tidak
 * ada di export â€” dan yang lebih menentukan, `Database/INSERT_SALVAGE_DETAILS.prc` yang
 * kita punya **tidak punya parameter untuk keduanya**. Prosedurnya hanya menerima satu
 * angka, `tTOTALHARGA`, dan angka itu sudah dipakai kolom "Jumlah Item".
 *
 * Keduanya karena itu digambar dalam keadaan mati, dengan alasannya tertulis di bawah
 * tabel â€” bukan dihilangkan, supaya ketiadaannya terlihat, dan bukan pula dibuat aktif,
 * supaya tidak ada yang mengetik nilai yang diam-diam hilang saat disimpan.
 */
function DetailItemTable({
  items,
  uploadColumns,
  uploadNote,
  uploadError,
  isUploading,
  fileInput,
  onChooseFile,
  onAdd,
  onChange,
  onRemove,
  failure,
  actions,
}: DetailProps) {
  return (
    <section className="space-y-3">
      <h3 className="text-sm font-semibold text-slate-900">Detail Item Salvage</h3>

      {/* Tautan Tambah dan Hapus berada DI ATAS tabel, sama seperti di layar lama. */}
      <div className="flex flex-wrap items-center gap-4">
        <button
          type="button"
          onClick={onAdd}
          className="rounded-kontrol text-sm font-medium text-blue-700 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
        >
          + Tambah
        </button>

        {/*
          "Hapus" membuang baris TERAKHIR.

          Di layar lama ia membuang baris yang sedang dipilih, dan pemilihan baris itu
          sendiri tidak tergambar. Membuang yang terakhir adalah padanan yang paling
          mendekati tanpa menambah kolom aksi yang tidak ada di layar lama â€” dan baris
          terakhir pula yang baru saja ditambahkan, jadi itulah yang paling sering
          dibatalkan orang.
        */}
        <button
          type="button"
          onClick={() => onRemove(items.length - 1)}
          disabled={items.length === 0}
          className="rounded-kontrol text-sm font-medium text-blue-700 hover:underline disabled:text-slate-400 disabled:no-underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
        >
          Hapus
        </button>

        <input
          ref={fileInput}
          type="file"
          accept=".csv,text/csv"
          onChange={onChooseFile}
          className="hidden"
          aria-hidden="true"
          tabIndex={-1}
        />
        <Button
          type="button"
          tone="kedua"
          disabled={isUploading}
          onClick={() => fileInput.current?.click()}
        >
          {isUploading ? 'Membaca berkasâ€¦' : 'Upload Detail Salvage'}
        </Button>
      </div>

      {uploadError != null && (
        <ErrorMessage
          title="Berkas tidak dapat dibaca"
          description={messageOf(uploadError)}
          tone="penolakan"
        />
      )}
      {uploadNote !== '' && (
        <p className="rounded-kontrol bg-amber-50 px-3 py-2 text-sm text-amber-900">
          {uploadNote}
        </p>
      )}
      {failure !== undefined && (
        <p className="text-sm text-red-700" role="alert">
          {failure}
        </p>
      )}

      {/* Tabel di kiri, tombol di kanan â€” sebaris, sebagaimana di layar lama. */}
      <div className="flex flex-wrap items-start gap-6">
        <div className="min-w-0 grow overflow-x-auto rounded-kartu border border-slate-200">
          <table className="w-full min-w-max text-sm" aria-label="Detail Item Salvage">
            <thead className="bg-slate-50 text-left text-xs font-semibold tracking-wide text-slate-600 uppercase">
              <tr>
                <th scope="col" className="px-3 py-2.5">
                  Nama Item
                </th>
                <th scope="col" className="px-3 py-2.5">
                  Jumlah Item / Qty
                </th>
                <th scope="col" className="px-3 py-2.5">
                  Remark
                </th>
              </tr>
            </thead>

            <tbody className="divide-y divide-slate-100">
              {items.length === 0 ? (
                <tr>
                  {/* Teks kekosongan disalin apa adanya dari layar lama. */}
                  <td colSpan={3} className="px-3 py-3 text-sm text-slate-500">
                    Data Tidak Ada
                  </td>
                </tr>
              ) : (
                items.map((item, index) => (
                  // Kunci barisnya POSISI, bukan isinya.
                  //
                  // Baris baru selalu lahir kosong, dan dua baris kosong berisi hal yang
                  // sama persis. Kunci berbasis isi akan membuat keduanya bertukar tempat
                  // saat salah satunya diketik.
                  <tr key={index}>
                    <td className="px-3 py-2">
                      <input
                        aria-label={`Nama Item baris ${index + 1}`}
                        className={SEL_INPUT}
                        value={item.nama_item}
                        onChange={(event) =>
                          onChange(index, { nama_item: event.target.value })
                        }
                      />
                    </td>

                    <td className="px-3 py-2">
                      <input
                        aria-label={`Jumlah Item baris ${index + 1}`}
                        inputMode="decimal"
                        className={`${SEL_INPUT} text-right tabular-nums`}
                        value={item.jumlah_item}
                        onChange={(event) =>
                          onChange(index, { jumlah_item: event.target.value })
                        }
                      />
                    </td>

                    <td className="px-3 py-2">
                      <input
                        aria-label={`Remark baris ${index + 1}`}
                        className={SEL_INPUT}
                        value={item.remark}
                        onChange={(event) =>
                          onChange(index, { remark: event.target.value })
                        }
                      />
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="flex flex-wrap items-center gap-3 pt-2">{actions}</div>
      </div>

      <p className="text-sm text-slate-600">
        Berkas CSV berkolom{' '}
        <span className="font-mono text-xs">{uploadColumns.join(', ')}</span>. Pemisah
        antarkolom adalah <strong>koma</strong> â€” berkas yang disimpan Excel dengan setelan
        Indonesia memakai titik koma dan tidak akan terbaca. Baris dari berkas DITAMBAHKAN
        ke yang sudah ada, bukan menggantikannya.
      </p>

      {/*
        Kolom "Satuan" ADA di berkas CSV tetapi TIDAK digambar di tabel ini â€” layar lama
        pun hanya menggambar tiga kolom. Baris yang diketik langsung karena itu tersimpan
        tanpa satuan, sementara baris dari CSV membawa satuannya.
      */}
    </section>
  )
}

/**
 * today mengembalikan tanggal hari ini berbentuk `YYYY-MM-DD`.
 *
 * Ia dipakai sebagai nilai awal isian "Tanggal Input" saja. Tanggal yang benar-benar
 * tersimpan tetap yang dikirim form ini, dan server tidak menggantinya â€” berbeda dari
 * `INSERT_PLADLA` yang membuang tanggal pilihan pengguna dan memakai waktu sistem
 * (`D-49` butir 7).
 */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}

/** emptyItem adalah satu baris Detail Item Salvage yang baru ditambahkan. */
function emptyItem(): DetailItem {
  return { nama_item: '', jumlah_item: '', satuan: '', remark: '' }
}

/**
 * formatDateTime menggambar "Tanggal Input" sebagaimana layar lama menggambarnya â€”
 * `dd/mm/yyyy HH:MM`.
 *
 * Jamnya diambil saat form digambar, bukan dari isian: yang dikirim saat menyimpan tetap
 * tanggalnya saja, dan jam di sini hanya menyatakan kapan pengisian dimulai.
 */
function formatDateTime(isoDate: string): string {
  const now = new Date()
  const jam = String(now.getHours()).padStart(2, '0')
  const menit = String(now.getMinutes()).padStart(2, '0')

  const bagian = isoDate.split('-')
  if (bagian.length !== 3) return `${isoDate} ${jam}:${menit}`

  return `${bagian[2]}/${bagian[1]}/${bagian[0]} ${jam}:${menit}`
}

function today(): string {
  const now = new Date()
  const bulan = String(now.getMonth() + 1).padStart(2, '0')
  const tanggal = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${bulan}-${tanggal}`
}

// Grid "Detail History Salvage" hidup di `RiwayatSalvage.tsx` — ia dipakai DUA layar,
// form ini dan panel rincian, sebagaimana di Pega satu section yang sama disisipkan
// `TambahData_Salvage` maupun `DataDetail_Salvage`.
