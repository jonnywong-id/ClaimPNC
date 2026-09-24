import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { AddIcon, ReloadIcon, SearchIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'
import { formatDate } from '@/components/format'
import { useSelectedPortal } from '@/app/portal'

import {
  useClaimReportList,
  useClaimReportOptions,
  useCreateClaimReport,
  useExportClaimReport,
} from './api'
import { EMPTY_QUERY, type ClaimReport, type ClaimReportQuery, type ReportCategory } from './types'

/**
 * Inbox Laporan Klaim — menu `MENU_ID 64`, pengganti harness `InboxRCVApp_Harness`.
 *
 * Judul yang dibaca pengguna di sistem lama adalah "Inbox Reporting Claim", dan isinya
 * adalah laporan kerugian yang masuk lewat surel, kurir, atau aplikasi sebelum menjadi
 * klaim bernomor.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/ViewStatusReceiveDocument-Section.xml` apa adanya: sembilan tab
 * berlencana, tiga penyaring di atas tabel, empat tombol tindakan, lalu grid berhalaman.
 * Judul tab dan nama kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega
 * supaya pengguna tidak perlu belajar ulang, dan itulah teks yang selama ini mereka baca.
 *
 * # Satu tombol Pega yang TIDAK ada di sini
 *
 * "Tarik data". Di sistem lama ia memanggil activity yang sama dengan Refresh
 * (`SetListRCV_Act`) — keduanya memuat ulang daftar, dan hanya berbeda nama. Membawa
 * dua tombol yang mengerjakan hal yang sama persis berarti membawa pertanyaan "apa
 * bedanya" yang tidak punya jawaban.
 */
export function ClaimReportInboxPage() {
  const [query, setQuery] = useState<ClaimReportQuery>(EMPTY_QUERY)
  const [keyword, setKeyword] = useState('')

  const navigate = useNavigate()
  const portal = useSelectedPortal((state) => state.alias)
  const options = useClaimReportOptions()
  const list = useClaimReportList(query)
  const create = useCreateClaimReport()
  const exportData = useExportClaimReport()

  // Lencana datang bersama daftar; sebelum daftar pertama tiba, tab digambar dari
  // pilihan yang lencananya masih kosong. Tanpa itu, deret tab baru muncul setelah
  // permintaan pertama selesai dan seluruh layar melompat.
  const category: ReportCategory[] = list.data?.kategori ?? options.data?.kategori ?? []
  const active = category.find((c) => c.kode === query.kategori)

  /** Mengubah penyaring SELALU mengembalikan ke halaman pertama. */
  function applyFilter(change: Partial<ClaimReportQuery>) {
    setQuery((previous) => ({ ...previous, ...change, halaman: 1 }))
  }

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Berkas laporan klaim milik satu badan hukum, dan aplikasi ini melayani ' +
            'empat. Pilih portal di bilah atas untuk membuka daftarnya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  return (
    <PageFrame>
      <CategoryTabs
        category={category}
        active={query.kategori}
        onSelect={(kode) => applyFilter({ kategori: kode })}
      />

      <form
        className="mt-4 grid gap-4 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut sm:grid-cols-2 lg:grid-cols-3"
        onSubmit={(e) => {
          e.preventDefault()
          applyFilter({ cari: keyword })
        }}
      >
        <SelectField
          id="kanwil"
          label="Pilih Kanwil"
          value={query.kanwil}
          onChange={(e) => applyFilter({ kanwil: e.target.value })}
          options={(options.data?.kanwil ?? []).map((k) => ({ value: k.kode, label: k.nama }))}
          emptyText="— seluruh kanwil —"
        />

        <SelectField
          id="bisnis"
          label="Bisnis"
          value={query.bisnis}
          onChange={(e) => applyFilter({ bisnis: e.target.value })}
          options={(options.data?.bisnis ?? [])
            // Pilihan "Semua bisnis" sudah diwakili baris kosong di puncak daftar;
            // menampilkannya dua kali membuat pengguna menebak apakah keduanya sama.
            .filter((b) => b.kode !== '')
            .map((b) => ({ value: b.kode, label: b.nama }))}
          emptyText="— semua bisnis —"
        />

        <div className="flex items-end gap-2">
          <div className="min-w-0 flex-1">
            <Field
              id="cari"
              label="Case ID"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="RCV-0001"
              autoComplete="off"
              icon={<SearchIcon className="h-4 w-4" />}
              hint="Dicocokkan persis, bukan sebagian."
            />
          </div>
          <Button type="submit" tone="kedua" className="mb-[26px]">
            Cari
          </Button>
        </div>
      </form>

      {options.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Pilihan saringan tidak dapat dimuat"
            description={messageOf(options.error)}
            tone="gangguan"
          />
        </div>
      )}

      {create.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Laporan baru tidak dapat dibuat"
            description={messageOf(create.error)}
            tone={toneOf(create.error)}
          />
        </div>
      )}

      {exportData.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Berkas ekspor tidak dapat diunduh"
            description={messageOf(exportData.error)}
            tone={toneOf(exportData.error)}
          />
        </div>
      )}

      <div className="mt-4">
        <DataTable<ClaimReport>
          columns={columnsFor(active?.komunikasi ?? false)}
          rows={list.data?.laporan ?? []}
          rowKey={(r) => r.id}
          title={active?.judul ?? 'Laporan klaim'}
          description={descriptionFor(active)}
          hideSearch
          isLoading={list.isPending}
          error={
            list.isError ? (
              <ErrorMessage
                title="Daftar laporan tidak dapat dimuat"
                description={messageOf(list.error)}
                tone="gangguan"
              />
            ) : undefined
          }
          emptyMessage="Tidak ada laporan pada tab ini."
          pagination={{
            page: list.data?.halaman.halaman ?? 1,
            size: list.data?.halaman.ukuran ?? 10,
            total: list.data?.halaman.total ?? 0,
            totalPage: list.data?.halaman.total_halaman ?? 1,
            onPageChange: (halaman) => setQuery((previous) => ({ ...previous, halaman })),
            isLoading: list.isFetching,
          }}
          actions={
            <>
              <BranchScope code={list.data?.batas_cabang ?? ''} />
              {/*
                Buat Baru MEMBUKA form isiannya, tidak berhenti setelah berkasnya dibuat.
                Itulah yang dilakukan alur lama: `CreateNewCaseRCV` membuat berkas kosong,
                lalu `Flow/InputReceiveDocument.xml` meneruskannya ke assignment "Receive
                Document" yang merender form `InputReceiveDocument`.

                Berhenti di sini — seperti versi pertama layar ini — menerbitkan berkas
                yang tidak dapat diapa-apakan, dan berkasnya pun mendarat di tab lain
                daripada yang sedang dibuka. Dari kursi petugas, tombolnya tampak tidak
                bekerja sama sekali.
              */}
              <Button
                type="button"
                tone="utama"
                onClick={() =>
                  create.mutate(undefined, {
                    onSuccess: (hasil) =>
                      navigate(`/inbox/laporan-klaim/${encodeURIComponent(hasil.laporan.id)}`),
                  })
                }
                disabled={create.isPending}
              >
                <AddIcon className="mr-1.5 h-4 w-4" />
                {create.isPending ? 'Membuat…' : 'Buat Baru'}
              </Button>
              <Button
                type="button"
                tone="kedua"
                onClick={() => exportData.mutate(query)}
                disabled={exportData.isPending}
              >
                {exportData.isPending ? 'Menyiapkan…' : 'Export Data'}
              </Button>
              <Button
                type="button"
                tone="kedua"
                onClick={() => void list.refetch()}
                disabled={list.isFetching}
              >
                <ReloadIcon className="mr-1.5 h-4 w-4" />
                {list.isFetching ? 'Memuat…' : 'Refresh'}
              </Button>
            </>
          }
        />
      </div>

      <p className="mt-4 text-xs text-slate-500">
        Tekan nomor berkas untuk membuka isiannya. Berkas bertanda{' '}
        <span className="font-medium">Pega</span> dibuka dalam modus baca saja — selama masa
        paralel, hanya satu sistem yang boleh menulis sebuah berkas.
      </p>
    </PageFrame>
  )
}

function PageFrame({ children }: { children: React.ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Laporan Klaim</h1>
        <p className="mt-1 text-sm text-slate-600">
          Laporan kerugian yang masuk dan menunggu diproses menjadi klaim.
        </p>
      </header>
      {children}
    </div>
  )
}

/**
 * Deret tab beserta lencananya.
 *
 * # Kenapa `<nav>` berisi tombol, bukan tautan
 *
 * Karena berpindah tab TIDAK mengubah alamat halaman. Menjadikannya tautan berarti
 * berjanji alamatnya dapat disalin dan dibuka kembali — janji yang tidak dipenuhi selama
 * penyaringnya hidup di state komponen. Bila kelak penyaring dipindahkan ke alamat,
 * tombol-tombol ini yang berubah menjadi tautan.
 */
function CategoryTabs({
  category,
  active,
  onSelect,
}: {
  category: ReportCategory[]
  active: string
  onSelect: (kode: string) => void
}) {
  if (category.length === 0) return null

  return (
    <nav aria-label="Kelompok laporan" className="mt-6 flex flex-wrap gap-2">
      {category.map((c) => {
        const selected = c.kode === active
        return (
          <button
            key={c.kode}
            type="button"
            aria-current={selected ? 'true' : undefined}
            /*
              Lencananya diberi nama, bukan dibiarkan terbaca sebagai angka telanjang.
              Tanpa ini, pembaca layar menyebut "Replied from ASM0" — angkanya menempel
              tanpa jeda karena JSX membuang spasi antarelemen, dan bahkan bila ada
              jedanya, "0" sendirian tidak menyatakan nol apa.
            */
            aria-label={c.jumlah === null ? c.judul : `${c.judul}, ${c.jumlah} berkas`}
            onClick={() => onSelect(c.kode)}
            className={[
              'inline-flex items-center gap-2 rounded-kontrol border px-3 py-1.5 text-sm',
              'transition-[background-color,border-color,box-shadow] duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
              selected
                ? 'border-blue-600 bg-blue-600 font-medium text-white shadow-aksen'
                : 'border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:bg-slate-50',
            ].join(' ')}
          >
            {c.judul}
            {/*
              Lencana hanya digambar bila jumlahnya BENAR-BENAR dihitung. Tab "Data
              rejected" tidak punya angka di sistem lama — kueri pencacahnya justru
              mengecualikan berkas yang ditolak — dan lencana bertuliskan 0 akan
              menyatakan "tidak ada berkas ditolak", yang tidak benar.
            */}
            {c.jumlah !== null && (
              <span
                aria-hidden="true"
                className={[
                  'rounded-full px-1.5 py-0.5 text-xs font-medium tabular-nums',
                  selected ? 'bg-white/20 text-white' : 'bg-slate-100 text-slate-600',
                ].join(' ')}
              >
                {c.jumlah}
              </span>
            )}
          </button>
        )
      })}
    </nav>
  )
}

/**
 * Kolom tabel, dengan nama persis seperti di layar Pega.
 *
 * "Last message" hanya digambar pada ketiga tab komunikasi — pada tab lain kolomnya
 * selalu kosong, dan kolom yang selalu kosong hanya memakan lebar layar.
 */
function columnsFor(message: boolean): Column<ClaimReport>[] {
  const column: Column<ClaimReport>[] = [
    {
      key: 'id',
      title: 'Case ID',
      value: (r) => r.id,
      width: '12rem',
      // Nomor berkas menjadi tautan pembuka form. Di layar lama, barisnya dibuka dengan
      // menekannya; tautan dipilih di sini supaya alamatnya dapat disalin, dibuka di tab
      // baru, dan dijangkau papan ketik — tiga hal yang tidak diberikan baris yang hanya
      // menanggapi klik.
      render: (r) => (
        <span className="inline-flex flex-wrap items-center gap-1.5">
          <Link
            to={`/inbox/laporan-klaim/${encodeURIComponent(r.id)}`}
            className="font-medium text-blue-700 underline decoration-blue-300 underline-offset-2 hover:text-blue-900 hover:decoration-blue-600"
          >
            {r.id}
          </Link>
          <OriginBadge origin={r.asal} />
        </span>
      ),
    },
    // Urutan kolom mengikuti layar lama apa adanya (`D-13`): Polis no mendahului Case
    // PNC, dan Business Name mendahului Insured Name. Urutan yang "lebih masuk akal"
    // tetap urutan yang berbeda, dan petugas membaca layar ini setiap hari.
    { key: 'polis', title: 'Polis no', value: (r) => r.nomor_polis, width: '10rem' },
    { key: 'klaim', title: 'Case PNC', value: (r) => r.nomor_klaim, width: '9rem' },
    { key: 'rujukan', title: 'Reference no', value: (r) => r.nomor_rujukan, width: '9rem' },
    { key: 'bisnis', title: 'Business Name', value: (r) => r.nama_bisnis, width: '10rem' },
    { key: 'tertanggung', title: 'Insured Name', value: (r) => r.tertanggung },
    {
      key: 'kejadian',
      title: 'Date of loss',
      value: (r) => r.tanggal_kejadian,
      width: '9rem',
      render: (r) => formatDate(r.tanggal_kejadian),
    },
    {
      key: 'masuk',
      title: 'Input Date',
      value: (r) => r.tanggal_masuk,
      width: '9rem',
      render: (r) => formatDate(r.tanggal_masuk),
    },
    { key: 'pembuat', title: 'Creator', value: (r) => r.pembuat, width: '10rem' },
    { key: 'cabang', title: 'Cabang Klaim', value: (r) => r.nama_cabang, width: '11rem' },
    // Dua kolom umur, berdampingan seperti di layar lama. "Aging" dibaca dari kolomnya
    // sendiri di basis data; "Total Aging" dihitung aplikasi dari tanggal aging.
    { key: 'aging', title: 'Aging', value: (r) => r.aging, width: '6rem' },
    {
      key: 'total-aging',
      title: 'Total Aging',
      value: (r) => String(r.umur_hari),
      width: '8rem',
      render: (r) => <AgingBadge days={r.umur_hari} />,
    },
    {
      key: 'posisi',
      title: 'Position',
      value: (r) => r.posisi,
      width: '9rem',
      render: (r) => <PositionBadge position={r.posisi} />,
    },
  ]

  if (message) {
    column.push({ key: 'pesan', title: 'Last message', value: (r) => r.pesan_akhir })
  }
  return column
}

/**
 * Asal sebuah berkas: ditulis Pega, atau diterbitkan aplikasi ini.
 *
 * Ia TIDAK ada di layar lama — di sana tabelnya memang hanya satu. Ia ada di sini karena
 * selama masa paralel daftar ini menggabungkan dua tabel, dan pertanyaan pertama pada
 * setiap selisih adalah "baris ini ditulis siapa".
 */
function OriginBadge({ origin }: { origin: string }) {
  const fromPega = origin === 'pega'
  return (
    <span
      title={
        fromPega
          ? 'Berkas warisan; masih dikelola Pega dan hanya dibaca dari sini.'
          : 'Berkas yang diterbitkan aplikasi ini.'
      }
      className={[
        'rounded-full px-1.5 py-0.5 text-[0.65rem] font-medium uppercase tracking-wide',
        fromPega ? 'bg-slate-100 text-slate-600' : 'bg-blue-100 text-blue-800',
      ].join(' ')}
    >
      {fromPega ? 'Pega' : 'Baru'}
    </span>
  )
}

/**
 * Umur berkas, dengan penegasan pada yang sudah lama menunggu.
 *
 * Ambangnya BUKAN aturan bisnis: sistem lama tidak punya ambang umur pada layar ini, dan
 * TAT sungguhan adalah lingkup `S-7`. Ia murni penanda baca — karena itu angkanya tetap
 * terbaca apa adanya, dan warnanya tidak pernah menjadi satu-satunya pembeda.
 */
/**
 * Umur berkas dalam bentuk ringkas, seperti layar lama: `6y ago`.
 *
 * # Apa yang benar-benar terbukti, dan apa yang tidak
 *
 * Dari layar lama hanya bentuk TAHUN yang terlihat — seluruh baris pada tangkapan layar
 * berbunyi `6y ago`. Satuan bulan dan hari mengikuti bentuk ringkas yang sama karena itu
 * satu-satunya bentuk yang dapat disandarkan pada bukti; bila Pega ternyata menuliskannya
 * lain, yang berubah hanya fungsi ini.
 *
 * Teksnya sengaja TIDAK diterjemahkan (`D-13`, `D-80`): ia teks layar yang dibaca petugas
 * setiap hari, dan mengubahnya berarti mengubah layar, bukan menerjemahkan kode.
 */
function AgingBadge({ days }: { days: number }) {
  return <span className="tabular-nums">{relativeAge(days)}</span>
}

function relativeAge(days: number): string {
  if (days <= 0) return 'today'
  if (days >= 365) return `${Math.floor(days / 365)}y ago`
  if (days >= 30) return `${Math.floor(days / 30)}mo ago`
  return `${days}d ago`
}

function PositionBadge({ position }: { position: string }) {
  const style: Record<string, string> = {
    Outstanding: 'bg-blue-50 text-blue-800 border-blue-200',
    'Not Registered': 'bg-amber-50 text-amber-800 border-amber-200',
    'Not Transferred': 'bg-slate-50 text-slate-700 border-slate-200',
  }
  return (
    <span
      className={[
        'inline-block rounded-full border px-2 py-0.5 text-xs font-medium',
        style[position] ?? 'border-slate-200 bg-slate-50 text-slate-700',
      ].join(' ')}
    >
      {position}
    </span>
  )
}

function descriptionFor(category: ReportCategory | undefined): string {
  if (!category) return ''
  if (category.komunikasi) {
    return 'Percakapan cabang dan kantor pusat pada berkas yang Anda buat.'
  }
  return 'Urut menurut tanggal aging, yang terbaru di atas — sama seperti layar lama.'
}

/** Keterangan batas cabang yang sedang berlaku, digambar di kanan judul tabel. */
function BranchScope({ code }: { code: string }) {
  if (code === '') return null
  return (
    <span className="rounded-kontrol bg-slate-100 px-2 py-1 text-xs text-slate-600">
      Cabang <span className="font-medium text-slate-900">{code}</span>
    </span>
  )
}

function messageOf(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/**
 * toneOf memilih nada pesan dari GALATNYA, bukan dari tempat pesan itu muncul.
 *
 * Kedua nada punya arti yang tegas (lihat ErrorMessage): `penolakan` berarti ada yang
 * dapat pengguna perbaiki, `gangguan` berarti sistemnya yang bermasalah dan mengulang
 * tidak menolong. Satu tombol dapat gagal karena keduanya — isian yang belum benar
 * (422) atau tabel penyimpanan yang belum dibuat DBA (503) — sehingga nada yang dipatok
 * di satu tempat pasti salah untuk salah satunya.
 *
 * Yang paling merugikan adalah arah ini: kegagalan pemasangan yang ditampilkan sebagai
 * penolakan membuat petugas mengubah-ubah isiannya berkali-kali, padahal tidak ada
 * isian yang salah.
 */
function toneOf(failure: unknown): 'penolakan' | 'gangguan' {
  if (failure instanceof APIError && failure.status >= 500) return 'gangguan'
  return 'penolakan'
}
