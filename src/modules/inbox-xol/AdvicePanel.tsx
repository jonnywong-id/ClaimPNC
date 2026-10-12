import { useState } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { downloadURL, useAdviceSearch, useCauseOfLoss, useUnduhBerkas } from './api'
import { isValidationError, messageOf, violationsOf } from './errors'
import { EMPTY_ADVICE_FORM, type Advice, type AdviceForm, type AdviceType } from './types'

/**
 * Panel PLA/DLA — isi "Cari Data DLA PLA XOL", sekaligus grid bawah "Generated DLA PLA XOL".
 *
 * # Kenapa SATU panel dipakai dua wadah
 *
 * Karena di sistem lama keduanya menjalankan hal yang sama. Wadah "Generated" (`source=='1'`)
 * dan wadah "Cari Data" (`source=='2'`) sama-sama memanggil `BrowseDataXOLPLADLAGenerated`
 * dengan tiga parameter yang sama — tahun, penyebab kerugian, dan tipe. Yang membedakannya
 * hanyalah dari mana ketiga nilai itu diambil: yang satu dari baris yang sedang disorot,
 * yang lain dari isian pencarian.
 *
 * Menulis dua komponen berisi tabel yang sama akan menggandakan kolom, format, dan
 * perlakuan galatnya — lalu membiarkan keduanya menyimpang diam-diam.
 *
 * # Kolomnya
 *
 * Diambil apa adanya dari `Section/InboxClaimXOL-Section.xml`: NO PLA / DLA · Nama
 * Insurance · Nama Layer · Tahun · Kurs (IDR) · Share Percent · Email · Remark.
 */
export function AdvicePanel() {
  const [form, setForm] = useState<AdviceForm>(EMPTY_ADVICE_FORM)
  const [picks, setPicks] = useState<Picks>(NOTHING_PICKED)
  const [blocked, setBlocked] = useState(false)
  const [submitted, setSubmitted] = useState<AdviceForm | null>(null)

  const causes = useCauseOfLoss()
  const search = useAdviceSearch(submitted ?? EMPTY_ADVICE_FORM, submitted !== null)

  const violations = violationsOf(search.error)
  const rows = search.data?.pemberitahuan ?? []
  const tipe = resolveType(picks)

  const causeOptions = (causes.data?.sebab_kerugian ?? []).map((cause) => ({
    // Nilainya DESKRIPSI, bukan ID: itulah yang tersimpan di kolom CAUSEOFLOSS pada
    // tabel PLA/DLA. Mengirim ID akan menghasilkan daftar kosong tanpa pesan galat.
    value: cause.deskripsi,
    label: cause.deskripsi,
  }))

  return (
    <div className="mt-4 space-y-4">
      <form
        className="rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut"
        onSubmit={(event) => {
          event.preventDefault()

          // Tanpa PLA maupun DLA, aktivitas lama menempuh cabang "DATA" yang tidak
          // menjalankan satu pun kueri PLA/DLA. Permintaannya DITAHAN di sini, bukan
          // dikirim untuk ditolak server: `tipe` kosong menghasilkan galat validasi yang
          // menyalahkan pengguna atas pilihan yang memang belum ada padanannya.
          if (tipe === '') {
            setBlocked(true)
            return
          }

          setBlocked(false)
          setSubmitted({ ...form, tipe })
        }}
      >
        <div className="max-w-xl space-y-4">
          <SelectField
            id="xol-sebab"
            label="Cause Of Loss"
            options={causeOptions}
            emptyText={causes.isPending ? '— memuat —' : '— pilih penyebab —'}
            value={form.sebab_kerugian}
            error={violations['sebab_kerugian']}
            onChange={(event) => setForm({ ...form, sebab_kerugian: event.target.value })}
          />

          <Field
            id="xol-tahun"
            label="Date Of Loss"
            inputMode="numeric"
            placeholder="2024"
            hint={
              'Diisi TAHUN perjanjian. Judulnya dipertahankan seperti di aplikasi lama, ' +
              'tetapi kueri lama membandingkannya dengan kolom TAHUN — bukan dengan tanggal.'
            }
            value={form.tahun}
            error={violations['tahun']}
            onChange={(event) => setForm({ ...form, tahun: event.target.value })}
          />
        </div>

        <TypePicker
          picks={picks}
          onChange={(next) => {
            setPicks(next)
            setBlocked(false)
          }}
          error={violations['tipe']}
        />

        <div className="mt-4 flex flex-wrap items-center gap-2">
          {/*
            Labelnya "Cari Data", bukan "Cari Data DLA PLA XOL".

            Yang kedua adalah nama tombol PEMBUKA panel ini di deret atas; yang menjalankan
            pencariannya di dalam panel bernama "Cari Data" (`pyButtonLabel Cari Data` pada
            `Section/Sec_Detail_claim_XOL-Section.xml:5423`). Menyamakan keduanya membuat
            satu layar memuat tiga tombol bernama persis sama yang berbuat hal berbeda.
          */}
          <Button type="submit" tone="utama" disabled={search.isFetching}>
            {search.isFetching ? 'Mencari…' : 'Cari Data'}
          </Button>

          {submitted !== null && rows.length > 0 && (
            <DownloadButton form={submitted} />
          )}
        </div>

        {blocked && <DataModeNotice />}

        {causes.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Daftar penyebab kerugian tidak dapat dimuat"
              description={messageOf(causes.error)}
              tone="gangguan"
            />
          </div>
        )}
      </form>

      {submitted !== null && (
        <DataTable<Advice>
          columns={adviceColumns}
          rows={rows}
          rowKey={(row) => `${row.nomor_asli}|${row.revisi}|${row.nama_layer}`}
          title={`Pemberitahuan ${submitted.tipe} — ${submitted.tahun} · ${submitted.sebab_kerugian}`}
          // Kotak cari bawaan disembunyikan: panel ini sudah punya formulir pencariannya
          // sendiri di atas, dan dua kotak yang mencari hal berbeda tidak dapat
          // dibedakan pengguna.
          hideSearch
          isLoading={search.isPending}
          error={
            search.isError && !isValidationError(search.error) ? (
              <ErrorMessage
                title="Pencarian tidak dapat dijalankan"
                description={messageOf(search.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          emptyMessage="Belum ada pemberitahuan yang diterbitkan untuk tahun dan penyebab kerugian ini."
        />
      )}
    </div>
  )
}

/**
 * Picks adalah ketiga kotak centang apa adanya — tiga penanda yang BERDIRI SENDIRI.
 *
 * Sengaja tidak dimodelkan sebagai satu pilihan tunggal, karena di layar lama ketiganya
 * memang kotak centang yang dapat dicentang bersamaan. Yang menyatukannya menjadi satu
 * tipe adalah `resolveType`, persis seperti di aktivitasnya.
 *
 * Ketiganya terikat ke properti yang namanya tidak ada hubungannya dengan isinya —
 * `Province`, `ProvinceID`, dan `Imei` — sisa penggunaan ulang properti bawaan yang
 * namanya tidak pernah dibetulkan. Di sini namanya dibetulkan; perilakunya tidak.
 */
type Picks = { pla: boolean; dla: boolean; data: boolean }

const NOTHING_PICKED: Picks = { pla: false, dla: false, data: false }

/**
 * resolveType menyatukan tiga kotak centang menjadi satu tipe, memakai URUTAN PRIORITAS.
 *
 * Aturannya disalin apa adanya dari `Activity/BrowseDataXOLPLADLAGenerated-Act.xml:976`:
 *
 *	Local.tipe = @if(Province=="true","PLA", @if(ProvinceID=="true","DLA",""))
 *
 * Artinya PLA MENANG atas DLA bila keduanya dicentang, dan kotak "DATA" tidak ikut
 * menentukan apa pun — ia hanya berarti "tidak keduanya". Sistem lama mengubah hasil
 * kosong itu menjadi label "DATA" (`:2257`), bukan menjadi jenis pencarian ketiga.
 *
 * Tiga kotak centang untuk tiga pilihan yang saling meniadakan sebenarnya kelompok radio
 * yang ditulis keliru. Ia TIDAK dibetulkan menjadi radio: mencentang PLA dan DLA sekaligus
 * masih sah di layar lama, dan mengubahnya menjadi radio akan menghalangi kombinasi yang
 * hari ini dapat dilakukan pengguna.
 */
function resolveType(picks: Picks): AdviceType | '' {
  if (picks.pla) return 'PLA'
  if (picks.dla) return 'DLA'
  return ''
}

/** TypePicker menggambar ketiga kotak centang sebaris, seperti di layar lama. */
function TypePicker({
  picks,
  onChange,
  error,
}: {
  picks: Picks
  onChange: (next: Picks) => void
  error: string | undefined
}) {
  const boxes: Array<{ key: keyof Picks; label: string }> = [
    { key: 'pla', label: 'PLA' },
    { key: 'dla', label: 'DLA' },
    { key: 'data', label: 'DATA' },
  ]

  return (
    <fieldset className="mt-4">
      <legend className="text-sm font-medium text-slate-700">Tipe</legend>

      <div className="mt-2 flex flex-wrap items-center gap-x-10 gap-y-2">
        {boxes.map((box) => (
          <label key={box.key} className="flex items-center gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              checked={picks[box.key]}
              onChange={(event) => onChange({ ...picks, [box.key]: event.target.checked })}
              className="size-4 rounded border-slate-300"
            />
            {box.label}
          </label>
        ))}
      </div>

      {error && <p className="mt-1.5 text-sm text-red-700">{error}</p>}
    </fieldset>
  )
}

/**
 * DataModeNotice menjawab penekanan "Cari Data" tanpa PLA maupun DLA.
 *
 * Diam adalah jawaban terburuk di sini: tombol yang ditekan tanpa akibat apa pun
 * dilaporkan sebagai kerusakan, dan yang dicari pengguna berikutnya adalah tombolnya —
 * bukan pilihannya.
 */
function DataModeNotice() {
  return (
    <p className="mt-3 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
      Centang <span className="font-medium">PLA</span> atau{' '}
      <span className="font-medium">DLA</span> lebih dulu. Pilihan{' '}
      <span className="font-medium">DATA</span> saja tidak menjalankan pencarian — di
      aplikasi lama pun tidak ada kueri PLA/DLA untuk pilihan itu.
    </p>
  )
}

/**
 * DownloadButton menggantikan tombol "Print Perhitungan" — sementara, dan dinyatakan
 * demikian.
 *
 * Tombol lama menyusun dokumen PLA/DLA lewat engine cetak Pega. `D-11` menetapkan
 * dokumen dibuat sendiri di Go, dan Work Owner memutuskan 2026-09-20 bahwa tahap ini
 * cukup mengeluarkan berkas tabel lebih dulu.
 *
 * Labelnya karena itu TIDAK berbunyi "Print Perhitungan". Berkas yang tampak seperti
 * dokumen resmi padahal bukan adalah kekeliruan yang berakibat ke luar perusahaan —
 * PLA dan DLA dikirim kepada reasuradur.
 *
 * # Kenapa tombol, bukan `<a href>`
 *
 * Ia SEMPAT ditulis sebagai tautan polos dengan alasan "unduhan adalah navigasi, dan
 * peramban sudah menanganinya". Itu keliru: sesi dikirim sebagai header `Authorization`,
 * bukan cookie, dan peramban tidak mengirim header apa pun pada navigasi biasa — tautan
 * polos selalu dijawab `sesi_tidak_sah`.
 */
function DownloadButton({ form }: { form: AdviceForm }) {
  const unduh = useUnduhBerkas()

  return (
    <div>
      <button
        type="button"
        className="inline-flex items-center rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 shadow-lembut transition hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-4 disabled:text-slate-400"
        disabled={unduh.isPending}
        onClick={() =>
          unduh.mutate({
            alamat: downloadURL(form),
            namaBerkas: `perhitungan-${form.tipe.toLowerCase()}-${form.tahun}-${form.sebab_kerugian}.csv`,
          })
        }
      >
        {unduh.isPending ? 'Menyiapkan…' : 'Unduh perhitungan (CSV)'}
      </button>

      {unduh.isError && (
        <div className="mt-2">
          <ErrorMessage
            title="Berkas tidak dapat diunduh"
            description={messageOf(unduh.error)}
            tone="gangguan"
          />
        </div>
      )}
    </div>
  )
}

const adviceColumns: Column<Advice>[] = [
  {
    key: 'nomor',
    title: 'NO PLA / DLA',
    value: (row) => row.nomor,
    width: '12rem',
  },
  {
    key: 'nama_insurance',
    title: 'Nama Insurance',
    value: (row) => row.nama_insurance,
  },
  {
    key: 'nama_layer',
    title: 'Nama Layer',
    value: (row) => row.nama_layer,
    width: '8rem',
  },
  {
    key: 'tahun',
    title: 'Tahun',
    value: (row) => row.tahun,
    width: '6rem',
  },
  {
    key: 'kurs',
    title: 'Kurs (IDR)',
    value: (row) => String(row.kurs),
    render: (row) => new Intl.NumberFormat('id-ID').format(row.kurs),
    alignRight: true,
    width: '8rem',
  },
  {
    key: 'share_percent',
    title: 'Share Percent',
    value: (row) => String(row.share_percent),
    // Tanda persen ditambahkan DI SINI, bukan di SQL. Nilai bersatuan tidak dapat
    // dijumlahkan maupun diurutkan — kueri lama merangkainya sebagai `PERCENT || ' %'`.
    render: (row) =>
      `${new Intl.NumberFormat('id-ID', { maximumFractionDigits: 4 }).format(row.share_percent)} %`,
    alignRight: true,
    width: '8rem',
  },
  {
    key: 'email',
    title: 'Email',
    value: (row) => row.email,
  },
  {
    key: 'remark',
    title: 'Remark',
    value: (row) => row.remark,
  },
  {
    key: 'status',
    title: 'Status',
    noSort: true,
    width: '9rem',
    value: (row) => (row.sudah_disetujui ? 'Disetujui' : 'Menunggu'),
    render: (row) =>
      row.sudah_disetujui ? (
        <span className="inline-flex items-center rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 ring-1 ring-emerald-200">
          Disetujui
        </span>
      ) : (
        <span className="inline-flex items-center rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600 ring-1 ring-slate-200">
          Menunggu komite
        </span>
      ),
  },
]
