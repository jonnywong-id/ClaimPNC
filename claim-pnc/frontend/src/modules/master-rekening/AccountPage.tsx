import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { AccountStatus, type Account } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { AddIcon, EditIcon, ReloadIcon } from '@/components/Icon'
import { TabBar } from '@/components/TabBar'

import { AccountForm } from './AccountForm'
import { useAccountList, useDecideAccount, type AccountFilter } from './api'

/**
 * EMPAT tab, dengan nama dan urutan persis seperti layar Pega.
 *
 * Diambil dari `Section/BrowseMasterRekening-Section.xml`, yang memuat keempat
 * `pyTitle`-nya berurutan beserta section yang dipasang di tiap tab:
 *
 * | Urutan | `pyTitle` | Section |
 * |---|---|---|
 * | 1 | `Approve` | `BrowseMasterRekeningApprove` |
 * | 2 | `Reject` | `BrowseMasterRekeningReject` |
 * | 3 | `Waiting Approval` | `BrowseMasterRekeningApproval` |
 * | 4 | `Komite Approval` | `ApprovalMasterRekening` |
 *
 * # Dua koreksi terhadap versi sebelumnya
 *
 * **Tab "Cari Data Rekening" dicabut.** `BrowseMasterCariDataRekening` memang ada di
 * export, tetapi ia **bukan tab** — tidak punya `pyTitle`, dan isinya alur yang
 * berbeda: "Apakah Ingin Merubah Data Rekening?", "Pilih Perubahan Rekening",
 * "Verifikasi Rekening", tombol `Proses Rekening`. Itu alur **perubahan rekening yang
 * sudah dipakai klaim**, bukan daftar. Menjadikannya tab adalah salah baca.
 *
 * **Nama tab memakai kata Pega apa adanya**, termasuk yang berbahasa Inggris. Versi
 * sebelumnya menerjemahkannya ("Sudah Disetujui", "Menunggu Approval") dengan alasan
 * membedakannya dari tombol `Approve`/`Reject`. Alasan itu tidak cukup kuat melawan
 * `D-13`: pengguna mengenali layar ini dari kata yang tertulis di sana, dan tab versus
 * tombol sudah terbedakan oleh tempatnya.
 */
const TABS = [
  { id: 'approve', label: 'Approve' },
  { id: 'reject', label: 'Reject' },
  { id: 'waiting', label: 'Waiting Approval' },
  { id: 'komite', label: 'Komite Approval' },
] as const

type TabId = (typeof TABS)[number]['id']

function filterFor(tab: TabId, cari: string, halaman: number): AccountFilter {
  const base: AccountFilter = {
    cari,
    batas: UKURAN_HALAMAN,
    lewati: (halaman - 1) * UKURAN_HALAMAN,
  }

  switch (tab) {
    case 'approve':
      return { ...base, status: AccountStatus.disetujui }
    case 'reject':
      return { ...base, status: AccountStatus.ditolak }
    case 'waiting':
      return { ...base, status: AccountStatus.menunggu }
    case 'komite':
      return { ...base, status: AccountStatus.menunggu, komiteSaya: true }
  }
}

/**
 * Satu halaman berisi 25 baris.
 *
 * Layar lama memaginasi juga — bilah halamannya terlihat di bawah grid. Angkanya tidak
 * terbaca dari export, sehingga 25 dipilih supaya bilah halaman muncul pada data nyata
 * (portal ASM memuat ratusan rekening) tanpa membuat satu halaman terlalu panjang.
 */
const UKURAN_HALAMAN = 25

/**
 * AccountPage adalah layar pengelolaan master rekening.
 *
 * Bentuknya mengikuti layar master lain — Master Tipe Surveyors, Master Status Klaim —
 * yakni `DataTable` bersama dengan tombol Refresh dan Tambah di kanan judul, panel form
 * di atas tabel, dan aksi Ubah pada kolom terakhir.
 *
 * # Kolom berbeda PER TAB, dan itu memang begitu di Pega
 *
 * Keempat section grid-nya tidak memuat kolom yang sama. Enam kolom pertama sama;
 * sisanya mengikuti apa yang relevan pada tahap itu — lihat `columnsFor`.
 *
 * # Pencarian
 *
 * Tiga kotak cari terpisah milik Pega diganti SATU kotak cari milik `DataTable`, yang
 * menelusuri empat kolom di server. Sama dengan master lain, dan alasannya sama: jumlah
 * baris yang disaring tidak sebanding dengan tiga perjalanan jaringan.
 */
export function AccountPage() {
  const portal = useSelectedPortal((state) => state.alias)
  const [tab, setTab] = useState<TabId>('approve')

  // null = sedang menambah; berisi = sedang mengubah baris itu.
  //
  // Barisnya disimpan utuh, bukan sekadar kuncinya, supaya formulir dapat terisi
  // seketika tanpa satu permintaan tambahan ke server — daftar sudah membawa seluruh
  // kolom yang dibutuhkan.
  const [beingEdited, setBeingEdited] = useState<Account | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [cari, setCari] = useState('')
  const [halaman, setHalaman] = useState(1)

  const list = useAccountList(filterFor(tab, cari, halaman))
  const rows = list.data?.rekening ?? []
  const total = list.data?.jumlah ?? 0

  function openAdd() {
    setBeingEdited(null)
    setFormOpen(true)
  }

  function openEdit(account: Account) {
    setBeingEdited(account)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setBeingEdited(null)
  }

  // Berpindah tab atau mengubah kata kunci mengembalikan ke halaman pertama. Tanpa ini,
  // pengguna yang sedang di halaman 4 lalu berpindah tab melihat tabel kosong — tab
  // tujuan mungkin hanya punya satu halaman.
  function gantiTab(id: TabId) {
    setTab(id)
    setHalaman(1)
  }

  function gantiCari(nilai: string) {
    setCari(nilai)
    setHalaman(1)
  }

  const columns = columnsFor(tab, openEdit)

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Master Data</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Rekening</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Master Rekening</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Rekening tujuan pembayaran klaim. Rekening baru menunggu keputusan komite, lalu
          didaftarkan ke sistem Kasir — sebelum keduanya selesai, ia belum dapat dipakai
          membayar klaim.
        </p>

        {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
            empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
            hanya diandaikan pengguna (ADR-0030, R-20). */}
        <p className="mt-3 text-xs text-slate-500">
          Portal entitas: <span className="font-medium text-slate-700">{portal ?? '—'}</span>
        </p>
      </header>

      <TabBar
        tabs={TABS.map((t) => ({ kode: t.id, nama: t.label }))}
        active={tab}
        onSelect={(kode) => gantiTab(kode as TabId)}
        label="Tab master rekening"
      />

      {formOpen && (
        <div className="my-6">
          <AccountForm account={beingEdited} onClose={closeForm} />
        </div>
      )}

      <div className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : (
          <DataTable
            columns={columns}
            rows={rows}
            rowKey={(a) => `${a.kode_bank}-${a.nomor_rekening}`}
            title="Daftar Rekening"
            description={
              list.data
                ? `${total} rekening pada tab ini.`
                : 'Memuat daftar rekening…'
            }
            searchLabel="Cari nomor rekening, nama account, atau bank"
            // Pencarian dan paginasi keduanya diserahkan ke SERVER. Daftar dipotong per
            // halaman, sehingga menyaring di peramban hanya menyentuh halaman yang
            // sedang terbuka — pengguna mencari rekening yang ada di halaman berikutnya
            // lalu diberi tahu bahwa ia tidak ada.
            serverSearch={{
              value: cari,
              onChange: gantiCari,
              matchCount: list.data?.jumlah,
            }}
            pagination={{
              page: halaman,
              size: UKURAN_HALAMAN,
              total,
              totalPage: Math.max(1, Math.ceil(total / UKURAN_HALAMAN)),
              onPageChange: setHalaman,
              isLoading: list.isFetching,
            }}
            emptyMessage="Tidak ada rekening yang cocok dengan pencarian Anda."
            isLoading={list.isPending}
            error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
            actions={
              <>
                <Button
                  tone="kedua"
                  onClick={() => {
                    list.refetch()
                  }}
                  disabled={list.isFetching}
                >
                  <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
                  {list.isFetching ? 'Memuat…' : 'Refresh'}
                </Button>
                <Button tone="utama" onClick={openAdd} disabled={formOpen && beingEdited === null}>
                  <AddIcon className="h-4 w-4" />
                  Tambah
                </Button>
              </>
            }
          />
        )}

      </div>
    </div>
  )
}

/**
 * columnsFor menyusun kolom tabel untuk satu tab.
 *
 * Enam kolom pertama SAMA di keempat tab — keenamnya ada di keempat section Pega:
 *
 *	No Rek · Nama Bank · Nama Account · Nama Cabang Bank · Tipe Account · Komite Approval
 *
 * Sisanya berbeda, dan perbedaannya diambil dari caption yang benar-benar ada di tiap
 * section:
 *
 * | Tab | Kolom tambahan | Asal |
 * |---|---|---|
 * | Approve | `ID Kasir`, `Response Kasir` | `BrowseMasterRekeningApprove` |
 * | Reject | `Alasan Reject Komite`, `Alasan Reject Kasir` | `BrowseMasterRekeningReject` |
 * | Waiting Approval | — | `BrowseMasterRekeningApproval` |
 * | Komite Approval | `User Input` | `ApprovalMasterRekening` |
 *
 * # Tiga koreksi terhadap versi sebelumnya
 *
 * 1. **`Nama Bank` dan `Nama Cabang Bank` dipisah.** Sebelumnya digabung menjadi satu
 *    kolom "Bank" berisi `BCA · TASIKMALAYA`. Pega memisahkannya, dan penggabungan
 *    membuat kolomnya tidak dapat diurutkan maupun dibaca sebagai dua hal berbeda.
 * 2. **`Komite Approval` ditambahkan.** Datanya sudah ada di DTO sejak awal
 *    (`komite_approval`) tetapi tidak pernah ditampilkan — padahal itulah kolom yang
 *    memberi tahu siapa yang harus ditagih persetujuannya.
 * 3. **Kolom "Status" dicabut.** Ia tidak ada di Pega, dan memang tidak dibutuhkan:
 *    setiap tab sudah tersaring ke satu status, sehingga kolomnya akan berisi nilai
 *    yang sama di seluruh baris.
 */
function columnsFor(tab: TabId, onEdit: (account: Account) => void): Column<Account>[] {
  const base: Column<Account>[] = [
    {
      key: 'nomor_rekening',
      title: 'No Rek',
      width: '11rem',
      value: (a) => a.nomor_rekening,
      render: (a) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-700 ring-1 ring-slate-200">
          {a.nomor_rekening}
        </span>
      ),
    },
    {
      key: 'nama_bank',
      title: 'Nama Bank',
      width: '9rem',
      value: (a) => a.nama_bank,
    },
    {
      key: 'nama_pemilik',
      title: 'Nama Account',
      value: (a) => a.nama_pemilik,
      render: (a) => <span className="font-medium text-slate-900">{a.nama_pemilik}</span>,
    },
    {
      key: 'cabang_bank',
      title: 'Nama Cabang Bank',
      width: '10rem',
      value: (a) => a.cabang_bank,
      render: (a) => <Teks nilai={a.cabang_bank} />,
    },
    {
      key: 'tipe_rekening',
      title: 'Tipe Account',
      width: '7rem',
      value: (a) => a.tipe_rekening,
      render: (a) => <Teks nilai={a.tipe_rekening} />,
    },
    {
      key: 'komite_approval',
      title: 'Komite Approval',
      width: '12rem',
      value: (a) => a.komite_approval,
      render: (a) => <Teks nilai={a.komite_approval} />,
    },
  ]

  const extra: Column<Account>[] =
    tab === 'approve'
      ? [
          {
            key: 'id_rekening_kasir',
            title: 'ID Kasir',
            width: '8rem',
            value: (a) => a.id_rekening_kasir,
            render: (a) => <Teks nilai={a.id_rekening_kasir} />,
          },
          {
            key: 'respons_kasir',
            title: 'Response Kasir',
            width: '12rem',
            value: (a) => a.respons_kasir,
            // Kegagalan pendaftaran ke Kasir ditandai merah, bukan disamarkan sebagai
            // teks biasa: rekening yang disetujui komite tetapi gagal didaftarkan akan
            // menahan pembayaran, dan satu-satunya orang yang dapat menindaklanjutinya
            // adalah petugas yang melihat layar ini.
            render: (a) => (
              <span className={a.status_layanan === 'BERHASIL' ? 'text-slate-700' : 'text-red-700'}>
                {a.respons_kasir || <Teks nilai="" />}
              </span>
            ),
          },
        ]
      : tab === 'reject'
        ? [
            {
              key: 'alasan_reject_komite',
              title: 'Alasan Reject Komite',
              value: (a) => a.catatan,
              render: (a) => <Teks nilai={a.catatan} />,
            },
            {
              key: 'alasan_reject_kasir',
              title: 'Alasan Reject Kasir',
              value: (a) => a.respons_kasir,
              render: (a) => <Teks nilai={a.respons_kasir} />,
            },
          ]
        : tab === 'komite'
          ? [
              {
                key: 'diinput_oleh',
                title: 'User Input',
                width: '10rem',
                value: (a) => a.diinput_oleh,
                render: (a) => <Teks nilai={a.diinput_oleh} />,
              },
            ]
          : []

  const action: Column<Account> =
    tab === 'komite'
      ? {
          key: 'aksi',
          title: 'Aksi',
          width: '18rem',
          noSort: true,
          value: () => '',
          render: (a) => <CommitteeAction account={a} />,
        }
      : {
          key: 'aksi',
          title: 'Aksi',
          width: '7rem',
          noSort: true,
          alignRight: true,
          value: () => '',
          /*
            Tombol `Ubah` ada di SETIAP baris, termasuk pada rekening yang sudah
            disetujui — sama dengan layar lama.

            Versi sebelumnya menyembunyikannya pada baris yang sudah diputuskan, dengan
            alasan server akan menolaknya. Yang keliru ternyata servernya: layar lama
            memang menyediakan tombol itu di tab `Approve`, dan
            `UpdateMasterRekening-SQL.xml` menulis ulang `approval`, `OLDBANID`, dan
            `OLDACCOUNT_NO` sekaligus. Yang berubah sekarang: menyimpan perubahan atas
            rekening yang sudah disetujui MENCABUT persetujuannya — ia kembali menunggu
            keputusan komite atas data yang baru.
          */
          render: (a) => (
            <Button
              tone="halus"
              onClick={() => onEdit(a)}
              aria-label={`Ubah rekening ${a.nomor_rekening}`}
            >
              <EditIcon className="h-3.5 w-3.5" />
              Ubah
            </Button>
          ),
        }

  return [...base, ...extra, action]
}

/**
 * Teks menggambar satu nilai, dengan tanda hubung bila kosong.
 *
 * Sel yang benar-benar kosong tidak dapat dibedakan dari sel yang gagal dimuat; tanda
 * hubung menyatakan "memang tidak ada isinya".
 */
function Teks({ nilai }: { nilai: string }) {
  if (nilai.trim() === '') {
    return <span className="text-slate-400">—</span>
  }
  return <span className="text-slate-700">{nilai}</span>
}

/**
 * CommitteeAction menampilkan tombol setujui dan tolak.
 *
 * Keterangan approval wajib diisi untuk KEDUANYA di layar ini, walaupun server hanya
 * mewajibkannya saat menyetujui. Alasannya: komite yang menolak tanpa alasan membuat
 * pengaju mengulang pengajuan yang sama persis, karena tidak ada yang memberitahunya
 * apa yang salah.
 */
function CommitteeAction({ account }: { account: Account }) {
  const decide = useDecideAccount()
  const [note, setNote] = useState(account.catatan)
  const id = `catatan-${account.kode_bank}-${account.nomor_rekening}`

  const send = (status: typeof AccountStatus.disetujui | typeof AccountStatus.ditolak) => {
    decide.mutate({
      kodeBank: account.kode_bank,
      nomorRekening: account.nomor_rekening,
      status,
      catatan: note,
    })
  }

  return (
    <div className="flex w-full min-w-[16rem] flex-col gap-2">
      <label className="sr-only" htmlFor={id}>
        Keterangan approval atasan untuk rekening {account.nomor_rekening}
      </label>
      <input
        id={id}
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder="Keterangan approval atasan"
        disabled={decide.isPending}
        className="w-full rounded-kontrol border border-slate-300 px-2 py-1 text-sm shadow-lembut focus:border-blue-500 focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20"
      />

      <div className="flex gap-2">
        <Button tone="utama" disabled={decide.isPending} onClick={() => send(AccountStatus.disetujui)}>
          Approve
        </Button>
        {/*
          Menolak perlu terlihat berbeda dari menyetujui, dan `Button` baku hanya punya
          tiga nada — tidak ada nada "bahaya". Warnanya ditimpa di sini, BUKAN dengan
          menambah nada baru ke komponen bersama: nada baru akan menyentuh seluruh layar
          yang memakainya, dan itu di luar lingkup pekerjaan ini. Bila kelak ada layar
          kedua yang membutuhkannya, nada `bahaya` layak dibuat dan penimpaan ini dicabut.
        */}
        <Button
          tone="kedua"
          disabled={decide.isPending}
          onClick={() => send(AccountStatus.ditolak)}
          className="border-red-300 text-red-700 hover:border-red-400 hover:bg-red-50 hover:text-red-800 focus-visible:ring-red-500/30"
        >
          Reject
        </Button>
      </div>

      {decide.isError && <DecisionMessage error={decide.error} />}
    </div>
  )
}

function DecisionMessage({ error }: { error: unknown }) {
  if (error instanceof APIError && error.kode === 'isian_tidak_sah') {
    return (
      <p role="alert" className="text-xs text-red-700">
        {Object.values(error.violations()).join(' ')}
      </p>
    )
  }
  if (error instanceof APIError && error.kode === 'keputusan_sudah_diambil') {
    return (
      <p role="alert" className="text-xs text-red-700">
        Rekening ini sudah diputuskan komite lain. Muat ulang daftar.
      </p>
    )
  }
  return (
    <p role="alert" className="text-xs text-red-700">
      Keputusan tidak tersimpan. Coba lagi.
    </p>
  )
}

/**
 * Gagal memuat dibedakan dari gagal menyimpan.
 *
 * Yang di sini selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat
 * salah — ia baru membuka layarnya.
 */
function LoadErrorMessage({ error }: { error: unknown }) {
  const message = loadMessage(error)
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

function loadMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Daftar rekening belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    return {
      title: 'Daftar rekening gagal dimuat',
      description: error.message,
      tone: 'gangguan',
    }
  }
  return {
    title: 'Daftar rekening gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
