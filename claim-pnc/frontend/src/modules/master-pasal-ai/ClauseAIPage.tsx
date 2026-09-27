import { useState, type FormEvent } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, Paginator, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'

import { useClauseAIList } from './api'
import type { ClauseAI } from './types'

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data ini dimiliki masing-masing entitas. Pilih portal entitas di bagian atas ' +
            'halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar Pasal AI tidak dapat dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
    tone: 'gangguan',
  }
}

/**
 * Layar Master Pasal AI.
 *
 * Pengganti `Harness/DetailMasterPasalAI-harness.xml` (MENU_ID 36).
 *
 * # Apa yang ditampilkan layar ini
 *
 * **Daftar wording polis** yang dipakai penilaian AI atas sebuah klaim — nomor pasalnya,
 * ayatnya, dan kejadian yang dicakupnya.
 *
 * # Ia BACA-SAJA, dan itu terbaca dari layar lamanya
 *
 * `Section/DetailMasterPasalAI_sect.xml` ber-`pyEditingMode = readOnly`, dan satu-satunya
 * tombol di sana adalah **Cari** dan **Refresh**. Tidak ada Tambah, Simpan, Ubah, maupun
 * Hapus — dipastikan dari `pyDeleteActivity` dan `pyAppendActivity` yang seluruhnya kosong.
 *
 * Itu membedakannya dari **Master Pasal Kerugian** (MENU_ID 27), yang justru menjadi
 * asal-usulnya: section ini Save-As dari `BrowsePasalDeatailMaster`, lalu dipangkas menjadi
 * layar pencarian.
 *
 * # Tabelnya `POOLDATA.MST_PASAL_AI`
 *
 * Ketiga kolomnya `WP_PASAL`, `WP_AYAT`, dan `WP_KEJADIAN`. Nama itu **tidak terbaca dari
 * layar Pega sama sekali**: propertinya di sana bernama `.City`, `.CityID`, dan `.District`
 * — sisa Save-As berlapis dari layar surveyor tahun 2017. Ia baru terbaca dari activity dan
 * kedua Connect-SQL-nya.
 *
 * # Paginasinya di SERVER — satu-satunya layar master yang begitu
 *
 * Bukan pilihan kami. Grid Pega-nya ber-`pyPageMode = None`, dan jendelanya dihitung activity
 * lewat `FirstRow`/`LastRow` dengan `PageSize = 25`. Seluruh layar master lain di aplikasi
 * ini memaginasi di peramban karena layar lamanya pun begitu.
 *
 * Akibatnya di sini: **pencarian dan paginasi menembak server**, dan pengurutan kolom
 * dimatikan — mengurutkan satu halaman dari sepuluh bukan pengurutan. Layar lamanya pun tidak
 * punya pengurutan sama sekali (`pySortType = NONE` pada ketiga kolom).
 *
 * # Mencari dengan menekan tombol, bukan sambil mengetik
 *
 * Kotak "Cari" di Pega **tidak mencari saat diketik** — action set pada even `change`-nya
 * kosong. Yang menjalankan pencarian adalah tombol **Cari**.
 *
 * Perilaku itu ditiru apa adanya (`D-13`), dan di sini ia kebetulan juga pilihan yang benar:
 * setiap ketukan tombol berarti satu permintaan ke basis data, dan mencari sambil mengetik
 * akan menembaknya sekali per huruf.
 */
export function ClauseAIPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Dua keadaan yang sengaja dipisah: apa yang sedang DIKETIK, dan apa yang sedang DICARI.
  // Menyatukannya berarti mencari sambil mengetik — persis yang tidak dilakukan layar lama.
  const [draft, setDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)

  const list = useClauseAIList(keyword, page)

  const rows = list.data?.pasal_ai ?? []
  const pagination = list.data?.paginasi

  function submitSearch(event: FormEvent) {
    event.preventDefault()
    // Kembali ke halaman pertama: hasil pencarian baru tidak punya halaman ketujuh.
    setPage(1)
    setKeyword(draft.trim())
  }

  function refresh() {
    // Meniru tombol Refresh Pega apa adanya: ia MENGOSONGKAN kotak cari lebih dulu
    // (`setValue TempSearch.Country := ""`), baru memuat ulang dari halaman pertama.
    setDraft('')
    setKeyword('')
    setPage(1)
    void list.refetch()
  }

  /*
    Tiga kolom, satu-lawan-satu dengan grid Pega — No Pasal, Ayat, Kejadian.

    Ketiganya `noSort`: grid lamanya ber-`pySortType = NONE` pada ketiga kolom, dan
    barisnya di sini hanya satu halaman dari server sehingga mengurutkannya di peramban
    akan mengurutkan halaman, bukan daftar.

    Lebarnya mengikuti perbandingan lebar sel aslinya — 48 px, 47 px, dan 283 px.
  */
  const columns: Column<ClauseAI>[] = [
    {
      key: 'no_pasal',
      title: 'No Pasal',
      width: '7rem',
      noSort: true,
      value: (row) => row.no_pasal,
      render: (row) => <span className="font-medium text-slate-800">{row.no_pasal}</span>,
    },
    {
      key: 'ayat',
      title: 'Ayat',
      width: '7rem',
      noSort: true,
      value: (row) => row.ayat,
    },
    {
      key: 'kejadian',
      title: 'Kejadian',
      noSort: true,
      value: (row) => row.kejadian,
      // Teks panjang dibiarkan membungkus, bukan dipotong: ia isi ketentuan polis, dan
      // potongannya tidak dapat dipakai menilai apa pun.
      render: (row) => <span className="whitespace-pre-line">{row.kejadian}</span>,
    },
  ]

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya dibaca dari `pyCaption Detail Pasal AI` pada
              Section/GridDetailMasterPasalAI (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Detail Pasal AI</h1>
          <p className="text-sm text-slate-600">
            Daftar wording polis yang dipakai penilaian AI atas klaim pada entitas ini.
          </p>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        Portal entitas:{' '}
        <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
      </p>

      {/* Formulir pencarian terpisah dari tabel, meniru layar lama: satu kotak, dua tombol.

          Ia <form> supaya menekan Enter di dalam kotak ikut menjalankan pencarian — di layar
          lama pun tombol Cari adalah tombol utamanya (`pyStyleName = Strong`). */}
      <form
        onSubmit={submitSearch}
        className="mt-5 flex flex-wrap items-end gap-3 rounded-kartu border border-slate-200 bg-white p-4"
      >
        <div className="min-w-64 flex-1">
          <Field
            id="cari-pasal-ai"
            label="Cari"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="Nomor pasal, ayat, atau kata pada kejadian"
            maxLength={200}
            hint="Satu kata kunci dicocokkan ke ketiga kolom sekaligus."
          />
        </div>
        <div className="flex items-center gap-2 pb-1">
          {/* Caption kedua tombol dibaca dari `pyLabel` pada section aslinya (D-13). */}
          <Button tone="utama" type="submit" disabled={list.isFetching}>
            Cari
          </Button>
          <Button tone="kedua" onClick={refresh} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </form>

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data ini dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar Pasal AI…</p>
        ) : list.isError ? (
          (() => {
            const message = loadMessage(list.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <div className="overflow-hidden rounded-kartu border border-slate-200 bg-white">
            <DataTable
              columns={columns}
              rows={rows}
              // Kunci barisnya WP_ID — kunci sungguhan, bukan gabungan No Pasal dan Ayat.
              // Tidak ada constraint yang diketahui melarang dua baris berpasangan sama
              // (`R-08`), dan kunci yang dapat kembar membuat React menggambar ulang baris
              // yang salah.
              rowKey={(row) => row.id}
              hideSearch
              emptyMessage={
                keyword === ''
                  ? 'Belum ada Pasal AI pada entitas ini.'
                  : `Tidak ada Pasal AI yang cocok dengan "${keyword}".`
              }
            />

            {/* Paginator sisi server, memakai komponen yang sama dengan tabel lain supaya
                tidak ada dua gaya paginasi hidup berdampingan.

                Barisnya dihitung dari jendela yang dikembalikan server, bukan dari panjang
                senarai — panjang senarai hanya menceritakan halaman yang sedang terbuka. */}
            {pagination !== undefined && rows.length > 0 && (
              <Paginator
                firstRow={(pagination.halaman - 1) * pagination.ukuran_halaman + 1}
                lastRow={(pagination.halaman - 1) * pagination.ukuran_halaman + rows.length}
                totalRows={pagination.jumlah_baris}
                currentPage={pagination.halaman}
                totalPages={pagination.jumlah_halaman}
                onPick={setPage}
              />
            )}
          </div>
        )}
      </section>

      {/* Dua keterbatasan yang nyata, dinyatakan di kaki halaman alih-alih ditemukan
          pengguna sendiri. */}
      <footer className="mt-6 space-y-2 border-t border-slate-200 pt-4 text-xs text-slate-500">
        <p>
          <strong className="text-slate-700">Layar ini hanya menampilkan.</strong>{' '}
          Wording polis di sini tidak dapat ditambah, diubah, maupun dihapus — layar lamanya
          pun tidak punya jalurnya. Perubahannya dilakukan di tempat lain.
        </p>
        <p>
          <strong className="text-slate-700">Pencarian dan halaman dikerjakan server.</strong>{' '}
          Kata kunci dicocokkan ke seluruh baris, bukan hanya yang sedang tampil — dan karena
          itu kolomnya tidak dapat diurutkan dari layar.
        </p>
      </footer>
    </main>
  )
}
