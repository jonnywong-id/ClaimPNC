import { useMemo } from 'react'

import { SparepartStatus, type Sparepart } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useApprovalTabs } from '@/components/masterpage/useApprovalTabs'
import { useCrudForm } from '@/components/masterpage/useCrudForm'
import { ApprovalPanel } from '@/components/masterpage/ApprovalPanel'
import {
  AddButton,
  ListHeader,
  RefreshButton,
  editColumn,
} from '@/components/masterpage/MasterPage'
import { DataTable, type Column } from '@/components/DataTable'
import { selectColumn } from '@/components/ApprovalControls'

import { useCreateSparepart, useDecideSparepart, useSaveSparepart, useSparepartList } from './api'
import { SparepartForm } from './SparepartForm'
import { collectKnownValues } from '@/components/masterpage/knownValues'

/**
 * Tiga tab, sama persis dengan layar lama — termasuk URUTANNYA.
 *
 * `Section/BrowseMasterSparepartHE-Section.xml` memuat tiga section yang ketiganya membaca
 * `BrowseSparepartHE_RD` yang sama dan hanya berbeda pada nilai APPROVAL-nya. Urutan
 * kemunculannya di dalam section itu:
 *
 *	BrowseMasterSparepartHEApprove   APPROVAL="1"
 *	BrowseMasterSparepartHEReject    APPROVAL="2"
 *	BrowseMasterSparepartHEApproval  APPROVAL="0"
 *
 * Reject berada di TENGAH, bukan di ujung — sama seperti Master Panel dan Master Bengkel.
 * Urutan itu tidak intuitif, tetapi ia yang dilihat petugas hari ini, dan `D-13` menuntut
 * tata letak yang sama supaya pengguna tidak perlu belajar ulang.
 */
const TABS = [
  {
    id: 'approve',
    label: 'Approve',
    status: SparepartStatus.disetujui,
    description: 'Sparepart yang sudah disetujui dan berlaku.',
  },
  {
    id: 'reject',
    label: 'Reject',
    status: SparepartStatus.ditolak,
    description: 'Pengajuan yang ditolak. Dapat diperbaiki lalu diajukan ulang.',
  },
  {
    id: 'menunggu',
    label: 'Waiting Approval',
    status: SparepartStatus.menunggu,
    description:
      'Pengajuan dan perubahan yang belum diputuskan. Centang barisnya untuk menyetujui atau menolak.',
  },
] as const

/**
 * Ukuran halaman diambil dari `pyPageSize` pada ketiga section tab Master Sparepart.
 *
 * Ketiganya bernilai **30** — berbeda dari Master Panel yang 15 pada satu tab dan 50 pada
 * dua tab lainnya, dan dari Master Bengkel yang 20. Tidak ada satu angka yang benar untuk
 * seluruh layar; angkanya milik layar, bukan milik komponen tabel.
 */
const PAGE_SIZE = 30

/** Kolom penanda yang nilai sahnya tidak ada di export; pilihannya dikumpulkan dari data. */
const MARK_COLUMNS = [
  'jenis_sparepart',
  'satuan',
  'status_aktif',
  'status_sparepart',
] as const

/**
 * Kalimat yang mengisi badan grid ketika tidak ada satu pun baris yang tergambar.
 *
 * # Satu kalimat untuk SETIAP keadaan nol baris
 *
 * Grid Pega menggambar kepala kolomnya beserta satu pesan di bawahnya saat hasilnya nol,
 * dan pesannya berbunyi "data tidak ada". Layar ini mengikutinya apa adanya (`D-13`):
 * berhasil-tetapi-kosong dan gagal-dibaca memakai kalimat yang SAMA.
 *
 * Teksnya sendiri TIDAK dapat dibaca dari export — grid lama mengambilnya dari field value
 * `GridNoResultsOnLoad`, dan tidak ada satu pun direktori `Field Value/` di sana (`R-16`).
 * Ia datang dari Work Owner yang membaca layar Pega sungguhan (2026-09-24). Tanpa nama tab,
 * persis seperti Pega: pesannya sama di ketiga tab.
 *
 * # Yang hilang karenanya, dan di mana menggantinya
 *
 * Layar TIDAK LAGI membedakan "tabelnya memang kosong" dari "tabelnya gagal dibaca".
 * Keduanya tampil identik. Pada saat tulisan ini dibuat, view `POOLDATA.SPAREPART_HE`
 * sedang rusak di Oracle (`ORA-04063`), dan layar ini menampilkannya sebagai data kosong.
 *
 * Keputusan Work Owner, diambil setelah akibatnya disampaikan tiga kali (2026-09-24). Yang
 * menggantikan pembedaan itu ada di dua tempat yang TIDAK dilihat pengguna:
 *
 *	log backend          setiap kegagalan tercatat lengkap dengan galat Oracle-nya
 *	claimpnc -periksa    menyebut objek dan galatnya, beserta kueri katalog penjawabnya
 *
 * Satu pengecualian yang dipertahankan: sebelum portal dipilih, kuerinya belum pernah
 * dijalankan sama sekali — tidak ada "hasil nol" untuk dilaporkan, dan yang dibutuhkan
 * pengguna adalah petunjuk tindakan, bukan keterangan data.
 */
function emptyMessageFor(hasPortal: boolean): string {
  if (!hasPortal) {
    return 'Pilih portal entitas di bagian atas halaman untuk menampilkan daftarnya.'
  }
  return 'Data tidak ada'
}

/**
 * Layar Master Sparepart.
 *
 * Pengganti `Harness/SparePart_HE-Harness.xml` atas tabel POOLDATA.SPAREPART_HE
 * (MENU_ID 31).
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **suku cadang alat berat** beserta harga jual, dimensi, batas stok, dan
 * penggolongannya. Setiap sparepart menunjuk satu Kategori dan satu Tipe yang dibaca dari
 * dua tabel acuan.
 *
 * # Lima kolom, bukan dua puluh tiga
 *
 * Grid Pega hanya menampilkan ID, Nama, Harga Jual, User Update, dan Tanggal Update; sisanya
 * hanya terlihat saat sebuah baris dibuka. Susunan itu ditiru apa adanya atas keputusan Work
 * Owner (2026-09-20), termasuk tidak menambahkan kolom yang menurut kami berguna.
 *
 * Satu kolom Pega TIDAK digambar: sel yang di layar lama terikat pada `.TELP_BENGKEL` —
 * properti yang tidak ada di SPAREPART_HE, jadi selalu kosong. Ia sisa salin-tempel dari
 * grid Master Bengkel, dan menggambar kolom yang selalu kosong bukan kesetaraan melainkan
 * peniruan cacat.
 *
 * # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
 *
 * Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
 * `Activity/UpdateSparepartHE_act` menetapkan `APPROVAL := "0"` tanpa syarat apa pun.
 *
 * # Kenapa Approve dan Reject ada DI SINI, bukan di Inbox Manager
 *
 * Di Pega keduanya ada di layar lain: `Section/ApprovalMasterSparepartHE` dipakai Inbox
 * Manager, dan keputusannya dijalankan `Activity/SetApprovalAllMaster` yang melayani
 * bengkel, panel, dan sparepart sekaligus.
 *
 * Inbox Manager belum dibangun. Menunda keputusannya sampai layar itu ada berarti setiap
 * sparepart yang ditambah tertahan di Waiting Approval tanpa satu pun cara menyelesaikannya.
 * Yang dipakai sebagai gantinya adalah BENTUK yang sama persis: centang beberapa baris, lalu
 * satu tombol untuk seluruh pilihan. Perlakuannya sama dengan Master Panel dan Master
 * Bengkel.
 *
 * # Tanpa isian Catatan, berbeda dari Master Panel
 *
 * `POOLDATA.SPAREPART_HE` tidak punya kolom penampung alasan penolakan — kedua puluh tiga
 * kolomnya terbaca lengkap dari `BrowseSparepartHE_RD`, dan tidak satu pun menampungnya.
 * Menggambar isian yang diam-diam membuang isinya lebih buruk daripada tidak menggambarnya.
 */
export function SparepartPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const { tab, setTab, active, chosen, setChosen, toggle } = useApprovalTabs(TABS, 'approve')

  const list = useSparepartList(active.status)
  const create = useCreateSparepart()
  const save = useSaveSparepart()
  const form = useCrudForm(create, save, (row: Sparepart, input) => ({
    id: row.id_sparepart,
    input,
  }))
  const editing = form.openedRow
  const isFormOpen = form.isOpen
  const { closeForm, openCreate: openAdd, openEdit } = form
  const decide = useDecideSparepart()

  const rows = list.data?.sparepart ?? []

  /*
    Pilihan nilai untuk keempat penanda yang daftar pilihannya tidak ada di export (R-16).

    Dikumpulkan dari baris yang SEDANG TERMUAT, bukan dari daftar yang dikarang. Ia jawaban
    terbaik yang tersedia atas pertanyaan "nilai apa yang sah di kolom ini" — lihat
    ChoiceField.
  */
  const knownValues = useMemo(() => collectKnownValues(rows, MARK_COLUMNS), [rows])

  function runDecision(status: string) {
    decide.mutate(
      { id_sparepart: [...chosen], status },
      { onSuccess: () => setChosen(new Set()) },
    )
  }

  /*
    Susunan kolom mengikuti grid Pega APA ADANYA — kelimanya berdampingan pada urutan yang
    sama.

    Dua kolomnya di layar lama TIDAK punya judul sama sekali: selnya bertuliskan
    `.HARGA_JUAL` dan `.TGL_UPDATE_HARGA`, yakni nama propertinya sendiri yang bocor ke
    layar. Judulnya di sini diisi kata yang benar — "Harga Jual" dan "Tanggal Update" —
    karena nama properti yang bocor bukan tata letak yang layak ditiru, melainkan cacat.
  */
  const columns: Column<Sparepart>[] = [
    ...selectColumn<Sparepart>({
      enabled: tab === 'menunggu',
      chosen,
      idOf: (row) => row.id_sparepart,
      nameOf: (row) => row.nama_sparepart,
      disabled: decide.isPending,
      onToggle: toggle,
    }),
    {
      key: 'id',
      title: 'ID Sparepart',
      width: '9rem',
      value: (row) => row.id_sparepart,
    },
    {
      key: 'nama',
      title: 'Nama Sparepart',
      width: '16rem',
      value: (row) => row.nama_sparepart,
    },
    {
      key: 'harga',
      title: 'Harga Jual',
      width: '10rem',
      alignRight: true,
      // Yang dicari dan diurutkan adalah teks aslinya, sedangkan yang dilihat pengguna
      // adalah bentuk berpemisah ribuan. Mengurutkan bentuk terformat akan menaruh
      // "1.000.000" sebelum "900.000".
      value: (row) => row.harga_jual,
      render: (row) => (
        <span className="tabular-nums text-sm text-slate-900">{rupiah(row.harga_jual)}</span>
      ),
    },
    {
      key: 'user',
      title: 'User Update',
      width: '10rem',
      value: (row) => row.user_update,
      render: (row) => (
        <span className="text-sm text-slate-900">{row.user_update || '—'}</span>
      ),
    },
    {
      key: 'tanggal',
      title: 'Tanggal Update',
      width: '12rem',
      value: (row) => row.tanggal_update_harga ?? '',
      render: (row) => (
        <span className="text-sm text-slate-900">{tanggalWIB(row.tanggal_update_harga)}</span>
      ),
    },
    editColumn<Sparepart>({
      onEdit: openEdit,
      width: '7rem',
      tone: 'halus',
      disabled: save.isPending,
    }),
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      {/* Judulnya dibaca dari `pyCaption Master Sparepart HE` pada
          Harness/SparePart_HE — "HE" ikut, karena itulah yang tertulis di layar lama
          (D-13). */}
      <ListHeader
        title="Master Sparepart HE"
        description="Suku cadang alat berat beserta harga jual, dimensi, dan batas stoknya."
      >
        <AddButton onClick={openAdd} disabled={isFormOpen} />
        <RefreshButton query={list} />
      </ListHeader>

      {/*
        Kedua tombol unggah layar Pega — "Upload Document" dan "Upload Data Master Sparepart"
        (`pyButtonLabel` pada Section/BrowseMasterSparepartHE) — SENGAJA TIDAK DIGAMBAR.

        Keduanya memanggil local action `UploadDocument` dan `PNCUploadMasterSparepartCSV`;
        tidak satu pun ada di export (`R-16`), sehingga susunan kolom CSV-nya, validasinya,
        dan — yang paling menentukan — apakah baris hasil unggah masuk antrean persetujuan,
        seluruhnya tidak diketahui.

        Sempat digambar dalam keadaan mati supaya ketiadaannya terbaca dari layar. Work Owner
        memilih menariknya sama sekali (2026-09-24), menyamakannya dengan Master Panel (§28);
        lihat docs/keputusan-implementasi.md §30.9.
      */}
      <ApprovalPanel
        tabLabel="Tab Master Sparepart"
        tabs={TABS}
        active={active}
        onSwitch={setTab}
        onBeforeSwitch={closeForm}
        portal={list.data?.portal ?? portal}
        decide={decide}
        feedbackNoun="sparepart"
        barNoun="sparepart"
        chosenCount={chosen.size}
        onApprove={() => runDecision(SparepartStatus.disetujui)}
        onReject={() => runDecision(SparepartStatus.ditolak)}
        onClear={() => setChosen(new Set())}
      />

      {isFormOpen && (
        <section className="mt-5">
          <SparepartForm
            editing={editing}
            knownValues={knownValues}
            isSaving={form.isSaving}
            error={form.saveError}
            onSave={form.submit}
            onCancel={closeForm}
          />
        </section>
      )}

      {/*
        Tabelnya digambar dalam SETIAP keadaan — termuat, kosong, gagal, bahkan sebelum
        portal dipilih. Kolomnya karena itu selalu terlihat, persis seperti grid Pega yang
        menggambar kepala kolomnya beserta pyGridNoResultsMessage di bawahnya.

        Tidak ada lagi kotak galat yang menggantikan tabelnya: keterangan apa pun tinggal di
        dalam grid lewat emptyMessage. Lihat emptyMessageFor untuk alasan gagal dan kosong
        tetap dibedakan kalimatnya.
      */}
      <section className="mt-6">
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.id_sparepart}
          description="Sumber: POOLDATA.SPAREPART_HE"
          searchLabel="Cari sparepart"
          pageSize={PAGE_SIZE}
          isLoading={portal !== null && list.isPending}
          showHeaderWhenEmpty
          emptyMessage={emptyMessageFor(portal !== null)}
        />
      </section>
    </main>
  )
}

/**
 * rupiah menampilkan nilai harga dengan pemisah ribuan.
 *
 * Nilai yang BUKAN angka ditampilkan apa adanya, tidak dikosongkan: baris lama dapat memuat
 * apa saja di kolom itu — sistem lama tidak pernah memeriksanya — dan menyembunyikannya
 * berarti petugas tidak punya cara mengetahui bahwa isinya perlu diperbaiki.
 */
function rupiah(value: string): string {
  const clean = value.trim()
  if (clean === '') return '—'

  const number = Number(clean.replace(',', '.'))
  if (!Number.isFinite(number)) return clean

  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(number)
}

/**
 * tanggalWIB menampilkan stempel UTC dari server dalam waktu Jakarta.
 *
 * Konversi terjadi DI SINI, di satu tempat, lewat Intl — tidak ada satu pun penambahan 7 jam
 * manual (`F-5`, `08-TECHNICAL-STRATEGY.md` §4.4).
 */
function tanggalWIB(value: string | undefined): string {
  if (!value) return '—'

  const at = new Date(value)
  if (Number.isNaN(at.getTime())) return value

  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Asia/Jakarta',
  }).format(at)
}
