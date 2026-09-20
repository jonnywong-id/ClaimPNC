import { useState } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, SUPPLIER_ACTIVE, type Supplier, type SupplierCode } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useCreateSupplier, useSaveSupplier, useSupplierCodeList, useSupplierList } from './api'
import { SupplierForm, type SupplierFormValues } from './SupplierForm'

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
          title: 'Daftar supplier tidak dapat dimuat',
          description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Daftar supplier tidak dapat dimuat',
    description: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}

/**
 * Layar Master Supplier.
 *
 * Pengganti `Harness/MasterSupplier-Harness.xml` atas tabel `M_SUPPLIER` (MENU_ID 29).
 *
 * # Apa yang dikelola layar ini
 *
 * Daftar **supplier rekanan** beserta syarat dagangnya: ke mana pembayarannya dikirim,
 * berapa tenggat bayar dan tenggat kirimnya, ia rekanan atau bukan, dan apakah ia termasuk
 * supplier Heavy Equipment. Sebagian kolomnya menentukan ke rekening siapa uang berpindah
 * — ia master komersial, bukan sekadar daftar alamat.
 *
 * # Kenapa TANPA tab, berbeda dari Master Bengkel
 *
 * Karena layar lamanya memang tanpa tab. `Section/InboxMasterSupplier-Section.xml` hanya
 * punya satu grid dengan tiga tombol — New Supplier, Edit, dan Refresh. Tidak ada
 * penyaring status di mana pun, dan kolom POSISI yang ditampilkannya milik antrean
 * persetujuan, bukan penyaring.
 *
 * # Dua jalur penyimpanan yang berperilaku BERBEDA
 *
 * Perbedaannya menyentuh supplier mana yang boleh dipakai, sehingga layar menyebutnya
 * terang-terangan alih-alih membiarkannya ditemukan:
 *
 *	tambah  STS_AKTIF selalu "0", berapa pun yang dipilih di form, dan persetujuan
 *	        SELALU diminta (`CreateNewMasterSupplier_post` step 6 dan 12)
 *	ubah    STS_AKTIF menyusul pilihan di form, dan persetujuan diminta HANYA bila
 *	        supplier-nya diaktifkan (`EditMasterSupplier_post` step 7 dan 12)
 *
 * Akibat yang paling perlu disadari: **menonaktifkan supplier berlaku seketika, tanpa
 * persetujuan siapa pun.** Itu satu-satunya jalur di modul ini yang mengubah keadaan tanpa
 * melewati antrean mana pun, dan itu perilaku sistem lama yang ditiru apa adanya.
 *
 * # Kenapa tidak ada tombol Approve dan Reject di sini
 *
 * Berbeda dari Master Bengkel, yang layar persetujuannya ada di Pega dan hanya belum
 * dibangun: sisi pemutus antrean `pooldata.proteksi_klaimmbu` **tidak ada di export sama
 * sekali** (`R-16`). Tidak ada satu pun rule yang membacanya, menyetujuinya, menolaknya,
 * atau memajukan POSISI-nya.
 *
 * Membangunnya di sini berarti mengarang aturan yang menentukan supplier mana yang boleh
 * dipakai — dan itu keputusan bisnis, bukan keputusan layar. Yang dikerjakan modul ini
 * berhenti pada menyisipkan permintaannya, persis seperti sistem lama.
 */
export function SupplierPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [isAdding, setAdding] = useState(false)
  const [editing, setEditing] = useState<Supplier | null>(null)

  const list = useSupplierList()
  const codeList = useSupplierCodeList()
  const create = useCreateSupplier()
  const save = useSaveSupplier()

  const isFormOpen = isAdding || editing !== null
  const rows = list.data?.supplier ?? []

  const codes = {
    status_rekanan: codeList.data?.status_rekanan ?? [],
    status_supply: codeList.data?.status_supply ?? [],
    jenis_supplier: codeList.data?.jenis_supplier ?? [],
    status_aktif: codeList.data?.status_aktif ?? [],
    status_autopayment: codeList.data?.status_autopayment ?? [],
  }

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

  function openEdit(row: Supplier) {
    create.reset()
    save.reset()
    setAdding(false)
    setEditing(row)
  }

  function submit(values: SupplierFormValues) {
    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan pada form dua puluh tiga isian,
    // itu kehilangan yang tidak dapat dimaafkan.
    if (editing) {
      save.mutate({ id: editing.id_supplier, input: values }, { onSuccess: closeForm })
      return
    }
    create.mutate(values, { onSuccess: closeForm })
  }

  /*
    Kesembilan kolom mengikuti grid Pega SATU-LAWAN-SATU — urutan dan caption yang sama,
    terbaca dari `Section/InboxMasterSupplier-Section.xml`:

      ID · NAMA · ALAMAT · TELP · JENIS SUPPLIER · STATUS REKANAN · STATUS AKTIF ·
      POSISI · OPTION

    # Kenapa TIDAK digabung menjadi kolom majemuk

    Percobaan pertama menggabungkannya menjadi tujuh kolom — nama beserta ID di bawahnya,
    kota beserta cabang, dan seterusnya — dengan alasan yang disalin dari Master Bengkel:
    grid empat puluh kolom tidak dapat dibaca pada layar mana pun.

    Alasan itu TIDAK berlaku di sini, dan menyalinnya adalah kekeliruan. Grid supplier
    hanya SEMBILAN kolom, dan sembilan kolom memang dapat dibaca. Yang tersisa dari
    penggabungan itu hanyalah biayanya: petugas yang hafal urutan kolom Pega harus
    mencarinya kembali — dan `D-13` menuntut justru sebaliknya.

    # Caption "Input Nama" berasal dari layar produksi, bukan dari export

    Export menuliskannya `NAMA`. Layar yang dipakai hari ini menuliskannya **Input Nama**.
    Yang diikuti adalah layar yang dilihat pengguna (`D-13`); selisihnya dicatat sebagai
    bukti bahwa export tertinggal dari produksi (`R-09`).

    # Dua kolom yang ISINYA belum dapat ditiru

    `JENIS SUPPLIER` dan `STATUS REKANAN` di Pega menampilkan `.JENIS_STATUS_NOTE` dan
    `.STS_REKANAN_NOTE` — **labelnya, bukan sandinya**. Kedua nama itu hanya muncul di
    harness dan section layar ini, dan tidak ada satu pun kueri di export yang memuatnya
    (`R-16`): keduanya berasal dari `ListMasterSupllier` yang memang hilang.

    Yang dikerjakan di sini: label dicari dari daftar sandi yang sudah dikirim server
    lewat `/sandi`. Bila artinya diketahui, labelnya yang tampil; bila tidak, **sandinya
    sendiri** yang tampil — bukan tebakan. Begitu daftar Field Value-nya diterima, kedua
    kolom ini ikut menampilkan label tanpa satu baris kode pun berubah.

    # Satu kolom yang isinya selalu kosong

    `POSISI` milik `pooldata.proteksi_klaimmbu`, dan tidak ada satu pun rule di export yang
    membacanya. Kolomnya TETAP digambar supaya susunannya sama dengan Pega — dan pada
    layar Pega yang sesungguhnya pun kolom itu tampak kosong.
  */
  const columns: Column<Supplier>[] = [
    { key: 'id', title: 'ID', width: '8rem', value: (row) => row.id_supplier },
    { key: 'nama', title: 'Input Nama', value: (row) => row.nama },
    { key: 'alamat', title: 'Alamat', value: (row) => row.alamat },
    { key: 'telp', title: 'Telp', width: '9rem', value: (row) => row.telepon },
    {
      key: 'jenis_supplier',
      title: 'Jenis Supplier',
      width: '9rem',
      value: (row) => labelOf(codes.status_supply, row.status_supply),
    },
    {
      key: 'status_rekanan',
      title: 'Status Rekanan',
      width: '9rem',
      value: (row) => labelOf(codes.status_rekanan, row.status_rekanan),
    },
    {
      key: 'status_aktif',
      title: 'Status Aktif',
      width: '9rem',
      value: (row) => labelOf(codes.status_aktif, row.status_aktif_berlaku),
      render: (row) => (
        <StatusMark
          label={labelOf(codes.status_aktif, row.status_aktif_berlaku)}
          waiting={!row.aktif && row.status_aktif === SUPPLIER_ACTIVE}
        />
      ),
    },
    {
      key: 'posisi',
      title: 'Posisi',
      width: '7rem',
      noSort: true,
      value: () => '',
    },
    {
      key: 'option',
      title: 'Option',
      width: '6rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      // Caption tombolnya "Edit", mengikuti layar lama apa adanya (`D-13`) — bukan "Ubah".
      render: (row) => (
        <Button tone="halus" onClick={() => openEdit(row)} disabled={save.isPending}>
          Edit
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-7xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Master Supplier</h1>
          <p className="text-sm text-slate-600">
            Supplier rekanan beserta syarat dagangnya — rekening pembayaran, tenggat bayar,
            dan tenggat kirim.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/* Caption tombolnya mengikuti layar lama apa adanya (D-13):
              `Harness/MasterSupplier-Harness.xml` memakai "Refresh" dan "New Supplier". */}
          <Button tone="kedua" onClick={() => void list.refetch()} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
          <Button tone="utama" onClick={openAdd} disabled={isFormOpen}>
            New Supplier
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20). */}
      <p className="mt-3 text-xs text-slate-500">
        Total data: <span className="font-medium text-slate-700">{rows.length}</span>
        <span className="ml-3">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {/*
        Daftar sandi yang gagal dimuat TIDAK menghentikan layar, tetapi form-nya tidak
        dapat dipakai: kelima dropdown-nya akan kosong, dan kelimanya wajib diisi. Itu
        dinyatakan di sini supaya petugas tidak mengira form-nya yang rusak.
      */}
      {codeList.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Daftar pilihan status tidak dapat dimuat"
            description="Menambah dan mengubah supplier belum dapat dilakukan sampai daftar ini terbaca. Muat ulang halaman, atau hubungi administrator Claim PNC."
            tone="gangguan"
          />
        </div>
      )}

      {isFormOpen && (
        <section className="mt-5">
          <SupplierForm
            editing={editing}
            codes={codes}
            isSaving={create.isPending || save.isPending}
            error={editing ? save.error : create.error}
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
          <p className="text-sm text-slate-500">Memuat daftar supplier…</p>
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
            rows={rows}
            rowKey={(row) => row.id_supplier}
            description="Sumber: M_SUPPLIER"
            searchLabel="Cari supplier"
            emptyMessage="Belum ada supplier pada entitas ini."
            /*
              Dua puluh baris per halaman, dibaca langsung dari
              `Section/InboxMasterSupplier-Section.xml`:

                pyPageMode  = Numeric   nomor halaman, bukan "muat lebih banyak"
                pyPageSize  = 20

              Angkanya TIDAK seragam antarlayar — Master Bengkel juga 20, sementara Master
              Rekening, Status Klaim, Status Progres, Pasal Kerugian, dan Penolakan Klaim
              memakai 15 (`pyPageSizeOther`). Karena itu ia prop per layar, bukan bawaan
              komponen.

              Paginasinya di peramban, bukan di server: grid Pega pun terikat pada page
              list klipboard (`pyPageListProperty = ListMasterSupllier.pxResults`) dan
              memotong daftar yang sudah dimuat. Paginasi keyset sisi server adalah
              `TKT-U2-001`, dan Steering menyebutnya perubahan perilaku — bukan
              pemeliharaan.
            */
            pageSize={20}
          />
        )}
      </section>
    </main>
  )
}

/**
 * labelOf mencari label sebuah sandi pada daftar yang dikirim server lewat `/sandi`.
 *
 * Bila sandinya tidak ada di daftar — atau labelnya memang berisi sandinya sendiri karena
 * artinya tidak diketahui (`R-16`) — yang dikembalikan adalah sandinya apa adanya.
 * **Tidak pernah tebakan.**
 *
 * Ia yang membuat kolom JENIS SUPPLIER dan STATUS REKANAN menampilkan label begitu daftar
 * Field Value-nya diterima, tanpa satu baris kode pun berubah.
 */
function labelOf(list: SupplierCode[], value: string): string {
  return list.find((code) => code.nilai === value)?.label ?? value
}

/**
 * StatusMark menggambar isi kolom STATUS AKTIF.
 *
 * Teksnya mengikuti Pega apa adanya — layar lama menampilkan "Aktif", bukan sandi `"1"`
 * dan bukan lencana berwarna. Labelnya diambil dari daftar sandi yang sama dengan kedua
 * kolom bersandi lain, sehingga ketiganya tidak pernah berbeda cara.
 *
 * # Satu keterangan yang DITAMBAHKAN
 *
 * "menunggu persetujuan" muncul tepat pada keadaan yang paling membingungkan: supplier
 * yang baru ditambahkan sebagai aktif, tetapi belum berlaku aktif karena
 * `CreateNewMasterSupplier_post` step 6 menetapkan `STS_AKTIF := "0"` tanpa syarat.
 *
 * Tanpa keterangan itu, petugas melihat supplier yang "sudah disimpan sebagai aktif"
 * tampil sebagai tidak aktif dan tidak punya cara mengetahui sebabnya. Ia keterangan di
 * DALAM kolom yang sama, bukan kolom tambahan — susunan kolomnya tetap sembilan.
 */
function StatusMark({ label, waiting }: { label: string; waiting: boolean }) {
  return (
    <span className="inline-flex flex-col">
      <span className="text-sm text-slate-900">{label || '—'}</span>
      {waiting && <span className="text-xs text-amber-700">menunggu persetujuan</span>}
    </span>
  )
}
