import { useMemo, useState, type FormEvent, type ReactNode } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column as TableColumn } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { FormField } from '@/components/FormField'
import { SelectField, type SelectOption } from '@/components/SelectField'
import { formatDate, formatPercent, formatRupiah } from '@/components/format'

import { RemarkCell } from './RemarkCell'
import { UKURAN_HALAMAN, useCaseStudyList, useCaseStudyMetadata, useExportCaseStudy } from './api'
import type { CaseStudyRow, Column, FilterInput, MetadataResponse } from './types'

/**
 * **Case Study Claim** — `MENU_ID 74`, pengganti harness `PNCStudyClaim`.
 *
 * Layar TELAAH atas klaim besar: hanya klaim yang salah satu baris settlement-nya
 * melampaui **Rp 5.000.000.000** yang muncul di sini.
 *
 * # Ia BUKAN Inbox, meski butir menunya di bawah kelompok INBOX
 *
 * Menurut `D-79`, Inbox adalah daftar PEKERJAAN milik seorang pengguna — barisnya hilang
 * setelah ditindaklanjuti dan punya tenggat. Layar ini tidak: barisnya klaim yang sudah
 * ada, tidak hilang setelah dicatat, dan tidak punya tenggat. Yang membuatnya masuk daftar
 * hanyalah NILAI.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/PNCStudyClaim-Section.xml` apa adanya: judul, lalu empat penyaring
 * berjajar (Awal, Akhir, Status, Bisnis), dua tombol (Lihat Data, Export Data), lalu grid
 * 24 kolom. Nama kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega, dan
 * itulah teks yang selama ini dibaca pengguna, termasuk huruf besar semua pada sebagian
 * judul dan salah ketik "SUBROGARATION".
 *
 * # Kenapa grid KOSONG sampai tombol ditekan
 *
 * Karena layar lama pun begitu: `StudyClaim_act` baru menjalankan kuerinya saat "Lihat
 * Data" ditekan. Memuat otomatis akan menembak kueri berisi delapan subkueri agregat tanpa
 * ada yang memintanya.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena daftar 24 kolom beserta judulnya adalah hasil pembacaan export Pega, dan tempat
 * pembacaan itu tercatat adalah `internal/casestudyclaim/columns.go`. Menyalinnya ke sini
 * berarti daftar yang sama hidup di dua tempat — dan yang paling mudah tertinggal adalah
 * berkas CSV, yang tidak dilihat siapa pun sampai ada yang mengunduhnya.
 */
export function CaseStudyClaimPage() {
  const [draft, setDraft] = useState<FilterInput>(emptyFilter)
  const [applied, setApplied] = useState<FilterInput | null>(null)
  const [page, setPage] = useState(1)

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useCaseStudyMetadata()

  const active = applied ?? emptyFilter
  const searched = applied !== null

  const list = useCaseStudyList(active, page, searched)
  const exportFile = useExportCaseStudy()

  const columns = useMemo(
    () => buildColumns(meta.data?.kolom ?? []),
    [meta.data?.kolom],
  )

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Klaim telaah milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  if (meta.isError) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    setApplied(draft)
    setPage(1)
  }

  const total = list.data?.total ?? 0
  const totalPage = Math.max(1, Math.ceil(total / UKURAN_HALAMAN))

  return (
    <PageFrame>
      <form
        className="mt-4 space-y-4 rounded-kartu border border-slate-200 bg-white p-4"
        onSubmit={submit}
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <FormField
            id="case-study-awal"
            label="Awal"
            type="date"
            value={draft.dari}
            onChange={(event) => setDraft({ ...draft, dari: event.target.value })}
          />
          <FormField
            id="case-study-akhir"
            label="Akhir"
            type="date"
            value={draft.sampai}
            onChange={(event) => setDraft({ ...draft, sampai: event.target.value })}
          />
          <SelectField
            id="case-study-status"
            label="Status"
            options={toSelectOptions(meta.data?.status)}
            emptyText="----- Pilih -----"
            value={draft.status}
            onChange={(event) => setDraft({ ...draft, status: event.target.value })}
          />
          <SelectField
            id="case-study-bisnis"
            label="Bisnis"
            options={toSelectOptions(meta.data?.bisnis)}
            emptyText="----- Pilih -----"
            value={draft.bisnis}
            onChange={(event) => setDraft({ ...draft, bisnis: event.target.value })}
          />
        </div>

        {/*
          Keterangan bahwa hanya TAHUN dari kedua tanggal yang dipakai.

          Ia datang dari server, bukan ditulis tetap di sini, supaya penjelasan dan
          penyaringnya tidak dapat menyimpang. Tanpa keterangan ini, daftar yang memuat
          klaim di luar bulan yang dipilih akan dilaporkan berulang kali sebagai kerusakan.
        */}
        {meta.data?.catatan_periode && (
          <p className="text-xs text-slate-600">{meta.data.catatan_periode}</p>
        )}

        <div className="flex flex-wrap items-center gap-2">
          <Button type="submit" tone="utama">
            Lihat Data
          </Button>
          <Button
            onClick={() => exportFile.mutate(active)}
            disabled={!searched || exportFile.isPending}
          >
            {exportFile.isPending ? 'Menyiapkan berkas…' : 'Export Data'}
          </Button>

          {/*
            "Last Refresh" pada layar lama. Ia menyatakan kapan angka di grid diambil —
            pada layar telaah yang dibuka berjam-jam, itu perbedaan yang menentukan.
          */}
          {searched && list.dataUpdatedAt > 0 && (
            <span className="ml-auto text-xs text-slate-500" role="status">
              Last Refresh: {formatClock(list.dataUpdatedAt)}
            </span>
          )}
        </div>
      </form>

      {exportFile.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Berkas tidak dapat diunduh"
            description={messageOf(exportFile.error)}
            tone="gangguan"
          />
        </div>
      )}

      {!searched ? (
        <p className="mt-4 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-600">
          Pilih periode, lalu tekan <strong>Lihat Data</strong>. Layar lama pun menampilkan
          grid kosong sampai tombol itu ditekan.
        </p>
      ) : (
        <div className="mt-4">
          <DataTable<CaseStudyRow>
            columns={columns}
            rows={list.data?.baris ?? []}
            rowKey={(row) => row.nomor_klaim}
            label="Case Study Claim"
            // Kotak cari bawaan disembunyikan: layar lama tidak punya pencarian, dan kotak
            // bawaan hanya akan menyaring halaman yang sedang terbuka — hasilnya
            // menyesatkan pada data berhalaman.
            hideSearch
            showHeaderWhenEmpty
            isLoading={list.isPending}
            error={
              list.isError ? (
                <ErrorMessage
                  title="Data tidak dapat dimuat"
                  description={messageOf(list.error)}
                  tone="gangguan"
                />
              ) : undefined
            }
            emptyMessage={emptyMessageFor(meta.data)}
            pagination={{
              page,
              size: UKURAN_HALAMAN,
              total,
              totalPage,
              onPageChange: setPage,
              isLoading: list.isFetching,
            }}
          />

          {/*
            Periode yang BENAR-BENAR dipakai, disebutkan setelah data datang.

            Pengguna memilih dua tanggal; yang menyaring hanya tahunnya. Menyebutkannya di
            sini membedakan "penyaringnya tidak bekerja" dari "memang begitu cara kerjanya".
          */}
          {list.data && (
            <p className="mt-3 text-xs text-slate-500" role="status">
              Disaring menurut tahun registrasi {list.data.periode.tahun_awal}
              {list.data.periode.tahun_akhir !== list.data.periode.tahun_awal &&
                `–${list.data.periode.tahun_akhir}`}
              .
            </p>
          )}
        </div>
      )}
    </PageFrame>
  )
}

function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[110rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Case Study Claim</h1>
        <p className="mt-1 text-sm text-slate-600">
          Telaah klaim bernilai besar: nilai settlement, share ASM, dan catatan hasil
          telaahnya.
        </p>
      </header>
      {children}
    </div>
  )
}

/** Penyaring kosong — keadaan awal formulir. */
const emptyFilter: FilterInput = { dari: '', sampai: '', bisnis: '', status: '' }

/**
 * buildColumns menyusun kolom tabel dari bentuk yang ditetapkan server.
 *
 * Kolom aksi ditambahkan di ujung, bukan disebut server: ia bukan DATA melainkan kontrol,
 * dan backend tidak tahu apa pun tentang tombol.
 */
function buildColumns(columns: Column[]): TableColumn<CaseStudyRow>[] {
  const result: TableColumn<CaseStudyRow>[] = columns.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => cellText(row, column),
    alignRight: column.jenis === 'uang' || column.jenis === 'persen',

    // Kolom catatan digambar sebagai isian, bukan teks. Ia satu-satunya sel yang dapat
    // disunting di seluruh layar — `pyEditOptions=Editable` pada section Pega.
    ...(column.jenis === 'catatan'
      ? {
          render: (row: CaseStudyRow) => <RemarkCell row={row} />,

          // Tidak dapat diurutkan: yang digambar isian, bukan teks, dan mengurutkan
          // kolom berisi isian yang sedang disunting akan memindahkan barisnya di bawah
          // kursor pengguna.
          noSort: true,
          width: '20rem',
        }
      : {}),
  }))

  return result
}

/**
 * cellText menyusun teks satu sel.
 *
 * Nilai kosong menjadi tanda pisah — bukan sel kosong yang tidak dapat dibedakan dari
 * kolom yang gagal dimuat.
 *
 * Uang dan persentase diformat lewat `shared` yang sama dengan seluruh layar lain, bukan
 * ditulis ulang di sini: `TKT-U4-001` menuntut angka uang di layar transaksi TAMPIL SAMA
 * dengan yang muncul di laporan, dan selama pemformatan disalin per layar, "sama" adalah
 * harapan — bukan sifat.
 */
function cellText(row: CaseStudyRow, column: Column): string {
  const value = row[column.kunci as keyof CaseStudyRow]

  if (value === null || value === undefined || value === '') return '—'

  switch (column.jenis) {
    case 'uang':
      return formatRupiah(value as number)
    case 'persen':
      return formatPercent(value as number)
    case 'tanggal':
      return formatDate(String(value))
    default:
      return String(value)
  }
}

/** toSelectOptions mengubah pilihan dari server menjadi bentuk yang dipakai SelectField. */
function toSelectOptions(options: { kode: string; label: string }[] | undefined): SelectOption[] {
  return (options ?? []).map((item) => ({ value: item.kode, label: item.label }))
}

/**
 * emptyMessageFor menyusun pesan grid kosong yang MENJELASKAN, bukan sekadar "tidak ada
 * data".
 *
 * Hasil kosong di layar ini hampir selalu punya satu sebab yang sama: tidak ada klaim yang
 * melampaui ambang pada periode itu. Menyebut ambangnya menjawab "kenapa klaim saya tidak
 * muncul" tanpa pengguna perlu bertanya.
 */
function emptyMessageFor(metadata: MetadataResponse | undefined): string {
  if (!metadata) return 'Tidak ada klaim yang cocok dengan penyaring ini.'

  return (
    'Tidak ada klaim yang cocok. Layar ini hanya memuat klaim yang salah satu nilai ' +
    `settlement-nya melampaui ${formatRupiah(metadata.ambang_nilai_klaim)}.`
  )
}

/**
 * Panel "Yang perlu diketahui" DIHAPUS atas permintaan Work Owner (2026-09-26).
 *
 * Isinya tidak hilang, hanya berpindah ke tempat yang lebih dekat dengan kejadiannya:
 *
 *	keterangan periode   tepat di bawah kedua isian tanggal
 *	ambang nilai klaim   pada pesan grid ketika hasilnya kosong
 *
 * Kedua kalimat itu masih dikirim server (`catatan_periode`, `catatan_kolom_kembar`,
 * `ambang_nilai_klaim`) dan TIDAK dibuang dari kontrak — layar lain, atau layar ini kelak,
 * masih dapat memakainya.
 *
 * Yang benar-benar hilang dari layar hanyalah `catatan_kolom_kembar`. Kekembaran "Nature of
 * Loss" dan "Cause of Loss" karena itu tidak lagi dinyatakan di layar; ia tetap tercatat di
 * kode dan di dokumen, dan `P-5` yang menjaganya direplikasi.
 */

/**
 * formatClock menuliskan jam pengambilan data terakhir.
 *
 * Hanya JAMNYA, tanpa tanggal: layar ini dibuka dan disegarkan dalam satu sesi kerja, dan
 * tanggal hanya menambah panjang tanpa menambah keterangan.
 */
function formatClock(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
