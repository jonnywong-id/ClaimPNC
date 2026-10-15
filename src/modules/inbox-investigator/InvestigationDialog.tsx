import { useEffect, useMemo, useState } from 'react'

import type { InvestigatorTask } from '@/api/types'
import { Button } from '@/components/Button'
import { DateField } from '@/components/DateField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'
import { TextAreaField } from '@/components/TextAreaField'
import { DocumentTab } from '@/modules/registrasi/EstimateTabs'

import {
  useInvestigation,
  useSubmitInvestigation,
  type InvestigationForm,
  type InvestigationVisibility,
} from './api'

/*
  Pilihan keenam isian radio, DISALIN dari rule Property-nya.

  Work Owner menyerahkan keenam berkasnya pada 2026-10-06, dan isinya mengoreksi EMPAT dari
  enam tebakan sebelumnya — yang seluruhnya "Ya"/"Tidak":

    IsInvestigated           Ya / Tidak                      tebakan benar
    SelectRS                 Rumah Sakit / Non Rumah Sakit   tebakan hampir benar
    PasienTerdaftar          YA / TIDAK                      huruf besar, bukan "Ya"
    FlagDOB                  Sesuai / Tidak Sesuai           BUKAN ya/tidak
    KonfirmasiModelKwitansi  SESUAI / TIDAK SESUAI / TIDAK ADA   TIGA pilihan, bukan dua
    TagihanLunasBelumLunas   LUNAS / BELUM LUNAS             BUKAN ya/tidak

  Dua di antaranya bukan sekadar beda kata: `FlagDOB` menanyakan KESESUAIAN, dan
  `KonfirmasiModelKwitansi` punya pilihan ketiga yang tidak akan pernah dapat dipilih
  pengguna bila ditebak dua.

  Huruf besar pada `PasienTerdaftar`, `KonfirmasiModelKwitansi`, dan
  `TagihanLunasBelumLunas` DIPERTAHANKAN apa adanya. Ia bentuk yang dibaca pengguna di layar
  lama (`D-13`); menyeragamkannya berarti menyunting layar yang sedang ditiru.
*/

/** `IsInvestigated` — "Dapat Diinvestigasi". */
const INVESTIGATED = [
  { value: '1', label: 'Ya' },
  { value: '0', label: 'Tidak' },
]

/** `SelectRS` — "Tempat Kejadian". */
const PLACE_KIND = [
  { value: '1', label: 'Rumah Sakit' },
  { value: '0', label: 'Non Rumah Sakit' },
]

/** `PasienTerdaftar` — "Peserta Terdaftar Di RS". */
const PATIENT_REGISTERED = [
  { value: '1', label: 'YA' },
  { value: '0', label: 'TIDAK' },
]

/** `FlagDOB` — "Verifikasi Tanggal Lahir". */
const BIRTH_VERIFIED = [
  { value: '1', label: 'Sesuai' },
  { value: '0', label: 'Tidak Sesuai' },
]

/** `KonfirmasiModelKwitansi` — "Konfirmasi Model Kwitansi". TIGA pilihan. */
const RECEIPT_CONFIRMATION = [
  { value: '1', label: 'SESUAI' },
  { value: '0', label: 'TIDAK SESUAI' },
  { value: '2', label: 'TIDAK ADA' },
]

/** `TagihanLunasBelumLunas` — "Tagihan Lunas/Belum Lunas". */
const BILL_SETTLED = [
  { value: '1', label: 'LUNAS' },
  { value: '0', label: 'BELUM LUNAS' },
]

/** Isian kosong, dipakai sebelum jawaban server tiba. */
const EMPTY: InvestigationForm = {
  referensi: '',
  urutan_survei: 1,
  urutan: 1,
  tanggal_investigasi: null,
  dapat_diinvestigasi: '',
  tempat_kejadian: '',
  nama_rumah_sakit: '',
  nama_tempat_lainnya: '',
  alamat_rs_klinik: '',
  nomor_rekam_medik: '',
  nama_pasien: '',
  tanggal_lahir: null,
  verifikasi_tanggal_lahir: '',
  keterangan_tanggal_lahir: '',
  peserta_terdaftar: '',
  keterangan_pendaftaran: '',
  tanggal_perawatan: null,
  tanggal_selesai_perawatan: null,
  total_pengajuan: '',
  tagihan_lunas: '',
  bayar_pasien: 'false',
  bayar_perusahaan: 'false',
  bayar_asuransi_lain: 'false',
  tidak_ada_pembayaran: 'false',
  nama_asuransi_lain: '',
  konfirmasi_kwitansi: '',
  nama_pic_rs: '',
  nama_penelepon: '',
  nama_karyawan: '',
  kode_area_telepon: '',
  nomor_telepon: '',
  ekstensi_telepon: '',
  hasil_investigasi: '',
}

/** Dua tab formulir, dari `pyTitle` section lamanya. */
const TABS = ['Investigasi', 'Unggah Dokumen'] as const

type Props = {
  task: InvestigatorTask
  onClose: () => void
}

/**
 * Formulir kerja Investigator.
 *
 * Pengganti Flow Action `InputInvestigator` — yang di Pega terbuka sebagai **modal**
 * berukuran 80 × 82, bukan sebagai halaman (`Flow Action/InputInvestigator-FA.xml:883`,
 * `:890`). Bentuk itu ditiru: ia dialog di atas daftar, sehingga tidak ada rute baru dan
 * daftar di belakangnya tetap pada tempatnya.
 *
 * # Menekan Simpan MEMINDAHKAN klaim
 *
 * `SetStatusInvestigator_Act` menyetel `SurveyStatus = 5`, `PNCStatus = 5`,
 * `StatusClaim = "1151"` (Analyst), `InvestTfDate`, dan `AnalystTransferDate`. Pekerjaannya
 * HILANG dari antrean setelah itu — ciri kedua Inbox pada `D-79` — dan dialog ini
 * menyebutkannya sebelum tombolnya ditekan.
 *
 * # Keenam isian bersyarat diputuskan SERVER
 *
 * `tampil` datang dari jawaban server, tidak dihitung ulang di sini. Syaratnya aturan yang
 * dibaca dari `pyCondition` section lama, dan aturan yang hidup di dua tempat akan berbeda
 * pada perubahan berikutnya.
 */
export function InvestigationDialog({ task, onClose }: Props) {
  const form = useInvestigation(task.referensi)
  const submit = useSubmitInvestigation()

  const [draft, setDraft] = useState<InvestigationForm>(EMPTY)
  const [loaded, setLoaded] = useState(false)

  // Dua tab, persis seperti di layar lama: Investigasi dan Unggah Dokumen.
  const [tab, setTab] = useState<(typeof TABS)[number]>('Investigasi')

  // Isian server dipakai SEKALI, saat jawaban pertama tiba. Menyalinnya pada setiap render
  // akan membuang ketikan pengguna setiap kali TanStack Query menyegarkan datanya.
  useEffect(() => {
    if (form.data && !loaded) {
      setDraft(form.data.investigasi)
      setLoaded(true)
    }
  }, [form.data, loaded])

  // Escape menutup dialog, seperti dialog mana pun yang dikenal pengguna. Tidak saat sedang
  // menyimpan: menutupnya di tengah jalan menyembunyikan hasilnya tanpa membatalkannya.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape' && !submit.isPending) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose, submit.isPending])

  /*
    Keadaan tampil dihitung dari JAWABAN SERVER, lalu disesuaikan dengan isian yang sedang
    diketik.

    Keduanya perlu: server yang menentukan aturannya, tetapi pengguna yang mengubah
    "Tempat Kejadian" harus melihat isiannya berganti SEKETIKA — bukan setelah menyimpan.
    Yang disalin di sini hanyalah penerapan aturan yang sama atas isian terbaru.
  */
  const shown: InvestigationVisibility = useMemo(() => {
    const hospital = draft.tempat_kejadian === '1'
    return {
      nama_rumah_sakit: hospital,
      nama_tempat_lainnya: !hospital,
      alamat_rs_klinik: hospital,
      // Dua tanda tanya, bukan satu. `tampil` dapat tidak ada sama sekali — pada jawaban
      // yang bentuknya tak terduga, pada pemuatan pertama, dan pada versi server yang
      // belum mengirimnya. Satu tanda tanya hanya menjaga `data`, dan sisanya melempar.
      tanggal_selesai_perawatan: form.data?.tampil?.tanggal_selesai_perawatan ?? true,
      nama_asuransi_lain:
        draft.bayar_perusahaan === 'true' || draft.bayar_asuransi_lain === 'true',
    }
  }, [draft, form.data])

  function set<K extends keyof InvestigationForm>(key: K, value: InvestigationForm[K]) {
    setDraft((before) => ({ ...before, [key]: value }))
  }

  /** toggle membalik satu kotak centang; nilainya teks `"true"`/`"false"`, bukan boolean. */
  function toggle(key: keyof InvestigationForm, checked: boolean) {
    set(key, (checked ? 'true' : 'false') as InvestigationForm[typeof key])
  }

  function save() {
    submit.mutate(
      { referensi: task.referensi, isian: draft },
      { onSuccess: onClose },
    )
  }

  const failure = submit.error instanceof Error ? submit.error.message : null

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-900/40 p-4 sm:p-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-investigasi"
    >
      <div className="w-full max-w-4xl rounded-kartu bg-white p-6 shadow-angkat">
        {/* Judulnya DISALIN dari rule field-value `pyCaption Form Investigasi Personal
            Accident` pada section lama, bukan dikarang. "Input Investigator" yang dipakai
            sebelumnya adalah nama Flow Action-nya — yang di Pega muncul sebagai judul
            jendela, bukan sebagai judul formulir yang dibaca petugas. */}
        <h2 id="judul-investigasi" className="text-lg font-semibold text-slate-900">
          Form Investigasi Personal Accident
        </h2>

        {/* Klaim yang sedang dikerjakan, BACA-SAJA. Keempatnya ada di formulir lama dan
            seluruhnya milik klaim, bukan milik baris investigasi. */}
        <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
          <dt className="text-slate-500">Nomor Case</dt>
          <dd className="font-medium text-slate-900">{task.nomor_case}</dd>
          <dt className="text-slate-500">Nama Tertanggung</dt>
          <dd className="text-slate-900">{task.nama_tertanggung || '—'}</dd>
          <dt className="text-slate-500">Nama Peserta</dt>
          <dd className="text-slate-900">{task.nama_peserta || '—'}</dd>
        </dl>

        {/* Dua tab milik Flow Action ini — `pyTitle` "Investigasi" dan "Unggah Dokumen"
            pada section lamanya. Ia BUKAN ketiga tab Pega di luar (Assignment ·
            Information · Audit), yang milik kerangka kasus, bukan milik formulir ini. */}
        <div className="mt-5 flex gap-1 border-b border-slate-200" role="tablist">
          {TABS.map((one) => (
            <button
              key={one}
              type="button"
              role="tab"
              aria-selected={tab === one}
              onClick={() => setTab(one)}
              className={
                'rounded-t-kontrol px-4 py-2 text-sm transition-colors ' +
                (tab === one
                  ? 'border-b-2 border-blue-600 font-medium text-blue-700'
                  : 'text-slate-600 hover:text-slate-900')
              }
            >
              {one}
            </button>
          ))}
        </div>

        {form.isPending ? (
          <p className="mt-6 text-sm text-slate-500">Memuat formulir investigasi…</p>
        ) : form.isError ? (
          <div className="mt-6">
            <ErrorMessage
              title="Formulir investigasi tidak dapat dimuat"
              description={
                form.error instanceof Error
                  ? form.error.message
                  : 'Coba tutup lalu buka kembali.'
              }
              tone="gangguan"
            />
          </div>
        ) : (
          <div className="mt-6 space-y-6">
            {/*
              URUTAN ISIAN DISALIN DARI PEGA, bukan disusun sendiri.

              Section lamanya memuat tiga kelompok ber-`pyTitle`, dan urutan sel di dalam
              kelompok "Investigasi" terbaca utuh:

                Tanggal Investigasi · Tempat Kejadian · Alamat RS/Klinik
                -- kelompok "Tanggal Perawatan/Kejadian" --
                Nomor Rekam Medik · Peserta Terdaftar · Total Pengajuan
                Tanggal Lahir · Verifikasi Tanggal Lahir · Keterangan Tambahan
                Konfirmasi Model Kwitansi · Dapat Diinvestigasi
                -- Yang Melakukan Pembayaran --
                Nama Asuransi/Perusahaan Lain · kontak · Hasil Investigasi

              Susunan sebelumnya menaruh "Dapat Diinvestigasi" di urutan KEDUA; di Pega ia
              urutan ke-13. Itu bukan selera — petugas yang hafal layar lamanya membaca
              formulir ini dari atas ke bawah.
            */}
            {/* Kepala kelompok pertanyaan. Ia caption nyata di section lama
                (`pyCaption Daftar Pertanyaan`), bukan penanda yang ditambahkan sendiri. */}
            {tab === 'Investigasi' && (
              <>
            <h3 className="text-base font-semibold text-slate-900">Daftar Pertanyaan</h3>

            <section className="grid gap-3 sm:grid-cols-2">
              <DateField
                id="inv-tanggal-investigasi"
                label="Tanggal Investigasi"
                value={isoDay(draft.tanggal_investigasi)}
                onChange={(iso) => set('tanggal_investigasi', dayToMoment(iso))}
                disabled={submit.isPending}
              />
              <SelectField
                id="inv-tempat-kejadian"
                label="Tempat Kejadian"
                options={PLACE_KIND}
                value={draft.tempat_kejadian}
                onChange={(event) => set('tempat_kejadian', event.target.value)}
                disabled={submit.isPending}
              />
              {shown.nama_rumah_sakit && (
                <Field
                  id="inv-nama-rumah-sakit"
                  label="Nama Rumah Sakit"
                  value={draft.nama_rumah_sakit}
                  onChange={(event) => set('nama_rumah_sakit', event.target.value)}
                  disabled={submit.isPending}
                />
              )}
              {shown.nama_tempat_lainnya && (
                <Field
                  id="inv-nama-tempat-lainnya"
                  label="Nama Tempat Lainnya"
                  value={draft.nama_tempat_lainnya}
                  onChange={(event) => set('nama_tempat_lainnya', event.target.value)}
                  disabled={submit.isPending}
                />
              )}
            </section>

            {shown.alamat_rs_klinik && (
              <TextAreaField
                id="inv-alamat"
                label="Alamat RS/Klinik/Lokasi Kejadian *"
                value={draft.alamat_rs_klinik}
                onChange={(event) => set('alamat_rs_klinik', event.target.value)}
                disabled={submit.isPending}
              />
            )}

            {/* Kelompok TERSENDIRI di Pega, ber-`pyTitle` sendiri. Tanggal Keluar
                bersyarat rawat inap. */}
            <fieldset className="rounded-kartu border border-slate-200 p-4">
              <legend className="px-1 text-sm font-medium text-slate-700">
                Tanggal Perawatan/Kejadian
              </legend>
              <div className="grid gap-3 sm:grid-cols-2">
                <DateField
                  id="inv-tanggal-perawatan"
                  label="Tanggal Masuk"
                  value={isoDay(draft.tanggal_perawatan)}
                  onChange={(iso) => set('tanggal_perawatan', dayToMoment(iso))}
                  disabled={submit.isPending}
                />
                {shown.tanggal_selesai_perawatan && (
                  <DateField
                    id="inv-tanggal-selesai"
                    label="Tanggal Keluar"
                    value={isoDay(draft.tanggal_selesai_perawatan)}
                    onChange={(iso) => set('tanggal_selesai_perawatan', dayToMoment(iso))}
                    disabled={submit.isPending}
                  />
                )}
              </div>
            </fieldset>

            <section className="grid gap-3 sm:grid-cols-2">
              <Field
                id="inv-rekam-medik"
                label="Nomor Rekam Medik *"
                value={draft.nomor_rekam_medik}
                onChange={(event) => set('nomor_rekam_medik', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-nama-pasien"
                label="Nama Pasien"
                value={draft.nama_pasien}
                onChange={(event) => set('nama_pasien', event.target.value)}
                disabled={submit.isPending}
              />
              <SelectField
                id="inv-peserta-terdaftar"
                label="Peserta Terdaftar Di RS"
                options={PATIENT_REGISTERED}
                value={draft.peserta_terdaftar}
                onChange={(event) => set('peserta_terdaftar', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-keterangan-pendaftaran"
                label="PT / Reg"
                value={draft.keterangan_pendaftaran}
                onChange={(event) => set('keterangan_pendaftaran', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-total-pengajuan"
                label="Total Pengajuan"
                value={draft.total_pengajuan}
                onChange={(event) => set('total_pengajuan', event.target.value)}
                disabled={submit.isPending}
              />
              <SelectField
                id="inv-tagihan-lunas"
                label="Tagihan Lunas/Belum Lunas"
                options={BILL_SETTLED}
                value={draft.tagihan_lunas}
                onChange={(event) => set('tagihan_lunas', event.target.value)}
                disabled={submit.isPending}
              />
              <DateField
                id="inv-tanggal-lahir"
                label="Tanggal Lahir"
                value={isoDay(draft.tanggal_lahir)}
                onChange={(iso) => set('tanggal_lahir', dayToMoment(iso))}
                disabled={submit.isPending}
              />
              <SelectField
                id="inv-verifikasi-lahir"
                label="Verifikasi Tanggal Lahir"
                options={BIRTH_VERIFIED}
                value={draft.verifikasi_tanggal_lahir}
                onChange={(event) => set('verifikasi_tanggal_lahir', event.target.value)}
                disabled={submit.isPending}
              />
            </section>

            <TextAreaField
              id="inv-keterangan-lahir"
              label="Keterangan Tambahan"
              value={draft.keterangan_tanggal_lahir}
              onChange={(event) => set('keterangan_tanggal_lahir', event.target.value)}
              disabled={submit.isPending}
            />

            <section className="grid gap-3 sm:grid-cols-2">
              <SelectField
                id="inv-konfirmasi-kwitansi"
                label="Konfirmasi Model Kwitansi"
                options={RECEIPT_CONFIRMATION}
                value={draft.konfirmasi_kwitansi}
                onChange={(event) => set('konfirmasi_kwitansi', event.target.value)}
                disabled={submit.isPending}
              />
              <SelectField
                id="inv-dapat-diinvestigasi"
                label="Dapat Diinvestigasi"
                options={INVESTIGATED}
                value={draft.dapat_diinvestigasi}
                onChange={(event) => set('dapat_diinvestigasi', event.target.value)}
                disabled={submit.isPending}
              />
            </section>

            {/* SATU kelompok pilihan, bukan empat pertanyaan yang berdiri sendiri — itulah
                bentuknya di formulir lama. */}
            <fieldset className="rounded-kartu border border-slate-200 p-4">
              <legend className="px-1 text-sm font-medium text-slate-700">
                Yang Melakukan Pembayaran
              </legend>
              <div className="grid gap-2 sm:grid-cols-2">
                {(
                  [
                    ['bayar_pasien', 'Pasien'],
                    ['bayar_perusahaan', 'Perusahaan'],
                    ['bayar_asuransi_lain', 'Asuransi Lain'],
                    ['tidak_ada_pembayaran', 'Tidak Ada Pembayaran'],
                  ] as const
                ).map(([key, label]) => (
                  <label key={key} className="flex items-center gap-2 text-sm text-slate-700">
                    <input
                      type="checkbox"
                      checked={draft[key] === 'true'}
                      disabled={submit.isPending}
                      onChange={(event) => toggle(key, event.target.checked)}
                    />
                    {label}
                  </label>
                ))}
              </div>

              {shown.nama_asuransi_lain && (
                <div className="mt-3">
                  <Field
                    id="inv-asuransi-lain"
                    label="Nama Asuransi/Perusahaan Lain"
                    value={draft.nama_asuransi_lain}
                    onChange={(event) => set('nama_asuransi_lain', event.target.value)}
                    disabled={submit.isPending}
                  />
                </div>
              )}
            </fieldset>

            <section className="grid gap-3 sm:grid-cols-2">
              <Field
                id="inv-pic-rs"
                label="Nama PIC Yang Dapat Dihubungi *"
                value={draft.nama_pic_rs}
                onChange={(event) => set('nama_pic_rs', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-penelepon"
                label="Nama Penelepon"
                value={draft.nama_penelepon}
                onChange={(event) => set('nama_penelepon', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-nama-karyawan"
                label="Nama Karyawan"
                value={draft.nama_karyawan}
                onChange={(event) => set('nama_karyawan', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-kode-area"
                label="Kode"
                value={draft.kode_area_telepon}
                onChange={(event) => set('kode_area_telepon', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-telepon"
                label="Nomor Telepon Yang Dapat Dihubungi *"
                value={draft.nomor_telepon}
                onChange={(event) => set('nomor_telepon', event.target.value)}
                disabled={submit.isPending}
              />
              <Field
                id="inv-ekstensi"
                label="No. Ext."
                value={draft.ekstensi_telepon}
                onChange={(event) => set('ekstensi_telepon', event.target.value)}
                disabled={submit.isPending}
              />
            </section>

            <TextAreaField
              id="inv-hasil"
              label="Hasil Investigasi"
              value={draft.hasil_investigasi}
              onChange={(event) => set('hasil_investigasi', event.target.value)}
              disabled={submit.isPending}
            />
              </>
            )}

            {/* Panel yang SAMA dengan tab Unggah Dokumen pada layar Registrasi — kategori,
                kewajiban unggah, dan tombol unggahnya. Ia dipakai ulang, bukan dibangun
                kedua kalinya: dua tafsir atas satu daftar dokumen akan berbeda pada
                perubahan berikutnya.

                `claimID` diisi `referensi`, yaitu pzInsKey — dan itu memang bentuk yang
                disimpan `T_CLAIM_OBJECTLIST.CLAIMID`, bukan nomor kasusnya. */}
            {tab === 'Unggah Dokumen' && (
              <DocumentTab claimID={task.referensi} line={task.lini_bisnis} />
            )}

            {/* Akibat menekan Simpan ditulis SEBELUM tombolnya, bukan sesudahnya.

                Berbeda dari keterangan tempelan yang dicabut di layar ini, kalimat ini
                menjelaskan AKIBAT SEBUAH TINDAKAN — dan tanpa itu barisnya hilang dari
                antrean tanpa satu pun tanda bahwa itu memang yang terjadi. */}
            <p className="rounded-kartu border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900">
              Menyimpan memindahkan klaim ini ke <span className="font-medium">Analyst</span>.
              Pekerjaannya akan hilang dari antrean investigator.
            </p>

            {failure && (
              <ErrorMessage
                title="Formulir belum tersimpan"
                description={failure}
                tone="gangguan"
              />
            )}

            <div className="flex flex-wrap justify-end gap-2">
              <Button tone="kedua" onClick={onClose} disabled={submit.isPending}>
                Back
              </Button>
              <Button onClick={save} disabled={submit.isPending}>
                {submit.isPending ? 'Menyimpan…' : 'Submit'}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

/**
 * isoDay memangkas stempel waktu kontrak menjadi tanggal `YYYY-MM-DD`.
 *
 * `DateField` bekerja dengan tanggal, sedangkan kontraknya membawa waktu penuh — bentuk
 * yang sama dengan yang disimpan Pega. Pemangkasan terjadi di sini, satu-satunya tempat
 * keduanya bertemu.
 */
function isoDay(moment: string | null): string {
  if (!moment) return ''
  return moment.slice(0, 10)
}

/**
 * dayToMoment mengembalikan tanggal menjadi stempel waktu kontrak.
 *
 * Tengah malam UTC, bukan tengah malam waktu mesin pengguna. Tanpa penetapan itu, dua
 * petugas di zona berbeda akan mengirim dua tanggal berbeda untuk hari yang sama — kelas
 * kesalahan yang melahirkan 118 penyesuaian tujuh jam di sistem lama (`R-12`).
 */
function dayToMoment(day: string): string | null {
  if (!day) return null
  return `${day}T00:00:00Z`
}
