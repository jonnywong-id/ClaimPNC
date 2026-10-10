import {
  Fragment,
  useEffect,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
} from 'react'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { formatRupiah } from '@/lib/money'

import { useOSClaimPerCabangDetail } from './api'
import type {
  DetailCoverage,
  DetailEstimation,
  DetailHeader,
  DetailItem,
  DetailMessage,
  DetailObject,
  DetailProgress,
} from './types'

/**
 * Popup Detail — pengganti harness `View_DetailKlaimCabang_Harness`.
 *
 * Dibuka tombol "Detail" pada grid, persis seperti di layar lama. Isinya empat bagian dengan
 * urutan yang sama dengan `Section/DetailKlaimCabang_Sect-Section.xml`:
 *
 *   1. ringkasan             delapan nilai berlabel
 *   2. objek pertanggungan   grid, kolomnya berbeda menurut lini bisnis
 *   3. riwayat progres       grid 8 kolom
 *   4. komunikasi adjuster   grid 5 kolom
 *
 * # Hanya NOMOR klaim yang dikirim
 *
 * Tombol Detail di Pega mengirim lima parameter dari baris yang diklik — nomor, nilai
 * cadangan, umur, lini bisnis, dan catatan PIC. Di sini hanya nomornya. Empat sisanya dibaca
 * ulang peladen, karena nilai uang yang ditentukan peramban dapat diubah lewat alat
 * pengembang biasa.
 *
 * Akibat yang disadari: bila datanya berubah antara daftar dimuat dan popup dibuka, angka di
 * popup berbeda dari angka di barisnya. Yang benar adalah yang di popup.
 */
export function DetailDialog({
  nomorKlaim,
  onTutup,
}: {
  nomorKlaim: string
  onTutup: () => void
}) {
  const detail = useOSClaimPerCabangDetail(nomorKlaim)

  // Escape menutup popup, seperti dialog mana pun yang dikenal pengguna.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') onTutup()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onTutup])

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-detail-os-cabang"
    >
      <div className="w-full max-w-5xl rounded-kartu bg-white p-6 shadow-angkat">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2
              id="judul-detail-os-cabang"
              className="text-lg font-semibold text-slate-900"
            >
              Detail Klaim
            </h2>
            <p className="mt-1 font-mono text-sm text-slate-600">{nomorKlaim}</p>
          </div>
          <Button tone="kedua" onClick={onTutup}>
            Tutup
          </Button>
        </div>

        {detail.isLoading ? (
          <p className="mt-6 text-sm text-slate-500">Memuat detail klaim…</p>
        ) : null}

        {detail.error ? (
          <div className="mt-6">
            {/*
              Nadanya PENOLAKAN ketika klaimnya tidak ditemukan, dan GANGGUAN untuk
              selebihnya. Keduanya menuntut tindakan yang berbeda: yang pertama menyuruh
              pengguna memuat ulang daftarnya, yang kedua menyuruhnya mencoba lagi nanti.
            */}
            <ErrorMessage
              tone={isNotFound(detail.error) ? 'penolakan' : 'gangguan'}
              title={
                isNotFound(detail.error)
                  ? 'Klaim tidak ditemukan'
                  : 'Detail klaim tidak dapat dimuat'
              }
              description={messageOf(detail.error)}
            />
          </div>
        ) : null}

        {detail.data ? (
          <div className="mt-6 space-y-8">
            <Summary header={detail.data.ringkasan} />

            <Section title="Objek Pertanggungan">
              <DataTable
                columns={objectColumnsFor(detail.data.ringkasan.cob)}
                rows={detail.data.objek}
                // Kunci baris memakai OBJECTID, bukan gabungan kolom tampilan. Nama objek
                // DAPAT berulang pada satu klaim — dua unit di lokasi berbeda kerap bernama
                // sama — dan sejak baris dapat dibuka, kunci yang bertabrakan berarti dua
                // baris terbuka bersamaan menampilkan coverage yang sama.
                rowKey={(row) => row.id}
                hideSearch
                emptyMessage="Tidak ada objek pertanggungan pada klaim ini."
                // Baris objek dapat dibuka — sama seperti di Pega, yang menggambarnya
                // sebagai master-detail (`pyRowEditing = masterDetail`) dengan aksi
                // `ViewObjectItem`.
                //
                // `null` untuk objek tanpa coverage: DataTable memakai itu untuk mematikan
                // kemampuan membuka pada baris itu, sehingga tidak ada baris yang mengundang
                // diklik lalu membuka panel kosong.
                expandedRow={(row) =>
                  row.coverage.length === 0 ? null : <CoverageTable rows={row.coverage} />
                }
              />
            </Section>

            <Section title="Riwayat Progress">
              <DataTable
                columns={PROGRESS_COLUMNS}
                rows={detail.data.riwayat_progres}
                rowKey={(row) =>
                  `${row.tanggal_input}|${row.status_progres_1}|${row.keterangan}`
                }
                hideSearch
                emptyMessage="Belum ada catatan progres untuk klaim ini."
              />
            </Section>

            <Section title="KOMUNIKASI DENGAN LOSS ADJUSTER">
              <DataTable
                columns={MESSAGE_COLUMNS}
                rows={detail.data.komunikasi_adjuster}
                rowKey={(row) => `${row.tanggal_proses}|${row.nama_user}|${row.pesan}`}
                hideSearch
                emptyMessage="Belum ada komunikasi dengan loss adjuster."
              />
            </Section>

          </div>
        ) : null}
      </div>
    </div>
  )
}

/**
 * Summary menggambar delapan nilai berlabel pada kepala popup.
 *
 * Label ditulis PERSIS seperti di layar lama, termasuk bahasa campurnya — "Occupation :",
 * "Kronologi :", "Note dari PIC :". `D-13` menetapkan teks yang dilihat pengguna mengikuti
 * Pega, dan menerjemahkan sebagiannya justru membuat layar tidak dikenali lagi.
 */
function Summary({ header }: { header: DetailHeader }) {
  return (
    <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2">
      <Entry label="COB">{header.cob || '—'}</Entry>
      <Entry label="Occupation :">{header.occupation || '—'}</Entry>
      <Entry label="Total Sum Insured :">{formatRupiah(header.total_sum_insured)}</Entry>
      <Entry label="Total Reserve :">{formatRupiah(header.total_reserve)}</Entry>

      <Entry label="Aging :">
        {/*
          Ditandai dengan cara yang SAMA dengan barisnya di grid, dan keterangannya ikut
          terbawa lewat `title` — bukan hanya lewat warna, yang tidak sampai ke semua orang.
        */}
        <span
          className={header.perlu_perhatian ? 'font-semibold text-red-700' : undefined}
          title={
            header.perlu_perhatian
              ? 'Umur klaim melewati ambang yang ditetapkan.'
              : undefined
          }
        >
          {header.aging_hari} hari
        </span>
      </Entry>

      <Entry label="Dominant Factor :">{header.dominant_factor || '—'}</Entry>

      <Entry label="Kronologi :" wide>
        {header.kronologi || '—'}
      </Entry>
      <Entry label="Claim Recommendation :" wide>
        {header.claim_recommendation || '—'}
      </Entry>
      <Entry label="Note dari PIC :" wide>
        {header.note_pic || '—'}
      </Entry>
    </dl>
  )
}

/** Entry menggambar satu pasang label dan nilai. */
function Entry({
  label,
  children,
  wide = false,
}: {
  label: string
  children: ReactNode
  wide?: boolean
}) {
  return (
    <div className={wide ? 'sm:col-span-2' : undefined}>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </dt>
      <dd className="mt-1 whitespace-pre-wrap text-sm text-slate-900">{children}</dd>
    </div>
  )
}

/** Section membungkus satu grid beserta judulnya. */
function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h3 className="mb-2 text-sm font-semibold text-slate-900">{title}</h3>
      {children}
    </section>
  )
}

/**
 * objectColumnsFor memilih varian kolom grid objek menurut lini bisnis.
 *
 * # TIGA varian, dan urutannya terbaca dari section
 *
 * Section lama memuat tiga susunan kolom berbeda, masing-masing didahului rujukan when rule.
 * Posisi keduanya di dalam berkas menunjukkan pasangannya:
 *
 *	IsAneka · IsMarineCargo · IsFire  @136k  ->  "Lokasi Object"  @167k   varian A
 *	IsPA                             @210k  ->  "Pekerjaan"      @247k   varian B
 *	IsTravel                         @292k  ->  "Nama Peserta"   @328k   varian C
 *
 * # Kelima when rule ADA di export, dan isinya sudah dibaca
 *
 *	IsPA            pyWorkPage.Policy.Quotation.GroupPanel = "002"
 *	IsTravel        Kode Bisnis = "77"
 *	IsAneka         Kode Bisnis = 24 | 18 | 17 | 10 | 07 | 06
 *	IsMarineCargo   Kode Bisnis = 24 | 18 | 17 | 10 | 07 | 06   (IDENTIK dengan IsAneka)
 *	IsFire          kosong — "Double click to add condition"
 *
 * Hanya `IsPA` yang dapat dipetakan tepat: `GroupPanel = "002"` sama persis dengan `cob`
 * bernilai `"PA"` pada kueri daftar.
 *
 * # Dua hal yang masih tebakan, dan dinyatakan sebagai tebakan
 *
 * `IsTravel` menyaring **Kode Bisnis** `"77"`, bukan `GroupPanel`. Layar ini tidak membawa
 * kode bisnis, sehingga yang dipakai `cob === 'Travel'` — hasil terjemahan `grouppanel = 005`.
 * Keduanya BELUM dibuktikan menunjuk himpunan klaim yang sama.
 *
 * `IsAneka` dan `IsMarineCargo` bersyarat IDENTIK, dan `IsFire` tanpa syarat sama sekali.
 * Ketiganya karena itu tidak dapat saling dibedakan — dan memang tidak perlu: ketiganya
 * mengarah ke varian yang sama.
 *
 * # SATU klaim = SATU lini bisnis — dijawab Work Owner 2026-10-09
 *
 * *"1 PNC pasti salah satu bisnis (ANEKA, FIRE, PA, TRAVEL) tidak mungkin lebih dari 1"*.
 *
 * Itu menutup pertanyaan yang tidak pernah saya ajukan secara terbuka: apakah satu klaim bisa
 * menuntut DUA varian kolom sekaligus — misalnya sebagian objeknya orang dan sebagian lagi
 * lokasi. Tidak bisa. Karena itu varian dipilih SEKALI dari `ringkasan.cob`, bukan per baris
 * objek. Penyimpanannya pun setuju: `cob` diturunkan dari satu kolom `T_CLAIM_PNC.GROUPPANEL`,
 * sehingga secara bentuk data memang mustahil ada dua.
 *
 * Dua hal yang diukur dan TIDAK sejalan dengan daftar empat itu — keduanya dilaporkan, bukan
 * diserap diam-diam. Diukur pada populasi layar ini (1.117 klaim berstatus belum selesai):
 *
 *	002 PA     378	006 Fire   319	003 Aneka  246
 *	<kosong>   123	005 Travel  33	004 Marine Cargo  18
 *
 * `004` Marine Cargo **ada**, 18 klaim. Dan **123 klaim tidak punya GROUPPANEL sama sekali**,
 * sehingga kolom COB-nya kosong di grid.
 *
 * Keduanya TIDAK mengubah perilaku: ketiganya — Aneka, Marine Cargo, dan yang kosong — jatuh
 * ke varian A yang sama. Yang kosong bahkan sejalan dengan Pega, karena `IsFire` di sana
 * memang tanpa syarat apa pun sehingga menjadi varian bawaan.
 *
 * # Kenapa pemilihannya di sini, bukan di peladen
 *
 * Peladen mengirim SELURUH kolom ketiga varian. Menaruh pemilihan di layar membuat kedua
 * tebakan di atas dapat diperbaiki tanpa menyentuh penyimpanan; menaruhnya di kueri akan
 * menguncinya ke dalam SQL.
 */
function objectColumnsFor(cob: string): Column<DetailObject>[] {
  // Varian B — PA. `IsPA` berpasangan dengan susunan berkolom "Pekerjaan", BUKAN dengan
  // susunan peserta. Keduanya sempat tertukar di sini, dan tertukarnya tidak menghasilkan
  // galat apa pun: grid tetap terisi, hanya kolomnya yang milik lini lain.
  if (cob === 'PA') {
    return [
      { key: 'nama', title: 'Nama Objek', value: (row) => row.nama || '—' },
      { key: 'pekerjaan', title: 'Pekerjaan', value: (row) => row.pekerjaan || '—' },
      {
        key: 'tanggal_lahir',
        title: 'Tanggal lahir',
        value: (row) => formatDate(row.tanggal_lahir) || '—',
      },
    ]
  }

  // Varian C — Travel. Ejaan "Tanggal Lahir" di sini memang berbeda dari "Tanggal lahir"
  // pada varian PA; keduanya dibawa apa adanya (`D-13`).
  if (cob === 'Travel') {
    return [
      { key: 'nama', title: 'Nama Peserta', value: (row) => row.nama || '—' },
      {
        key: 'status_peserta',
        title: 'Status',
        value: (row) => row.status_peserta || '—',
      },
      { key: 'ktp_paspor', title: 'KTP/Paspor', value: (row) => row.ktp_paspor || '—' },
      {
        key: 'tanggal_lahir',
        title: 'Tanggal Lahir',
        value: (row) => formatDate(row.tanggal_lahir) || '—',
      },
    ]
  }

  // Varian A — Aneka, Marine Cargo, Fire, dan selebihnya.
  return [
    { key: 'nama', title: 'Nama Objek', value: (row) => row.nama || '—' },
    { key: 'lokasi', title: 'Lokasi Object', value: (row) => row.lokasi || '—' },
  ]
}

/**
 * PROGRESS_COLUMNS adalah delapan kolom riwayat progres.
 *
 * Judulnya HURUF BESAR seperti di layar lama, dan ejaan "Keterangan" yang berhuruf kecil di
 * antara tujuh yang kapital juga dibawa apa adanya (`D-13`). Di Pega judul terakhir itu
 * bahkan ditulis `<b>Keterangan<b>` dengan tag penutup yang salah ketik; yang dibawa
 * teksnya, bukan salah ketiknya.
 */
/**
 * CoverageTable menggambar panel yang terbuka di bawah sebuah baris objek.
 *
 * # Kenapa tiga kolom, dan hanya tiga
 *
 * `Section/ViewObjectCoverage-Section.xml` menggambar `.ObjectCoverageList` dengan tepat
 * tiga kolom, dan judulnya diambil apa adanya dari sana (`D-13`):
 *
 *	Coverage    <- .CoverageNote, berisi NAMA jaminan ("FLEXAS"), bukan catatan
 *	Mata Uang   <- .Currency
 *	TSI         <- .SumTSI
 *
 * Urutan itu juga urutan layar lama. Menambah kolom di luar ketiganya adalah PENAMBAHAN,
 * bukan penyamaan (`P-5`).
 *
 * Ia dirender sebagai tabel biasa, bukan `DataTable` bersarang: komponen itu membawa
 * pencarian, pengurutan, dan paginasinya sendiri, dan ketiganya tidak punya arti pada
 * segelintir baris di dalam panel yang terbuka.
 */
function CoverageTable({ rows }: { rows: DetailCoverage[] }) {
  // Satu coverage terbuka pada satu waktu, sama seperti baris objek di atasnya.
  //
  // Baris coverage DAPAT DIBUKA, dan itu perilaku layar lama: `ViewObjectCoverage` pun
  // `pyRowEditing = masterDetail` dengan `pyEditAction = ViewObjectCoverageObjectItem`.
  // Serahan sebelumnya menggambar seluruh isinya sekaligus begitu objek dibuka — keliru, dan
  // pada klaim bercoverage banyak ia menumpahkan semuanya tanpa diminta.
  const [terbuka, setTerbuka] = useState<string | null>(null)

  // TANPA kepala "Coverage" di atas tabelnya: kolom pertamanya sudah bernama Coverage, dan
  // layar lama tidak memberinya judul kedua. Permintaan Work Owner 2026-10-09.
  //
  // Alasnya PUTIH, dengan bingkai, sama seperti baris objek di atasnya. Panel yang terbuka
  // milik `DataTable` beralas `bg-slate-50/70` — abu-abu — sehingga tanpa alas sendiri baris
  // coverage tampak kelabu dan terbaca seolah tidak dapat dipencet. `DataTable` TIDAK diubah:
  // warnanya dipakai belasan layar lain, dan mengubahnya di sana adalah perubahan di luar
  // lingkup modul ini.
  return (
    <div className="rounded-md border border-slate-200 bg-white p-2">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-slate-500">
            <th className="pb-1 pr-3 font-medium">Coverage</th>
            <th className="pb-1 pr-3 font-medium">Mata Uang</th>
            <th className="pb-1 text-right font-medium">TSI</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const isi =
              row.object_item.length > 0 ||
              row.list_spreading.length > 0 ||
              row.co_member.length > 0
            const dibuka = isi && terbuka === row.id

            return (
            <Fragment key={row.id}>
              <tr
                className={[
                  'border-b border-slate-100 transition-colors duration-150 ease-halus',
                  // Disamakan PERSIS dengan baris objek di DataTable: sorotan biru pada
                  // SETIAP baris, penunjuk tangan hanya pada baris yang dapat dibuka, biru
                  // lebih pekat saat terbuka. Tanpa penanda lain — Work Owner meminta baris
                  // coverage tampak sama dengan baris objek sebelum dipencet (2026-10-09).
                  'hover:bg-blue-50/50',
                  isi ? 'cursor-pointer' : '',
                  dibuka ? 'bg-blue-50/60' : '',
                ].join(' ')}
                {...(isi
                  ? {
                      role: 'button' as const,
                      tabIndex: 0,
                      'aria-expanded': dibuka,
                      onClick: () => setTerbuka(dibuka ? null : row.id),
                      onKeyDown: (event: ReactKeyboardEvent<HTMLTableRowElement>) => {
                        if (event.key !== 'Enter' && event.key !== ' ') return
                        event.preventDefault()
                        setTerbuka(dibuka ? null : row.id)
                      },
                    }
                  : {})}
              >
                <td className="py-1.5 pr-3 text-slate-900">{row.coverage || '—'}</td>
                <td className="py-1.5 pr-3 text-slate-700">{row.mata_uang || '—'}</td>
                {/*
                  TANPA awalan "Rp". Layar lama menggambar `2.800.000` polos, dan kolom di
                  sebelahnya sudah menyatakan mata uangnya — yang tidak selalu rupiah.
                  Menempelkan "Rp" pada baris bermata uang USD akan salah secara terang.
                */}
                <td className="py-1.5 text-right tabular-nums text-slate-900">
                  {formatRupiah(row.tsi, { withoutSymbol: true })}
                </td>
              </tr>
              {dibuka && (
                <tr className="border-b border-slate-100 last:border-0">
                  <td colSpan={3} className="space-y-3 pb-2 pl-3">
                    {row.object_item.length > 0 && <ItemTable rows={row.object_item} />}
                    {/*
                      Keduanya berdampingan, persis seperti layar lama — bukan bertumpuk.
                      Keduanya juga TETAP digambar saat kosong, dengan "Data Tidak Ada",
                      karena begitulah layar lama menyatakannya.
                    */}
                    <div className="grid gap-3 lg:grid-cols-2">
                      <ShareTable
                        title="List Spreading"
                        firstHeading="Tipe Treaty"
                        rows={row.list_spreading.map((s) => ({
                          key: s.tipe_treaty,
                          first: s.tipe_treaty,
                          currency: s.currency,
                          estimasi: s.estimasi_value,
                          persen: s.pembagian_persentase,
                          result: s.result_value,
                        }))}
                      />
                      <ShareTable
                        title="CO MEMBER"
                        firstHeading="Asuransi"
                        rows={row.co_member.map((m) => ({
                          key: m.asuransi,
                          first: m.asuransi,
                          currency: m.currency,
                          estimasi: m.estimasi_value,
                          persen: m.pembagian_persentase,
                          result: m.result_value,
                        }))}
                      />
                    </div>
                  </td>
                </tr>
              )}
            </Fragment>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

/**
 * ShareTable menggambar "List Spreading" dan "CO MEMBER" — keduanya berkolom sama.
 *
 * Satu komponen untuk dua grid karena susunan kolomnya memang identik di layar lama; hanya
 * judul kolom pertamanya berbeda ("Tipe Treaty" versus "Asuransi"). Membuat dua komponen
 * kembar berarti dua tempat yang harus diperbaiki setiap kali satu kolom berubah.
 *
 * Tabel yang KOSONG tetap digambar, dengan "Data Tidak Ada" — begitulah layar lama
 * menyatakannya, dan kedua contoh dari Work Owner memperlihatkannya persis demikian.
 */
function ShareTable({
  title,
  firstHeading,
  rows,
}: {
  title: string
  firstHeading: string
  rows: {
    key: string
    first: string
    currency: string
    estimasi: string
    persen: string
    result: string
  }[]
}) {
  return (
    <div className="rounded-md border border-slate-200 bg-white p-2">
      <p className="mb-1 text-xs font-semibold text-slate-700">{title}</p>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-slate-500">
            <th className="pb-1 pr-3 font-medium">{firstHeading}</th>
            <th className="pb-1 pr-3 font-medium">Currency</th>
            <th className="pb-1 text-right font-medium">Estimasi Value</th>
            <th className="pb-1 text-right font-medium">Pembagian Persentase (%)</th>
            <th className="pb-1 text-right font-medium">Result Value</th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={5} className="py-2 text-slate-500">
                Data Tidak Ada
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr key={row.key} className="border-b border-slate-100 last:border-0">
                <td className="py-1.5 pr-3 text-slate-900">{row.first || '—'}</td>
                <td className="py-1.5 pr-3 text-slate-700">{row.currency || '—'}</td>
                <td className="py-1.5 text-right tabular-nums text-slate-900">
                  {formatRupiah(row.estimasi, { withoutSymbol: true })}
                </td>
                <td className="py-1.5 text-right tabular-nums text-slate-700">
                  {/* Empat desimal, mengikuti layar lama yang menuliskannya `100,0000%`. */}
                  {row.persen}%
                </td>
                <td className="py-1.5 text-right tabular-nums text-slate-900">
                  {formatRupiah(row.result, { withoutSymbol: true })}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

/**
 * ItemTable menggambar grid "Object Item" beserta estimasi di bawahnya.
 *
 * # Kenapa TIDAK dapat dibuka-tutup seperti tingkat di atasnya
 *
 * Karena layar lama memang tidak menyembunyikannya: pada contoh Work Owner, Object Item dan
 * Estimasi tergambar SEKALIGUS begitu coverage dibuka — bukan menunggu klik berikutnya.
 * Membuatnya dapat dibuka-tutup akan menambah satu langkah yang tidak ada di Pega.
 *
 * # Nama dan deskripsi yang kosong itu benar
 *
 * `T_CLAIM_OBJECTITEMLIST` memuat 1 baris di seluruh tabel, sedangkan 22.004 klaim punya
 * estimasi. Barisnya dibentuk dari estimasi, dan kedua sel itu memang tidak punya sumber —
 * persis seperti yang digambar layar lama.
 */
function ItemTable({ rows }: { rows: DetailItem[] }) {
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50/60 p-2">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-slate-500">
            <th className="pb-1 font-medium">Object Item</th>
            <th className="pb-1 font-medium">Deskripsi Item</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <Fragment key={row.id}>
              <tr className="border-b border-slate-100">
                <td className="py-1.5 text-slate-900">{row.object_item || '—'}</td>
                <td className="py-1.5 text-slate-700">{row.deskripsi_item || '—'}</td>
              </tr>
              {row.estimasi.length > 0 && (
                <tr className="border-b border-slate-100 last:border-0">
                  <td colSpan={2} className="pb-2 pl-3">
                    <EstimationTable rows={row.estimasi} />
                  </td>
                </tr>
              )}
            </Fragment>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/**
 * EstimationTable menggambar grid "Estimasi" — tingkat terdalam.
 *
 * Keenam kolomnya diambil dari layar lama apa adanya (`D-13`). Nilai estimasi DAPAT negatif
 * dan digambar apa adanya: contoh Fire dari Work Owner memuat `100 / -100 / 200 / -200 / 100`,
 * yakni koreksi yang saling meniadakan.
 */
function EstimationTable({ rows }: { rows: DetailEstimation[] }) {
  return (
    <div className="rounded-md border border-slate-200 bg-white p-2">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-slate-500">
            <th className="pb-1 font-medium">Estimasi Ke</th>
            <th className="pb-1 font-medium">Tanggal Estimasi</th>
            <th className="pb-1 font-medium">Tipe Estimasi</th>
            <th className="pb-1 font-medium">Mata Uang</th>
            <th className="pb-1 text-right font-medium">Nilai Kurs (IDR)</th>
            <th className="pb-1 text-right font-medium">Nilai Estimasi</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.estimasi_ke} className="border-b border-slate-100 last:border-0">
              <td className="py-1.5 text-slate-700">{row.estimasi_ke || '—'}</td>
              {/*
                Digambar APA ADANYA, tidak diformat ulang. Peladen sudah mengirimnya siap
                tampil — sama seperti kolom tanggal pada grid Riwayat Progress dan Komunikasi
                Adjuster di bawah. `formatDate` menerima tanggal SAJA, dan memberinya cap
                waktu berjam membuatnya mengembalikan tanda pisah; itulah sebab kedua kolom
                ini sempat kosong di layar.
              */}
              <td className="py-1.5 text-slate-700">{row.tanggal_estimasi || '—'}</td>
              <td className="py-1.5 text-slate-700">{row.tipe_estimasi || '—'}</td>
              <td className="py-1.5 text-slate-700">{row.mata_uang || '—'}</td>
              <td className="py-1.5 text-right tabular-nums text-slate-700">
                {formatRupiah(row.nilai_kurs, { withoutSymbol: true })}
              </td>
              <td className="py-1.5 text-right tabular-nums text-slate-900">
                {formatRupiah(row.nilai_estimasi, { withoutSymbol: true })}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

const PROGRESS_COLUMNS: Column<DetailProgress>[] = [
  {
    key: 'tanggal_input',
    title: 'TANGGAL INPUT',
    value: (row) => row.tanggal_input || '—',
  },
  { key: 'no_klaim', title: 'NOMOR KLAIM', value: (row) => row.no_klaim || '—' },
  {
    key: 'status_progres_1',
    title: 'STATUS PROGRESS 1',
    value: (row) => row.status_progres_1 || '—',
  },
  {
    key: 'status_progres_2',
    title: 'STATUS PROGRESS 2',
    value: (row) => row.status_progres_2 || '—',
  },
  { key: 'user_input', title: 'USER INPUT', value: (row) => row.user_input || '—' },
  {
    key: 'tanggal_next_followup',
    title: 'TANGGAL NEXT FOLLOWUP',
    value: (row) => formatDate(row.tanggal_next_followup) || '—',
  },
  { key: 'status', title: 'STATUS', value: (row) => row.status || '—' },
  { key: 'keterangan', title: 'Keterangan', value: (row) => row.keterangan || '—' },
]

/**
 * MESSAGE_COLUMNS adalah lima kolom komunikasi dengan loss adjuster.
 *
 * Kolom pertama menandai pengirim INTERNAL. Penandanya tidak digambar sebagai kolom
 * tersendiri — ia melekat pada nama, karena tanpa itu percakapan terbaca seolah satu pihak
 * saja dan pembacanya tidak dapat tahu mana pesan keluar.
 */
const MESSAGE_COLUMNS: Column<DetailMessage>[] = [
  {
    key: 'nama_user',
    title: 'Nama User',
    value: (row) => row.nama_user || '—',
    render: (row) => (
      <span>
        {row.nama_user || '—'}
        {row.internal ? (
          <span className="ml-2 rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-600">
            internal
          </span>
        ) : null}
      </span>
    ),
  },
  {
    key: 'tanggal_proses',
    title: 'Tanggal Proses',
    value: (row) => row.tanggal_proses || '—',
  },
  { key: 'pesan', title: 'Pesan', value: (row) => row.pesan || '—' },
  {
    key: 'tanggal_balas',
    title: 'Tanggal Balas',
    value: (row) => row.tanggal_balas || '—',
  },
  { key: 'jawaban', title: 'Jawaban', value: (row) => row.jawaban || '—' },
]

/**
 * isNotFound membedakan "klaim tidak ditemukan" dari gangguan lain.
 *
 * Dicocokkan lewat KODE, bukan lewat teks pesan: teks dapat berubah kapan saja tanpa
 * mengubah artinya, dan pencocokan teks akan diam-diam berhenti bekerja ketika itu terjadi.
 */
function isNotFound(error: unknown): boolean {
  return error instanceof APIError && error.kode === 'klaim_tidak_ditemukan'
}

/** messageOf membaca pesan galat yang layak dibaca pengguna. */
function messageOf(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  return 'Detail klaim tidak dapat dimuat. Coba lagi beberapa saat lagi.'
}
