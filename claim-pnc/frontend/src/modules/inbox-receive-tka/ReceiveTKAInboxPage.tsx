import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, ReceiveTKAErrorCode, type ReceiveTKATask } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useCompleteReceiveTKA, useReceiveTKAInbox } from './api'

/**
 * Ukuran halaman.
 *
 * **50**, dibaca langsung dari `pyPageSize` pada grid layar lama
 * (`Section/InboxTKA_Section-Section.xml`), yang juga memakai `pyGridPaginator`.
 *
 * Ia sama dengan Inbox Investigator dan berbeda dari 15 dan 20 yang dipakai layar master.
 * Perbedaan itu memang ada di sistem lama: ukuran halaman ditetapkan per grid, bukan per
 * aplikasi.
 */
const PAGE_SIZE = 50

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
            'Daftar klaim TKA dimiliki masing-masing entitas. Pilih portal entitas di ' +
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
          title: 'Daftar Inbox Receive TKA tidak dapat dimuat',
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
 * Mengubah galat penyimpanan menjadi pesan yang menyebut TINDAKAN yang perlu diambil.
 *
 * Ketiga kode galat modul ini dibedakan justru karena tindakannya berbeda — menyegarkan
 * daftar menolong pada yang pertama dan tidak menolong sama sekali pada dua sisanya.
 */
function saveMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Tanggal belum tersimpan',
      description:
        'Server Claim PNC tidak dapat dihubungi. Tanggal yang Anda isi masih ada di ' +
        'layar; coba tekan Submit lagi setelah koneksi pulih.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ReceiveTKAErrorCode.taskNotFound:
        return {
          title: 'Klaim ini sudah tidak ada di daftar',
          description:
            'Kemungkinan tanggalnya sudah diisi petugas lain. Tekan Refresh untuk ' +
            'melihat daftar terbaru.',
          tone: 'penolakan',
        }
      case ReceiveTKAErrorCode.claimMissing:
        return {
          title: 'Klaim tidak ditemukan pada data klaim utama',
          description:
            'Tanggalnya tidak dapat disimpan, dan menyegarkan daftar tidak akan ' +
            'menolong — barisnya akan muncul lagi. Laporkan nomor klaim ini ke ' +
            'administrator Claim PNC.',
          tone: 'penolakan',
        }
      case ReceiveTKAErrorCode.claimAmbiguous:
        return {
          title: 'Nomor klaim ini menunjuk lebih dari satu data',
          description:
            'Tidak jelas mana yang harus diperbarui, sehingga tidak ada yang diubah. ' +
            'Laporkan nomor klaim ini ke administrator Claim PNC.',
          tone: 'penolakan',
        }
      case ReceiveTKAErrorCode.dateRequired:
        return {
          title: 'Tanggal belum diisi',
          description: error.message,
          tone: 'penolakan',
        }
      default:
        return {
          title: 'Tanggal belum tersimpan',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Tanggal belum tersimpan',
    description: 'Terjadi kesalahan pada sistem. Bila berulang, hubungi administrator.',
    tone: 'gangguan',
  }
}

/**
 * formatDate menuliskan tanggal ISO dari server sebagai tanggal yang dibaca pengguna.
 *
 * # Ia TIDAK memakai `new Date()`, dan itu disengaja
 *
 * Server mengirim tanggal murni (`YYYY-MM-DD`) tanpa jam, karena kolomnya memang tanggal
 * kalender. Menguraikannya dengan `new Date('2026-09-14')` akan memperlakukannya sebagai
 * tengah malam UTC, dan menampilkannya dalam WIB menghasilkan **15 September** — kelas cacat
 * yang `R-12` catat.
 *
 * Yang dilakukan di sini adalah menyusun ulang teksnya, tanpa pernah menyentuh zona waktu.
 */
function formatDate(value: string | null): string {
  if (!value) return '—'
  const [year, month, day] = value.split('-')
  if (!year || !month || !day) return value
  const monthName = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'Mei',
    'Jun',
    'Jul',
    'Agu',
    'Sep',
    'Okt',
    'Nov',
    'Des',
  ][Number(month) - 1]
  return monthName ? `${day} ${monthName} ${year}` : value
}

/**
 * formatAging menuliskan lama menunggu sejak tanggal registrasi.
 *
 * # Ia meniru tampilan Pega, bukan menyimpan angkanya
 *
 * Kolom "Aging" pada layar lama menampilkan `.ClaimData.RegisterDate` yang dirender sebagai
 * waktu relatif — "2 years 6 months ago", "3 years ago". Yang tersimpan adalah tanggalnya;
 * kalimatnya disusun saat ditampilkan.
 *
 * Perhitungannya karena itu ada DI SINI, bukan di server: "berapa lama menunggu" bergantung
 * pada kapan ia dibaca, dan nilai yang dihitung server akan membeku pada saat permintaan.
 *
 * # Kalimatnya bahasa Indonesia, dan itu pilihan yang disengaja
 *
 * Pega menuliskannya dalam bahasa Inggris karena begitulah bawaan kontrolnya — bukan karena
 * ada teks yang ditulis seseorang. Ia tidak muncul sebagai kalimat di rule mana pun,
 * sehingga tidak ada teks Pega yang sedang kita simpangi (`D-13`). Yang tetap ditiru adalah
 * **bentuk keterangannya**: satuan terbesar lebih dulu, bulan hanya disebut bila masih
 * dalam hitungan tahun.
 */
function formatAging(value: string | null): string {
  if (!value) return '—'
  const [year, month, day] = value.split('-').map(Number)
  if (!year || !month || !day) return '—'

  const now = new Date()
  let months = (now.getFullYear() - year) * 12 + (now.getMonth() + 1 - month)
  if (now.getDate() < day) months -= 1
  if (months < 0) return 'baru saja'

  const years = Math.floor(months / 12)
  const restMonths = months % 12

  if (years === 0) {
    return months === 0 ? 'bulan ini' : `${months} bulan lalu`
  }
  return restMonths === 0 ? `${years} tahun lalu` : `${years} tahun ${restMonths} bulan lalu`
}

/**
 * Layar Inbox Receive TKA.
 *
 * Pengganti `Harness/InboxTKA_Harness-Harness.xml` (MENU_ID 49).
 *
 * # Apa yang ditampilkan dan dikerjakan layar ini
 *
 * **Klaim TKA yang tanggal kelengkapan dokumennya belum diisi.** Pengguna mengisi tanggal
 * itu langsung di dalam tabel lalu menekan Submit pada barisnya; begitu tersimpan, barisnya
 * HILANG dari daftar.
 *
 * Ia INBOX menurut keempat ciri `D-79` — dan berbeda dari Inbox Investigator yang baca-saja,
 * di sini ciri kedua (baris hilang setelah dikerjakan) benar-benar terjadi lewat layar ini
 * sendiri.
 *
 * # Submit berada PER BARIS, bukan satu tombol untuk seluruh tabel
 *
 * Itu mengikuti sistem lama. Grid pada `Section/InboxTKA_Section-Section.xml` menggambar
 * tombol Submit yang mengirim parameter milik BARISNYA — `Inskey`, `ClaimNo`, `NoPolis`,
 * `QQName`, `Insured`, `DOL`, dan `Tanggal` — dan `SubmitTanggalLengkapTKA` menerima
 * ketujuhnya dalam bentuk tunggal, membuka satu klaim, menyimpan satu tanggal.
 *
 * # Ketiga penyaring layar lama TIDAK muncul sebagai kendali
 *
 * Report Definition-nya menyaring `TKA = "1"`, `TanggalDokLengkap IS NULL`, dan status
 * pekerjaan — ketiganya nilai TETAP, bukan pilihan pengguna. Tidak ada satu pun kendali
 * penyaring di harness lamanya, dan tidak ada yang ditambahkan di sini.
 *
 * # Satu penyaring yang HILANG dari layar ini dibanding aslinya, dan sebabnya
 *
 * Penyaring status pekerjaan (`pyStatusWork != "Resolved-Completed"`) diterapkan lewat
 * gabungan ke data klaim, karena `POOLDATA.T_CLAIM_TKA_H` tidak punya kolom kunci apa pun.
 * Baris yang klaimnya TIDAK ditemukan tetap ditampilkan — disembunyikan berarti pekerjaan
 * hilang tanpa jejak — tetapi Submit atasnya ditolak, dan layar menandainya lebih dulu.
 */
export function ReceiveTKAInboxPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const inbox = useReceiveTKAInbox()
  const complete = useCompleteReceiveTKA()
  const rows = inbox.data?.tugas ?? []

  /*
    Tanggal yang sedang diketik, disimpan per nomor klaim.

    Ia state LAYAR, bukan state server: sampai Submit ditekan, tanggal itu belum ada di mana
    pun. Menyimpannya per baris — bukan satu nilai bersama — karena grid lamanya pun
    menggambar isian pada SETIAP baris, dan petugas dapat mengisi beberapa baris sebelum
    menekan Submit yang pertama.
  */
  const [draft, setDraft] = useState<Record<string, string>>({})

  /* Baris yang sedang dikirim, supaya hanya tombolnya sendiri yang berubah menjadi
     "Menyimpan…" — bukan seluruh tombol di tabel. */
  const [sending, setSending] = useState<string | null>(null)

  /* Hasil penyimpanan terakhir, ditampilkan di atas tabel. */
  const [saved, setSaved] = useState<{ claim: string; notified: boolean; attempted: boolean } | null>(
    null,
  )
  const [failure, setFailure] = useState<MessageContent | null>(null)

  function submitRow(row: ReceiveTKATask) {
    const value = (draft[row.nomor_klaim] ?? '').trim()
    if (value === '') return

    setSending(row.nomor_klaim)
    setFailure(null)
    setSaved(null)

    complete.mutate(
      { nomor_klaim: row.nomor_klaim, tanggal_dokumen_lengkap: value },
      {
        onSuccess: (result) => {
          setSaved({
            claim: result.nomor_klaim,
            notified: result.pemberitahuan_terkirim,
            attempted: result.pemberitahuan_dicoba,
          })
          // Draf barisnya dibuang: barisnya sudah hilang dari daftar, dan menyisakan
          // tanggalnya akan membuatnya muncul kembali bila nomor klaim yang sama kelak
          // masuk lagi ke inbox.
          setDraft((previous) => {
            const next = { ...previous }
            delete next[row.nomor_klaim]
            return next
          })
        },
        onError: (error) => setFailure(saveMessage(error)),
        onSettled: () => setSending(null),
      },
    )
  }

  /*
    TUJUH kolom pada urutan grid layar lama, ditambah satu kolom aksi.

    Urutannya dihitung dari pemasangan caption-ke-sel pada
    `Section/InboxTKA_Section-Section.xml`, satu lawan satu menurut urutan kemunculannya di
    dalam berkas.

    Kolom ketiga dan keempat sempat diragukan karena Report Definition melabelinya TERBALIK.
    Yang dipakai adalah Section, dan buktinya dikuatkan sumber ketiga yang berdiri sendiri:
    badan surel `NotificationKelengkapanTKA` memberi label "Nama Tertanggung" pada nilai yang
    diisi dari `Param.QQName`.
  */
  const columns: Column<ReceiveTKATask>[] = [
    {
      key: 'nomor_klaim',
      title: 'Nomor Klaim',
      width: '11rem',
      value: (row) => row.nomor_klaim,
      render: (row) => (
        <span className="font-medium text-slate-800">{row.nomor_klaim}</span>
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
      key: 'nama_peserta',
      title: 'Nama Peserta',
      value: (row) => row.nama_peserta,
      render: (row) =>
        row.nama_peserta ? row.nama_peserta : <span className="text-slate-400">—</span>,
    },
    {
      /* Judulnya "Date Of Loss" mengikuti caption layar lama apa adanya (`D-13`), meski
         nama field kontraknya `tanggal_kejadian` mengikuti `CONTEXT.md` (`D-80`). */
      key: 'tanggal_kejadian',
      title: 'Date Of Loss',
      width: '9rem',
      value: (row) => row.tanggal_kejadian ?? '',
      render: (row) => formatDate(row.tanggal_kejadian),
    },
    {
      /*
        Aging. Judulnya mengikuti caption layar lama; ISINYA tanggal registrasi yang
        dirender sebagai lama menunggu — persis seperti Pega, yang menuliskan
        "2 years 6 months ago" untuk klaim ber-REGISTERDATE_1 20240319.

        Pengurutannya memakai TANGGALNYA (ISO), bukan kalimatnya: teks "3 tahun lalu" akan
        jatuh sebelum "5 bulan lalu" secara abjad, padahal yang pertama jauh lebih lama.
        Tanggal ISO berurut secara abjad sama dengan urutan kronologisnya.
      */
      key: 'tanggal_registrasi',
      title: 'Aging',
      width: '11rem',
      value: (row) => row.tanggal_registrasi ?? '',
      render: (row) =>
        row.tanggal_registrasi ? (
          <span title={`Terdaftar ${formatDate(row.tanggal_registrasi)}`}>
            {formatAging(row.tanggal_registrasi)}
          </span>
        ) : (
          <span className="text-slate-400">—</span>
        ),
    },
    {
      /*
        Kolom KETUJUH adalah ISIAN, bukan tampilan.

        Di grid lamanya pun begitu: selnya ber-`pyEditOptions = Editable` dan
        `pyReadOnly = false`. Nilainya selalu kosong saat dimuat — daftar ini menurut
        definisinya hanya memuat baris yang tanggalnya belum diisi.
      */
      key: 'tanggal_dokumen_lengkap',
      title: 'Tanggal Dokumen Lengkap',
      width: '12rem',
      noSort: true,
      value: (row) => draft[row.nomor_klaim] ?? '',
      render: (row) => (
        <input
          type="date"
          aria-label={`Tanggal dokumen lengkap untuk klaim ${row.nomor_klaim}`}
          className="w-full rounded-kontrol border border-slate-300 bg-white px-2 py-1.5 text-sm text-slate-800 shadow-lembut focus:border-blue-500 focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/25 disabled:cursor-not-allowed disabled:bg-slate-100"
          value={draft[row.nomor_klaim] ?? ''}
          disabled={!row.klaim_tersedia}
          onChange={(event) =>
            setDraft((previous) => ({ ...previous, [row.nomor_klaim]: event.target.value }))
          }
        />
      ),
    },
    {
      /*
        Kolom aksi. Tombol Submit ada PER BARIS karena activity lamanya memang menerima satu
        baris; lihat banner komponen.

        Tombolnya mati pada dua keadaan, dan keduanya disebut alasannya lewat `title`:
        tanggalnya belum diisi, dan barisnya tidak punya klaim pasangan. Yang kedua tidak
        dibiarkan dicoba lalu ditolak — penolakannya sudah pasti, dan membiarkan pengguna
        menekannya hanya membuang waktunya.
      */
      key: 'aksi',
      title: '',
      width: '8rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => {
        const orphan = !row.klaim_tersedia
        const empty = (draft[row.nomor_klaim] ?? '').trim() === ''
        const busy = sending === row.nomor_klaim

        return (
          <Button
            tone="utama"
            disabled={orphan || empty || busy}
            onClick={() => submitRow(row)}
            title={
              orphan
                ? 'Klaim ini tidak ditemukan pada data klaim utama, sehingga tanggalnya tidak dapat disimpan.'
                : empty
                  ? 'Isi tanggal dokumen lengkap lebih dulu.'
                  : undefined
            }
          >
            {busy ? 'Menyimpan…' : 'Submit'}
          </Button>
        )
      },
    },
  ]

  const orphanCount = rows.filter((row) => !row.klaim_tersedia).length

  return (
    <main className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya "Inbox Receive TKA" mengikuti MENU_DESC pada
              Database/m_menu_aplikasi_pnc.csv (MENU_ID 49), dan caption berformat Heading 1
              pada section lamanya berbunyi sama (`D-13`). */}
          <h1 className="text-xl font-semibold text-slate-900">Inbox Receive TKA</h1>
          <p className="text-sm text-slate-600">
            Klaim TKA yang menunggu tanggal penerimaan dokumen asli diisi.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => void inbox.refetch()} disabled={inbox.isFetching}>
            {inbox.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Pada layar yang MENULIS ini
          lebih dari sekadar kerapian: Submit mengubah tanggal pada data klaim, dan "klaim
          milik siapa" tidak boleh hanya diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        Daftar ini milik entitas yang sedang dibuka, dan Submit mengubah data klaim entitas
        itu.
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{inbox.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {/* Pemotongan DINYATAKAN, tidak dibiarkan senyap seperti pyMaxRecords=500 pada sistem
          lama. Batas yang diketahui adalah batas; batas yang senyap adalah data yang hilang. */}
      {inbox.data?.terpotong && (
        <p className="mt-4 rounded-kartu border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900">
          Daftar ini <span className="font-medium">lebih panjang</span> daripada yang dapat
          ditampilkan sekaligus. Yang tampil {inbox.data.batas_baris} pekerjaan pertama;
          sisanya belum terlihat. Pakai kotak pencarian untuk mempersempit daftar.
        </p>
      )}

      {/* Baris yatim ditandai SEBELUM pengguna mencoba mengisinya. Penolakannya sudah pasti,
          dan menemukannya sendiri satu per satu hanya membuang waktu petugas. */}
      {orphanCount > 0 && (
        <p className="mt-4 rounded-kartu border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm text-amber-900">
          <span className="font-medium">{orphanCount} baris</span> tidak ditemukan pada data
          klaim utama, sehingga tanggalnya tidak dapat disimpan. Isian dan tombol Submit pada
          baris itu dimatikan. Laporkan nomor klaimnya ke administrator Claim PNC.
        </p>
      )}

      {saved && (
        <p className="mt-4 rounded-kartu border border-emerald-200 bg-emerald-50/80 px-4 py-3 text-sm text-emerald-900">
          Tanggal dokumen lengkap untuk klaim{' '}
          <span className="font-medium">{saved.claim}</span> tersimpan.
          {/* Ketiga keadaan pemberitahuan dibedakan; menyatukan dua yang terakhir akan
              membuat lingkungan tanpa server surel terus-menerus menampilkan peringatan. */}
          {saved.attempted && saved.notified && ' Pemberitahuan sudah dikirim.'}
          {saved.attempted && !saved.notified && (
            <span className="text-amber-900">
              {' '}
              Namun pemberitahuannya <span className="font-medium">gagal dikirim</span> —
              tanggalnya tetap aman, tetapi penerimanya perlu dikabari secara lain.
            </span>
          )}
          {!saved.attempted && ' Pemberitahuan lewat surel belum aktif di lingkungan ini.'}
        </p>
      )}

      {failure && (
        <div className="mt-4">
          <ErrorMessage
            title={failure.title}
            description={failure.description}
            tone={failure.tone}
          />
        </div>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Daftar klaim TKA dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : inbox.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar Inbox Receive TKA…</p>
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
              Kunci barisnya `referensi` — `pzInsKey`, yang unik per pekerjaan dan SELALU
              terisi karena sumbernya tabel yang menggerakkan kuerinya. Memakai nomor klaim
              tidak salah hari ini, tetapi keunikannya tidak dibuktikan DDL mana pun
              (`R-08`), dan kunci baris yang kembar membuat React menganggap dua baris
              sebagai satu.
            */
            rowKey={(row) => row.referensi}
            description="Sumber: antrean klaim TKA pada tabel kerja Pega"
            searchLabel="Cari nomor klaim, polis, tertanggung, atau peserta"
            pageSize={PAGE_SIZE}
            emptyMessage="Tidak ada klaim TKA yang menunggu tanggal kelengkapan dokumen."
          />
        )}
      </section>

      <footer className="mt-6 space-y-2 border-t border-slate-200 pt-4 text-xs text-slate-500">
        <p>
          <strong className="text-slate-700">Baris hilang setelah tanggalnya diisi.</strong>{' '}
          Itu memang cara kerjanya: daftar ini hanya memuat klaim TKA yang tanggal
          penerimaan dokumen aslinya belum diisi. Tanggal yang sudah tersimpan tidak dapat
          diubah dari layar ini — sama seperti aplikasi lama.
        </p>
        <p>
          <strong className="text-slate-700">Kolom “Aging” dihitung dari tanggal pendaftaran</strong>{' '}
          klaim, sama seperti aplikasi lama. Arahkan kursor ke atasnya untuk melihat tanggal
          aslinya. Bertanda hubung berarti tanggal pendaftarannya tidak tercatat.
        </p>
        <p>
          <strong className="text-slate-700">
            Tanggal yang Anda isi belum langsung terlihat di aplikasi lama.
          </strong>{' '}
          Selama kedua aplikasi masih berjalan berdampingan, layar TKA di Pega dapat tetap
          menampilkan klaim ini sebagai belum lengkap sampai datanya tersinkron. Tanggalnya
          sendiri sudah tersimpan.
        </p>
        <p>
          <strong className="text-slate-700">Daftar tidak menyegarkan dirinya sendiri.</strong>{' '}
          Petugas lain dapat mengisi tanggal pada baris yang sama, dan baris baru masuk tanpa
          tindakan Anda. Tekan Refresh untuk melihat keadaan terbaru.
        </p>
      </footer>
    </main>
  )
}
