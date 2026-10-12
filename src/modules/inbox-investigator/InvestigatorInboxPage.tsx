import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type InvestigatorTask } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { ExportPanel } from './ExportPanel'
import { InvestigationDialog } from './InvestigationDialog'
import { useInvestigatorInbox } from './api'

/**
 * Ukuran halaman.
 *
 * **50**, dibaca langsung dari `pyPageSize` pada grid layar lama
 * (`Section/InputInvestigator_Section-Section.xml`), yang juga ber-`pyPageMode = Numeric` —
 * nomor halaman, bukan tombol "muat lebih banyak".
 *
 * Ia berbeda dari 15 dan 20 yang dipakai layar master, dan perbedaannya memang ada di sistem
 * lama: ukuran halaman di sana ditetapkan per grid, bukan per aplikasi.
 */
const PAGE_SIZE = 50

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat pemuatan antrean menjadi pesan yang dapat ditindaklanjuti. */
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
            'Antrean pekerjaan dimiliki masing-masing entitas. Pilih portal entitas di ' +
            'bagian atas halaman ini lebih dulu.',
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
          title: 'Antrean investigator tidak dapat dimuat',
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
 * formatDate menuliskan waktu UTC dari server sebagai tanggal WIB.
 *
 * Konversi zona waktu terjadi DI SINI, di tempat waktu ditampilkan — satu-satunya tempat
 * yang boleh melakukannya (`08-TECHNICAL-STRATEGY.md` §4.4). Tidak ada penambahan 7 jam
 * manual di mana pun; `Asia/Jakarta` disebut namanya supaya hasilnya tidak bergantung pada
 * zona waktu mesin pengguna.
 *
 * Bentuknya disamakan dengan layar Master Penolakan Klaim supaya kedua layar tidak terasa
 * dirakit dari dua aplikasi berbeda.
 */
function formatDate(value: string | null): string {
  if (!value) return '—'
  const moment = new Date(value)
  if (Number.isNaN(moment.getTime())) return '—'
  return moment.toLocaleDateString('id-ID', {
    timeZone: 'Asia/Jakarta',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  })
}


/**
 * Layar Inbox Investigator.
 *
 * Pengganti `Harness/InboxInvestigator_Harness-Harness.xml` (MENU_ID 48).
 *
 * # Apa yang ditampilkan layar ini
 *
 * **Daftar pekerjaan yang menunggu Investigator** — klaim yang penugasannya berada di
 * workbasket `InvestigatorPNC` dan belum selesai dikerjakan.
 *
 * Ia INBOX menurut keempat ciri `D-79`: barisnya pekerjaan, baris hilang setelah selesai
 * dikerjakan, "hanya yang jadi tanggung jawab saya" adalah aturan kewenangan, dan barisnya
 * punya tenggat. Itu yang membedakannya dari layar master dan dari View History Claim.
 *
 * # Layar ini MENYIMPAN satu hal
 *
 * Lingkup awalnya "layar inbox saja" (2026-09-23), dan formulir kerja Investigator
 * ditambahkan 2026-10-05 setelah Work Owner menunjukkan bahwa Nomor Case di Pega
 * menjalankan Flow Action `InputInvestigator` — bukan membuka tampilan.
 *
 * Yang MASIH belum ada: mengambil pekerjaan dari antrean (`openAssignment`). Itu bagian
 * dari modul Penugasan, bukan dari layar ini.
 *
 * # Susunan layar, dan asal setiap bagiannya
 *
 *	Judul + tombol Refresh
 *	Spanduk terpotong    hanya bila server menyatakan antreannya terpotong
 *	Grid 9 kolom         paginasi 50 baris, mengikuti pyPageSize grid lamanya
 *	Panel ekspor         "Dari", "Sampai", "Pilih Investigation", dan tombolnya
 *	Keterangan kaki      tiga keterbatasan yang tidak terlihat dari layar
 *
 * # Satu hal dari layar lama yang TIDAK dibawa
 *
 * **Grid kedua.** Section lama memuat DUA grid dengan kolom dan parameter identik — sisa
 * Save-As dari inbox Compliance; yang kedua bahkan mengeja "Nama Bisinis".
 *
 * # Dua selisih tata letak yang disengaja
 *
 *  1. **Panel ekspor ada di BAWAH daftar**, sementara di Pega ketiga kendalinya berada di
 *     kepala layar berdampingan dengan penyaring grid. Alasannya di ExportPanel: keduanya
 *     menyaring hal yang berbeda, dan kedekatan posisi membuat pengguna menduga sebaliknya.
 *
 *  2. **Panel ekspor dan formulir investigasi berbagi satu halaman.** Di Pega keduanya
 *     terpisah: kendali ekspor di kepala layar, formulir sebagai modal. Di sini modalnya
 *     tetap modal; yang berpindah hanya panel ekspornya.
 *
 * # Yang MENYIMPAN di layar ini
 *
 * Satu: formulir kerja Investigator, dibuka dengan menekan Nomor Case. Menyimpannya
 * memindahkan klaim ke Analyst dan menghilangkan barisnya dari antrean. Selebihnya —
 * daftar, pencarian, ekspor — hanya membaca.
 */
export function InvestigatorInboxPage() {
  const portal = useSelectedPortal((state) => state.alias)

  /*
    Pekerjaan yang formulirnya sedang dibuka, atau null bila dialognya tertutup.

    Barisnya DISIMPAN UTUH, bukan hanya referensinya: dialog menampilkan Nomor Case, Nama
    Tertanggung, dan Nama Peserta sebagai keterangan baca-saja — ketiganya sudah ada di
    tangan, dan mengambilnya ulang dari server hanya menambah satu permintaan untuk data
    yang tidak berubah.
  */
  const [workingOn, setWorkingOn] = useState<InvestigatorTask | null>(null)

  const inbox = useInvestigatorInbox()
  const rows = inbox.data?.tugas ?? []

  /*
    SEMBILAN kolom, pada urutan grid layar lama — dihitung dari pemasangan caption-ke-sel
    pada `Section/InputInvestigator_Section-Section.xml`, satu lawan satu.

    Kolom yang ADA di Report Definition tetapi TIDAK digambar grid — SobName, GroupPanel,
    UserTeknis, PNCStatus, StatusClaim, isComplianceTransfer, TanggalBuatCompliance —
    sengaja tidak ditampilkan. Menampilkan kolom yang tidak pernah dilihat pengguna berarti
    mengarang kegunaan, dan pada grid sembilan kolom ia langsung berbiaya keterbacaan.
  */
  const columns: Column<InvestigatorTask>[] = [
    {
      /*
        Nomor case adalah TAUTAN — sel-nya di Pega ber-`pyFormat = pxLink` atas `.pyID`.

        # Apa yang dibukanya

        Flow Action `InputInvestigator`, yaitu FORMULIR KERJA yang menyimpan:

          Flow Action/InputInvestigator-FA.xml:128   pra-proses  PresetInvestigation
          Flow Action/InputInvestigator-FA.xml:49    formulir    InputClaimInvestigasiDetail
          Flow Action/InputInvestigator-FA.xml:144   simpan      SetStatusInvestigator_Act

        Ia MODAL di sana (80 × 82), dan modal pula di sini — sehingga tidak ada rute baru
        dan daftar di belakangnya tetap pada tempatnya.

        # Ia BUKAN halaman rincian klaim

        Sempat ditautkan ke sana dan DICABUT. Dua sebabnya, masing-masing cukup sendirian:
        halaman itu menampilkan sedangkan yang ini mengerjakan, dan halaman itu gagal untuk
        klaim warisan — `T_CLAIM_OBJECTLIST.URUTAN` tidak pernah diisi Pega.
      */
      key: 'nomor_case',
      title: 'Nomor Case',
      width: '11rem',
      value: (row) => row.nomor_case,
      render: (row) =>
        row.nomor_case ? (
          <button
            type="button"
            className="font-medium text-blue-700 underline underline-offset-2 hover:text-blue-900"
            title={`Input Investigator untuk ${row.nomor_case}`}
            onClick={() => setWorkingOn(row)}
          >
            {row.nomor_case}
          </button>
        ) : (
          <span className="text-slate-400">—</span>
        ),
    },
    {
      key: 'nomor_polis',
      title: 'No Polis',
      width: '12rem',
      value: (row) => row.nomor_polis,
    },
    {
      key: 'nama_tertanggung',
      title: 'Nama Tertanggung',
      value: (row) => row.nama_tertanggung,
    },
    {
      /*
        Captionnya "Nama Peserta", bukan "Nama Objek", dan itu dipakai apa adanya (`D-13`).
        Isinya objek pertanggungan PERTAMA; klaim berobjek banyak tampil seolah berobjek
        satu, persis sistem lama (`P-5`).
      */
      key: 'nama_peserta',
      title: 'Nama Peserta',
      value: (row) => row.nama_peserta,
      render: (row) =>
        row.nama_peserta ? row.nama_peserta : <span className="text-slate-400">—</span>,
    },
    {
      key: 'nama_bisnis',
      title: 'Nama Bisnis',
      width: '12rem',
      value: (row) => row.nama_bisnis,
    },
    {
      key: 'nama_cabang',
      title: 'Nama Cabang',
      width: '12rem',
      value: (row) => row.nama_cabang,
    },
    {
      /*
        `pyOrigUserID` — PEMBUAT kasus, bukan pemegangnya. Pekerjaan di workbasket memang
        belum bertuan (`D-26`); itulah yang membuatnya antrean bersama.

        Kosong berarti kasusnya dibuat proses terjadwal, bukan orang (`D-57`) — dan itu
        ditulis apa adanya alih-alih dibiarkan sebagai sel kosong yang terbaca seperti data
        hilang.
      */
      key: 'nama_admin',
      title: 'Nama Admin',
      width: '11rem',
      value: (row) => row.nama_admin,
      render: (row) =>
        row.nama_admin ? (
          row.nama_admin
        ) : (
          <span className="text-slate-500 italic">proses terjadwal</span>
        ),
    },
    {
      key: 'tanggal_pendaftaran',
      title: 'Tanggal Pendaftaran',
      width: '10rem',
      value: (row) => formatDate(row.tanggal_pendaftaran),
    },
    {
      /*
        Kolom KESEMBILAN. Judulnya "Lama Masuk Inbox" mengikuti caption layar lama (`D-13`),
        tetapi ISINYA tanggal survei — sel Pega-nya terikat
        `.ClaimData.SurveyResults(1).SurveyDate`.

        Ketidakcocokan judul dan isi itu ADA di sistem lama dan direplikasi apa adanya
        (`P-5`). Ia sisa Save-As dari inbox Compliance, tempat kolom bernama sama memang
        berisi lama menunggu. Keterangan di kaki halaman yang menjelaskannya kepada pengguna.
      */
      key: 'tanggal_survey',
      title: 'Lama Masuk Inbox',
      width: '10rem',
      value: (row) => formatDate(row.tanggal_survey),
    },
  ]

  return (
    <main className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya "Inbox Investigator" mengikuti MENU_DESC pada
              Database/m_menu_aplikasi_pnc.csv (MENU_ID 48), dan caption section lamanya
              berbunyi sama (`D-13`). */}
          <h1 className="text-xl font-semibold text-slate-900">Inbox Investigator</h1>
          <p className="text-sm text-slate-600">
            Daftar pekerjaan yang menunggu diselidiki, dari antrean bersama Investigator.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => { inbox.refetch() }} disabled={inbox.isFetching}>
            {inbox.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "antrean siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20).

          Pada layar ini ia berarti lebih dari sekadar kerapian: barisnya memuat nama
          tertanggung dan nama peserta klaim satu badan hukum. */}
      <p className="mt-3 text-xs text-slate-500">
        Antrean ini milik entitas yang sedang dibuka.
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{inbox.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {/* Pemotongan DINYATAKAN, tidak dibiarkan senyap seperti pyMaxRecords=500 pada sistem
          lama. Batas yang diketahui adalah batas; batas yang senyap adalah data yang hilang. */}
      {inbox.data?.terpotong && (
        <p className="mt-4 rounded-kartu border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900">
          Antrean ini <span className="font-medium">lebih panjang</span> daripada yang dapat
          ditampilkan sekaligus. Yang tampil {inbox.data.batas_baris} pekerjaan pertama;
          sisanya belum terlihat. Pakai kotak pencarian untuk mempersempit daftar.
        </p>
      )}

      {/* Panel ekspor berada DI ATAS daftar, mengikuti letaknya di layar lama. */}
      {portal !== null && <ExportPanel />}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Antrean pekerjaan dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : inbox.isPending ? (
          <p className="text-sm text-slate-500">Memuat antrean investigator…</p>
        ) : inbox.isError ? (
          (() => {
            const message = loadMessage(inbox.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <DataTable
            columns={columns}
            rows={rows}
            /*
              Kunci barisnya `referensi` — pzInsKey, yang unik per pekerjaan. Memakai nomor
              case tidak salah hari ini, tetapi tidak ada DDL yang membuktikan keunikannya
              (`R-08`), dan kunci baris yang kembar membuat React menganggap dua baris
              sebagai satu.
            */
            rowKey={(row) => row.referensi}
            description="Sumber: antrean workbasket InvestigatorPNC"
            searchLabel="Cari nomor case, polis, tertanggung, peserta, bisnis, cabang, atau admin"
            pageSize={PAGE_SIZE}
            emptyMessage="Tidak ada pekerjaan yang menunggu di antrean investigator."
          />
        )}
      </section>

      {/* Formulir kerja Investigator — padanan Flow Action `InputInvestigator`, yang di
          Pega pun terbuka sebagai modal (80 × 82), bukan sebagai halaman. */}
      {workingOn !== null && (
        <InvestigationDialog task={workingOn} onClose={() => setWorkingOn(null)} />
      )}
    </main>
  )
}
