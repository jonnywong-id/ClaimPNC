import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Surveyor } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { AddIcon, EditIcon, ReloadIcon } from '@/components/Icon'
import { useSurveyorTypeList } from '@/modules/master-tipe-surveyors/api'

import { useSurveyorList, type SurveyorFilter } from './api'
import { SurveyorDecisionPanel } from './SurveyorDecisionPanel'
import { SurveyorForm } from './SurveyorForm'

/**
 * Lima tab: keempat tab layar lama, ditambah satu tab pencarian.
 *
 * Saringan tiap tab BUKAN tafsiran. Layar lama memanggil Report Definition
 * `BrowseVDSurveyors_RD` dengan dua parameter, dan nilainya terbaca langsung di tiap
 * section (`pyRDParams`):
 *
 *	tab              section                              Approve   Komite
 *	Waiting Approval BrowseDetailSuveryorsWaiting-Section  "0"      (kosong)
 *	Komite Approval  BrowseDetailSuveryorsKomite-Section   "0"      OperatorID.pyUserIdentifier
 *	Approve          BrowseDetailSuveryorsApprove-Section  "1"      (kosong)
 *	Reject           BrowseDetailSuveryorsReject-Section   "2"      (kosong)
 *
 * Baris kedua itu yang membuktikan dua hal sekaligus: tab "Antrean Komite Saya" menyaring
 * dengan identitas operator yang sedang masuk, dan kolom KOMITE memang berisi Operator ID
 * — bukan kode bisnis seperti yang dijanjikan alias `BUSINESS_CODE`.
 *
 * Tab "Cari" adalah TAMBAHAN yang tidak ada di layar lama: keempat tab Pega tidak punya
 * satu pun tampilan tanpa saringan status, sehingga mencari surveyor yang statusnya belum
 * diketahui menuntut membuka tab satu per satu.
 */
const TABS = [
  { id: 'approve', label: 'Approve' },
  { id: 'reject', label: 'Reject' },
  { id: 'waiting', label: 'Waiting Approval' },
  { id: 'komite', label: 'Komite Approval' },
] as const

type TabId = (typeof TABS)[number]['id']

/**
 * Saringan tiap tab, diambil dari `pyRDParams` masing-masing section — bukan ditafsirkan.
 *
 *	tab               Approve   Komite
 *	Approve           "1"       (kosong)
 *	Reject            "2"       (kosong)
 *	Waiting Approval  "0"       (kosong)
 *	Komite Approval   "0"       OperatorID.pyUserIdentifier
 */
function filterFor(tab: TabId): SurveyorFilter {
  switch (tab) {
    case 'approve':
      return { status: '1' }
    case 'reject':
      return { status: '2' }
    case 'waiting':
      return { status: '0' }
    case 'komite':
    default:
      return { antrean_saya: true }
  }
}

/**
 * Layar Master Surveyors.
 *
 * Menggantikan harness `DetailSurveyorsInbox` beserta section-nya. Butir menunya
 * `MENU_ID 15` pada POOLDATA.M_MENU_APLIKASI_PNC.
 *
 * # Apa yang dikelola di sini
 *
 * Daftar ORANG dan LEMBAGA yang melakukan survei — bukan golongannya. Golongannya ada
 * satu tingkat di atas, di butir menu "Master Tipe Surveyors" (`MENU_ID 14`, harness
 * `SurveyorsInbox`). Keduanya mudah tertukar, dan judul layar ini sengaja menyebut
 * kaitannya.
 *
 * # Yang ditiru dari layar lama
 *
 * Alur persetujuan komite, keempat posisinya, tombol **Tambah** dan **Refresh**, serta
 * aksi **Ubah** per baris. **Tidak ada Hapus** — layar Pega pun tidak punya.
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Empat grid | empat section terpisah | satu tabel, saringan per tab |
 * | Judul kolom | nama kolom mentah | nama yang dibaca manusia (`D-19`) |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Nama ganda | ditolak lewat activity | ditolak, dan juga oleh indeks unik |
 * | Menyunting baris yang sudah diputus | kembali ke antrean komite | **sama** |
 * | Akun surveyor | dibuat otomatis dengan sandi sementara | BELUM dibuat — menunggu `F-3` |
 * | Entitas | disimpulkan dari nama server | dipilih pengguna dan disebut di layar |
 *
 * Baris terakhir adalah penyimpangan yang paling perlu diketahui pengguna, dan karena itu
 * ia disebut di layar — bukan hanya di dokumen.
 */
export function SurveyorPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Tab yang terbuka pertama adalah "Approve", sama seperti layar lama.
  const [tab, setTab] = useState<TabId>('approve')
  const list = useSurveyorList(filterFor(tab))

  // Daftar tipe dibaca dari modul tetangga, bukan disalin. Ia sumber yang sama dengan
  // yang dikelola layar Master Tipe Surveyors, sehingga tipe yang baru ditambahkan di
  // sana langsung dapat dipilih di sini.
  const types = useSurveyorTypeList()

  const [beingEdited, setBeingEdited] = useState<Surveyor | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [beingDecided, setBeingDecided] = useState<Surveyor | null>(null)

  function openAdd() {
    setBeingEdited(null)
    setBeingDecided(null)
    setFormOpen(true)
  }

  function openEdit(surveyor: Surveyor) {
    setBeingEdited(surveyor)
    setBeingDecided(null)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setBeingEdited(null)
  }

  function openDecision(surveyor: Surveyor) {
    setFormOpen(false)
    setBeingEdited(null)
    setBeingDecided(surveyor)
  }

  const rows = list.data?.surveyor ?? []

  // KOLOM DAN JUDULNYA mengikuti grid layar lama persis, dibaca dari properti yang
  // dirujuk keempat section (`.D_SURVEY_ID`, `sub1.DESCRIPTION`, `.NAME`, `.BRANCHNAME`,
  // `.TELEPHONE`, `.EMAIL`, `.KOMITE`, lalu tautan Ubah).
  //
  // Kolom "Status" yang sempat ada di sini DIBUANG: layar lama tidak memilikinya, dan
  // memang tidak perlu — tab yang sedang dibuka sudah menyatakan statusnya.
  const columns: Column<Surveyor>[] = [
    {
      key: 'id',
      title: 'ID',
      width: '7rem',
      value: (s) => s.id,
      render: (s) => <span className="font-mono text-xs text-slate-700">{s.id}</span>,
    },
    {
      key: 'nama_tipe',
      title: 'ID Survei Master',
      width: '11rem',
      value: (s) => s.nama_tipe || s.kode_tipe,
      render: (s) => (
        <span className="text-slate-700">
          {s.nama_tipe || (
            // Tipe yang kodenya tidak lagi ada di master. Barisnya tetap ditampilkan —
            // menyembunyikannya adalah kelas cacat yang paling sulit disadari — tetapi
            // keadaannya ditandai supaya dapat diperbaiki.
            <span title={`Kode tipe ${s.kode_tipe} tidak ada di master tipe surveyor`}>
              <span className="font-mono text-xs">{s.kode_tipe}</span>
              <span className="ml-1 text-amber-600">(tipe tidak dikenal)</span>
            </span>
          )}
        </span>
      ),
    },
    {
      key: 'nama',
      // "Nama", bukan "Input Nama": teks itu nol kemunculan di seluruh export.
      title: 'Nama',
      value: (s) => s.nama,
      render: (s) => <span className="font-medium text-slate-900">{s.nama}</span>,
    },
    {
      key: 'nama_cabang',
      title: 'Cabang',
      width: '10rem',
      value: (s) => s.nama_cabang || s.kode_cabang,
      render: (s) => <span className="text-slate-700">{s.nama_cabang || s.kode_cabang || '—'}</span>,
    },
    {
      key: 'telepon',
      title: 'Telp',
      width: '9rem',
      value: (s) => s.telepon,
      render: (s) => <span className="text-slate-700">{s.telepon || '—'}</span>,
    },
    {
      key: 'email',
      title: 'Email',
      value: (s) => s.email,
      render: (s) => <span className="break-all text-slate-700">{s.email || '—'}</span>,
    },
    {
      key: 'komite',
      title: 'Komite',
      width: '10rem',
      value: (s) => s.komite,
      render: (s) => <span className="font-mono text-xs text-slate-700">{s.komite || '—'}</span>,
    },
    {
      key: 'aksi',
      title: '',
      width: '11rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      // Tombol Approve dan Reject HANYA muncul di tab "Komite Approval" — sama seperti
      // layar lama, yang menaruh keduanya di section `-Komite` saja. Tab Approve, Reject,
      // dan Waiting Approval hanya punya tautan Ubah.
      render: (s) => (
        <div className="flex justify-end gap-1.5">
          {tab === 'komite' && s.status === '0' && (
            <Button
              tone="halus"
              onClick={() => openDecision(s)}
              aria-label={`Putuskan sebagai komite untuk surveyor ${s.nama}`}
            >
              Approve / Reject
            </Button>
          )}
          <Button
            tone="halus"
            onClick={() => openEdit(s)}
            aria-label={`Ubah surveyor ${s.nama}`}
          >
            <EditIcon className="h-3.5 w-3.5" />
            Ubah
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Master Data</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Surveyors</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Master Surveyors</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Daftar petugas dan lembaga yang melakukan survei kerugian. Setiap surveyor
          digolongkan ke salah satu tipe pada Master Tipe Surveyors, dan baru dapat
          ditugaskan pada klaim setelah komite menyetujuinya.
        </p>

        {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
            empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
            hanya diandaikan pengguna (ADR-0030, R-20). */}
        <p className="mt-3 text-xs text-slate-500">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </p>
      </header>

      {/* Penyimpangan yang paling perlu diketahui pengguna disebut di layar, bukan hanya
          di dokumen: surveyor internal yang ditambahkan di sini BELUM mendapat akun. */}
      <div className="mb-6">
        <ErrorMessage
          title="Akun aplikasi surveyor belum dibuat otomatis"
          description="Di sistem lama, menambah Internal Surveyor sekaligus menerbitkan akun aplikasinya. Di sini nama loginnya tersimpan dan keunikannya dijaga, tetapi akunnya belum terbit — pembuatan akun menunggu modul Identitas & Akses. Sampaikan ke administrator bila surveyor baru perlu segera masuk."
          tone="gangguan"
        />
      </div>

      {formOpen && (
        <div className="mb-6">
          <SurveyorForm
            surveyor={beingEdited}
            surveyorTypes={types.data?.tipe_surveyor ?? []}
            onClose={closeForm}
          />
        </div>
      )}

      {beingDecided && (
        <div className="mb-6">
          <SurveyorDecisionPanel
            surveyor={beingDecided}
            onClose={() => setBeingDecided(null)}
          />
        </div>
      )}

      {portal === null ? (
        <ErrorMessage
          title="Portal entitas belum dipilih"
          description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu."
          tone="penolakan"
        />
      ) : (
        <>
          <div
            role="tablist"
            aria-label="Tab master surveyors"
            className="mb-4 flex flex-wrap gap-1 border-b border-slate-200"
          >
            {TABS.map((t) => (
              <button
                key={t.id}
                type="button"
                role="tab"
                aria-selected={tab === t.id}
                onClick={() => setTab(t.id)}
                className={
                  '-mb-px rounded-t-md border-b-2 px-3 py-2 text-sm font-medium transition ' +
                  (tab === t.id
                    ? 'border-blue-600 text-blue-700'
                    : 'border-transparent text-slate-500 hover:text-slate-700')
                }
              >
                {t.label}
              </button>
            ))}
          </div>

          <DataTable
            columns={columns}
            rows={rows}
            rowKey={(s) => s.id}
            title="Daftar Surveyor"
            description={
              list.data
                ? `${list.data.total} surveyor pada tab ini.`
                : 'Memuat daftar surveyor…'
            }
            searchLabel="Cari nama surveyor, tipe, atau cabang"
            emptyMessage={
              tab === 'komite'
                ? 'Tidak ada surveyor yang menunggu keputusan Anda.'
                : 'Belum ada surveyor pada tab ini.'
            }
            isLoading={list.isPending}
            error={list.isError ? <LoadErrorMessage error={list.error} /> : undefined}
            actions={
              <>
                <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
                  <ReloadIcon className={`h-4 w-4 ${list.isFetching ? 'animate-spin' : ''}`} />
                  {list.isFetching ? 'Memuat…' : 'Refresh'}
                </Button>
                <Button tone="utama" onClick={openAdd} disabled={formOpen && !beingEdited}>
                  <AddIcon className="h-4 w-4" />
                  Tambah
                </Button>
              </>
            }
          />
        </>
      )}
    </div>
  )
}

/**
 * Gagal memuat dibedakan dari gagal menyimpan.
 *
 * Yang di sini selalu bernada gangguan: pengguna belum melakukan apa pun yang dapat salah
 * — ia baru membuka layarnya. Kecuali soal portal, yang justru dapat ia perbaiki sendiri.
 */
function LoadErrorMessage({ error }: { error: unknown }) {
  const message = loadMessage(error)
  return (
    <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
  )
}

function loadMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Daftar surveyor belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
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
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar surveyor gagal dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Daftar surveyor gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
