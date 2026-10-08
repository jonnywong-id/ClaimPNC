import { useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { ReloadIcon } from '@/components/Icon'
import { offsetPagination } from '@/components/inbox/TaskQueueTable'
import { policyNumberColumn, protectionTrailingColumns } from '@/components/inbox/protectionColumns'
import { DokumenPenunjangPanel } from '@/modules/dokumen-penunjang/DokumenPenunjangPanel'

import {
  PAGE_SIZE,
  useAcceptQueue,
  useAllowedQueues,
  useDecideProtection,
  useProtectionDetail,
} from './api'
import {
  protectionTypeLabel,
  type ChangeDetail,
  type Protection,
  type ProtectionDetail,
  type Queue,
} from './types'

/**
 * Layar Inbox Accept Open Protection.
 *
 * Migrasi dari **`Harness/InputProtection_Harness-Harness.xml`** beserta
 * `Section/InputProtection_Section-Section.xml` dan form
 * `Section/AcceptProtectionSection-Section.xml`.
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Ketujuh kolom | `Section/InputProtection_Section-Section.xml` |
 * | Antrean NON PREMI | `Report Definition/InboxOpenProtection2_RD-RD.xml` |
 * | Antrean PREMI | `Report Definition/InboxOpenProtection2_RD_collection-RD.xml` |
 * | Setuju / Tolak | `Flow/CreateProtection_Flow.xml` · `When/IsAcceptProtection-When.xml` |
 *
 * # Dua antrean, dan asal pemisahannya
 *
 * Layar lama memuat tiga grid berisi kolom yang sama persis, dibedakan syarat tampilnya.
 * Dua di antaranya nyata: PREMI (`TypeProtection = "2"`, hanya bagi peran penagihan premi)
 * dan NON PREMI (`!= "2"`).
 *
 * # Grid ketiga TIDAK dibawa
 *
 * Grid ketiga hanya muncul untuk **satu alamat Gmail pribadi**
 * (`Section/InputProtection_Section-Section.xml:10342`), dan penyaringnya lebih longgar —
 * tanpa syarat nomor klaim. `D-15` melarang nilai bisnis di-hardcode dan `D-67` melarang
 * akun pribadi dibawa ke sistem baru.
 *
 * Ini SELISIH TERENCANA (`P-5`): orang tersebut akan melihat antrean yang berbeda dari hari
 * ini. Bila keleluasaan itu memang dibutuhkan bisnis, ia harus kembali sebagai PERAN di
 * master data.
 *
 * # Kewenangan ditegakkan sejak 2026-09-25
 *
 * Tab yang digambar hanyalah antrean yang MENJADI HAK pemanggil, dan daftarnya datang dari
 * server — bukan ditebak di peramban. Sumbernya `POOLDATA.M_LOGIN_GROUP_PNC.GROUP_ID`, yang
 * berisi nama access group Pega tanpa awalan `GCNMFW:`.
 *
 * Penegakan sesungguhnya ada di SERVER: setiap permintaan diperiksa, sehingga menyembunyikan
 * tab hanyalah kenyamanan tampilan. Itu yang membedakannya dari sistem lama, yang hanya
 * menyembunyikan menu.
 */
export function AcceptQueuePage() {
  const [pickedQueue, setPickedQueue] = useState<Queue | null>(null)
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)
  const [opened, setOpened] = useState<string | null>(null)

  const portal = useSelectedPortal((state) => state.alias)

  const allowed = useAllowedQueues()
  const allowedQueues = allowed.data?.antrean ?? []

  // Antrean aktif: yang dipilih pengguna bila masih menjadi haknya, selain itu yang pertama
  // menjadi haknya. Menahan pilihan yang sudah tidak berlaku akan menembak server untuk
  // jawaban yang pasti 403.
  const queue: Queue | null =
    pickedQueue && allowedQueues.includes(pickedQueue)
      ? pickedQueue
      : (allowedQueues[0] ?? null)

  const list = useAcceptQueue(
    { queue: queue ?? 'non-premi', search, offset },
    { enabled: queue !== null },
  )
  const detail = useProtectionDetail(opened)
  const decide = useDecideProtection()

  const rows = list.data?.proteksi ?? []
  const total = list.data?.total ?? 0

  // 403 atas daftar antrean berarti access group pemanggil tidak berwenang atas layar ini.
  // Dibedakan dari gangguan supaya pesannya menjelaskan apa yang harus diminta pengguna.
  const forbidden = allowed.error instanceof APIError && allowed.error.status === 403

  function changeQueue(next: Queue) {
    setPickedQueue(next)
    // Halaman dan pencarian dikembalikan: keduanya milik antrean sebelumnya, dan
    // mempertahankannya akan menampilkan halaman empat dari daftar yang hanya punya satu.
    setOffset(0)
    setSearch('')
    setOpened(null)
    decide.reset()
  }

  function changeSearch(next: string) {
    setSearch(next)
    setOffset(0)
  }

  async function submit(decision: 'setuju' | 'tolak') {
    if (opened === null) return

    // Galat DITELAN dengan sengaja; ia sudah ditampilkan lewat `decide.error`. Membiarkan
    // promise ini menolak akan menghasilkan unhandled rejection di peramban.
    //
    // Panel TIDAK ditutup saat gagal, dan justru itu yang penting di sini: kegagalan yang
    // paling sering terjadi adalah `409` — proteksi sudah diputuskan petugas lain. Menutup
    // panelnya akan menyembunyikan pesan yang menjelaskannya.
    try {
      await decide.mutateAsync({ number: opened, decision })
      setOpened(null)
    } catch {
      // Sudah tercatat di decide.error; lihat komentar di atas.
    }
  }

  const columns = protectionColumns((nomor) => {
    decide.reset()
    setOpened(nomor)
  })

  const decisionFailure = decide.error instanceof APIError ? decide.error.message : null

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Inbox Accept Open Protection</h1>
        <p className="mt-1 text-sm text-slate-600">
          Permintaan pembukaan proteksi yang sudah lengkap dan menunggu keputusan. Baris
          hilang begitu diputuskan — juga bila yang memutuskan petugas lain.
        </p>
      </header>

      {portal === null && (
        <div className="mt-6">
          <ErrorMessage
            title="Pilih entitas lebih dulu"
            description="Layar ini membaca proteksi milik satu badan hukum, sehingga entitasnya harus dipilih di bilah atas."
            tone="gangguan"
          />
        </div>
      )}

      {forbidden && (
        <div className="mt-6">
          <ErrorMessage
            title="Anda tidak berwenang atas layar ini"
            description="Access group Anda tidak termasuk yang boleh mengakseptasi permintaan proteksi. Hubungi administrator bila seharusnya berhak."
            tone="penolakan"
          />
        </div>
      )}

      {/*
        Tab hanya digambar untuk antrean yang MENJADI HAK pemanggil.

        Layar lama tidak punya pemilihan sama sekali — grid yang bukan haknya tidak pernah
        dirender. Menggambar tab yang pasti dijawab 403 hanya membuat pengguna mencobanya.

        Bila haknya cuma satu, tabnya tetap digambar: ia menyebutkan antrean mana yang
        sedang dilihat, dan tanpa itu daftar PREMI tidak dapat dibedakan dari NON PREMI.
      */}
      {allowedQueues.length > 0 && (
        <div className="mt-6 flex flex-wrap items-center gap-2" role="tablist" aria-label="Antrean akseptasi">
          {allowedQueues.includes('non-premi') && (
            <QueueTab current={queue} value="non-premi" label="Proteksi Klaim NON PREMI" onPick={changeQueue} />
          )}
          {allowedQueues.includes('premi') && (
            <QueueTab current={queue} value="premi" label="Proteksi Klaim PREMI" onPick={changeQueue} />
          )}
        </div>
      )}

      {allowed.isSuccess && allowedQueues.length === 0 && (
        <div className="mt-6">
          <ErrorMessage
            title="Tidak ada antrean yang dapat dibuka"
            description="Access group Anda berwenang atas layar ini, tetapi tidak atas satu antrean pun. Laporkan ke administrator — kemungkinan besar keanggotaan group Anda belum lengkap."
            tone="gangguan"
          />
        </div>
      )}

      {opened !== null && (
        <section className="mt-6 rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <h2 className="mb-4 text-base font-semibold text-slate-900">Akseptasi Proteksi</h2>

          {decisionFailure && (
            <div className="mb-4">
              <ErrorMessage
                title="Keputusan tidak dapat disimpan"
                description={decisionFailure}
                tone="penolakan"
              />
            </div>
          )}

          <ProtectionDetailBody
            isPending={detail.isPending}
            error={detail.error}
            data={detail.data}
            deciding={decide.isPending}
            onDecide={(decision) => { submit(decision) }}
            onClose={() => setOpened(null)}
          />
        </section>
      )}

      <div className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(p) => p.nomor_proteksi}
          title={queue === 'premi' ? 'Proteksi Klaim PREMI' : 'Proteksi Klaim NON PREMI'}
          label={`Antrean akseptasi ${queue}`}
          // Prop tidak dikirim sama sekali saat kosong: tsconfig memakai
          // exactOptionalPropertyTypes, yang membedakan "tidak ada" dari "undefined".
          {...(total > 0 ? { description: `${total} permintaan menunggu keputusan.` } : {})}
          isLoading={list.isPending && portal !== null}
          searchLabel="Cari No Proteksi / No Polis / No Klaim"
          emptyMessage="Tidak ada permintaan proteksi yang menunggu keputusan pada antrean ini."
          serverSearch={{ value: search, onChange: changeSearch, matchCount: total }}
          pagination={offsetPagination(offset, PAGE_SIZE, total, setOffset, list.isFetching)}
          actions={
            <Button
              tone="halus"
              onClick={() => { list.refetch() }}
              disabled={list.isFetching || portal === null}
            >
              <ReloadIcon className="h-4 w-4" />
              {list.isFetching ? 'Memuat…' : 'Muat ulang'}
            </Button>
          }
        />
      </div>

      <p className="mt-4 text-xs text-slate-500">
        Pemisahan antrean di sistem lama mengikuti peran pengguna, bukan pilihan. Selama
        tabel peran belum terisi, kedua antrean dapat dibuka siapa pun yang berhak masuk
        layar ini.
      </p>
    </div>
  )
}

/**
 * Ketujuh kolom antrean. Dipisahkan dari komponen layar supaya fungsi layarnya tetap
 * terbaca (temuan SonarQube S3776); `onOpen` membuka panel akseptasi untuk satu proteksi.
 */
function protectionColumns(onOpen: (nomor: string) => void): Column<Protection>[] {
  return [
    {
      key: 'nomor_proteksi',
      title: 'No Proteksi',
      width: '11rem',
      value: (p) => p.nomor_proteksi,
      render: (p) => (
        <button
          type="button"
          onClick={() => onOpen(p.nomor_proteksi)}
          className="truncate rounded font-mono text-xs font-medium text-blue-700 underline-offset-2 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
        >
          {p.nomor_proteksi}
        </button>
      ),
    },
    policyNumberColumn<Protection>(),
    {
      key: 'nomor_klaim',
      title: 'No Klaim',
      width: '11rem',
      value: (p) => p.nomor_klaim,
      render: (p) => (
        <span className="truncate font-mono text-xs text-slate-700">{p.nomor_klaim || '—'}</span>
      ),
    },
    {
      key: 'tipe_proteksi',
      title: 'Tipe Proteksi',
      width: '12rem',
      value: (p) => p.tipe_proteksi,
      render: (p) => <span className="truncate">{protectionTypeLabel(p.tipe_proteksi, p.nama_tipe_proteksi)}</span>,
    },
    ...protectionTrailingColumns<Protection>(),
  ]
}

/** Satu tab antrean. */
function QueueTab({
  current,
  value,
  label,
  onPick,
}: Readonly<{
  // Boleh null selama daftar antrean yang menjadi hak pemanggil belum tiba — pada saat itu
  // belum ada tab yang aktif, dan menandai salah satunya akan keliru separuh waktu.
  current: Queue | null
  value: Queue
  label: string
  onPick: (queue: Queue) => void
}>) {
  const active = current === value

  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={() => onPick(value)}
      className={
        active
          ? 'rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white'
          : 'rounded-lg bg-slate-100 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-200'
      }
    >
      {label}
    </button>
  )
}

/** Satu pasang label dan nilai pada rincian proteksi. */
/**
 * Panel "Detail Perubahan" — apa yang sebenarnya diminta berubah.
 *
 * # Kenapa ia dipisahkan dari daftar field di atasnya
 *
 * Layar lama pun memisahkannya: `Section/AcceptProtectionSection-Section.xml` membungkusnya
 * dalam container bersyarat berjudul sendiri. Bagi tipe `'7'` dan `'8'`, inilah yang
 * ditimbang petugas — sisanya konteks.
 *
 * Pasangan nilai digambar sebagai **sebelum → sesudah** supaya arah perubahannya terbaca
 * tanpa membandingkan dua label. Pada layar sempit panahnya berputar menjadi menurun; nilai
 * yang panjang tidak boleh memaksa tabel menggeser mendatar.
 */
function ChangeDetailPanel({ detail }: Readonly<{ detail: ChangeDetail }>) {
  // Tipe '7' mengisi pasangan tanggal, tipe '8' mengisi pasangan penyebab. Keduanya tidak
  // pernah terisi bersamaan — backend yang memastikannya.
  const pasangan = detail.dol_sebelum || detail.dol_sesudah
    ? {
        label: 'Current Date Of Loss',
        labelSesudah: 'Next Date Of Loss',
        sebelum: detail.dol_sebelum ? formatDate(detail.dol_sebelum) : '',
        sesudah: detail.dol_sesudah ? formatDate(detail.dol_sesudah) : '',
      }
    : {
        label: 'Cause Of Loss Dipilih',
        labelSesudah: 'Next Cause Of Loss',
        sebelum: detail.penyebab_sebelum,
        sesudah: detail.penyebab_sesudah,
      }

  return (
    <section className="mt-6 rounded-lg border border-slate-200 bg-slate-50 p-4">
      <h3 className="text-sm font-semibold text-slate-900">{detail.judul}</h3>

      {detail.kosong ? (
        // Panel tetap tampil. Menyembunyikannya akan membuat permintaan perubahan tampak
        // seolah tidak mengubah apa pun — dan baris warisan Pega seluruhnya begini.
        <p className="mt-2 text-sm text-slate-600">
          Rincian perubahan tidak tersedia untuk permintaan ini.
        </p>
      ) : (
        <>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-stretch sm:gap-4">
            <ChangeSide label={pasangan.label} value={pasangan.sebelum} />
            <div
              aria-hidden="true"
              className="flex items-center justify-center text-slate-400 sm:px-1"
            >
              <span className="sm:hidden">↓</span>
              <span className="hidden sm:inline">→</span>
            </div>
            <ChangeSide label={pasangan.labelSesudah} value={pasangan.sesudah} highlight />
          </div>

          {(detail.nama_objek || detail.nama_cabang) && (
            <dl className="mt-4 grid gap-x-6 gap-y-3 border-t border-slate-200 pt-3 sm:grid-cols-2">
              <Detail label="Object Name" value={detail.nama_objek} />
              <Detail label="Branch Name" value={detail.nama_cabang} />
            </dl>
          )}
        </>
      )}
    </section>
  )
}

/**
 * Satu sisi pasangan perubahan.
 *
 * Sisi "sesudah" ditandai supaya nilai yang diminta menonjol dari nilai yang berlaku
 * sekarang — bukan dengan warna saja, melainkan dengan bingkai dan ketebalan, supaya
 * pembedanya tetap terbaca tanpa membedakan warna.
 */
function ChangeSide({
  label,
  value,
  highlight,
}: Readonly<{
  label: string
  value: string
  highlight?: boolean
}>) {
  return (
    <div
      className={
        highlight
          ? 'flex-1 rounded-md border-2 border-blue-300 bg-white px-3 py-2'
          : 'flex-1 rounded-md border border-slate-200 bg-white px-3 py-2'
      }
    >
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</p>
      <p
        className={
          highlight
            ? 'mt-0.5 break-words text-sm font-semibold text-slate-900'
            : 'mt-0.5 break-words text-sm text-slate-700'
        }
      >
        {value || '—'}
      </p>
    </div>
  )
}

function Detail({ label, value, mono }: Readonly<{ label: string; value: string; mono?: boolean }>) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className={mono ? 'mt-0.5 font-mono text-sm text-slate-800' : 'mt-0.5 text-sm text-slate-800'}>
        {value || '—'}
      </dd>
    </div>
  )
}

/**
 * Isi panel "Akseptasi Proteksi": memuat, galat, atau rincian beserta tombol keputusannya.
 *
 * Dipisahkan dari layar supaya ketiga keadaannya terbaca sebagai `if` berurutan, bukan
 * ternary bersarang (temuan SonarQube S3358 dan S3776). Urutan pemeriksaannya sama persis
 * dengan sebelumnya: memuat lebih dulu, lalu galat, lalu data.
 */
function ProtectionDetailBody({
  isPending,
  error,
  data,
  deciding,
  onDecide,
  onClose,
}: Readonly<{
  isPending: boolean
  error: Error | null
  data: ProtectionDetail | undefined
  deciding: boolean
  onDecide: (decision: 'setuju' | 'tolak') => void
  onClose: () => void
}>) {
  if (isPending) {
    return <p className="text-sm text-slate-500">Memuat rincian…</p>
  }

  if (error) {
    return (
      <ErrorMessage
        title="Rincian tidak dapat dibuka"
        description={
          error instanceof APIError
            ? error.message
            : 'Terjadi kesalahan saat memuat rincian.'
        }
        tone="gangguan"
      />
    )
  }

  if (!data) return null

  return (
    <>
      <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
        <Detail label="No Proteksi" value={data.nomor_proteksi} mono />
        <Detail label="No Polis" value={data.nomor_polis} mono />
        <Detail label="No Klaim" value={data.nomor_klaim} mono />
        <Detail label="Nama Tertanggung" value={data.nama_tertanggung} />
        <Detail
          label="Start Date Time"
          value={data.polis_mulai ? formatDate(data.polis_mulai) : ''}
        />
        <Detail
          label="End Date Time"
          value={data.polis_akhir ? formatDate(data.polis_akhir) : ''}
        />
        <Detail
          label="Tipe Proteksi"
          value={protectionTypeLabel(data.tipe_proteksi, data.nama_tipe_proteksi)}
        />
        <Detail
          label="Tanggal Input"
          value={
            data.tanggal_proteksi ? formatDate(data.tanggal_proteksi) : ''
          }
        />
        <Detail label="User Create" value={data.user_create} />
        <div className="sm:col-span-2 lg:col-span-3">
          <Detail label="Keterangan" value={data.keterangan} />
        </div>
      </dl>

      {data.detail_perubahan && (
        <ChangeDetailPanel detail={data.detail_perubahan} />
      )}

      {/*
        Dokumen penunjang menempel pada KLAIM, bukan pada permintaan proteksinya —
        karena itu yang diserahkan nomor klaimnya.

        Permintaan yang sudah diputuskan menjadi baca-saja: dokumennya tetap perlu
        dibuka untuk ditinjau, tetapi menambah berkas pada permintaan yang sudah
        selesai tidak lagi bermakna.
      */}
      <DokumenPenunjangPanel
        nomorKlaim={data.nomor_klaim}
        readOnly={!data.menunggu_keputusan}
      />

      <div className="mt-5 flex flex-wrap items-center gap-3">
        {data.menunggu_keputusan ? (
          <>
            <Button
              tone="utama"
              onClick={() => { onDecide('setuju') }}
              disabled={deciding}
            >
              {deciding ? 'Menyimpan…' : 'Setujui'}
            </Button>
            <Button
              tone="kedua"
              onClick={() => { onDecide('tolak') }}
              disabled={deciding}
            >
              Tolak
            </Button>
          </>
        ) : (
          // Sudah diputuskan: tombolnya tidak ditampilkan sama sekali, dan yang
          // ditampilkan adalah keputusannya beserta pelakunya. Menampilkan tombol
          // yang pasti ditolak hanya membuat pengguna mencobanya.
          <p className="text-sm text-slate-600">
            Sudah{' '}
            <span className="font-medium">
              {data.status_akseptasi === '1' ? 'disetujui' : 'ditolak'}
            </span>
            {data.diaksep_oleh && <> oleh {data.diaksep_oleh}</>}
            {data.tanggal_akseptasi && (
              <> pada {formatDate(data.tanggal_akseptasi)}</>
            )}
            .
          </p>
        )}

        <Button tone="halus" onClick={onClose} disabled={deciding}>
          Tutup
        </Button>
      </div>
    </>
  )
}
