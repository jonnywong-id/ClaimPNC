import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { AutoClaimStatus, ErrorCode, type AutoClaim } from '@/api/types'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'
import { useSelectedPortal } from '@/app/portal'

import { useAutoClaimList, useSaveAutoClaim } from './api'
import { AutoClaimForm, type AutoClaimFormValues } from './AutoClaimForm'
import { useCreateAutoClaim } from './api'

/**
 * Empat tab, sama persis dengan layar lama.
 *
 * Harness Pega menyusunnya sebagai empat section terpisah yang isinya nyaris sama —
 * `BrowseAutoKlaim`, `BrowseAutoKlaimKomite`, `BrowseAutoKlaimApproval`, dan
 * `BrowseAutoKlaimReject` — dan karena itu berbeda-beda di tempat yang tidak disengaja
 * (yang satu punya kolom KOMITE, yang lain tidak; yang satu punya tombol tambah, yang
 * lain tidak). Di sini keempatnya satu layar dengan saringan berbeda.
 *
 * # Caption dan URUTANNYA dibaca dari `pyTitle` tiap container, bukan dikarang
 *
 * `Section/MasterAutoKlaim-Section.xml` adalah layout group ber-`pyHeaderType=TABBED`,
 * `pyTabAlignment=Top`, `pyStretchTab=false`. Keempat tabnya muncul pada urutan dokumen
 * berikut, dan caption inilah yang dilihat pengguna:
 *
 *	offset   pyTitle            section dibawahnya
 *	 84.789  Approve            BrowseAutoKlaim
 *	122.544  Reject             BrowseAutoKlaimReject
 *	164.383  Waiting Approval   BrowseAutoKlaimApproval
 *	202.564  Komite Approval    BrowseAutoKlaimKomite
 *
 * **"Master Auto Klaim" BUKAN nama tab** — ia judul layar, sebaris dengan tombol Tambah
 * dan Refresh di atas keempat tab. Versi pertama layar ini keliru menjadikannya caption
 * tab pertama, sehingga tab "Approve" hilang sama sekali dan urutannya ikut bergeser.
 *
 * Keempat saringannya dibaca langsung dari `pyDeferLoadRetrievalActivityParams` tiap
 * section, bukan ditebak:
 *
 *	BrowseAutoKlaim          stsapprove="1"
 *	BrowseAutoKlaimKomite    stsapprove="0"  komite="ya"
 *	BrowseAutoKlaimApproval  stsapprove="0"
 *	BrowseAutoKlaimReject    stsapprove="2"
 *
 * # Ketiga perbedaan antartab, dan semuanya nyata di Pega
 *
 * Keempat section itu BUKAN salinan yang sama. Perbedaannya dibaca dari header grid dan
 * dari tombol yang benar-benar ada di tiap section:
 *
 *	tab               kolom  tombol baris  tombol form
 *	Approve             9     Update        Simpan            (stsapprove="0")
 *	Reject              9     Update        Simpan            (stsapprove="0")
 *	Waiting Approval    9     Update        —                 (tidak ada sama sekali)
 *	Komite Approval     8     Update        Approve · Reject  (stsapprove="1" / "2")
 *
 * Tab Komite tidak punya kolom KOMITE karena seluruh barisnya memang milik pemanggil,
 * dan tab Waiting Approval tidak punya tombol simpan apa pun — ia baca saja.
 *
 * # Satu nama yang muncul dua kali, dan itu memang begitu di Pega
 *
 * "Approve" adalah caption TAB PERTAMA sekaligus caption TOMBOL pada form tab Komite.
 * Keduanya memang bernama sama di sistem lama. Uji yang mencarinya WAJIB menyebut
 * lingkupnya — `within(form)` atau `within(table)` — karena pencarian global akan
 * menemukan keduanya. Kekeliruan yang sama sudah pernah menjatuhkan uji Master Rekening.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: AutoClaimStatus.disetujui,
    committeeOnly: false,
    /** Kolom KOMITE ikut ditampilkan. */
    showCommittee: true,
    /** Form menyimpan perubahan dengan status "0". */
    canSave: true,
    /** Form menampilkan Approve dan Reject. */
    canDecide: false,
    description: 'Sumber bisnis yang sudah disetujui komite dan dipakai pembuatan klaim otomatis.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: AutoClaimStatus.ditolak,
    committeeOnly: false,
    showCommittee: true,
    canSave: true,
    canDecide: false,
    description: 'Pengajuan yang ditolak komite. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: AutoClaimStatus.menunggu,
    committeeOnly: false,
    showCommittee: true,
    canSave: false,
    canDecide: false,
    description:
      'Seluruh pengajuan yang belum diputuskan, siapa pun penyetujunya. Baca saja — yang memutuskan adalah komite yang ditunjuk.',
  },
  {
    id: 'komite',
    label: 'Komite Approval',
    status: AutoClaimStatus.menunggu,
    committeeOnly: true,
    showCommittee: false,
    canSave: false,
    canDecide: true,
    description: 'Menunggu keputusan Anda sebagai komite.',
  },
] as const

type TabId = (typeof TABS)[number]['id']

type MessageContent = { title: string; description: string; tone: ErrorTone }

function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan Anda, lalu muat ulang.',
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
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
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
          title: 'Daftar tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return { title: 'Daftar tidak dapat dimuat', description: 'Coba beberapa saat lagi.', tone: 'gangguan' }
}

/**
 * Layar Master Auto Claim.
 *
 * Pengganti `Harness/AutoKlaim-Harness.xml` atas tabel POOLDATA.M_AUTO_CLAIM_PNC.
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **Sumber Bisnis yang klaimnya boleh dibuat otomatis**, beserta ke mana ganti
 * ruginya dibayarkan. Barisnya dipakai `RDB List/GetReceiverClaimAsuransiKredit-SQL.xml`
 * saat klaim asuransi kredit dibuat otomatis, dengan penyaring
 * `claim_allowed = 1 AND APPROVAL = '1'`.
 *
 * Syarat kedua itu tidak terlihat di kolom mana pun — sama seperti di Pega. Ia
 * dilaporkan sebagai CATATAN DI ATAS TABEL bila memang ada barisnya, bukan sebagai kolom
 * tambahan: grid ini mengikuti Pega kolom per kolom (`D-13`), dan keterangan yang tidak
 * ada di sana tidak boleh menyelinap masuk sebagai kolom.
 *
 * # Menyimpan dan memutuskan adalah satu operasi
 *
 * Karena di sistem lama memang satu: `Activity/UpdateMstAutoClaim_act` melayani tombol
 * Update, Approve, dan Reject sekaligus, dibedakan hanya oleh parameter `stsapprove`.
 * Keputusan Work Owner 2026-09-19 mempertahankan bentuk itu.
 *
 * Akibatnya yang paling terasa bagi pengguna: **menyunting baris yang sudah disetujui
 * mengembalikannya ke antrean persetujuan.** Itu bukan efek samping melainkan perilaku
 * yang benar untuk master yang menentukan ke mana uang dikirim.
 */
export function AutoClaimPage() {
  const portal = useSelectedPortal((state) => state.alias)

  // Tab pertama Pega adalah "Approve", bukan judul layarnya.
  const [tab, setTab] = useState<TabId>('approve')
  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<AutoClaim | null>(null)

  const active = TABS.find((t) => t.id === tab) ?? TABS[0]
  const list = useAutoClaimList(active.status, active.committeeOnly)
  const create = useCreateAutoClaim()
  const save = useSaveAutoClaim()

  const isFormOpen = isAdding || editing !== null

  function closeForm() {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(null)
  }

  function openAdd() {
    create.reset()
    save.reset()
    setEditing(null)
    setAdding(true)
  }

  function openEdit(row: AutoClaim) {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(row)
  }

  /**
   * submit melayani KETIGA tombol form — Simpan, Approve, dan Reject.
   *
   * Statusnya datang dari tombol yang ditekan, bukan dari isian: itu bentuk sistem lama,
   * tempat `UpdateMstAutoClaim_act` dibedakan hanya oleh parameter `stsapprove`.
   *
   *	Simpan   "0"  perubahan mengembalikan baris ke antrean persetujuan
   *	Approve  "1"
   *	Reject   "2"
   *
   * Pada penambahan, status diabaikan — baris baru selalu lahir menunggu.
   */
  function submit(values: AutoClaimFormValues, status: string) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal.
    if (editing) {
      save.mutate(
        {
          inisial: editing.inisial,
          input: {
            nama_bank: values.nama_bank,
            no_rekening: values.no_rekening,
            pct_max: values.pct_max,
            pic_lapor: values.pic_lapor,
            email_lapor: values.email_lapor,
            alamat_penerima: values.alamat_penerima,
            id_client: values.client?.id ?? '',
            nama_client: values.client?.nama ?? '',
            status,
          },
        },
        { onSuccess: closeForm },
      )
      return
    }

    // Tombol Simpan dinonaktifkan sampai sumber bisnis dipilih, sehingga cabang ini
    // tidak tercapai lewat layar. Ia tetap ada supaya tipe-nya tidak dipaksakan.
    if (!values.sumberBisnis) return

    create.mutate(
      {
        inisial: values.sumberBisnis.id,
        nama_penerima: values.sumberBisnis.nama,
        nama_bank: values.nama_bank,
        no_rekening: values.no_rekening,
        pct_max: values.pct_max,
        pic_lapor: values.pic_lapor,
        email_lapor: values.email_lapor,
        alamat_penerima: values.alamat_penerima,
        id_client: values.client?.id ?? '',
        nama_client: values.client?.nama ?? '',
      },
      { onSuccess: closeForm },
    )
  }

  // Baris yang DISETUJUI tetapi CLAIM_ALLOWED-nya bukan "1". Ia tidak pernah dipakai
  // pembuatan klaim otomatis, dan tidak ada satu pun kolom — di sini maupun di Pega —
  // yang menjelaskannya. Dihitung untuk catatan di atas tabel.
  const blockedRows = (list.data?.auto_claim ?? []).filter(
    (row) => row.status === AutoClaimStatus.disetujui && !row.dapat_dipakai,
  )

  // Susunan kolom mengikuti header grid Pega APA ADANYA — satu kolom Pega menjadi satu
  // kolom di sini, tanpa penggabungan. Terbaca dari
  // `Section/BrowseAutoKlaim-Section.xml`, beserta lebar pikselnya:
  //
  //	<b>INISIAL<b>          .CaseID           INISIALID        72px
  //	<b>NAMA PENERIMA<b>    .City             NAMA_PENERIMA   177px
  //	<b>BANK PENERIMA<b>    .CityID           BANK_PENERIMA   167px
  //	<b>NO REKENING<b>      .District         NO_REKENING     105px
  //	<b>ALAMAT PENERIMA<b>  .CountryID        ALAMAT_PENERIMA 139px
  //	<b>EMAIL LAPOR<b>      .DistrictID       EMAIL_LAPOR     163px
  //	<b>PCT MAX<b>          .Country          PCT_MAX          60px
  //	<b>PIC<b>              .AlasanTerlambat  PIC_LAPOR        90px
  //	<b>KOMITE<b>           .ProdKe           KOMITE          115px
  //	(tanpa judul)                            tombol Update    93px
  //
  // Perhatikan alias-aliasnya: tidak satu pun mencerminkan isinya, dan `District`
  // (NO_REKENING) tidak berpasangan dengan `DistrictID` (EMAIL_LAPOR). Yang dipetakan
  // adalah kolomnya, bukan aliasnya (D-19).
  //
  // KOLOM KOMITE hanya muncul pada tab yang memilikinya di Pega. Tab Komite Approval
  // tidak punya — seluruh barisnya memang milik pemanggil, sehingga kolomnya tidak
  // memberi tahu apa pun.
  //
  // KOLOM AKSI TANPA JUDUL, sama seperti Pega. Sel judul yang kosong tidak memberi nama
  // pada kolomnya bagi pembaca layar; yang menutupinya adalah tombol di dalam tiap sel,
  // yang bernama "Update" dan menyebut inisial barisnya.
  //
  // CLIENT ID dan NAMA CLIENT TIDAK ada di grid ini, dan itu benar: keempat header yang
  // memuatnya (`INISIAL`, `SUMBER BISNIS`, `CLIENT ID`, `NAMA CLIENT`) milik dua grid
  // pencarian DI DALAM FORM, bukan grid daftar. Keduanya tetap dibaca API — lihat
  // `auto_claim_list` pada berkas .sql.
  const columns: Column<AutoClaim>[] = [
    { key: 'inisial', title: 'Inisial', width: '72px', value: (row) => row.inisial },
    { key: 'nama_penerima', title: 'Nama penerima', width: '177px', value: (row) => row.nama_penerima },
    { key: 'nama_bank', title: 'Bank penerima', width: '167px', value: (row) => row.nama_bank },
    { key: 'no_rekening', title: 'No rekening', width: '105px', value: (row) => row.no_rekening },
    { key: 'alamat_penerima', title: 'Alamat penerima', width: '139px', value: (row) => row.alamat_penerima },
    { key: 'email_lapor', title: 'Email lapor', width: '163px', value: (row) => row.email_lapor },
    { key: 'pct_max', title: 'PCT max', width: '60px', value: (row) => row.pct_max },
    { key: 'pic_lapor', title: 'PIC', width: '90px', value: (row) => row.pic_lapor },
    ...(active.showCommittee
      ? [{ key: 'komite', title: 'Komite', width: '115px', value: (row: AutoClaim) => row.komite }]
      : []),
    {
      key: 'aksi',
      title: '',
      width: '93px',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button tone="halus" onClick={() => openEdit(row)} disabled={save.isPending}>
          Update
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/*
            "Klaim" dengan K, bukan "Claim" — itu teks judul pada
            `Section/MasterAutoKlaim-Section.xml`, dan `D-80` menetapkan teks yang
            dilihat pengguna mengikuti layar Pega apa adanya.

            Butir MENUNYA di kolom samping tetap "Master Auto Claim" dengan C, karena ia
            datang dari `POOLDATA.M_MENU_APLIKASI_PNC.MENU_DESC`. Kedua ejaan itu memang
            berbeda di sistem lama; keduanya direplikasi dari sumbernya masing-masing,
            bukan diseragamkan sepihak.
          */}
          <h1 className="text-xl font-semibold text-slate-900">Master Auto Klaim</h1>
          <p className="text-sm text-slate-600">
            Sumber bisnis yang klaimnya boleh dibuat otomatis, beserta rekening tujuan
            pembayarannya.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            Tambah
          </Button>
        </div>
      </header>

      <nav
        aria-label="Tab Master Auto Klaim"
        className="mt-4 flex flex-wrap gap-1 border-b border-slate-200"
      >
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            aria-current={tab === t.id ? 'page' : undefined}
            onClick={() => {
              closeForm()
              setTab(t.id)
            }}
            className={[
              'rounded-t px-3 py-2 text-sm font-medium',
              'transition-colors duration-150 ease-halus',
              'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50',
              tab === t.id
                ? 'border-b-2 border-blue-600 text-blue-700'
                : 'text-slate-500 hover:text-slate-800',
            ].join(' ')}
          >
            {t.label}
          </button>
        ))}
      </nav>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani
          empat badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh
          hanya diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        {active.description}{' '}
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {save.isError && !isFormOpen && (
        <div className="mt-4">
          <ErrorMessage
            title="Keputusan belum tersimpan"
            description={
              save.error instanceof APIError
                ? save.error.message
                : 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.'
            }
            tone="gangguan"
          />
        </div>
      )}

      {/* Catatan ini ADA DI LUAR GRID, dan itu disengaja: grid mengikuti Pega kolom per
          kolom, sehingga keterangan yang tidak ada di sana tidak boleh menyelinap masuk
          sebagai kolom tambahan. Ia hanya muncul bila barisnya memang ada. */}
      {blockedRows.length > 0 && (
        <div className="mt-4">
          <ErrorMessage
            title={`${blockedRows.length} baris disetujui tetapi tidak dipakai klaim otomatis`}
            description={`CLAIM_ALLOWED pada baris ${blockedRows
              .map((row) => row.inisial)
              .join(', ')} bukan “1”, sehingga tidak lolos penyaring pembuatan klaim otomatis. Menyimpan ulang barisnya akan membetulkannya.`}
            tone="penolakan"
          />
        </div>
      )}

      {isFormOpen && (
        <section className="mt-5">
          <AutoClaimForm
            editing={editing}
            isSaving={create.isPending || save.isPending}
            error={editing ? save.error : create.error}
            // Ketiga tombol form mengikuti tab, persis seperti Pega: Simpan hanya ada di
            // tab yang punya `stsapprove="0"`, Approve/Reject hanya di tab Komite, dan
            // Waiting Approval tidak punya satu pun.
            //
            // Penambahan SELALU dapat disimpan, di tab mana pun tombol Tambah ditekan —
            // tombol itu sendiri berada di tingkat harness, di atas keempat tab.
            canSave={editing === null || active.canSave}
            canDecide={editing !== null && active.canDecide}
            onSave={submit}
            onCancel={closeForm}
          />
        </section>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar auto claim…</p>
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
          <DataTable
            columns={columns}
            rows={list.data.auto_claim}
            rowKey={(row) => row.inisial}
            description="Sumber: POOLDATA.M_AUTO_CLAIM_PNC"
            emptyMessage={`Belum ada baris pada tab ${active.label}.`}
          />
        )}
      </section>
    </main>
  )
}
