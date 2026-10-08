import { useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'

import { useSelectedPortal } from '@/app/portal'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { NoteList, screenGate } from '@/components/inbox/InboxNotices'
import { apiMessageOf, isISODate } from '@/components/inbox/messages'
import { columnsWithAction } from '@/components/inbox/serverColumns'
import { useDebouncedCommit } from '@/components/inbox/useDebouncedCommit'
import { InboxPageFrame } from '@/components/inbox/InboxPageFrame'

import { ServiceCenterTabs } from './ServiceCenterTabs'
import { useInboxServiceCenterList, useInboxServiceCenterMetadata } from './api'
import {
  EMPTY_FILTER,
  type FilterForm,
  type ServiceClaim,
  type Tab,
  type TabColumn,
} from './types'

/**
 * Inbox Service Center — menu `MENU_ID 46`, pengganti harness `InboxServiceCenter`.
 *
 * Isinya daftar klaim portal rekanan: klaim perbaikan perangkat yang masuk dari portal mitra,
 * dipecah menjadi empat antrean menurut status persetujuannya. Per `D-79` ia benar-benar
 * Inbox — barisnya pekerjaan, dan ia berpindah tab begitu komite memutuskan.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/BrowseServiceCenter-Section.xml` apa adanya: bilah tab, kotak cari,
 * lalu grid berhalaman. Judul kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru
 * Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * # Kenapa kolomnya datang dari server
 *
 * Karena daftar kolom adalah hasil pembacaan export Pega, dan tempat pembacaan itu tercatat
 * adalah backend. Menyalinnya ke sini berarti daftar yang sama hidup di dua tempat.
 *
 * # Satu perilaku yang tampak seperti cacat, tetapi memang ditiru
 *
 * Begitu kotak cari terisi, bilah halaman MENGHILANG dan seluruh baris yang cocok tampil
 * sekaligus. Sistem lama mematikan paginasinya sendiri saat mencari (`P-5`); server
 * menyatakannya lewat `paginasi.aktif`, dan layar mengikutinya alih-alih menggambar tombol
 * halaman yang tidak melakukan apa pun.
 */
export function InboxServiceCenterPage() {
  const [filter, setFilter] = useState<FilterForm>(EMPTY_FILTER)
  const [page, setPage] = useState(1)

  /**
   * Isi kotak cari yang sedang DIKETIK, terpisah dari kata kunci yang sudah dikirim.
   *
   * Keduanya dipisah karena pencarian di layar ini mahal: paginasinya mati, sehingga satu
   * permintaan dapat menarik seluruh baris yang cocok. Mengirim satu permintaan per huruf
   * berarti melakukannya berulang kali untuk satu kata.
   */
  const [draft, setDraft] = useState('')

  // Ketikan menunggu jeda sebelum dikirim (`useDebouncedCommit`).
  useDebouncedCommit(draft, filter.cari, (value) => {
    setFilter((previous) => ({ ...previous, cari: value }))
    setPage(1)
  })

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useInboxServiceCenterMetadata()
  const list = useInboxServiceCenterList(filter, page, meta.isSuccess)

  const tabs: Tab[] = meta.data?.tab ?? []
  const active = filter.tab || meta.data?.tab_bawaan || ''
  const tab = tabs.find((candidate) => candidate.kode === active)

  /**
   * Berpindah tab MEMBERSIHKAN kotak cari dan mengembalikan ke halaman pertama.
   *
   * Membawa kata kunci lama ke tab baru akan menampilkan antrean yang tampak kosong padahal
   * isinya ada — dan pengguna tidak punya cara melihat bahwa penyebabnya kotak cari yang
   * masih terisi dari tab sebelumnya.
   */
  function selectTab(code: string) {
    setFilter({ ...EMPTY_FILTER, tab: code })
    setDraft('')
    setPage(1)
  }

  const gate = screenGate({
    portal,
    subject: 'Klaim portal rekanan',
    failed: meta.isError,
    error: meta.error,
    describe: apiMessageOf,
  })
  if (gate) return <PageFrame>{gate}</PageFrame>

  const info = list.data?.paginasi

  return (
    <PageFrame>
      <div className="mt-4">
        <ServiceCenterTabs tabs={tabs} active={active} onSelect={selectTab} />
      </div>

      {tab && (
        <>
          <p className="mt-3 text-sm text-slate-600">{tab.keterangan}</p>

          <SearchBar value={draft} onChange={setDraft} />

          <div className="mt-4">
            <DataTable<ServiceClaim>
              columns={columnsFor(tab)}
              rows={list.data?.baris ?? []}
              rowKey={(row) => `${row.id}|${row.repair_id}`}
              title={tab.nama}
              // Kotak cari bawaan disembunyikan: layar ini punya kotaknya sendiri di atas,
              // dan yang kedua hanya akan menyaring halaman yang sedang terbuka — hasilnya
              // menyesatkan pada data berhalaman.
              hideSearch
              isLoading={list.isPending}
              error={
                list.isError ? (
                  <ErrorMessage
                    title="Antrean tidak dapat dimuat"
                    description={apiMessageOf(list.error)}
                    tone="gangguan"
                  />
                ) : undefined
              }
              emptyMessage={emptyMessageFor(tab, filter)}
              /*
                Bilah halaman hanya digambar saat paginasinya memang aktif. Saat mencari,
                server mengirim seluruh baris yang cocok dalam satu halaman — menggambar
                "Berikutnya" di sana berarti tombol yang tidak pernah bisa ditekan.

                Disebar bersyarat, bukan diisi `undefined`: `exactOptionalPropertyTypes`
                membedakan "prop tidak diberikan" dari "prop bernilai undefined", dan
                hanya yang pertama yang sah bagi prop opsional.
              */
              {...(info?.aktif
                ? {
                    pagination: {
                      page: info.halaman,
                      size: info.ukuran,
                      total: info.total,
                      totalPage: info.total_halaman,
                      onPageChange: setPage,
                      isLoading: list.isFetching,
                    },
                  }
                : {})}
            />

            {info && !info.aktif && info.total > 0 && (
              <output className="block mt-3 text-sm text-slate-600">
                Menampilkan seluruh {info.total} baris yang cocok. Saat mencari, hasilnya
                tidak dibagi per halaman — sama seperti di layar lama.
              </output>
            )}
          </div>
        </>
      )}

      <NoteList title="Yang perlu diketahui" lines={meta.data?.keterbatasan ?? []} />
    </PageFrame>
  )
}

function PageFrame({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <InboxPageFrame
      title="Inbox Service Center"
      intro={
        'Klaim portal rekanan yang Anda tangani: registrasi baru, yang menunggu ' +
        'keputusan komite, serta yang sudah disetujui atau ditolak.'
      }
    >
      {children}
    </InboxPageFrame>
  )
}

/**
 * Kotak "Cari" di atas tabel.
 *
 * Jangkauannya dinyatakan tepat di bawah kotaknya, bukan hanya di catatan bawah. Alasannya
 * konkret: penyaring pencarian sistem lama adalah IRISAN dua kelompok, sehingga mengetik ID
 * atau No Klaim saja TIDAK menghasilkan baris. Tanpa keterangan itu, pengguna yang mengetik
 * ID akan menyimpulkan antreannya kosong.
 */
function SearchBar({
  value,
  onChange,
}: Readonly<{
  value: string
  onChange: (text: string) => void
}>) {
  return (
    <div className="mt-4 w-full sm:w-80">
      <label
        htmlFor="inbox-service-center-cari"
        className="block text-sm font-medium text-slate-700"
      >
        Cari
      </label>
      {/*
        Kotak ini sengaja TIDAK dinonaktifkan saat permintaan sedang berjalan.
        Menonaktifkannya berarti huruf yang diketik selama permintaan itu HILANG — dan
        karena setiap ketikan memicu permintaan, kotak yang menonaktifkan diri akan menelan
        sebagian besar kata yang diketik cepat.
      */}
      <input
        id="inbox-service-center-cari"
        type="search"
        value={value}
        placeholder="No Polis, Nasabah, atau IMEI"
        onChange={(event) => onChange(event.target.value)}
        className={[
          'mt-1 block w-full rounded-kontrol border border-slate-300 px-3 py-2 text-sm',
          'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
        ].join(' ')}
      />
      <p className="mt-1 text-xs text-slate-500">
        Menelusuri No Polis, Nasabah, dan IMEI. Mencari dengan ID atau No Klaim saja tidak
        menghasilkan baris — sama seperti di layar lama.
      </p>
    </div>
  )
}

/**
 * columnsFor menyusun kolom tabel dari bentuk yang ditetapkan server.
 *
 * Kolom aksi ditambahkan di ujung, bukan disebut server: ia bukan DATA melainkan kontrol, dan
 * backend tidak tahu apa pun tentang rute antarmuka.
 */
function columnsFor(tab: Tab): Column<ServiceClaim>[] {
  return columnsWithAction<ServiceClaim, TabColumn>(tab.kolom, cellText, (row) => (
    <DetailLink id={row.id} />
  ))
}

/**
 * Tautan ke rincian satu klaim.
 *
 * Tautan, bukan tombol: tujuannya sebuah alamat, sehingga klik-tengah dan "buka di tab baru"
 * bekerja seperti yang diharapkan pengguna. Petugas yang membandingkan beberapa klaim
 * sekaligus memang membukanya berdampingan.
 */
function DetailLink({ id }: Readonly<{ id: string }>) {
  if (id === '') return <span className="text-sm text-slate-400">—</span>

  return (
    <Link
      to={`/inbox-service-center/${encodeURIComponent(id)}`}
      className={[
        'inline-flex items-center rounded-kontrol border border-slate-300 px-3 py-1.5',
        'text-sm font-medium text-slate-700',
        'transition-colors duration-150 ease-halus hover:bg-slate-50 hover:text-slate-900',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
      ].join(' ')}
    >
      Lihat Rincian
    </Link>
  )
}

/**
 * cellText menyusun teks satu sel.
 *
 * Tanggal diformat ke `1 Juni 2026`; sisanya ditampilkan apa adanya. Nilai kosong menjadi
 * tanda pisah — bukan sel kosong yang tidak dapat dibedakan dari kolom yang gagal dimuat.
 */
function cellText(row: ServiceClaim, column: TabColumn): string {
  const value = row[column.kunci]

  if (value == null || value === '') return '—'

  const text = String(value)
  return isISODate(text) ? formatDate(text) : text
}

/**
 * emptyMessageFor menjelaskan antrean kosong menurut sebabnya.
 *
 * "Tidak ada data" tidak cukup: antrean yang kosong karena kata kunci berbeda jauh dari
 * antrean yang memang tidak punya pekerjaan, dan tindakannya pun berbeda.
 */
function emptyMessageFor(tab: Tab, filter: FilterForm): string {
  if (filter.cari.trim() !== '') {
    return (
      'Tidak ada baris yang cocok dengan pencarian ini. Kotak cari menelusuri No Polis, ' +
      'Nasabah, dan IMEI — bukan ID atau No Klaim.'
    )
  }
  return `Tidak ada klaim milik Anda pada antrean ${tab.nama}.`
}

