import type { SurveyorLogin } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { reloadLoadMessage, type MessageContent } from '@/components/masterpage/loadMessage'
import { useCrudForm } from '@/components/masterpage/useCrudForm'
import {
  AddButton,
  ListHeader,
  RefreshButton,
  renderListState,
} from '@/components/masterpage/MasterPage'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'

import {
  useCreateSurveyorLogin,
  useSaveSurveyorLogin,
  useSurveyorLoginList,
} from './api'
import { SurveyorLoginForm } from './SurveyorLoginForm'

/**
 * Ukuran halaman diambil dari `pyPageSizeOther` pada grid
 * `Section/BrowseLoginSurveyor-Section.xml` (`pyPageSize = "Other"`).
 *
 * **15**, sama dengan Master Rekening, Master Status Klaim, Master Status Progres, Master
 * Pasal Kerugian, dan Master Penolakan Klaim — dan berbeda dari Supplier, Bengkel, serta
 * Panel yang memakai 20. Tidak ada satu angka yang benar untuk seluruh layar; angkanya
 * milik layar, bukan milik komponen tabel.
 *
 * Satu selisih yang disadari: gridnya memakai `pyPageMode = "Next Previous"`, sedangkan
 * `DataTable` menomori halamannya. Menomori lebih mudah dipakai pada daftar yang panjang,
 * dan seluruh layar master lain di aplikasi ini sudah memakainya — memperkenalkan mode
 * kedua di satu layar akan membuat dua gaya paginasi hidup berdampingan tanpa alasan.
 */
const PAGE_SIZE = 15

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  return reloadLoadMessage(error, { failedTitle: 'Daftar login surveyor tidak dapat dimuat' })
}

/**
 * Layar Master Login.
 *
 * Pengganti `Harness/MasterLoginSurvey-Harness.xml` atas tabel
 * POOLDATA.MST_LOGIN_SURVEYOR (MENU_ID 37).
 *
 * # Apa yang dikelola layar ini
 *
 * **Daftar login surveyor** — siapa saja yang terdaftar bekerja sebagai surveyor pada satu
 * entitas, beserta surel, telepon, dan alamatnya.
 *
 * # Layar master paling sederhana di aplikasi ini, dan itu bukan kebetulan
 *
 * TANPA tab, TANPA persetujuan, TANPA penghapusan, dan TANPA tabel acuan. Ketiga rule SQL
 * yang menyentuh tabelnya menyebut tujuh kolom yang sama, dan tidak satu pun berupa
 * APPROVAL, pencatat pelaku, stempel waktu, maupun penanda aktif.
 *
 * Akibat yang harus disadari, dan yang dinyatakan di layar alih-alih ditutupi: **tidak ada
 * cara menyatakan sebuah login sudah tidak berlaku.** Bukan lewat penghapusan — `D-66`
 * melarangnya, dan sistem lama pun tidak punya — dan bukan lewat penonaktifan, karena
 * kolomnya memang tidak ada.
 *
 * # Login diturunkan dari Nama, dan itu menyetir seluruh layar
 *
 * `Activity/SetLoginSurveyor_act` membentuknya dengan membuang spasi, titik, koma, dan
 * tanda hubung. LOGIN itulah kunci barisnya — setiap pernyataan simpan menyaring
 * `where login = ...` — sehingga Nama TERKUNCI setelah baris tersimpan.
 *
 * # Akun aplikasi TIDAK diterbitkan, dan itu dinyatakan terang-terangan
 *
 * Di Pega, menyimpan baris baru ikut memanggil `GCNMCreateOperator` untuk menerbitkan akun
 * operator, dengan kata sandi yang sama untuk setiap orang, dan pesan suksesnya menyebut
 * kata sandi itu. Tiga hal menghalanginya di sini: tidak ada operator Pega untuk
 * diterbitkan, kontrak identitas `F-3` belum ada (`R-14`), dan mengumumkan kata sandi bagi
 * akun yang tidak diterbitkan adalah keterangan yang salah — petugas akan menyampaikannya
 * kepada surveyor yang kemudian tidak dapat masuk.
 *
 * Perlakuan yang sama dipakai Master Bengkel, yang menghadapi `GCNMCreateOperator` yang
 * sama persis.
 *
 * # Cakupan daftarnya adalah REKONSTRUKSI
 *
 * Rule yang mengisi grid Pega tidak ada di export (`R-16`); yang ada hanyalah pemuat satu
 * baris untuk tombol Ubah. Yang dipakai di sini adalah bentuk kueri yang benar-benar ada,
 * tanpa penyaring — sehingga daftarnya memuat seluruh baris entitas itu.
 *
 * Bacaan lain yang mungkin: daftarnya disaring per tim, sehingga seorang leader hanya
 * melihat anggotanya. Perbedaan keduanya menentukan siapa yang boleh menyunting login milik
 * tim lain, dan itu pertanyaan terbuka untuk Work Owner — dinyatakan di layar, bukan
 * ditebak diam-diam.
 */
export function SurveyorLoginPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useSurveyorLoginList()
  const create = useCreateSurveyorLogin()
  const save = useSaveSurveyorLogin()
  const form = useCrudForm(create, save, (row: SurveyorLogin, input) => ({
    login: row.login,
    input,
  }))
  const editing = form.openedRow
  const isFormOpen = form.isOpen
  const { closeForm, openCreate: openAdd, openEdit } = form

  const rows = list.data?.login_surveyor ?? []

  /*
    Lima kolom, satu-lawan-satu dengan grid Pega — Nama, Login, Email, Telp, Alamat —
    ditambah kolom aksi.

    Kelimanya dibaca dari `pyLabelFieldValue` pada
    `Section/BrowseLoginSurveyor-Section.xml`. Status Login dan Login Leader TIDAK ada di
    grid karena Pega pun tidak punya; keduanya hanya muncul saat sebuah baris dibuka.
  */
  const columns: Column<SurveyorLogin>[] = [
    {
      key: 'nama',
      title: 'Nama',
      width: '16rem',
      value: (row) => row.nama,
    },
    {
      key: 'login',
      title: 'Login',
      width: '14rem',
      value: (row) => row.login,
      render: (row) => <span className="font-medium text-slate-800">{row.login}</span>,
    },
    {
      key: 'email',
      title: 'Email',
      value: (row) => row.email,
    },
    {
      key: 'telp',
      title: 'Telp',
      width: '10rem',
      value: (row) => row.telp,
    },
    {
      key: 'alamat',
      title: 'Alamat',
      value: (row) => row.alamat,
    },
    {
      key: 'aksi',
      title: 'Aksi',
      width: '7rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button tone="halus" onClick={() => openEdit(row)} disabled={save.isPending}>
          Ubah
        </Button>
      ),
    },
  ]

  // Isi bagian daftar menurut keadaan portal dan kueri.
  function renderList() {
    return renderListState({
      portal,
      query: list,
      loadingText: 'Memuat daftar login surveyor…',
      toMessage: loadMessage,
      render: () => (
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(row) => row.login}
          description="Sumber: POOLDATA.MST_LOGIN_SURVEYOR"
          searchLabel="Cari nama, login, atau email"
          pageSize={PAGE_SIZE}
          emptyMessage="Belum ada login surveyor pada entitas ini."
        />
      ),
    })
  }

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      {/* Judulnya dibaca dari `pyCaption Master Login Surveyor` pada
          Section/LoginSurveyor (D-13).

          Kedua tombol ini dibaca dari `pyButtonLabel` pada Section/LoginSurveyor —
          Tambah dan Refresh, dengan caption itu apa adanya (D-13). */}
      <ListHeader
        title="Master Login Surveyor"
        description="Daftar surveyor yang terdaftar pada entitas ini, beserta email, telepon, dan alamatnya."
      >
        <AddButton onClick={openAdd} disabled={isFormOpen} />
        <RefreshButton query={list} />
      </ListHeader>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20).

          Pada layar ini ia lebih berarti daripada pada master penggolongan: yang tertera
          adalah orang, dan orang yang sama belum tentu terdaftar di entitas lain. */}
      <p className="mt-3 text-xs text-slate-500">
        <span>Daftar ini memuat seluruh login surveyor pada entitas yang sedang dibuka.</span>
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {isFormOpen && (
        <section className="mt-5">
          <SurveyorLoginForm
            editing={editing}
            isSaving={form.isSaving}
            error={form.saveError}
            onSave={form.submit}
            onCancel={closeForm}
          />
        </section>
      )}

      <section className="mt-6">
        {renderList()}
      </section>

      {/* Dua keterbatasan yang nyata, dinyatakan di kaki halaman alih-alih ditemukan
          pengguna sendiri.

          Keduanya berasal dari bentuk tabelnya, bukan dari pilihan modul ini — dan keduanya
          tidak terlihat dari layar bila tidak disebutkan. */}
      <footer className="mt-6 space-y-2 border-t border-slate-200 pt-4 text-xs text-slate-500">
        <p>
          <strong className="text-slate-700">Akun aplikasi belum diterbitkan.</strong>{' '}
          Layar ini mencatat login surveyor di master; ia belum membuat akun yang dapat
          dipakai masuk. Aplikasi lama menerbitkannya dengan kata sandi yang sama untuk
          setiap orang — perilaku yang tidak dibawa, dan penggantinya menunggu sistem
          identitas.
        </p>
        <p>
          <strong className="text-slate-700">Login tidak dapat dihapus.</strong>{' '}
          Tabelnya tidak punya penanda aktif, dan penghapusan data bernilai bisnis dilarang.
          Surveyor yang sudah tidak bertugas tetap tampil di daftar ini.
        </p>
      </footer>
    </main>
  )
}
