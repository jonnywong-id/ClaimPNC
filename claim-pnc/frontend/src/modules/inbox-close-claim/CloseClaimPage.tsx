import { useState } from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { ReloadIcon } from '@/components/Icon'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import {
  PAGE_SIZE,
  unduhKlaimTutupCSV,
  useAjukanPermintaan,
  useDaftarKlaimTutup,
  usePenyaringKlaimTutup,
} from './api'
import { RequestDialog } from './RequestDialog'
import type { JenisPermintaan, KlaimTutup, PenyaringKlaimTutup } from './types'

/**
 * Layar Inbox Close Claim.
 *
 * # Harness-nya TIDAK ADA di export
 *
 * Butir menu `MENU_ID 59` menunjuk `InboxCloseClaim_Harness`, dan **tidak ada satu pun
 * berkasnya** di antara 74 harness yang diekspor (`K-33`). Layar ini karena itu
 * direkonstruksi dari artefak yang memang ada — bukan dikarang:
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Judul layar dan 11 kolomnya | `Section/InboxManagerReopen1_Sec-Section.xml` |
 * | Baris yang tampil | `RDB List/GcnmBrowseReopenCase_SQL-SQL.xml` |
 * | Keenam penyaringnya | `Activity/GCNMGetManagerReopenCase_Act-Act.xml` |
 * | Tombol Export to Excel | `Activity/ExportDataCloseClaim-Act.xml` |
 *
 * Judul "Inbox Close Claim" bukan karangan: ia tertulis di dalam section rujukannya sendiri.
 *
 * # Ini layar PEMANTAUAN, bukan Inbox
 *
 * Isinya klaim yang sudah TUTUP — kebalikan tepat dari Inbox Outstanding, yang menyaring dua
 * nilai status yang sama dengan arah berlawanan. Menurut `D-79` ia bukan Inbox meski namanya
 * demikian: barisnya bukan pekerjaan pemanggil dan tidak punya tenggat. Namanya tetap
 * mengikuti butir menunya (`D-13`).
 *
 * # Yang sengaja dibuat berbeda dari Pega
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Kolom "Lama Waktu Klaim" | tanggal pendaftaran, lagi | umur klaim dalam hari |
 * | Total baris | mengabaikan penyaring Status Bayar | mengikuti seluruh penyaring |
 * | ReOpen dan Copy Klaim | mengubah klaim | mencatat PERMINTAAN |
 * | Penyaring dirangkai ke SQL | `{ASIS:…}` dari properti | parameter, disaring server |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 *
 * Ketiganya yang pertama dinyatakan di layar lewat `selisih_terencana`, bukan disembunyikan
 * sebagai detail teknis (`D-54`).
 */
export function CloseClaimPage() {
  const [cari, setCari] = useState('')
  const [noPolis, setNoPolis] = useState('')
  const [noKlaim, setNoKlaim] = useState('')
  const [pic, setPIC] = useState('')
  const [lini, setLini] = useState('')
  const [statusTransfer, setStatusTransfer] = useState('')
  const [statusBayar, setStatusBayar] = useState('')
  const [lewati, setLewati] = useState(0)

  const [dialog, setDialog] = useState<{ jenis: JenisPermintaan; klaim: KlaimTutup } | null>(null)
  const [kabar, setKabar] = useState<string | null>(null)
  const [galatUnduh, setGalatUnduh] = useState<string | null>(null)
  const [sedangUnduh, setSedangUnduh] = useState(false)

  const filter: PenyaringKlaimTutup = {
    cari,
    no_polis: noPolis,
    no_klaim: noKlaim,
    pic,
    lini,
    status_transfer: statusTransfer,
    status_bayar: statusBayar,
    lewati,
  }

  const daftar = useDaftarKlaimTutup(filter)
  const penyaring = usePenyaringKlaimTutup()
  const ajukan = useAjukanPermintaan()

  const portal = useSelectedPortal((state) => state.alias)
  const token = useSession((state) => state.token)

  const rows = daftar.data?.klaim ?? []
  const total = daftar.data?.total ?? 0

  /**
   * Kewenangan mengajukan datang dari SERVER, tidak pernah disimpulkan di layar.
   *
   * Sebelum daftarnya dimuat, nilainya belum diketahui — dan "belum diketahui" diperlakukan
   * sebagai TIDAK BOLEH. Menganggapnya boleh akan membuat tombolnya tampak aktif sekejap
   * lalu berubah, dan pengguna yang menekannya tepat pada saat itu menerima galat yang tidak
   * dapat ia jelaskan.
   */
  const bolehMengajukan = daftar.data?.boleh_mengajukan === true

  /**
   * Setiap perubahan penyaring mengembalikan halaman ke awal.
   *
   * Tanpa ini, menyaring dari halaman empat akan menampilkan tabel kosong yang tampak rusak
   * — dan pengguna menyimpulkan penyaringnya tidak menemukan apa pun.
   */
  function ubah(setter: (value: string) => void) {
    return (value: string) => {
      setter(value)
      setLewati(0)
    }
  }

  async function unduh() {
    if (token === null || portal === null) return

    setSedangUnduh(true)
    setGalatUnduh(null)
    try {
      await unduhKlaimTutupCSV(filter, token, portal)
    } catch (failure) {
      setGalatUnduh(pesanGalat(failure))
    } finally {
      setSedangUnduh(false)
    }
  }

  function kirimPermintaan(alasan: string) {
    if (dialog === null) return

    ajukan.mutate(
      { jenis: dialog.jenis, klaim_id: dialog.klaim.klaim_id, alasan },
      {
        onSuccess: (response) => {
          setDialog(null)
          setKabar(response.pesan)
        },
      },
    )
  }

  /**
   * Kolom mengikuti `Section/InboxManagerReopen1_Sec-Section.xml` — kesebelasnya, dengan
   * judul berbahasa Indonesia persis seperti di sana (`D-13`).
   *
   * Kolom "Pilih" layar lama berupa kotak centang yang menentukan baris mana yang dikenai
   * tombol. Di sini ia menjadi kolom AKSI pada barisnya sendiri: pilihan dan tindakan tidak
   * lagi terpisah, sehingga tidak mungkin menekan ReOpen tanpa menyadari baris mana yang
   * sedang dipilih — kelas kesalahan yang paling mahal pada layar ini.
   */
  const columns: Column<KlaimTutup>[] = [
    {
      key: 'no_klaim',
      title: 'No Klaim',
      width: '11rem',
      value: (c) => c.nomor_klaim,
      render: (c) =>
        c.nomor_klaim ? (
          <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
            {c.nomor_klaim}
          </span>
        ) : (
          <span className="text-slate-400">belum bernomor</span>
        ),
    },
    {
      key: 'no_polis',
      title: 'No Polis',
      width: '10rem',
      value: (c) => c.nomor_polis,
      render: (c) => <span className="truncate">{c.nomor_polis || '—'}</span>,
    },
    {
      key: 'tertanggung',
      title: 'Nama Tertanggung',
      width: '14rem',
      value: (c) => c.nama_tertanggung,
      render: (c) => <span className="truncate">{c.nama_tertanggung || '—'}</span>,
    },
    {
      key: 'nama_bisnis',
      title: 'Nama Bisnis',
      width: '10rem',
      value: (c) => c.nama_bisnis,
      render: (c) => <span className="truncate">{c.nama_bisnis || '—'}</span>,
    },
    {
      key: 'sumber_bisnis',
      title: 'Sumber Bisnis',
      width: '9rem',
      value: (c) => c.sumber_bisnis,
      render: (c) => <span className="truncate">{c.sumber_bisnis || '—'}</span>,
    },
    {
      key: 'nama_cabang',
      title: 'Nama Cabang',
      width: '9rem',
      value: (c) => c.nama_cabang,
      render: (c) => <span className="truncate">{c.nama_cabang || '—'}</span>,
    },
    {
      key: 'tanggal_pendaftaran',
      title: 'Tanggal Pendaftaran',
      width: '9rem',
      value: (c) => c.tanggal_pendaftaran,
      render: (c) => (
        <span className="tabular-nums">
          {c.tanggal_pendaftaran ? formatDate(c.tanggal_pendaftaran) : '—'}
        </span>
      ),
    },
    {
      key: 'lama_waktu',
      title: 'Lama Waktu Klaim',
      width: '8rem',
      value: (c) => String(c.lama_hari),
      render: (c) => <LamaKlaim hari={c.lama_hari} tanggalTutup={c.tanggal_tutup} />,
    },
    {
      key: 'pic_teknik',
      title: 'PIC Teknik',
      width: '10rem',
      value: (c) => c.pic_teknik,
      render: (c) => <span className="truncate">{c.pic_teknik || '—'}</span>,
    },
    {
      key: 'admin_pnc',
      title: 'Admin PNC',
      width: '10rem',
      value: (c) => c.admin_pnc,
      render: (c) => <span className="truncate">{c.admin_pnc || '—'}</span>,
    },
    {
      key: 'status',
      title: 'Status',
      width: '8rem',
      value: (c) => c.status_tampil,
      render: (c) => <StatusBadge status={c.status_tampil} label={c.status_klaim_label} />,
    },
    {
      key: 'aksi',
      title: 'Tindakan',
      width: '13rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (c) => (
        <BarisTindakan
          klaim={c}
          boleh={bolehMengajukan}
          alasan={daftar.data?.alasan_tidak_boleh ?? ''}
          nonaktif={portal === null || ajukan.isPending}
          onPilih={(jenis) => {
            setKabar(null)
            ajukan.reset()
            setDialog({ jenis, klaim: c })
          }}
        />
      ),
    },
  ]

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Close Claim</h1>
        <p className="mt-1 text-sm text-slate-600">
          Klaim yang sudah tutup pada entitas yang sedang dibuka, beserta permintaan untuk
          membukanya kembali atau menyalinnya menjadi klaim baru.
        </p>
      </header>

      {daftar.data && <SelisihTerencana butir={daftar.data.selisih_terencana} />}

      {/*
        Dua keterangan yang SALING MENIADAKAN, dan urutannya disengaja.

        Bila pengajuannya tertutup, keterangan tentang pelaksana tidak berlaku — menampilkan
        keduanya akan membuat pengguna mengira tombolnya mati karena pelaksananya belum ada,
        padahal sebabnya kewenangan. Server sudah memastikan hanya satu yang terisi.
      */}
      {daftar.data?.boleh_mengajukan === false && (
        <div className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 p-4">
          <p className="text-sm font-medium text-amber-900">
            Buka kembali dan salin klaim belum tersedia
          </p>
          <p className="mt-1 text-sm text-amber-800">{daftar.data.alasan_tidak_boleh}</p>
        </div>
      )}

      {daftar.data?.pelaksana_belum_ada === true && (
        <div className="mt-4 rounded-kartu border border-amber-200 bg-amber-50 p-4">
          <p className="text-sm font-medium text-amber-900">
            Permintaan tercatat, tetapi belum ada yang menjalankannya
          </p>
          <p className="mt-1 text-sm text-amber-800">
            Siapa yang mengeksekusi permintaan buka kembali dan salin klaim belum ditetapkan.
            Permintaan yang Anda ajukan tersimpan beserta waktunya, tetapi klaimnya belum akan
            berubah sampai pelaksananya ditentukan.
          </p>
        </div>
      )}

      {daftar.data?.permintaan_terbaca === false && (
        <div className="mt-4">
          <ErrorMessage
            title="Penanda permintaan tidak dapat ditampilkan"
            description="Tabel permintaan belum tersedia pada entitas ini, sehingga baris yang sudah diajukan tidak tertandai dan kedua tombol tindakan akan menjawab galat. Daftar klaim di bawah tetap benar."
            tone="gangguan"
          />
        </div>
      )}

      {portal === null && (
        <div className="mt-6">
          <ErrorMessage
            title="Pilih entitas lebih dulu"
            description="Layar ini membaca klaim milik satu badan hukum, sehingga entitasnya harus dipilih di bilah atas."
            tone="gangguan"
          />
        </div>
      )}

      {kabar && (
        <div
          className="mt-6 rounded-kartu border border-emerald-200 bg-emerald-50 p-4"
          role="status"
        >
          <p className="text-sm text-emerald-900">{kabar}</p>
        </div>
      )}

      {galatUnduh && (
        <div className="mt-6">
          <ErrorMessage title="Berkas tidak dapat diunduh" description={galatUnduh} tone="gangguan" />
        </div>
      )}

      <PanelPenyaring
        noPolis={noPolis}
        noKlaim={noKlaim}
        pic={pic}
        lini={lini}
        statusTransfer={statusTransfer}
        statusBayar={statusBayar}
        pilihanLini={penyaring.data?.lini_bisnis ?? []}
        pilihanTransfer={penyaring.data?.status_transfer ?? []}
        pilihanBayar={penyaring.data?.status_bayar ?? []}
        onNoPolis={ubah(setNoPolis)}
        onNoKlaim={ubah(setNoKlaim)}
        onPIC={ubah(setPIC)}
        onLini={ubah(setLini)}
        onStatusTransfer={ubah(setStatusTransfer)}
        onStatusBayar={ubah(setStatusBayar)}
        onBersihkan={() => {
          setNoPolis('')
          setNoKlaim('')
          setPIC('')
          setLini('')
          setStatusTransfer('')
          setStatusBayar('')
          setCari('')
          setLewati(0)
        }}
      />

      <div className="mt-4">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(c) => c.klaim_id}
          title="Klaim tutup"
          // Prop tidak dikirim sama sekali saat kosong, bukan dikirim bernilai undefined:
          // tsconfig memakai exactOptionalPropertyTypes, yang membedakan keduanya.
          {...(total > 0 ? { description: `${total} klaim sudah tutup.` } : {})}
          isLoading={daftar.isPending && portal !== null}
          searchLabel="Cari No Klaim / No Polis"
          emptyMessage="Tidak ada klaim tutup yang cocok."
          serverSearch={{ value: cari, onChange: ubah(setCari), matchCount: total }}
          actions={
            <>
              <Button
                tone="halus"
                onClick={() => void daftar.refetch()}
                disabled={daftar.isFetching || portal === null}
              >
                <ReloadIcon className="h-4 w-4" />
                {daftar.isFetching ? 'Memuat…' : 'Muat ulang'}
              </Button>
              <Button
                tone="kedua"
                onClick={() => void unduh()}
                disabled={sedangUnduh || portal === null || total === 0}
                title={total === 0 ? 'Tidak ada baris untuk diunduh.' : 'Unduh seluruh hasil sebagai CSV'}
              >
                {sedangUnduh ? 'Menyiapkan…' : 'Unduh CSV'}
              </Button>
            </>
          }
          error={
            daftar.isError ? (
              <ErrorMessage
                title="Daftar klaim tutup tidak dapat dimuat"
                description={pesanGalat(daftar.error)}
                tone="gangguan"
              />
            ) : undefined
          }
        />
      </div>

      <Paginasi
        lewati={lewati}
        tampil={rows.length}
        total={total}
        onChange={setLewati}
        sibuk={daftar.isFetching}
      />

      {dialog && (
        <RequestDialog
          jenis={dialog.jenis}
          klaim={dialog.klaim}
          sedangMengirim={ajukan.isPending}
          galat={ajukan.isError ? pesanGalat(ajukan.error) : null}
          onBatal={() => {
            setDialog(null)
            ajukan.reset()
          }}
          onKirim={kirimPermintaan}
        />
      )}
    </div>
  )
}

/**
 * Menyatakan perbedaan yang disengaja terhadap layar Pega.
 *
 * Ini bukan hiasan. `D-54` menetapkan selisih di luar 13 butir `P-5` menuntut persetujuan
 * Work Owner tertulis — dan menyatakannya di layar itulah yang membuat keputusan itu
 * terlihat oleh orang yang memakai layarnya, bukan hanya oleh yang membaca dokumen.
 */
function SelisihTerencana({ butir }: { butir: string[] }) {
  if (butir.length === 0) return null

  return (
    <details className="mt-4 rounded-kartu border border-slate-200 bg-slate-50 p-4">
      <summary className="cursor-pointer text-sm font-medium text-slate-800">
        Tiga hal yang berbeda dari layar lama
      </summary>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-slate-700">
        {butir.map((teks) => (
          <li key={teks}>{teks}</li>
        ))}
      </ul>
    </details>
  )
}

/**
 * Panel penyaring — pengganti `FilterDashboardClaimclose`, yang tidak ada di export.
 *
 * Keenam isiannya tetap disalin dari activity yang membacanya, sehingga tidak ada satu pun
 * penyaring yang dikarang maupun hilang.
 */
function PanelPenyaring(props: {
  noPolis: string
  noKlaim: string
  pic: string
  lini: string
  statusTransfer: string
  statusBayar: string
  pilihanLini: { nilai: string; label: string }[]
  pilihanTransfer: { nilai: string; label: string }[]
  pilihanBayar: { nilai: string; label: string }[]
  onNoPolis: (value: string) => void
  onNoKlaim: (value: string) => void
  onPIC: (value: string) => void
  onLini: (value: string) => void
  onStatusTransfer: (value: string) => void
  onStatusBayar: (value: string) => void
  onBersihkan: () => void
}) {
  const adaPenyaring =
    props.noPolis !== '' ||
    props.noKlaim !== '' ||
    props.pic !== '' ||
    props.lini !== '' ||
    props.statusTransfer !== '' ||
    props.statusBayar !== ''

  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-slate-800">Penyaring</h2>
        <Button tone="halus" onClick={props.onBersihkan} disabled={!adaPenyaring}>
          Bersihkan
        </Button>
      </div>

      <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <IsianTeks id="f-no-polis" label="No Polis" value={props.noPolis} onChange={props.onNoPolis} />
        <IsianTeks id="f-no-klaim" label="No Klaim" value={props.noKlaim} onChange={props.onNoKlaim} />
        <IsianTeks id="f-pic" label="PIC Teknik" value={props.pic} onChange={props.onPIC} />

        <IsianPilihan
          id="f-lini"
          label="Lini Bisnis"
          value={props.lini}
          options={props.pilihanLini}
          onChange={props.onLini}
        />
        <IsianPilihan
          id="f-transfer"
          label="Status Transfer Kasir"
          value={props.statusTransfer}
          options={props.pilihanTransfer}
          onChange={props.onStatusTransfer}
        />
        <IsianPilihan
          id="f-bayar"
          label="Status Bayar"
          value={props.statusBayar}
          options={props.pilihanBayar}
          onChange={props.onStatusBayar}
        />
      </div>
    </section>
  )
}

function IsianTeks({
  id,
  label,
  value,
  onChange,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        id={id}
        type="text"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 shadow-lembut transition-[border-color,box-shadow] duration-150 focus:outline-none focus-visible:border-blue-500 focus-visible:ring-4 focus-visible:ring-blue-500/25"
      />
    </div>
  )
}

function IsianPilihan({
  id,
  label,
  value,
  options,
  onChange,
}: {
  id: string
  label: string
  value: string
  options: { nilai: string; label: string }[]
  onChange: (value: string) => void
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <select
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 w-full rounded-kontrol border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 shadow-lembut transition-[border-color,box-shadow] duration-150 focus:outline-none focus-visible:border-blue-500 focus-visible:ring-4 focus-visible:ring-blue-500/25"
      >
        {options.map((option) => (
          <option key={option.nilai} value={option.nilai}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  )
}

/**
 * Tombol tindakan pada satu baris, beserta penanda permintaan yang sudah diajukan.
 *
 * Penandanya menggantikan tombolnya, tidak berdampingan dengannya: selama sebuah permintaan
 * masih menunggu, permintaan sejenis atas klaim yang sama akan ditolak server (409).
 * Menyisakan tombol yang pasti ditolak hanya mengundang penekanan yang berakhir sebagai
 * pesan galat.
 */
function BarisTindakan({
  klaim,
  boleh,
  alasan,
  nonaktif,
  onPilih,
}: {
  klaim: KlaimTutup
  boleh: boolean
  alasan: string
  nonaktif: boolean
  onPilih: (jenis: JenisPermintaan) => void
}) {
  const menunggu = new Set(klaim.permintaan_tertunda.map((p) => p.jenis))

  /*
    Tombolnya TETAP digambar saat pengajuan tertutup, hanya dinonaktifkan beserta alasannya.

    Menyembunyikannya sama sekali akan membuat kolom "Tindakan" kosong tanpa sebab yang
    terbaca — dan pengguna yang tahu fitur itu ada akan melaporkannya sebagai hilang.
    Tombol mati yang menjelaskan dirinya sendiri lebih jujur daripada ruang kosong.
  */
  const mati = nonaktif || !boleh
  const judulMati = boleh ? undefined : alasan

  return (
    <div className="flex flex-wrap justify-end gap-1.5">
      {menunggu.has('reopen') ? (
        <PenandaMenunggu teks="ReOpen diminta" />
      ) : (
        <Button tone="halus" disabled={mati} title={judulMati} onClick={() => onPilih('reopen')}>
          ReOpen
        </Button>
      )}
      {menunggu.has('salin') ? (
        <PenandaMenunggu teks="Salin diminta" />
      ) : (
        <Button tone="halus" disabled={mati} title={judulMati} onClick={() => onPilih('salin')}>
          Copy Klaim
        </Button>
      )}
    </div>
  )
}

function PenandaMenunggu({ teks }: { teks: string }) {
  return (
    <span
      className="inline-flex items-center rounded-md bg-amber-50 px-2 py-1 text-xs font-medium text-amber-800 ring-1 ring-amber-200"
      title="Permintaan sudah tercatat dan menunggu dijalankan. Klaim belum berubah."
    >
      {teks}
    </span>
  )
}

/**
 * Lama klaim berjalan, dalam hari.
 *
 * Tanggal tutupnya ikut disebut lewat `title` supaya angkanya dapat diperiksa — tanpa itu,
 * "41 hari" adalah angka yang tidak dapat dibantah maupun dibenarkan pengguna.
 *
 * Klaim yang tanggal tutupnya tidak diketahui ditandai berbeda: angkanya dihitung sampai
 * HARI INI, dan itu arti yang berbeda dari lamanya klaim berjalan.
 */
function LamaKlaim({ hari, tanggalTutup }: { hari: number; tanggalTutup: string }) {
  if (tanggalTutup === '') {
    return (
      <span
        className="tabular-nums text-slate-500"
        title="Tanggal tutup tidak tercatat; dihitung sampai hari ini."
      >
        {hari} hari*
      </span>
    )
  }

  return (
    <span className="tabular-nums text-slate-700" title={`Ditutup ${formatDate(tanggalTutup)}`}>
      {hari} hari
    </span>
  )
}

/**
 * Status yang dilihat pengguna; teksnya mengikuti layar Pega apa adanya (`D-13`).
 *
 * Label status klaim ikut ditampilkan di bawahnya — ia TIDAK ada di layar lama, tetapi
 * tanpanya penyaring Status Bayar menyaring sesuatu yang tidak terlihat di baris mana pun.
 */
function StatusBadge({ status, label }: { status: string; label: string }) {
  const tone =
    status === 'Reject'
      ? 'bg-red-50 text-red-700 ring-red-100'
      : 'bg-slate-100 text-slate-700 ring-slate-200'

  return (
    <div className="flex flex-col gap-0.5">
      <span
        className={`inline-flex w-fit items-center rounded-md px-2 py-0.5 text-xs font-medium ring-1 ${tone}`}
      >
        {status || '—'}
      </span>
      {label && <span className="truncate text-xs text-slate-500">{label}</span>}
    </div>
  )
}

/**
 * Paginasi maju-mundur satu halaman.
 *
 * `10-API-STRATEGY.md` §4 menghindari nomor halaman pada data besar. Di sini total memang
 * tersedia, sehingga keterangannya dapat menyebut angka — tetapi navigasinya tetap maju
 * mundur, bukan melompat ke halaman sekian.
 */
function Paginasi({
  lewati,
  tampil,
  total,
  onChange,
  sibuk,
}: {
  lewati: number
  tampil: number
  total: number
  onChange: (next: number) => void
  sibuk: boolean
}) {
  if (total === 0) return null

  const pertama = lewati + 1
  const terakhir = lewati + tampil

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm tabular-nums text-slate-600">
        Menampilkan {pertama}–{terakhir} dari {total}
      </p>
      <div className="flex gap-2">
        <Button
          tone="halus"
          onClick={() => onChange(Math.max(0, lewati - PAGE_SIZE))}
          disabled={lewati === 0 || sibuk}
        >
          Sebelumnya
        </Button>
        <Button
          tone="halus"
          onClick={() => onChange(lewati + PAGE_SIZE)}
          disabled={terakhir >= total || sibuk}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}

function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
