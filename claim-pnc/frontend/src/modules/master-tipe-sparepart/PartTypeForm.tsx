import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import {
  ErrorCode,
  PartTypeErrorCode,
  type PartType,
  type PartTypeCategory,
} from '@/api/types'
import { Field } from '@/components/Field'
import { SelectField, type SelectOption } from '@/components/SelectField'
import { MasterFormActions, MasterFormFrame, ReadOnlyIdRow, useServerViolations } from '@/components/masterform/BoxForm'
import { type CodeMessages, notFoundMessage, portalMessages, saveErrorMessage, validationMessage } from '@/components/masterform/saveErrorMessage'

/**
 * Batas panjang nama harus sama dengan mastertipesparepart.MaxNameLength di backend.
 *
 * Ia ASUMSI, bukan angka dari DDL: `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak ada DDL-nya di
 * export (`R-08`), dan layar lamanya tidak memasang satu pun `pyMaxLength`. Seratus dipilih
 * agar sama dengan batas nama pada Master Sparepart dan Master Kategori Sparepart — nilai
 * kolom ini muncul sebagai label di layar Master Sparepart berdampingan dengan nama
 * kategori, dan tiga batas berbeda pada tiga layar bertetangga hanya akan membingungkan.
 *
 * Diperiksa di dua tempat dengan sengaja: di sini supaya pengguna tahu sebelum mengirim,
 * dan di server karena API dapat ditembak tanpa melewati layar ini. Server tetap yang
 * berwenang — pemeriksaan di sini hanya kenyamanan.
 *
 * Bila angka ini berubah, `internal/mastertipesparepart/mastertipesparepart.go` harus ikut
 * berubah. Uji `TestMaxNameLengthMatchesFrontendForm` yang menjaganya.
 */
const MAX_NAME_LENGTH = 100

const schema = z.object({
  nama_tipe_sparepart: z
    .string()
    .trim()
    .min(1, 'Nama tipe sparepart wajib diisi.')
    .max(MAX_NAME_LENGTH, `Nama tipe sparepart paling panjang ${MAX_NAME_LENGTH} karakter.`),
  id_kategori_sparepart: z.string().trim().min(1, 'Kategori sparepart wajib dipilih.'),
})

export type PartTypeFormValues = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  editing: PartType | null
  /** Pilihan Kategori, dibaca dari basis data entitas yang sedang dibuka. */
  category: PartTypeCategory[]
  /** Daftar pilihan masih dimuat; dropdown dimatikan sementara. */
  isLoadingCategory: boolean
  /** Daftar pilihan terpotong pada batas lookup; dikatakan terang-terangan. */
  isCategoryTruncated: boolean
  isSaving: boolean
  error: unknown
  onSave: (values: PartTypeFormValues) => void
  onCancel: () => void
}

/** Pesan galat penyimpanan per kode; yang tidak dikenal jatuh ke pesan galat sistem. */
const saveMessages: CodeMessages = {
  // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya akan
  // mengulang hal yang sama.
  [ErrorCode.validationFailed]: validationMessage,
  // Ia sudah disorot pada isiannya, tetapi TETAP ditampilkan sebagai kotak pesan —
  // dan itu berbeda dari galat validasi biasa.
  //
  // Alasannya: perbaikannya bukan "betulkan isian" melainkan "cari tipe yang sudah
  // ada, atau pakai nama lain", dan yang memakainya bisa jadi baris di KATEGORI LAIN
  // atau baris di tab Reject — keduanya TIDAK terlihat dari tab yang sedang dibuka.
  // Sorotan di bawah isian tidak cukup menjelaskan ke mana pengguna harus mencari.
  [PartTypeErrorCode.nameTaken]: {
    title: 'Nama itu sudah dipakai',
    description:
      'Tipe lain sudah memakai nama ini. Keunikan nama berlaku di seluruh master — ' +
      'termasuk tipe di kategori yang berbeda, dan termasuk tipe yang sudah ditolak. ' +
      'Periksa juga tab Reject.',
    tone: 'penolakan',
  },
  // Penyebabnya BUKAN salah ketik: pengguna memilihnya dari dropdown. Yang berubah
  // adalah dunia di luar formnya — kategorinya ditolak petugas lain sementara form
  // terbuka. Perbaikannya karena itu "muat ulang pilihan", bukan "betulkan isian".
  [PartTypeErrorCode.categoryNotFound]: {
    title: 'Kategori itu sudah tidak tersedia',
    description:
      'Persetujuan kategori ini dicabut sementara form terbuka, atau kategorinya ' +
      'sudah tidak ada. Tutup form ini lalu buka kembali untuk memuat ulang daftar ' +
      'pilihannya.',
    tone: 'penolakan',
  },
  [ErrorCode.notFound]: notFoundMessage,
  ...portalMessages,
}

/** Isian yang dapat disorot pelanggaran server. */
const violationFields = ['nama_tipe_sparepart', 'id_kategori_sparepart'] as const

/**
 * PartTypeForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `Activity/SetMasterTipeSparepart_act` memuat
 * baris ke form yang sama lalu menandainya "Memperbaharui Data" — caption yang memang ada
 * di `Section/MasterTipeSparepartHEApproval-Section.xml`.
 *
 * # Dua isian yang dapat diubah
 *
 * Section lamanya punya empat label — "ID Tipe Sparepart", "Nama Tipe Sparepart",
 * "ID Kategori Sparepart", dan "Kategori Sparepart" — tetapi hanya dua yang dapat disentuh
 * pengguna. ID tipe tidak dapat diubah, dan ID kategori terisi sendiri dari dropdown
 * Kategori.
 *
 * # Kenapa dropdown, bukan ketik bebas
 *
 * Karena di Pega pun dropdown: `pxDropdown` dengan pilihan kosong bertuliskan
 * `---PILIH KATEGORI---`. Nilai yang disimpan adalah ID kategori, dan yang ditampilkan
 * adalah namanya — di layar lama keduanya dialiaskan `.District` dan `.DistrictID`, dua
 * nama yang justru tertukar terhadap pola "…ID" yang biasa.
 *
 * # ID tidak dapat disunting, dan pada penambahan ia belum ada
 *
 * `RDB List/UpdateMasterSparepartType_sql2-SQL.xml` memakai PART_SECTION_ID hanya sebagai
 * penyaring `WHERE`, tidak pernah sebagai kolom yang di-SET. Pada penambahan ia diterbitkan
 * server dari isi tabelnya sendiri, sehingga layar tidak punya cara menebaknya — dan tidak
 * boleh mencoba.
 *
 * # Tanpa isian Catatan
 *
 * Tabelnya tidak punya kolom penampung alasan penolakan. Menggambar isian yang diam-diam
 * membuang isinya lebih buruk daripada tidak menggambarnya.
 */
export function PartTypeForm({
  editing,
  category,
  isLoadingCategory,
  isCategoryTruncated,
  isSaving,
  error,
  onSave,
  onCancel,
}: Readonly<Props>) {
  const editMode = editing !== null

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<PartTypeFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama_tipe_sparepart: editing?.nama_tipe_sparepart ?? '',
      id_kategori_sparepart: editing?.id_kategori_sparepart ?? '',
    },
  })

  // Isian disesuaikan ketika baris yang disunting berganti tanpa form ditutup lebih dulu —
  // misalnya pengguna menekan "Ubah" pada baris lain.
  useEffect(() => {
    reset({
      nama_tipe_sparepart: editing?.nama_tipe_sparepart ?? '',
      id_kategori_sparepart: editing?.id_kategori_sparepart ?? '',
    })
  }, [editing, reset])

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5), dan
  // itu hanya berguna bila layar menyorotnya satu per satu.
  useServerViolations(error, setError, violationFields)

  const message = saveErrorMessage(error, saveMessages)
  const title = editMode ? 'Ubah Tipe Sparepart' : 'Tambah Tipe Sparepart'

  /*
    Kategori yang sedang dipegang baris TETAP muncul sebagai pilihan meski tidak ada di
    daftar — persis persoalan baris yatim, dan persis baris yang persetujuan kategorinya
    dicabut setelah tipenya tersimpan.

    Tanpa ini, membuka baris seperti itu akan mengosongkan dropdown-nya diam-diam, dan
    pengguna yang hanya ingin mengubah NAMA akan ikut memindahkan kategorinya tanpa
    menyadari. Pilihan tambahannya diberi keterangan supaya terlihat bahwa ia tidak sah.
  */
  const options: SelectOption[] = category.map((one) => ({
    value: one.kode,
    label: one.nama,
  }))
  const current = editing?.id_kategori_sparepart ?? ''
  const isCurrentMissing =
    current !== '' && !category.some((one) => one.kode === current)
  if (isCurrentMissing) {
    options.unshift({
      value: current,
      label: editing?.nama_kategori_sparepart
        ? `${editing.nama_kategori_sparepart} (tidak lagi disetujui)`
        : `${current} — kategori tidak ditemukan`,
    })
  }

  return (
    <MasterFormFrame onSubmit={handleSubmit(onSave)} title={title} message={message}>
      {/* ID hanya ditampilkan saat menyunting, dan tidak dapat diubah. Pada penambahan ia
          belum ada — nomornya diterbitkan server dari isi tabel. */}
      {editMode && <ReadOnlyIdRow label="ID Tipe Sparepart" value={editing.id_tipe_sparepart} />}

      <Field
        id="nama_tipe_sparepart"
        label="Nama Tipe Sparepart"
        type="text"
        autoFocus
        maxLength={MAX_NAME_LENGTH}
        error={errors.nama_tipe_sparepart?.message}
        {...register('nama_tipe_sparepart')}
      />

      <SelectField
        id="id_kategori_sparepart"
        label="Kategori Sparepart"
        options={options}
        emptyText="---PILIH KATEGORI---"
        disabled={isLoadingCategory}
        error={errors.id_kategori_sparepart?.message}
        {...register('id_kategori_sparepart')}
      />

      {isLoadingCategory && (
        <p className="text-xs text-slate-500">Memuat daftar kategori sparepart…</p>
      )}

      {/* Daftar yang terpotong dikatakan terang-terangan. Tanpa ini, pemotongan terjadi
          diam-diam dan terbaca pengguna sebagai "kategorinya belum dibuat". */}
      {isCategoryTruncated && (
        <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          Daftar kategori terpotong pada batas yang berlaku. Bila kategori yang Anda cari
          tidak ada di sini, hubungi administrator Claim PNC.
        </p>
      )}

      {!isLoadingCategory && category.length === 0 && (
        <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          Belum ada kategori sparepart yang disetujui pada entitas ini. Tipe sparepart tidak
          dapat disimpan sebelum ada kategori berstatus <strong>Approve</strong> di layar
          Master Kategori Sparepart.
        </p>
      )}

      {/* Akibat menyimpan dinyatakan di muka, bukan ditemukan setelah tombol ditekan.
          Baris yang sudah disetujui akan HILANG dari dropdown Tipe pada layar Master
          Sparepart selama ia menunggu persetujuan ulang — dan tidak ada apa pun di layar
          ini yang akan menunjukkannya bila tidak disebutkan di sini. */}
      <p className="rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
        Menyimpan mengembalikan tipe ini ke tab <strong>Waiting Approval</strong>, persis
        seperti aplikasi lama. Selama menunggu, tipe ini tidak muncul sebagai pilihan di
        layar Master Sparepart. Mengubah kategorinya juga memindahkan tipe ini ke golongan
        baru pada setiap layar yang menampilkannya.
      </p>

      <MasterFormActions isSaving={isSaving} onCancel={onCancel} />
    </MasterFormFrame>
  )
}
