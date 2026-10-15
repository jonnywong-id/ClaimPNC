import { useState, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { formatRupiah } from '@/lib/money'

import {
  useExportOSClaimPerCabang,
  useOSClaimPerCabangList,
  useOSClaimPerCabangSummary,
} from './api'
import { DetailDialog } from './DetailDialog'
import { SummaryPanel } from './SummaryPanel'
import type { Branch, PageInfo, WorkItem } from './types'

/**
 * Inbox OS Claim per Cabang — menu `MENU_ID 69`, pengganti harness
 * `OutstandingKlaimperCabang_Harness`.
 *
 * Isinya klaim yang BELUM TUNTAS milik cabang pemanggil. "OS" di sini adalah Outstanding
 * (`CONTEXT.md`), dan di layar ini artinya sempit dan tegas: satu penyaring pada kuerinya,
 * `pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')`. Tidak ada kaitannya
 * dengan status akseptasi.
 *
 * # Ia BUKAN inbox dalam arti `D-79`
 *
 * Barisnya bukan pekerjaan milik satu orang dan tidak hilang setelah dikerjakan seseorang;
 * ia pandangan PENYELIA atas seluruh klaim berjalan satu cabang. Nama butir menunya tetap
 * memakai kata "Inbox" karena begitulah ia tertulis di `M_MENU_APLIKASI_PNC`, dan `D-13`
 * menetapkan teks yang dilihat pengguna mengikuti sistem lama.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InboxOutstandingperCabang_Section-Section.xml` apa adanya: judul
 * yang menyebut cabang, tombol ekspor, lalu grid 16 kolom. Nama kolom TIDAK diterjemahkan.
 *
 * # Satu hal yang berbeda tempatnya dari layar lama
 *
 * Tombol "Detail" ada di dalam BARIS di sini, sedangkan di Pega ia berada di bilah atas dan
 * bekerja atas baris yang sedang disorot. Tabel ini tidak punya konsep "baris yang disorot",
 * dan tombol di bilah atas yang bekerja atas sesuatu yang tidak terlihat dipilih adalah
 * tombol yang menebak maksud pengguna.
 *
 * Yang dibukanya sama: `View_DetailKlaimCabang_Harness`, lihat DetailDialog.
 */
export function OSClaimPerCabangPage() {
  const [page, setPage] = useState(1)

  // Isi kotak cari. Ia dikirim ke peladen, bukan dipakai menyaring baris yang sudah di
  // tangan — lihat catatan pada DataTable di bawah.
  const [search, setSearch] = useState('')

  // Nomor klaim yang popupnya sedang terbuka, atau null bila tidak ada.
  //
  // Yang disimpan NOMOR, bukan barisnya. Menyimpan barisnya berarti popup menggambar
  // salinan data yang diambil saat daftar dimuat — dan bila datanya sudah berubah sejak
  // itu, popup menampilkan angka yang tidak berlaku lagi tanpa satu pun tanda.
  const [detailOf, setDetailOf] = useState<string | null>(null)

  const portal = useSelectedPortal((state) => state.alias)
  const list = useOSClaimPerCabangList(page, search)
  const summary = useOSClaimPerCabangSummary()

  function ubahPencarian(next: string) {
    setSearch(next)
    // Halaman dikembalikan ke awal. Tanpa ini, mencari dari halaman empat menampilkan
    // tabel kosong yang tampak rusak padahal hasilnya ada di halaman satu.
    setPage(1)
  }

  if (portal === null) {
    return (
      <PageFrame exportable={false}>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Klaim milik satu badan hukum, dan aplikasi ini melayani empat. Pilih portal ' +
            'di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  // Cabang yang tidak diketahui dijawab server sebagai galat tersendiri, bukan sebagai
  // daftar kosong — dan perbedaan itu yang paling penting untuk sampai ke pengguna.
  //
  // Daftar kosong berarti cabangnya memang tidak punya klaim berjalan; cabang yang tidak
  // diketahui berarti layarnya tidak dapat bekerja sama sekali, dan tindak lanjutnya
  // menghubungi Tim IT. Keduanya terlihat sama bila dijawab dengan tabel kosong.
  if (isBranchUnknown(list.error)) {
    return (
      <PageFrame exportable={false}>
        <ErrorMessage
          title="Cabang Anda belum terdaftar"
          description={messageOf(list.error)}
          tone="penolakan"
        />
      </PageFrame>
    )
  }

  const rows = list.data?.data ?? []
  const branch = list.data?.cabang

  return (
    // Tombol ekspor mengikuti jumlah berkas SELURUH CABANG, bukan jumlah yang cocok
    // dengan kotak cari. Berkasnya memang berisi seluruh cabang (lihat catatan pada
    // api.ts), sehingga mematikannya karena pencarian nihil akan mematikan tombol yang
    // sebenarnya masih menghasilkan berkas penuh.
    //
    // Jumlah dari panel ringkasan dipakai karena ia TIDAK ikut tersaring; jumlah daftar
    // dipakai hanya bila panelnya sendiri gagal dimuat.
    <PageFrame
      exportable={(summary.data?.total_berkas ?? list.data?.paginasi.total ?? 0) > 0}
      branch={branch}
    >
      {/*
        Panel ringkasan DI ATAS grid, bukan menggantikannya (keputusan Work Owner
        2026-10-08). Penyelia memperoleh gambaran menyeluruh sekaligus tetap dapat
        menelusuri per klaim lewat grid dan tombol Detail di bawahnya.

        Keduanya dimuat TERPISAH: panel yang gagal tidak mengosongkan grid, dan
        sebaliknya.
      */}
      <SummaryPanel
        data={summary.data}
        isLoading={summary.isPending}
        error={summary.error}
        onRefresh={() => {
          void summary.refetch()
          void list.refetch()
        }}
        isRefreshing={summary.isFetching || list.isFetching}
      />

      <div className="mt-4">
        <DataTable<WorkItem>
          columns={columnsFor(list.data?.ambang_aging ?? 0, setDetailOf)}
          rows={rows}
          rowKey={(row) => row.no_klaim}
          // Pencarian dikerjakan PELADEN, bukan komponen ini. Penyaringan di peramban
          // hanya menjangkau halaman yang sedang terbuka, sehingga pengguna akan diberi
          // tahu "tidak ada" untuk baris yang sebenarnya ada di halaman berikutnya — dan
          // gagalnya diam, tanpa satu pun tanda.
          //
          // `matchCount` memakai total dari peladen, yang sudah ikut tersaring karena
          // `COUNT(*) OVER ()` dihitung sesudah penyaringnya.
          //
          // KEMAMPUAN BARU: layar lama tidak punya kotak cari sama sekali. Diminta Work
          // Owner 2026-10-09, dan dinyatakan di daftar selisih terencana.
          searchLabel="Cari No Klaim / No Polis"
          serverSearch={{
            value: search,
            onChange: ubahPencarian,
            matchCount: list.data?.paginasi.total,
          }}
          isLoading={list.isPending}
          error={
            list.isError ? (
              <ErrorMessage
                title="Daftar tidak dapat dimuat"
                description={messageOf(list.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          // Pesan ini hanya berlaku saat kotak cari KOSONG. Saat terisi, `DataTable`
          // menggantinya sendiri dengan "Tidak ada baris yang cocok dengan …" — dan itu
          // yang benar: "tidak ada klaim berjalan di cabang ini" saat sedang mencari
          // adalah pernyataan yang keliru, yang kosong hasil pencariannya.
          emptyMessage="Tidak ada klaim berjalan di cabang ini."
        />

        {list.data && list.data.paginasi.total > 0 && (
          <Pagination
            info={list.data.paginasi}
            visible={rows.length}
            onMove={setPage}
            loading={list.isFetching}
          />
        )}
      </div>

      {list.data && (
        <>
          <RedRuleLegend threshold={list.data.ambang_aging} />
        </>
      )}

      {detailOf !== null && (
        <DetailDialog nomorKlaim={detailOf} onTutup={() => setDetailOf(null)} />
      )}
    </PageFrame>
  )
}

/**
 * Kerangka layar: judul yang menyebut cabang, lalu tombol ekspor.
 *
 * Judulnya mengikuti layar lama apa adanya, termasuk tanda kurungnya:
 * `Inbox Outstanding Claim per Cabang ( CABANG <nama> )`.
 */
function PageFrame({
  exportable,
  branch,
  children,
}: {
  exportable: boolean
  /**
   * Cabang pemanggil, atau undefined selama daftarnya belum termuat.
   *
   * `undefined` ditulis eksplisit, bukan hanya lewat `?`: proyek ini menyalakan
   * `exactOptionalPropertyTypes`, sehingga "boleh tidak diisi" dan "boleh bernilai
   * undefined" adalah dua hal yang berbeda — dan yang kedua itulah yang terjadi di sini.
   */
  branch?: Branch | undefined
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            Inbox Outstanding Claim per Cabang
            {branch && (
              <span className="ml-2 font-normal text-slate-600">
                ( CABANG {branch.nama || branch.kode} )
              </span>
            )}
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Klaim cabang Anda yang belum tuntas maupun belum ditolak, diurutkan dari yang
            paling lama menunggu.
          </p>
        </div>
        <ExportButton enabled={exportable} />
      </header>
      {children}
    </div>
  )
}

/**
 * Tombol "Export To Excel".
 *
 * Tombolnya dimatikan saat tidak ada yang dapat diekspor. Berkas kosong yang tetap terunduh
 * tidak dapat dibedakan pengguna dari ekspor yang gagal diam-diam.
 *
 * Namanya tetap "Export To Excel" meski berkasnya CSV. `D-13` menetapkan teks yang dilihat
 * pengguna mengikuti layar Pega, dan CSV memang dibuka Excel tanpa perantara.
 */
function ExportButton({ enabled }: { enabled: boolean }) {
  const ekspor = useExportOSClaimPerCabang()

  return (
    <div className="flex flex-col items-end gap-2">
      <Button
        tone="kedua"
        disabled={!enabled || ekspor.isPending}
        onClick={() => ekspor.mutate()}
      >
        {ekspor.isPending ? 'Menyiapkan berkas…' : 'Export To Excel'}
      </Button>
      {ekspor.isError && (
        <p className="text-sm text-red-700" role="alert">
          {messageOf(ekspor.error)}
        </p>
      )}
    </div>
  )
}

/**
 * Ke-16 kolom grid, dalam urutan yang sama dengan layar lama.
 *
 * Judulnya diambil apa adanya dari section, termasuk "Status Progress2" yang memang tanpa
 * spasi sebelum angkanya sementara kolom di sebelahnya memakai spasi. Ejaan layar lama
 * dibawa apa adanya (`D-13`), dan menyeragamkannya di sini akan memunculkan selisih pada
 * gerbang 1 tanpa satu pun manfaat.
 *
 * # Baris merah
 *
 * Ditandai lewat `render` per sel, bukan lewat kelas pada barisnya: `DataTable` adalah
 * komponen BERSAMA, dan menambah kemampuan pewarnaan baris ke sana menyentuh setiap layar
 * yang memakainya. Sampai kebutuhan itu muncul di layar kedua, penandaannya tinggal di sini.
 */
function columnsFor(
  threshold: number,
  onDetail: (nomorKlaim: string) => void,
): Column<WorkItem>[] {
  /**
   * text menyusun satu kolom teks biasa yang ikut ditandai merah.
   *
   * Ia closure di dalam columnsFor, bukan fungsi tingkat berkas, supaya ambangnya ditangkap
   * SEKALI — bukan diteruskan ulang pada keempat belas pemanggilan. Satu pemanggilan yang
   * lupa meneruskannya akan membuat satu kolom menjelaskan pewarnaannya dengan ambang yang
   * berbeda dari kolom di sebelahnya.
   */
  const text = (
    key: string,
    title: string,
    read: (row: WorkItem) => string,
  ): Column<WorkItem> => ({
    key,
    title,
    value: read,
    render: (row) => (
      <Marked row={row} threshold={threshold}>
        {read(row) || '—'}
      </Marked>
    ),
  })

  return [
    text('cabang', 'Cabang', (row) => row.cabang),
    text('sumbis', 'Sumbis', (row) => row.sumbis),
    text('cob', 'COB', (row) => row.cob),
    text('no_polis', 'Policy No', (row) => row.no_polis),
    text('nama_insured', 'Nama Insured', (row) => row.nama_insured),
    text('no_klaim', 'Claim No', (row) => row.no_klaim),
    text('tanggal_registrasi', 'Registration Date', (row) =>
      formatDate(row.tanggal_registrasi),
    ),
    text('tanggal_kejadian', 'DOL', (row) => formatDate(row.tanggal_kejadian)),
    text('col', 'COL', (row) => row.col),
    {
      key: 'nilai_estimasi',
      title: 'Reserve Claim ASM Share',
      value: (row) => row.nilai_estimasi,
      render: (row) => (
        <Marked row={row} threshold={threshold}>
          {formatRupiah(row.nilai_estimasi)}
        </Marked>
      ),
      alignRight: true,
    },
    text('tanggal_update_progres', 'Tgl Update Progress Terakhir', (row) =>
      formatDate(row.tanggal_update_progres),
    ),
    text('status_progres_1', 'Status Progress 1', (row) => row.status_progres_1),
    text('status_progres_2', 'Status Progress2', (row) => row.status_progres_2),
    text('adjuster', 'Adjuster', (row) => row.adjuster),
    text('pic', 'PIC', (row) => row.pic),
    {
      key: 'aging_hari',
      title: 'Aging (Hari)',
      value: (row) => String(row.aging_hari),
      render: (row) => (
        <Marked row={row} threshold={threshold}>
          {row.aging_hari}
        </Marked>
      ),
      alignRight: true,
    },
    {
      // Kolom aksi, bukan kolom data. Ia tidak ada di grid Pega sebagai kolom — di sana
      // tombolnya berada di bilah atas dan bekerja atas baris yang sedang DISOROT.
      //
      // Dipindahkan ke dalam baris di sini karena tabel ini tidak punya konsep "baris yang
      // disorot": tombol di bilah atas yang bekerja atas sesuatu yang tidak terlihat
      // dipilih adalah tombol yang menebak maksud pengguna.
      key: 'aksi',
      title: 'Detail',
      value: () => '',
      noSort: true,
      render: (row) => (
        <Button
          tone="halus"
          onClick={() => onDetail(row.no_klaim)}
          // Pembaca layar mendengar "Detail" delapan puluh kali tanpa keterangan ini.
          aria-label={`Lihat detail klaim ${row.no_klaim}`}
        >
          Detail
        </Button>
      ),
    },
  ]
}

/**
 * Marked mewarnai isi sel merah saat barisnya perlu perhatian.
 *
 * Warnanya TIDAK berdiri sendiri sebagai penanda. Pengguna yang tidak dapat membedakan
 * warna tetap memperoleh keterangannya lewat `title`, dan pembaca layar membacakannya.
 * Aturan yang hanya disampaikan lewat warna tidak sampai ke semua orang.
 */
function Marked({
  row,
  threshold,
  children,
}: {
  row: WorkItem
  threshold: number
  children: ReactNode
}) {
  if (!row.perlu_perhatian) return <>{children}</>

  return (
    <span className="font-medium text-red-700" title={reasonOf(row, threshold)}>
      {children}
    </span>
  )
}

/**
 * reasonOf menyebutkan mengapa sebuah baris ditandai.
 *
 * Ia menyebut HANYA syarat yang benar-benar terpenuhi. Menyebut umur pada baris yang merah
 * karena progresnya mandek — padahal umurnya masih jauh di bawah ambang — akan mengajari
 * pengguna aturan yang keliru, dan keterangan yang salah lebih buruk daripada tanpa
 * keterangan.
 *
 * Ambangnya datang dari jawaban server, sama dengan yang dipakai backend menghitung
 * `perlu_perhatian`. Menuliskannya di sini akan membuat keterangan dan pewarnaan dapat
 * berselisih.
 */
function reasonOf(row: WorkItem, threshold: number): string {
  const reasons: string[] = []

  if (row.aging_hari > threshold) {
    reasons.push(`umur klaim ${row.aging_hari} hari, lebih dari ${threshold}`)
  }
  if (row.progres_mandek) {
    reasons.push('progres tidak berubah pada tiga catatan terakhir')
  }

  // Daftar kosong berarti server menandai baris ini dengan syarat yang belum dikenal layar
  // — misalnya syarat ketiga yang ditambahkan kemudian. Barisnya tetap merah, dan
  // keterangannya menyatakan apa adanya alih-alih mengarang sebab.
  if (reasons.length === 0) return 'Perlu perhatian.'

  return `Perlu perhatian: ${reasons.join(' · ')}`
}

/**
 * Keterangan pewarnaan, di bawah tabel.
 *
 * Ambangnya datang dari server, bukan ditulis di sini: angka yang hidup di dua tempat akan
 * berbeda saat salah satunya diubah, dan yang berubah diam-diam adalah keterangannya —
 * bukan pewarnaannya.
 */
function RedRuleLegend({ threshold }: { threshold: number }) {
  return (
    <p className="mt-4 text-sm text-slate-600">
      <span className="font-medium text-red-700">Baris merah</span> menandai klaim yang
      berumur lebih dari {threshold} hari, atau yang progresnya tidak berubah pada tiga
      catatan terakhir.
    </p>
  )
}

function Pagination({
  info,
  visible,
  onMove,
  loading,
}: {
  info: PageInfo
  visible: number
  onMove: (page: number) => void
  loading: boolean
}) {
  const first = visible === 0 ? 0 : (info.halaman - 1) * info.ukuran + 1
  const last = (info.halaman - 1) * info.ukuran + visible

  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-slate-600" role="status">
        Menampilkan {first}–{last} dari {info.total} baris.
      </p>
      <div className="flex gap-2">
        <Button
          tone="kedua"
          onClick={() => onMove(Math.max(1, info.halaman - 1))}
          disabled={info.halaman <= 1 || loading}
        >
          Sebelumnya
        </Button>
        <Button
          tone="kedua"
          onClick={() => onMove(info.halaman + 1)}
          disabled={info.halaman >= info.total_halaman || loading}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  )
}

/**
 * isBranchUnknown membedakan "cabang Anda tidak diketahui" dari gangguan lain.
 *
 * Ia dicocokkan lewat KODE galat, bukan lewat teks pesannya: teks dapat berubah kapan saja
 * tanpa mengubah artinya, dan pesan yang ini justru diambil apa adanya dari layar Pega —
 * sehingga ia lebih mungkin berubah daripada kodenya.
 */
function isBranchUnknown(error: unknown): boolean {
  return error instanceof APIError && error.kode === 'cabang_tidak_diketahui'
}

function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error) return error.message
  return 'Terjadi kesalahan yang tidak dikenali.'
}
