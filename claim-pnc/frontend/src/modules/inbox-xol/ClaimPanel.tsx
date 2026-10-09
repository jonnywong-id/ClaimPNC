import { useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { SelectField } from '@/components/SelectField'

import {
  exportClaimDetailURL,
  uploadTemplateURL,
  useUploadSalvageMBU,
  useUploadInward,
  useUnduhBerkas,
  useGenerateAdvice,
  useBreakdown,
  useClaimSummary,
  useMasters,
  useRemoveDolCol,
  useClaimList,
  useSummaryBusiness,
} from './api'
import { messageOf } from './errors'
import { InsertDolColDialog } from './InsertDolColDialog'
import type {
  Breakdown,
  ClaimListItem,
  ClaimSummary,
  MasterXOL,
  SummaryBusiness,
  UploadResult,
} from './types'

/**
 * Dua blok teratas wadah `source=='1'` — tombol INSERT DOL DAN COL dan grid akumulasi.
 *
 * # Susunannya diambil dari mana
 *
 * `Section/InboxClaimXOL-Section.xml`: tombol "INSERT DOL DAN COL" (`:4621`) sendirian di
 * baris teratas, lalu grid "DATA XOL BASED ON DOL AND COL" (`:11601` — Date Of Loss ·
 * Cause Of Loss · Group Business · OS Value (USD) · Accepted Value (USD)), dan
 * `Sec_Detail_claim_XOL` sebagai rincian yang terbuka di balik satu baris.
 *
 * Judul kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega supaya
 * pengguna tidak perlu belajar ulang, dan itulah teks yang selama ini mereka baca.
 *
 * # Grid memuat SELURUH perjanjian, dan tidak ada yang "dipilih"
 *
 * `Activity/GetClaimXOL-Act.xml` step 4 me-loop `MstXOL.pxResults` — seluruh perjanjian,
 * tanpa batas awal maupun akhir — lalu meng-APPEND hasil tiap perjanjian ke satu daftar,
 * masing-masing dibagi kursnya sendiri. Itu sebabnya layar lama tidak punya pemilih
 * perjanjian, dan gridnya sudah terisi begitu wadah dibuka.
 *
 * Dua versi sebelumnya keliru di titik yang sama. Yang pertama menaruh dropdown
 * "Perjanjian XOL" yang tidak pernah ada di layar lama; yang kedua memilihkan perjanjian
 * terbaru sendiri — dan pada data nyata itu jatuh ke perjanjian 2027 yang belum punya
 * klaim, sehingga grid tetap kosong sementara Pega menampilkan baris 2018.
 *
 * Keduanya berakar pada satu anggapan yang tidak pernah diperiksa ke sumbernya: bahwa
 * grid ini milik SATU perjanjian.
 *
 * # Grid "PILIH MASTER XOL" tetap ada, tetapi bukan penyaring
 *
 * Ia hidup di dalam modal INSERT DOL DAN COL (`:7577`) dan menentukan perjanjian mana
 * yang akan DITULISI — bukan perjanjian mana yang ditampilkan.
 */
export function ClaimPanel() {
  const [insertOpen, setInsertOpen] = useState(false)

  const summary = useClaimSummary()

  return (
    <div className="mt-4 space-y-4">
      {/*
        Deret ini berisi SATU tombol saja, persis seperti baris teratas layar lama.
        Pemilih perjanjian tidak ada di sini melainkan di dalam modal yang dibukanya —
        grid "PILIH MASTER XOL" (`:7577`) memang hidup di sana.
      */}
      <div className="rounded-kartu border border-slate-200 bg-white p-3 shadow-lembut">
        <Button tone="kedua" onClick={() => setInsertOpen(true)}>
          INSERT DOL DAN COL
        </Button>
      </div>

      {insertOpen && <InsertDolColDialog onClose={() => setInsertOpen(false)} />}

      {/*
        Grid DIGAMBAR walau kosong.

        Di layar lama ia layout yang selalu ada — judul dan kolomnya terlihat sejak wadah
        terbuka, isinya saja yang kosong. Menggantinya dengan paragraf ajakan membuat judul
        "DATA XOL BASED ON DOL AND COL" hilang dari layar, dan itu menghapus satu-satunya
        petunjuk tentang apa yang akan muncul di situ.
      */}
      <ClaimTable
        rows={summary.data?.baris ?? []}
        loading={summary.isPending}
        error={summary.isError ? messageOf(summary.error) : null}
      />
    </div>
  )
}

type TableProps = {
  rows: ClaimSummary[]
  loading: boolean
  error: string | null
}

function ClaimTable({ rows, loading, error }: TableProps) {
  const columns: Column<ClaimSummary>[] = [
    {
      key: 'tanggal_kejadian',
      title: 'Date Of Loss',
      value: (row) => row.tanggal_kejadian,
      width: '9rem',
    },
    {
      key: 'sebab_kerugian',
      title: 'Cause Of Loss',
      value: (row) => row.sebab_kerugian,
    },
    {
      key: 'group_business',
      title: 'Group Business',
      value: (row) => row.group_business,
    },
    {
      key: 'nilai_outstanding',
      title: 'OS Value',
      value: (row) => String(row.nilai_outstanding),
      render: (row) => formatNumber(row.nilai_outstanding),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'nilai_akseptasi',
      title: 'Accepted Value',
      value: (row) => String(row.nilai_akseptasi),
      render: (row) => formatNumber(row.nilai_akseptasi),
      alignRight: true,
      width: '10rem',
    },
  ]

  return (
    <div className="space-y-4">
      <DataTable<ClaimSummary>
        columns={columns}
        rows={rows}
        rowKey={rowKey}
        title="DATA XOL BASED ON DOL AND COL"
        // Kursnya TIDAK disebut angkanya. Tiap baris dibagi kurs perjanjiannya
        // masing-masing, jadi satu angka di judul akan salah untuk sebagian baris —
        // dan justru terbaca meyakinkan.
        description="Seluruh perjanjian XOL, masing-masing dibagi kursnya sendiri — mengikuti perhitungan sistem lama."
        isLoading={loading}
        error={
          error ? (
            <ErrorMessage
              title="Akumulasi klaim tidak dapat dimuat"
              description={error}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={EMPTY_MESSAGE}
        /*
          Rincian terbuka dengan MENGEKLIK BARISNYA, bukan lewat tautan di kolom
          tersendiri — begitulah layar lama bekerja: barisnya tersorot, lalu isinya
          terbentang tepat di bawahnya.

          Versi sebelumnya menambahkan kolom "Detail Data" berisi tautan "Lihat rincian".
          Kolom itu tidak ada di layar lama, dan pengguna yang mengeklik barisnya —
          kebiasaan yang mereka bawa — tidak mendapat apa pun.
        */
        expandedRow={(row) => (
          <BreakdownTable
            masterID={row.id_master}
            lossDate={row.tanggal_kejadian}
            cause={row.sebab_kerugian}
          />
        )}
      />
    </div>
  )
}

/**
 * Pesan grid kosong.
 *
 * Ia TIDAK lagi menyebut satu perjanjian, karena grid ini bukan milik satu perjanjian —
 * ia gabungan seluruhnya. Kosong di sini berarti tidak ada satu pun perjanjian yang
 * menanggung klaim: entah `XOL_TABLE_ALL_KLAIM` memang belum diisi untuk tahun-tahun itu,
 * entah group business perjanjiannya belum dilengkapi.
 */
const EMPTY_MESSAGE =
  'Belum ada klaim XOL pada satu pun perjanjian. Penambahan Date Of Loss dan Cause Of ' +
  'Loss masih dilakukan lewat aplikasi Pega selama masa paralel.'


function BreakdownTable({
  masterID,
  lossDate,
  cause,
}: {
  masterID: string
  lossDate: string
  cause: string
}) {
  const breakdown = useBreakdown(masterID, lossDate, cause)
  const unduh = useUnduhBerkas()
  const rows = breakdown.data?.baris ?? []

  const columns: Column<Breakdown>[] = [
    {
      key: 'group_business',
      title: 'Group Business',
      value: (row) => row.group_business,
      render: (row) =>
        row.sumber === 'treaty' ? (
          <span>
            {row.group_business}
            <span className="ml-2 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-slate-200">
              treaty inward
            </span>
          </span>
        ) : (
          row.group_business
        ),
    },
    {
      // Judulnya "Number of Claim", bukan "Total Klaim". Keduanya ADA di sistem lama pada
      // grid yang berbeda; yang dipakai di sini adalah judul grid rincian.
      key: 'jumlah_klaim',
      title: 'Number of Claim',
      value: (row) => String(row.jumlah_klaim),
      alignRight: true,
      width: '9rem',
    },
    {
      key: 'nilai_outstanding',
      title: 'OS Value (USD)',
      value: (row) => String(row.nilai_outstanding),
      render: (row) => valueCell(row, row.nilai_outstanding),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'nilai_akseptasi',
      title: 'Accept Value (USD)',
      value: (row) => String(row.nilai_akseptasi),
      render: (row) => valueCell(row, row.nilai_akseptasi),
      alignRight: true,
      width: '11rem',
    },
    {
      /*
        Tombol "Export To Excel" di sistem lama berada DI DALAM baris grid ini, bukan di
        bawahnya: keempat parameternya diambil dari properti baris
        (`Section/DetailValueClaimXOL-Section.xml:4506` dan `:4530-4570`). Karena itu ia
        satu kolom, bukan satu tombol untuk seluruh grid.

        Berkasnya CSV meski labelnya berbunyi "Excel" — activity lamanya pun berakhir di
        `pxConvertResultsToCSV`, bukan di penulis XLSX. Label dipertahankan apa adanya
        (`D-13`); yang berubah hanya cara menyebut isinya di bawah ini.
      */
      key: 'export',
      title: 'Export',
      value: () => '',
      width: '9rem',
      render: (row) => (
        <button
          type="button"
          className="font-medium text-sky-700 underline underline-offset-2 hover:text-sky-900"
          disabled={unduh.isPending}
          onClick={(event) => {
            // Klik ini tidak boleh ikut membuka-tutup baris yang sedang terbentang —
            // seluruh baris adalah tombol pembuka rincian.
            event.stopPropagation()
            unduh.mutate({
              alamat: exportClaimDetailURL(
                lossDate,
                cause,
                row.sumber === 'treaty' ? 'treaty' : row.kode_group_business,
                row.group_business,
              ),
              namaBerkas: `Detail Claim XOL ${row.group_business}.csv`,
            })
          }}
        >
          Export To Excel
        </button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <DataTable<Breakdown>
        columns={columns}
        rows={rows}
        rowKey={(row) => `${row.sumber}-${row.kode_group_business || row.group_business}`}
        /*
          Tanpa judul dan tanpa kotak cari — begitulah grid ini di layar lama: ia langsung
          dimulai dari kepala kolomnya, tepat di bawah baris yang dibuka.

          Judul "Rincian — <tanggal> · <sebab>" yang sempat ada di sini mengulang isi
          baris yang barusan diklik, dan kotak cari atas empat baris group business tidak
          menolong siapa pun.
        */
        hideSearch
        isLoading={breakdown.isPending}
        error={
          breakdown.isError ? (
            <ErrorMessage
              title="Rincian tidak dapat dimuat"
              description={messageOf(breakdown.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage="Tidak ada rincian untuk tanggal dan penyebab kerugian ini."
      />

      {rows.some((row) => row.kurs_tidak_tersedia) && <MissingRateNotice />}

      <UploadBar />
      <MasterLayerPanel lossDate={lossDate} cause={cause} />
      <ClaimListPanel lossDate={lossDate} cause={cause} />
      <SummaryPanel lossDate={lossDate} cause={cause} />
      <GenerateBar lossDate={lossDate} cause={cause} />
    </div>
  )
}

/**
 * GenerateBar — dua tombol penutup layar rincian: "Generate PLA" dan "Generate DLA".
 *
 * Keduanya memanggil activity yang SAMA di sistem lama, `GenerateXOLByType`, dan hanya
 * dibedakan parameter `type` (`Section/DetailValueClaimXOL-Section.xml:23215` dan
 * `:24351`).
 *
 * "Generate DLA" digambar menonjol dan "Generate PLA" tidak — mengikuti layar lama, yang
 * memberi DLA gaya tombol utama. Urutan PLA lalu DLA juga mengikuti urutan penerbitannya:
 * PLA memberitahukan nilai estimasi, DLA memberitahukan nilai akseptasi.
 */
function GenerateBar({ lossDate, cause }: { lossDate: string; cause: string }) {
  const pla = useGenerateAdvice('pla')
  const dla = useGenerateAdvice('dla')
  const isian = { tanggal_kejadian: lossDate, sebab_kerugian: cause }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-3">
        <Button tone="halus" onClick={() => pla.mutate(isian)} disabled={pla.isPending}>
          Generate PLA
        </Button>
        <Button tone="utama" onClick={() => dla.mutate(isian)} disabled={dla.isPending}>
          Generate DLA
        </Button>
      </div>

      {pla.isError && (
        <ErrorMessage
          title="Generate PLA belum tersedia"
          description={messageOf(pla.error)}
          tone="gangguan"
        />
      )}
      {dla.isError && (
        <ErrorMessage
          title="Generate DLA belum tersedia"
          description={messageOf(dla.error)}
          tone="gangguan"
        />
      )}
    </div>
  )
}

/**
 * UploadBar menggambar empat kendali unggahan pada layar rincian, seperti layar lama:
 * dua tombol "Upload ..." dan dua tautan "Format ..."
 * (`Section/DetailValueClaimXOL-Section.xml:6323`, `:6793`, `:7094`, `:7480`).
 *
 * # Keduanya tidak sama derajatnya, dan itu disengaja
 *
 * Tautan **Format** berjalan penuh. Ia hanya MEMBACA — `DownloadFormatForAllUploadingFile`
 * menurunkan berkas contoh berisi baris judul saja, dan judul kolomnya ada lengkap di
 * activity itu, sehingga tidak ada yang perlu ditebak.
 *
 * Tombol **Upload** menulis, dan dua hal menghalanginya:
 *
 *  1. Tabel tujuannya — `POOLDATA.T_SALVAGE_MBU` dan `POOLDATA.T_CLAIM_INWARD_XOL` —
 *     selama masa paralel masih dimiliki Pega (`P-1`). Nama kueri lamanya menyebutkannya
 *     sendiri: `GetDataTrytyInwardFromUploadData`, "dari data upload".
 *  2. Aturan pembacaan berkasnya belum pernah dibaca siapa pun: flow action
 *     `PNCUploadClaimCSV_MBUSalvage` dan `PNCUploadClaimCSV_Inward` TIDAK ada di export —
 *     keduanya hanya dirujuk section itu, di `:6137`, `:6343`, `:6609`, dan `:6823`.
 *
 * Tombolnya tetap DIGAMBAR, dan yang menolak adalah server beserta sebabnya — pola yang
 * sama dengan "INSERT DOL DAN COL" dan "Remove All Data". Paragraf keterangan yang dulu
 * berdiri di sini menggantikan keempat kendali itu sekaligus, sehingga pengguna tidak
 * dapat mengambil berkas contoh yang sebenarnya sudah dapat diberikan.
 */
function UploadBar() {
  return (
    <div className="space-y-2 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <div className="flex flex-wrap items-center gap-2">
        <UploadSalvageButton />
        <UploadInwardButton />

        <FormatLink
          jenis="1"
          label="Format MBU Salvage"
          namaBerkas="Format Upload Salvage MBU.csv"
        />
        <FormatLink jenis="2" label="Format Inward" namaBerkas="Format Upload Inward.csv" />
      </div>

      <p className="text-xs text-slate-500">
        Berkas berformat CSV. Unduh contohnya lewat tautan Format di atas.
      </p>
    </div>
  )
}

/**
 * FormatLink — satu tautan unduh berkas contoh.
 *
 * # Kenapa tombol, bukan `<a href>`
 *
 * Karena sesi dikirim sebagai header `Authorization`, bukan cookie, dan peramban tidak
 * mengirim header apa pun pada navigasi biasa. Tautan polos dijawab `sesi_tidak_sah` —
 * bukan berkas. Lihat `useUnduhBerkas`.
 */
function FormatLink({
  jenis,
  label,
  namaBerkas,
}: {
  jenis: '1' | '2'
  label: string
  namaBerkas: string
}) {
  const unduh = useUnduhBerkas()

  return (
    <>
      <button
        type="button"
        className="text-sm font-medium text-sky-700 underline underline-offset-2 hover:text-sky-900 disabled:text-slate-400"
        disabled={unduh.isPending}
        onClick={() => unduh.mutate({ alamat: uploadTemplateURL(jenis), namaBerkas })}
      >
        {label}
      </button>

      {unduh.isError && (
        <div className="w-full">
          <ErrorMessage
            title={`${label} tidak dapat diunduh`}
            description={messageOf(unduh.error)}
            tone="gangguan"
          />
        </div>
      )}
    </>
  )
}

/**
 * UploadSalvageButton — tombol "Upload MBU Salvage" beserta laporan hasilnya.
 *
 * Tombolnya memicu pemilih berkas tersembunyi, bukan membuka dialog tersendiri. Layar
 * lama pun langsung meminta berkas; menambahkan satu dialog di tengahnya berarti satu
 * langkah yang tidak ada di sistem lama.
 */
function UploadSalvageButton() {
  const pilih = useRef<HTMLInputElement>(null)
  const unggah = useUploadSalvageMBU()
  const klien = useQueryClient()

  return (
    <>
      <Button tone="kedua" onClick={() => pilih.current?.click()} disabled={unggah.isPending}>
        {unggah.isPending ? 'Mengunggah…' : 'Upload MBU Salvage'}
      </Button>

      <input
        ref={pilih}
        type="file"
        accept=".csv,text/csv"
        className="hidden"
        aria-label="Berkas MBU Salvage"
        onChange={(event) => {
          const berkas = event.target.files?.[0]
          // Nilai input dikosongkan supaya memilih BERKAS YANG SAMA dua kali tetap
          // memicu onChange — tanpa ini, mengunggah ulang setelah perbaikan diam saja.
          event.target.value = ''
          if (!berkas) return
          unggah.mutate(berkas, {
            onSuccess: (hasil) => {
              // Grid rincian dan daftar klaim membaca tabel yang barusan ditulis.
              if (hasil.jumlah_tersimpan > 0) {
                void klien.invalidateQueries({ queryKey: ['inbox-xol'] })
              }
            },
          })
        }}
      />

      {unggah.isError && (
        <div className="w-full">
          <ErrorMessage
            title="Berkas tidak dapat diunggah"
            description={messageOf(unggah.error)}
            tone="gangguan"
          />
        </div>
      )}

      {unggah.isSuccess && <UploadReport hasil={unggah.data} />}
    </>
  )
}

/** UploadReport menyatakan berapa baris tersimpan, dan baris mana yang tidak. */
function UploadReport({ hasil }: { hasil: UploadResult }) {
  return (
    <div className="w-full space-y-1 rounded-kartu border border-slate-200 bg-white px-3 py-2 text-sm">
      <p className="text-slate-700">
        {hasil.jumlah_tersimpan} dari {hasil.jumlah_baris} baris tersimpan.
      </p>

      {hasil.ditolak.length > 0 && (
        <ul className="list-disc space-y-0.5 pl-5 text-slate-600">
          {/*
            Nomor baris disebut karena yang diperbaiki pengguna adalah BERKASNYA. Pada
            berkas ratusan baris, "ada yang gagal" saja tidak dapat ditindaklanjuti.
          */}
          {hasil.ditolak.map((baris) => (
            <li key={baris.baris}>
              Baris {baris.baris}
              {baris.no_klaim ? ` (${baris.no_klaim})` : ''}: {baris.alasan}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

/**
 * UploadInwardButton — tombol "Upload Inward", yang masih dijawab penolakan server.
 *
 * Ia TIDAK meminta berkas lebih dulu: activity pemetaan kolomnya,
 * `ConvertDataCsvInwardToPage`, tidak ada di export, sehingga isi berkas itu belum dapat
 * dibaca siapa pun. Meminta berkas hanya untuk menolaknya akan memindahkan data nasabah
 * tanpa satu pun kegunaan.
 */
function UploadInwardButton() {
  const unggah = useUploadInward()

  return (
    <>
      <Button tone="kedua" onClick={() => unggah.mutate()} disabled={unggah.isPending}>
        Upload Inward
      </Button>

      {unggah.isError && (
        <div className="w-full">
          <ErrorMessage
            title="Upload Inward belum tersedia"
            description={messageOf(unggah.error)}
            tone="gangguan"
          />
        </div>
      )}
    </>
  )
}

/**
 * MasterLayerPanel menggambar separuh bawah layar rincian: pemilih "Master tahun XOL" dan
 * grid perjanjian beserta batas layer terendahnya.
 *
 * # Datanya terbaca, labelnya tidak
 *
 * Section yang memuat layar ini TIDAK ada di export — "Master tahun XOL", "Min Limit",
 * dan "Min Limit IDR" nol kemunculan di seluruh berkas (`R-16`). Yang ada adalah SUMBER
 * DATANYA, dan ia cocok kolom per kolom dengan `RDB List/GetDataMasterXOL-SQL.xml:11`:
 *
 *	BranchID	a.ID					ID
 *	UserName	a.NAMA					Nama Master
 *	UserAdmin	a.TAHUN					Tahun
 *	Amount		a.KURSVALUE				Kurs
 *	AIDiterima	min(limit) dari MST_XOL_LAYER		Min Limit
 *	ClaimAmount	min(limit) * KURSVALUE			Min Limit IDR
 *
 * Jadi gridnya disusun dari bukti, bukan dari tangkapan layar. Judul kolomnya memang
 * diambil dari tangkapan layar — itu satu-satunya sumber yang ada untuknya, dan `D-13`
 * menuntut teks yang sudah dibaca pengguna dipertahankan.
 *
 * # Penyaring tahun
 *
 * Kueri lama menyisipkan penyaringnya sebagai teks SQL mentah
 * (`{ASIS:TempDataXOLPilihw.NoteKasir}`). Di sini penyaringan dilakukan di layar atas
 * daftar yang sudah dimuat: daftarnya pendek — satu baris per perjanjian — dan merangkai
 * SQL dari nilai layar adalah pola `{ASIS:...}` yang justru dilarang dibawa.
 */
function MasterLayerPanel({ lossDate, cause }: { lossDate: string; cause: string }) {
  const [year, setYear] = useState('')
  const [showAll, setShowAll] = useState(false)

  const masters = useMasters()
  const hapus = useRemoveDolCol()
  const all = masters.data?.perjanjian ?? []

  const years = [...new Set(all.map((master) => master.tahun).filter(Boolean))].sort()

  /*
    Grid mulai KOSONG, bukan berisi semuanya.

    Terbaca dari layar lama: dengan pemilih tahun masih pada "Pilih", gridnya berbunyi
    "Data Tidak Ada". Jadi ia menunggu tahun dipilih — atau "Show All Data" ditekan.

    Versi sebelumnya menampilkan seluruhnya saat belum ada tahun dipilih. Itu membuat
    "Show All Data" tidak berarti apa-apa, dan tombol yang tidak mengubah apa pun
    dilaporkan sebagai rusak.
  */
  const rows = showAll ? all : year === '' ? [] : all.filter((m) => m.tahun === year)

  const columns: Column<MasterXOL>[] = [
    { key: 'id', title: 'ID', value: (row) => row.id, width: '8rem' },
    { key: 'nama', title: 'Nama Master', value: (row) => row.nama },
    { key: 'tahun', title: 'Tahun', value: (row) => row.tahun, width: '7rem' },
    {
      key: 'kurs',
      title: 'Kurs',
      value: (row) => String(row.kurs),
      render: (row) => formatNumber(row.kurs),
      alignRight: true,
      width: '8rem',
    },
    {
      key: 'min_limit',
      title: 'Min Limit',
      value: (row) => String(row.min_limit),
      render: (row) => limitCell(row.min_limit),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'min_limit_idr',
      title: 'Min Limit IDR',
      value: (row) => String(row.min_limit_idr),
      render: (row) => limitCell(row.min_limit_idr),
      alignRight: true,
      width: '12rem',
    },
  ]

  return (
    <div className="space-y-3">
      <div className="max-w-xs">
        <SelectField
          id="xol-master-tahun"
          label="Master tahun XOL"
          options={years.map((value) => ({ value, label: value }))}
          emptyText={masters.isPending ? '— memuat —' : 'Pilih'}
          value={year}
          onChange={(event) => {
            setYear(event.target.value)
            // Memilih tahun membatalkan "Show All Data" — kalau tidak, gridnya tetap
            // memuat seluruh tahun sementara pemilihnya menunjuk satu tahun saja.
            setShowAll(false)
          }}
        />
      </div>

      <DataTable<MasterXOL>
        columns={columns}
        rows={rows}
        rowKey={(row) => row.id}
        // Judul kolom tetap terlihat walau gridnya kosong — begitulah layar lama: kepala
        // "ID · Nama Master · Tahun · Kurs · Min Limit · Min Limit IDR" sudah ada di atas
        // tulisan "Data Tidak Ada". Tanpa kepala itu, grid kosong tidak memberi petunjuk
        // apa pun tentang isinya nanti.
        showHeaderWhenEmpty
        isLoading={masters.isPending}
        error={
          masters.isError ? (
            <ErrorMessage
              title="Daftar perjanjian XOL tidak dapat dimuat"
              description={messageOf(masters.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={
          showAll || year !== ''
            ? `Tidak ada perjanjian XOL tahun ${year}.`
            : 'Pilih Master tahun XOL, atau tekan Show All Data.'
        }
      />

      {/*
        "Show All Data" mengatur TAMPILAN — ia menampilkan seluruh perjanjian tanpa
        menyaring tahun.

        "Remove All Data" MENGHAPUS. `RDB List/DeleteDataInXOLSummarybasedondol-SQL.xml`
        membuang baris dari `POOLDATA.XOL_TABLE_ALL_KLAIM`. Karena itu ia tidak dirakit
        sebagai pengosong tampilan — tombol yang namanya menjanjikan penghapusan tetapi
        berbuat lain adalah hal yang lebih buruk daripada tombol yang menolak.

        Ia tetap DIGAMBAR, dan yang menolak adalah server beserta sebabnya — pola yang
        sama dengan INSERT DOL DAN COL.
      */}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          tone="kedua"
          onClick={() => {
            setShowAll(true)
            setYear('')
          }}
        >
          Show All Data
        </Button>

        <Button
          tone="kedua"
          onClick={() => hapus.mutate({ tanggal_kejadian: lossDate, sebab_kerugian: cause })}
          disabled={hapus.isPending}
        >
          {hapus.isPending ? 'Menghapus…' : 'Remove All Data'}
        </Button>
      </div>

      {hapus.isError && (
        <ErrorMessage
          title="Belum dapat dihapus"
          description={messageOf(hapus.error)}
          tone="gangguan"
        />
      )}
    </div>
  )
}

/** limitCell menandai perjanjian yang belum punya satu pun layer. */
function limitCell(value: number) {
  if (value === 0) return <span className="text-slate-400">—</span>
  return formatNumber(value)
}

/**
 * valueCell menolak menampilkan angka yang kursnya tidak diketahui.
 *
 * Di sistem lama nilai seperti ini tampil sebagai angka biasa, karena fungsi kursnya
 * mengembalikan `1` saat kurs tidak ditemukan. Angka yang salah dan tampak benar jauh
 * lebih berbahaya daripada tanda hubung.
 */
function valueCell(row: Breakdown, value: number) {
  if (row.kurs_tidak_tersedia) {
    return <span className="text-slate-400">—</span>
  }
  return formatNumber(value)
}

function MissingRateNotice() {
  return (
    <p className="mt-2 rounded-kartu border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
      Sebagian baris treaty inward tidak dapat dihitung karena kurs mata uangnya pada
      tanggal kejadian tidak ada di master kurs. Nilainya sengaja tidak ditampilkan —
      lengkapi master kurs lebih dulu.
    </p>
  )
}

/**
 * rowKey menyusun kunci baris dari kedua kolom yang membentuk pengelompokannya.
 *
 * Tanggal saja tidak cukup: satu tanggal dapat punya beberapa penyebab kerugian, dan
 * kunci yang sama pada dua baris membuat React menggambar salah satunya saja.
 */
function rowKey(row: ClaimSummary): string {
  return `${row.tanggal_kejadian}|${row.sebab_kerugian}`
}

/**
 * formatNumber menampilkan angka dengan pemisah ribuan Indonesia.
 *
 * Ia TIDAK memakai formatRupiah dari `shared`: nilai di layar ini bukan rupiah melainkan
 * mata uang perjanjian, dan menempelkan "Rp" padanya akan menyatakan hal yang salah.
 * Desimalnya dibatasi dua — nilai hasil pembagian kurs nyaris selalu berkoma panjang.
 */
function formatNumber(value: number): string {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(value)
}

/**
 * ClaimListPanel — grid "No Klaim · Os Value · Accept Value · Currency".
 *
 * Kolomnya dari `Section/DetailValueClaimXOL-Section.xml:17050-17656`; isinya dari
 * `RDB List/BrowserT_claim_xolDesc-SQL.xml` lewat
 * `Activity/ShowDataKlaimXOLKlaimBeforeGenerated-Act.xml`.
 *
 * # Kolom "Button" tidak digambar
 *
 * Judulnya ada di section lama, tetapi isinya tombol yang menjalankan rule yang belum
 * ter-export. Kolom kosong berjudul "Button" tidak menyampaikan apa pun.
 */
function ClaimListPanel({ lossDate, cause }: { lossDate: string; cause: string }) {
  const daftar = useClaimList(lossDate, cause)
  const rows = daftar.data?.baris ?? []

  const columns: Column<ClaimListItem>[] = [
    {
      key: 'no_klaim',
      title: 'No Klaim',
      value: (row) => row.no_klaim,
      render: (row) =>
        row.sumber === 'treaty' ? (
          <span>
            {row.no_klaim}
            <span className="ml-2 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-slate-200">
              treaty inward
            </span>
          </span>
        ) : (
          row.no_klaim
        ),
    },
    {
      key: 'nilai_outstanding',
      title: 'Os Value',
      value: (row) => String(row.nilai_outstanding),
      render: (row) => claimValueCell(row, row.nilai_outstanding),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'nilai_akseptasi',
      title: 'Accept Value',
      value: (row) => String(row.nilai_akseptasi),
      render: (row) => claimValueCell(row, row.nilai_akseptasi),
      alignRight: true,
      width: '10rem',
    },
    {
      key: 'mata_uang',
      title: 'Currency',
      value: (row) => row.mata_uang,
      width: '8rem',
    },
  ]

  return (
    <div>
      <DataTable<ClaimListItem>
        columns={columns}
        rows={rows}
        rowKey={(row) => `${row.sumber}|${row.no_klaim}|${row.mata_uang}`}
        showHeaderWhenEmpty
        hideSearch
        isLoading={daftar.isPending}
        error={
          daftar.isError ? (
            <ErrorMessage
              title="Daftar klaim tidak dapat dimuat"
              description={messageOf(daftar.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage="Tidak ada klaim pada tanggal dan penyebab kerugian ini."
      />

      {rows.some((row) => row.kurs_tidak_tersedia) && <MissingRateNotice />}
    </div>
  )
}

/** claimValueCell menolak menggambar angka yang kursnya tidak diketahui (`D-48`). */
function claimValueCell(row: ClaimListItem, value: number) {
  if (row.kurs_tidak_tersedia) {
    return <span className="text-slate-400">—</span>
  }
  return formatNumber(value)
}

/**
 * SummaryPanel — judul "Summary Data XOL" beserta gridnya.
 *
 * Isinya dari `RDB List/GetBusinessnameXOLForSummerry-SQL.xml`: group business mana saja
 * yang menanggung klaim pada tanggal kejadian dan penyebab kerugian ini — klaim sendiri
 * dari `T_CLAIM_XOL`, ditambah satu baris "Treaty Inward" dari `T_CLAIM_INWARD_XOL`.
 *
 * Di layar lama gridnya berkolom tunggal "Business Name"; kode group business ikut
 * dibawa kuerinya dan dipakai di sini sebagai kunci baris, bukan sebagai kolom.
 */
function SummaryPanel({ lossDate, cause }: { lossDate: string; cause: string }) {
  const summary = useSummaryBusiness(lossDate, cause)
  const rows = summary.data?.baris ?? []

  const columns: Column<SummaryBusiness>[] = [
    {
      key: 'group_business',
      title: 'Business Name',
      value: (row) => row.group_business,
    },
  ]

  return (
    <DataTable<SummaryBusiness>
      columns={columns}
      rows={rows}
      rowKey={(row) => `${row.kode_group_business}|${row.group_business}`}
      title="Summary Data XOL"
      showHeaderWhenEmpty
      hideSearch
      isLoading={summary.isPending}
      error={
        summary.isError ? (
          <ErrorMessage
            title="Summary Data XOL tidak dapat dimuat"
            description={messageOf(summary.error)}
            tone="gangguan"
          />
        ) : undefined
      }
      emptyMessage="Tidak ada group business yang menanggung klaim pada tanggal dan penyebab kerugian ini."
    />
  )
}
