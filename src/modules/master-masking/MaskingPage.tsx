import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type Masking } from '@/api/types'
import { ReloadIcon, AddIcon, EditIcon, ShieldIcon, CloseIcon } from '@/components/Icon'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { DataTable, type Column } from '@/components/DataTable'
import { SelectField } from '@/components/SelectField'
import { Field } from '@/components/Field'
import { Button } from '@/components/Button'
import { useSelectedPortal } from '@/app/portal'

import {
  MaskingStatus,
  moduleChecklist,
  subModuleChecklist,
  useMaskingList,
  useSetMaskingStatus,
  type MaskingFilter,
  type SearchBy,
} from './api'
import { MaskingAddForm } from './MaskingAddForm'
import { MaskingEditForm } from './MaskingEditForm'

/**
 * Layar Master Masking.
 *
 * Menggantikan harness `MasterProteksiVisibilityData` beserta dua section-nya:
 * `MasterProteksi_Sec` (kerangka, grid, dan tombolnya) dan `ActionMaskingData_Sec` (aksi
 * EDIT dan DELETE per baris). Butir menunya `MENU_ID 17` pada
 * POOLDATA.M_MENU_APLIKASI_PNC.
 *
 * # Apa yang dikelola di sini
 *
 * KEWENANGAN MELIHAT DATA PRIBADI. Nomor KTP, surel, dan nomor telepon nasabah
 * ditampilkan tersamar bagi kebanyakan petugas; baris di layar inilah yang menyatakan
 * siapa yang boleh melihatnya utuh, pada modul apa, dan seberapa banyak data yang boleh
 * ia cari serta lihat.
 *
 * Penegakan masking-nya sendiri hidup di modul Proses Produksi dan belum dibangun. Layar
 * ini hanya mengelola daftarnya — persis seperti layar Pega yang digantikannya.
 *
 * # Yang ditiru dari layar lama
 *
 * Kolom grid dan urutannya diambil apa adanya dari `MasterProteksi_Sec`:
 *
 *	CABANG · LOGIN · STATUS AKTIF · KTP · EMAIL · NOTELP · MAX LIHAT · MAX CARI ·
 *	LIHAT MODUL · AKSI
 *
 * Perhatikan tiga hal yang mudah salah bila hanya menebak dari isi tabelnya:
 *
 *   - MODUL dan SUB MODUL **tidak ada di grid**. Keduanya hanya ada di form; di grid
 *     tempatnya digantikan kolom LIHAT MODUL berisi tombol VIEW.
 *   - KTP, EMAIL, dan NOTELP adalah **tiga kolom terpisah**, bukan satu kolom gabungan.
 *   - MAX LIHAT mendahului MAX CARI, bukan sebaliknya.
 *
 * Pencariannya juga sama: **empat** tipe, mengikuti `SearchData.Type` — semua, cabang,
 * login, dan status aktif.
 *
 * Tombol Ubah dan Nonaktifkan hanya muncul pada baris AKTIF, meniru
 * `ActionMaskingData_Sec` yang kedua tombolnya bersyarat `.STS_AKTF=='AKTIF'`.
 *
 * # Yang sengaja dibuat berbeda
 *
 * | Hal | Pega | Di sini |
 * |---|---|---|
 * | Tombol DELETE | bernama "hapus", tetapi hanya menonaktifkan | disebut apa adanya: Nonaktifkan |
 * | Tiga kolom izin | teks "Ya"/"Tidak" | lencana yang terbaca sekilas |
 * | Layar sempit | grid digulir menyamping | berubah menjadi kartu (`D-12`) |
 * | Entitas | disimpulkan dari nama server | dipilih pengguna dan disebut di layar |
 * | Pencarian | dirangkai ke teks SQL (`{ASIS:…}`) | parameter binding |
 */
export function MaskingPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Baku TIDAK menyaring — `SearchData.Type == "1"` di layar lama. Sepuluh dari 25 baris
  // produksi berstatus tidak aktif; menyembunyikannya secara baku akan membuat 40% isinya
  // lenyap dari layar tanpa penjelasan.
  const [filter, setFilter] = useState<MaskingFilter>({ cariDi: 'login', kataKunci: '' })

  // Isian pencarian punya keadaannya sendiri supaya permintaan hanya dikirim saat pengguna
  // menekan Cari — bukan pada setiap ketukan papan ketik.
  const [typeDraft, setTypeDraft] = useState<SearchBy>('login')
  const [keywordDraft, setKeywordDraft] = useState('')
  // Kosong berarti belum dipilih, sama seperti layar lama — bukan diam-diam terisi AKTIF.
  const [statusDraft, setStatusDraft] = useState('')
  const [searchWarning, setSearchWarning] = useState('')

  const list = useMaskingList(filter)
  const setStatus = useSetMaskingStatus()

  const [beingEdited, setBeingEdited] = useState<Masking | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  // Baris yang sedang dilihat modulnya lewat tombol VIEW.
  const [viewing, setViewing] = useState<Masking | null>(null)

  function openAdd() {
    setBeingEdited(null)
    setFormOpen(true)
    setViewing(null)
  }

  function openEdit(row: Masking) {
    setBeingEdited(row)
    setFormOpen(true)
    setViewing(null)
  }

  function closeForm() {
    setFormOpen(false)
    setBeingEdited(null)
  }

  /**
   * Menerapkan pencarian.
   *
   * Nilai yang dikirim bergantung pada tipenya, persis seperti layar lama: tipe cabang dan
   * login memakai kotak teks (`SearchData.Country`), tipe status memakai dropdown
   * (`SearchData.STS_AKTF`), dan tipe "semua" tidak mengirim nilai apa pun.
   */
  function search() {
    setSearchWarning('')

    if (typeDraft === '') {
      setFilter({ cariDi: '', kataKunci: '' })
      return
    }
    if (typeDraft === 'status' && statusDraft === '') {
      // Pesan yang sama dengan layar lama, yang menyiapkannya untuk keadaan ini persis
      // (`Activity/SearchDataMasking-Act.xml:727`). Menyaring dengan status kosong akan
      // menampilkan seluruh baris kepada pengguna yang mengira ia sedang menyaring.
      setSearchWarning('Pilih Status Aktif lebih dulu.')
      return
    }
    setFilter({
      cariDi: typeDraft,
      kataKunci: typeDraft === 'status' ? statusDraft : keywordDraft,
    })
  }

  const rows = list.data?.masking ?? []

  const columns: Column<Masking>[] = [
    {
      key: 'cabang',
      title: 'Cabang',
      width: '13rem',
      value: (m) => `${m.nama_cabang} ${m.cabang}`,
      render: (m) => (
        <div>
          <span className="block font-medium text-slate-900">
            {/* Cabang yang tidak dikenal TIDAK disembunyikan. Tidak ada yang menggantung
                hari ini, tetapi menyembunyikannya bila kelak terjadi berarti sebuah
                kewenangan hidup tanpa pernah terbaca benar di layar mana pun. */}
            {m.nama_cabang || <span className="text-amber-700">Cabang tidak dikenal</span>}
          </span>
          <span className="font-mono text-xs text-slate-500">{m.cabang}</span>
        </div>
      ),
    },
    {
      key: 'login',
      title: 'Login',
      width: '11rem',
      value: (m) => m.login,
      render: (m) => <span className="font-medium text-slate-900">{m.login}</span>,
    },
    {
      key: 'aktif',
      title: 'Status Aktif',
      width: '8rem',
      value: (m) => (m.aktif ? 'AKTIF' : 'TIDAK AKTIF'),
      render: (m) =>
        m.aktif ? (
          <span className="inline-flex items-center rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 ring-1 ring-emerald-100">
            AKTIF
          </span>
        ) : (
          <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-slate-200">
            TIDAK AKTIF
          </span>
        ),
    },
    // KTP, EMAIL, dan NOTELP adalah TIGA kolom terpisah di layar lama, bukan satu kolom
    // gabungan. Dipertahankan begitu supaya petugas dapat menyapu satu kolom dari atas ke
    // bawah — cara membaca yang berbeda dari membandingkan tiga lencana per baris.
    // Ketiganya ditampilkan sebagai teks "Ya"/"Tidak" apa adanya, sama dengan layar lama.
    // Lencana berwarna sempat dipakai di sini dan dicabut: ia bacaan yang berbeda dari
    // yang dikenal petugas, dan `D-13` menetapkan tampilan mengikuti Pega.
    {
      key: 'lihat_ktp',
      title: 'KTP',
      width: '6rem',
      value: (m) => yesNo(m.lihat_ktp),
      render: (m) => <span className="text-slate-700">{yesNo(m.lihat_ktp)}</span>,
    },
    {
      key: 'lihat_email',
      title: 'Email',
      width: '6rem',
      value: (m) => yesNo(m.lihat_email),
      render: (m) => <span className="text-slate-700">{yesNo(m.lihat_email)}</span>,
    },
    {
      key: 'lihat_notelp',
      title: 'Notelp',
      width: '6rem',
      value: (m) => yesNo(m.lihat_notelp),
      render: (m) => <span className="text-slate-700">{yesNo(m.lihat_notelp)}</span>,
    },
    // MAX LIHAT mendahului MAX CARI — urutan layar lama.
    //
    // Angkanya ditulis APA ADANYA tanpa pemisah ribuan. Pemisah sempat ditambahkan dan
    // dicabut: layar lama menulis `10000`, dan `10.000` adalah angka yang berbeda bentuk
    // dari yang dihafal petugas.
    {
      key: 'maks_lihat',
      title: 'Max Lihat',
      width: '7rem',
      value: (m) => String(m.maks_lihat),
      render: (m) => <span className="text-slate-700">{m.maks_lihat}</span>,
    },
    {
      key: 'maks_cari',
      title: 'Max Cari',
      width: '7rem',
      value: (m) => String(m.maks_cari),
      render: (m) => <span className="text-slate-700">{m.maks_cari}</span>,
    },
    {
      // Kolom LIHAT MODUL layar lama berisi TOMBOL VIEW, bukan teks modulnya.
      key: 'lihat_modul',
      title: 'Lihat Modul',
      width: '8rem',
      noSort: true,
      value: () => '',
      render: (m) => (
        <Button tone="halus" onClick={() => setViewing(m)} aria-label={`Lihat modul ${m.login}`}>
          VIEW
        </Button>
      ),
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '13rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (m) =>
        // Kedua tombol hanya muncul pada baris AKTIF, meniru `ActionMaskingData_Sec` yang
        // keduanya bersyarat `.STS_AKTF=='AKTIF'`. Baris nonaktif karena itu tidak dapat
        // disunting maupun dinonaktifkan ulang — persis seperti sistem lama.
        m.aktif ? (
          <div className="flex justify-end gap-2">
            {/* Nama tombolnya mengikuti layar lama: Edit dan Delete. */}
            <Button tone="halus" onClick={() => openEdit(m)} aria-label={`Edit masking ${m.login}`}>
              <EditIcon className="h-3.5 w-3.5" />
              Edit
            </Button>
            <Button
              tone="kedua"
              disabled={setStatus.isPending}
              onClick={() => setStatus.mutate({ id: m.id, aktif: false })}
              aria-label={`Delete masking ${m.login}`}
            >
              Delete
            </Button>
          </div>
        ) : (
          <span className="text-xs text-slate-400">—</span>
        ),
    },
  ]

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Master Data</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Masking</li>
          </ol>
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Master Masking</h1>
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Kewenangan melihat data pribadi nasabah tanpa disamarkan — nomor KTP, alamat
          surel, dan nomor telepon — beserta batas berapa banyak data yang boleh dicari dan
          dilihat setiap pengguna.
        </p>

        {/*
          Peringatan ini bukan hiasan. Isi layar ini menentukan siapa yang dapat membuka
          data pribadi nasabah, dan `D-59` menetapkan tidak ada pemisahan tugas formal —
          jejak audit adalah satu-satunya kontrol pengimbang yang tersisa.
        */}
        <div className="mt-4 flex gap-3 rounded-kartu border border-amber-200 bg-amber-50/70 px-4 py-3">
          <ShieldIcon className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
          <p className="text-xs leading-relaxed text-amber-900">
            Setiap perubahan di layar ini mengubah siapa yang dapat membuka data pribadi
            nasabah, dan tercatat lengkap dengan nama pelakunya. Berikan kewenangan
            seperlunya saja.
          </p>
        </div>

        {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
            empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
            hanya diandaikan pengguna (ADR-0030, R-20). */}
        <p className="mt-3 text-xs text-slate-500">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </p>
      </header>

      {formOpen &&
        (beingEdited === null ? (
          <div className="mb-6">
            <MaskingAddForm onClose={closeForm} />
          </div>
        ) : (
          <div className="mb-6">
            <MaskingEditForm masking={beingEdited} onClose={closeForm} />
          </div>
        ))}

      {viewing !== null && <ModuleView masking={viewing} onClose={() => setViewing(null)} />}

      {portal === null ? (
        <ErrorMessage
          title="Portal entitas belum dipilih"
          description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bilah atas halaman ini lebih dulu."
          tone="penolakan"
        />
      ) : (
        <>
          {/*
            Panel pencarian meniru bagian CARI DATA layar lama: satu pilihan tipe dan satu
            kotak kata kunci. Bedanya, kata kunci di sini dikirim sebagai PARAMETER —
            bukan dirangkai ke dalam teks SQL seperti `{ASIS:InputSearch.CARI1}`.
          */}
          <form
            className="mb-4 rounded-kartu border border-slate-200 bg-white p-4 shadow-sm"
            onSubmit={(event) => {
              event.preventDefault()
              search()
            }}
            aria-label="Cari data masking"
          >
            <div className="grid items-end gap-4 sm:grid-cols-[14rem_1fr_auto]">
              {/*
                Keempat pilihannya sama dengan `SearchData.Type` layar lama. Pilihan
                "semua" memakai opsi kosong bawaan SelectField — itu memang artinya:
                belum menyaring apa pun.
              */}
              <SelectField
                id="cariDi"
                label="Tipe Pencarian"
                emptyText="Semua data"
                options={[
                  { value: 'cabang', label: 'Nama cabang' },
                  { value: 'login', label: 'Nama pengguna' },
                  { value: 'status', label: 'Status aktif' },
                ]}
                value={typeDraft}
                onChange={(event) => {
                  setTypeDraft(event.target.value as SearchBy)
                  setSearchWarning('')
                }}
              />

              {/*
                Kontrol nilainya BERGANTI menurut tipe, persis seperti layar lama:
                `SearchData.Type=='2'||SearchData.Type=='3'` menampilkan kotak teks, dan
                `SearchData.Type=='4'` menampilkan dropdown status.
              */}
              {typeDraft === 'status' ? (
                <SelectField
                  id="statusPencarian"
                  label="Status Aktif"
                  // Teks kosongnya diambil apa adanya dari layar lama, yang memakai kalimat
                  // yang sama sebagai pesan bila status belum dipilih.
                  emptyText="Pilih Status Aktif"
                  options={[
                    { value: MaskingStatus.aktif, label: 'AKTIF' },
                    { value: MaskingStatus.tidakAktif, label: 'TIDAK AKTIF' },
                  ]}
                  error={searchWarning || undefined}
                  value={statusDraft}
                  onChange={(event) => {
                    setStatusDraft(event.target.value)
                    setSearchWarning('')
                  }}
                />
              ) : (
                <Field
                  id="kataKunci"
                  label="Nama Pencarian"
                  placeholder={
                    typeDraft === ''
                      ? 'Tidak dipakai untuk pencarian semua data'
                      : 'Ketik sebagian nama, lalu tekan Cari'
                  }
                  disabled={typeDraft === ''}
                  value={keywordDraft}
                  onChange={(event) => setKeywordDraft(event.target.value)}
                />
              )}

              <Button tone="utama" type="submit">
                Cari
              </Button>
            </div>
          </form>

          {setStatus.isError && (
            <div className="mb-4">
              <ErrorMessage
                title="Status tidak dapat diubah"
                description={
                  setStatus.error instanceof APIError
                    ? setStatus.error.message
                    : 'Periksa koneksi lalu coba lagi.'
                }
                tone="gangguan"
              />
            </div>
          )}

          <DataTable
            columns={columns}
            rows={rows}
            rowKey={(m) => m.id}
            title="Daftar Masking Data"
            description={
              list.data
                ? `${list.data.total} data masking pada entitas ini.`
                : 'Memuat data masking…'
            }
            searchLabel="Saring daftar yang sedang tampil"
            emptyMessage="Tidak ada data masking yang cocok."
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

/** Teks kewenangan, sama persis dengan yang tersimpan di kolom: "Ya" atau "Tidak". */
function yesNo(allowed: boolean) {
  return allowed ? 'Ya' : 'Tidak'
}

/**
 * Dialog ViewDataModul — isi tombol VIEW pada kolom LIHAT MODUL.
 *
 * Bentuknya ditiru dari layar lama: dialog melayang berjudul **ViewDataModul**, dengan dua
 * kelompok **kotak centang** — MODUL dan SUB MODUL — yang menunjukkan mana yang berlaku
 * bagi baris itu.
 *
 * Centangnya MATI; ia tampilan, bukan isian. Mengubah kewenangan ditempuh lewat Edit,
 * sama seperti di sistem lama.
 *
 * Panel inline sempat dipakai di sini dan dicabut: ia menuliskan nilai mentah kolom
 * (`PNCSearchKlaim`) alih-alih label yang dibaca petugas (`VIEW HISTORY KLAIM`), dan
 * menghilangkan sub modul yang TIDAK dicentang — padahal justru daftar lengkap beserta
 * keadaan centangnya yang membuat dialog ini berguna.
 */
function ModuleView({ masking, onClose }: { masking: Masking; onClose: () => void }) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/40 p-4 sm:p-8"
      role="presentation"
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="ViewDataModul"
        className="mt-10 w-full max-w-sm overflow-hidden rounded-kartu border border-slate-200 bg-white shadow-angkat"
      >
        <div className="flex items-center justify-between gap-4 border-b border-slate-200 px-5 py-3">
          <h3 className="text-base font-medium text-slate-900">ViewDataModul</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup"
            className="rounded-kontrol p-1 text-slate-500 transition hover:bg-slate-100 hover:text-slate-700"
          >
            <CloseIcon className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-5 px-5 py-4">
          <ChecklistGroup title="MODUL" item={moduleChecklist(masking.modul)} />
          <ChecklistGroup title="SUB MODUL" item={subModuleChecklist(masking.sub_modul)} />
        </div>
      </div>
    </div>
  )
}

/** Satu kelompok kotak centang mati pada dialog ViewDataModul. */
function ChecklistGroup({
  title,
  item,
}: {
  title: string
  item: { label: string; checked: boolean; dikenal: boolean }[]
}) {
  return (
    <section>
      <h4 className="text-sm font-medium text-slate-900">{title}</h4>
      {item.length === 0 ? (
        <p className="mt-2 text-sm text-slate-500">—</p>
      ) : (
        <ul className="mt-2 space-y-2">
          {item.map((row) => (
            <li key={row.label} className="flex items-center gap-2 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={row.checked}
                readOnly
                disabled
                aria-label={`${row.label}: ${row.checked ? 'berlaku' : 'tidak berlaku'}`}
                className="h-4 w-4 rounded border-slate-300 text-blue-600"
              />
              <span>{row.label}</span>
              {/* Nilai di luar daftar tetap ditampilkan dan ditandai — menyembunyikannya
                  akan membuat data yang perlu diperbaiki tidak pernah terlihat. */}
              {!row.dikenal && (
                <span className="rounded bg-amber-50 px-1.5 py-0.5 text-xs text-amber-800 ring-1 ring-amber-200">
                  di luar daftar
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
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
  return <ErrorMessage title={message.title} description={message.description} tone={message.tone} />
}

function loadMessage(error: unknown): { title: string; description: string; tone: ErrorTone } {
  if (error instanceof NetworkError) {
    return {
      title: 'Tidak dapat menghubungi server',
      description: 'Data masking belum dapat dimuat. Periksa koneksi lalu tekan Refresh.',
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
      case ErrorCode.malformedRequest:
        return {
          title: 'Pencarian tidak dapat dijalankan',
          description: 'Pilih tipe pencarian berdasarkan nama cabang atau nama pengguna.',
          tone: 'penolakan',
        }
      default:
        return {
          title: 'Data masking gagal dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Data masking gagal dimuat',
    description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    tone: 'gangguan',
  }
}
