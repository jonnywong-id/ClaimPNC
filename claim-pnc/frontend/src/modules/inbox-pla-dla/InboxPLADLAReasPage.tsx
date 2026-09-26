import { useMemo, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TabBar } from '@/components/TabBar'

import {
  useEksporReas,
  useReasCounts,
  useReasList,
  useReasMetadata,
  useReasXOL,
  type ParameterDaftar,
} from './api'
import { StatusSummary } from './StatusSummary'
import { XOLPanel } from './XOLPanel'
import { bukanReasuradur, formatTanggal, pesanGalat, pesanMuat } from './pesan'
import type { Baris, Daftar } from './types'

/**
 * Layar Inbox PLA DLA — milik REASURADUR.
 *
 * Pengganti `Harness/InboxPLADLA-Harness.xml` (`MENU_ID 45`).
 *
 * # Apa yang ditampilkan layar ini, dan untuk siapa
 *
 * **Klaim yang pemberitahuannya sudah DIKIRIMKAN kepada mitra reasuransi yang sedang
 * masuk.** Login pemanggil dicocokkan ke `POOLDATA.T_REINSURER.LOGIN`, dan hasil
 * pencocokan itulah yang menentukan klaim mana yang terlihat.
 *
 * Tiga tab:
 *
 *	PLA    PLA sudah dikirimkan kepada Anda, DLA belum
 *	DLA    DLA sudah dikirimkan kepada Anda
 *	Close  klaimnya sudah selesai dan tidak lagi menunggu penutupan
 *
 * # JANGAN tertukar dengan "Inbox PLA, DLA, Pre DLA" (`MENU_ID 44`)
 *
 * Penyaringnya BERLAWANAN ARAH. Layar itu menampilkan dokumen yang belum dikirim, untuk
 * petugas internal; layar ini menampilkan dokumen yang sudah dikirim, untuk penerimanya.
 *
 * # Petugas internal yang membuka layar ini DITOLAK, bukan dibiarkan melihat layar kosong
 *
 * Login yang tidak terdaftar sebagai mitra menerima pesan yang menjelaskan sebabnya dan
 * menunjuk menu yang benar. Di Pega ia hanya melihat layar kosong tanpa satu pun
 * keterangan — dan menyimpulkan sistemnya rusak.
 *
 * # Layar BACA-SAJA
 *
 * Ia baca-saja di Pega pula: keempat activity-nya hanya memuat data. Ketiga tombolnya —
 * "Detail Claim", "Detail", dan "DLA" — membuka layar rincian tersendiri yang belum
 * dianalisis, dan karena itu belum digambar di sini.
 */
export function InboxPLADLAReasPage() {
  const meta = useReasMetadata()

  const [tab, setTab] = useState('')
  const [page, setPage] = useState(1)

  // Kotak "Claim No" dipegang DUA KALI: yang sedang diketik, dan yang sudah dikirim.
  //
  // Layar lamanya memakai tombol "Search Data", bukan pencarian saat mengetik — dan
  // penyaringnya menyapu tabel klaim yang sama dengan modul lain.
  const [ketikan, setKetikan] = useState('')
  const [dicari, setDicari] = useState('')

  const daftar = meta.data?.daftar ?? []
  const aktif = tab || meta.data?.daftar_bawaan || ''
  const daftarAktif = daftar.find((item) => item.kode === aktif)

  const parameter: ParameterDaftar = useMemo(
    () => ({ tab: aktif, page, cari: dicari }),
    [aktif, page, dicari],
  )

  const siap = aktif !== ''
  const list = useReasList(parameter, siap)
  const ringkas = useReasCounts(parameter, siap)
  const ekspor = useEksporReas()

  // Grid XOL baru diambil SETELAH daftarnya berhasil.
  //
  // Menunggu keberhasilan, bukan sekadar memeriksa belum-ada-galat, dan perbedaannya
  // nyata: keduanya berangkat bersamaan pada render pertama, ketika galat daftarnya
  // belum tiba. Pemanggil yang bukan mitra terdaftar akan menerima DUA penolakan untuk
  // satu sebab yang sama — dan yang kedua digambar sebagai kerusakan di tengah halaman
  // yang sudah menjelaskan sebabnya di atas.
  //
  // Biayanya satu perjalanan yang tertunda sesaat; yang ditukar dengannya adalah
  // permintaan yang sudah pasti ditolak tidak pernah dikirim sama sekali.
  const ditolak = bukanReasuradur(list.error)
  const xol = useReasXOL(siap && list.isSuccess)

  function pilihDaftar(kode: string) {
    setTab(kode)
    setPage(1)
  }

  function cari() {
    setDicari(ketikan.trim())
    setPage(1)
  }

  if (meta.isPending) {
    return (
      <Bingkai>
        <p className="text-sm text-slate-600">Memuat keterangan layar…</p>
      </Bingkai>
    )
  }

  if (meta.isError) {
    const pesan = pesanMuat(meta.error)
    return (
      <Bingkai>
        <ErrorMessage
          title={pesan.title}
          description={pesan.description}
          tone={pesan.tone}
        />
      </Bingkai>
    )
  }

  // Penolakan "bukan mitra terdaftar" menggantikan SELURUH isi layar, bukan digambar
  // sebagai galat di atas tabel kosong.
  //
  // Ia bukan kegagalan sementara yang layak dicoba ulang: selama pendaftarannya belum
  // diubah, jawabannya akan sama. Menggambar tabel, tombol ekspor, dan bilah tab di
  // bawahnya hanya menawarkan hal-hal yang seluruhnya akan ditolak.
  if (ditolak) {
    const pesan = pesanMuat(list.error)
    return (
      <Bingkai>
        <ErrorMessage
          title={pesan.title}
          description={pesan.description}
          tone={pesan.tone}
        />
      </Bingkai>
    )
  }

  return (
    <Bingkai>
      <TabBar
        tabs={daftar.map((item) => ({
          kode: item.kode,
          nama: item.nama,
          keterangan: item.keterangan,
        }))}
        active={aktif}
        onSelect={pilihDaftar}
        label="Tahap pemberitahuan"
      />

      {daftarAktif && (
        <p className="text-sm text-slate-600">{daftarAktif.keterangan}</p>
      )}

      <StatusSummary
        rows={ringkas.data?.baris ?? []}
        isLoading={ringkas.isLoading}
      />

      <DataTable<Baris>
        columns={kolomDaftar(daftarAktif)}
        rows={list.data?.baris ?? []}
        rowKey={(row) => row.kunci_klaim}
        label={`Daftar ${daftarAktif?.nama ?? ''}`}
        isLoading={list.isLoading}
        error={
          list.isError ? (
            <ErrorMessage
              title="Daftar tidak dapat dimuat"
              description={pesanGalat(list.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={pesanKosong(dicari)}
        searchLabel="Claim No"
        serverSearch={{
          value: ketikan,
          onChange: setKetikan,
          matchCount: list.data?.paginasi.total,
        }}
        pagination={{
          page: list.data?.paginasi.halaman ?? 1,
          size: list.data?.paginasi.ukuran ?? 10,
          total: list.data?.paginasi.total ?? 0,
          totalPage: list.data?.paginasi.total_halaman ?? 1,
          onPageChange: setPage,
          isLoading: list.isFetching,
        }}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button type="button" onClick={cari} disabled={list.isFetching}>
              {list.isFetching ? 'Mencari…' : 'Search Data'}
            </Button>
            <Button
              type="button"
              tone="kedua"
              onClick={() => void list.refetch()}
              disabled={list.isFetching}
            >
              Refresh
            </Button>
            <Button
              type="button"
              tone="kedua"
              disabled={ekspor.isPending}
              onClick={() => ekspor.mutate(parameter)}
            >
              {ekspor.isPending ? 'Menyiapkan…' : 'Export To Excel'}
            </Button>
          </div>
        }
      />

      {ekspor.error != null && (
        <ErrorMessage
          title="Berkas ekspor tidak dapat diambil"
          description={pesanGalat(ekspor.error)}
          tone="gangguan"
        />
      )}

      <XOLPanel
        kolom={meta.data?.kolom_xol ?? []}
        rows={xol.data?.baris ?? []}
        isLoading={xol.isLoading}
        isError={xol.isError}
        error={xol.error}
      />

      <SelisihTerencana butir={meta.data?.selisih_terencana ?? []} />
    </Bingkai>
  )
}

/** Bingkai adalah judul layar beserta ruang isinya. */
function Bingkai({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-5 p-6">
      <header>
        <h1 className="text-lg font-semibold text-slate-900">Inbox PLA / DLA</h1>
        <p className="mt-1 text-sm text-slate-600">
          Klaim yang pemberitahuan PLA atau DLA-nya sudah dikirimkan kepada Anda sebagai
          mitra reasuransi.
        </p>
      </header>

      {children}
    </div>
  )
}

/**
 * kolomDaftar menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * Alias Pega-nya menyesatkan hampir seluruhnya — `"TSI"` berarti kunci klaim,
 * `"MARKETING"` berarti nama lini bisnis, `"CURRENCY"` berarti tanggal kejadian — dan
 * menulis ulang pemetaannya di layar berarti dua tempat yang dapat bergeser.
 */
function kolomDaftar(daftar: Daftar | undefined): Column<Baris>[] {
  if (!daftar) return []

  return daftar.kolom.map((item) => ({
    key: item.kunci,
    title: item.judul,
    value: (row) => nilaiSel(row, item.kunci),
    render: (row) => gambarSel(row, item.kunci, item.tanggal),
  }))
}

/** nilaiSel mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function nilaiSel(row: Baris, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/**
 * gambarSel menggambar satu sel.
 *
 * Kolom "Status" jatuh ke KODENYA bila artinya tidak ada di master. Kode status yang
 * tidak terdaftar memang terjadi — domainnya 33 kode dan data lama memuat kode di luar
 * itu — dan sel kosong akan membuat barisnya tampak rusak.
 */
function gambarSel(row: Baris, kunci: string, tanggal: boolean) {
  if (kunci === 'status') {
    const isi = row.status || row.kode_status
    if (isi === '') return <span className="text-slate-400">—</span>
    return isi
  }

  const isi = nilaiSel(row, kunci)
  if (tanggal) return formatTanggal(isi)
  if (isi === '') return <span className="text-slate-400">—</span>
  return isi
}

/** pesanKosong menjelaskan MENGAPA daftarnya kosong. */
function pesanKosong(dicari: string): string {
  if (dicari !== '') {
    return `Tidak ada klaim yang nomornya mengandung "${dicari}" pada daftar ini.`
  }
  return 'Belum ada klaim pada tahap ini yang pemberitahuannya dikirimkan kepada Anda.'
}

/** SelisihTerencana menggambar selisih terhadap layar Pega di kaki halaman. */
function SelisihTerencana({ butir }: { butir: string[] }) {
  if (butir.length === 0) return null

  return (
    <details className="rounded-kartu border border-slate-200 bg-slate-50 p-4">
      <summary className="cursor-pointer text-sm font-medium text-slate-800">
        Perbedaan yang disengaja terhadap layar Pega ({butir.length})
      </summary>
      <ul className="mt-3 space-y-2 text-sm text-slate-600">
        {butir.map((isi) => (
          <li key={isi} className="flex gap-2">
            <span aria-hidden className="text-slate-400">
              •
            </span>
            <span>{isi}</span>
          </li>
        ))}
      </ul>
    </details>
  )
}
