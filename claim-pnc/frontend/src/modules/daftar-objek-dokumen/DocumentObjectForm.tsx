import { z } from 'zod'

import type { Business, DocumentObject } from '@/api/types'
import { BusinessMappedForm } from '@/components/masterform/BusinessMappedForm'
import { businessListSchema } from '@/components/masterform/BusinessRowsField'

/**
 * Kedua batas harus sama dengan MaxDescriptionLength dan MaxBusinessNameLength di
 * `internal/daftarobjekdokumen/daftarobjekdokumen.go`.
 *
 * Keduanya PENJAGA TEKNIS, bukan aturan bisnis: lebar kolomnya belum diketahui (`R-08`),
 * dan tabel pemetaan bisnisnya bahkan belum ada. Angkanya diperiksa di dua tempat dengan
 * sengaja — di sini supaya pengguna tahu sebelum mengirim, dan di server karena API dapat
 * ditembak tanpa melewati layar ini. Server tetap yang berwenang.
 *
 * Bila angka ini berubah, berkas Go di atas harus ikut berubah.
 */
const MAX_DESCRIPTION_LENGTH = 100
const MAX_BUSINESS_NAME_LENGTH = 100

/**
 * Yang diperiksa di layar hanyalah PANJANG, dan itu disengaja.
 *
 * Layar Pega menandai seluruh isiannya `pyRequired=false` dan tidak punya satu pun Validate
 * rule untuk kelas ASM-FW-GCNMFW-Int-V_LST_DOC_OBJ, sehingga keterangan kosong maupun bisnis
 * kembar sama-sama sah. `P-5` menetapkan perilaku dipertahankan lebih dulu.
 *
 * Batas panjang tetap ada karena ia bukan aturan bisnis melainkan penjaga terhadap lebar
 * kolom basis data — tanpa itu, nilai yang kepanjangan ditolak Oracle dengan ORA-12899 yang
 * muncul sebagai galat 500 dan tidak dapat dibaca pengguna.
 */
const schema = z.object({
  objek_dokumen: z
    .string()
    .trim()
    .max(
      MAX_DESCRIPTION_LENGTH,
      `Daftar Objek Dokumen paling panjang ${MAX_DESCRIPTION_LENGTH} karakter.`,
    ),
  bisnis: businessListSchema(MAX_BUSINESS_NAME_LENGTH),
})

export type DocumentObjectFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: DocumentObject | null
  /** Saran bisnis, dibaca dari POOLDATA.BUSINESS milik entitas yang aktif. */
  businesses: Business[]
  /** Pemetaan bisnis baris yang disunting masih dimuat. */
  isLoadingBusinessMapping: boolean
  isSaving: boolean
  error: unknown
  onSave: (values: DocumentObjectFields) => void
  onCancel: () => void
}

/**
 * DocumentObjectForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `Section/BrowseDocumentObject-Section.xml`
 * memakai satu halaman `TempDocObj` untuk keduanya dan menandai modusnya lewat isian ID yang
 * hanya tampil bila terisi (`pyVisible=NOTBLANK` pada `:11530`). Isian yang dibandingkan
 * pengguna karena itu berada di tempat yang sama pada kedua mode.
 *
 * # Susunan isiannya disamakan PERSIS dengan Pega
 *
 * Diambil dari urutan kemunculannya di `Section/BrowseDocumentObject-Section.xml`
 * (`D-13`: tata letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *	baris 11500  ID                      TempDocObj.ID             read-only
 *	baris 11910  Daftar Objek Dokumen    TempDocObj.KET_DOC_OBJ
 *	baris 13720  ID Bisnis               TempDocObj.LIST_LBU_ID    repeat grid, banyak baris
 *
 * Judulnya pun SAMA untuk kedua modus — "Memperbaharui Data" (`:10859`) — persis seperti
 * layar lama, yang menandai modusnya lewat ada-tidaknya baris ID dan bukan lewat judul.
 *
 * # Satu isian Pega yang sengaja TIDAK dibawa
 *
 * `TempDocObj.pyNote` berlabel "Catatan" (`:3097`, `:3160`) adalah kotak READ-ONLY tempat
 * Pega menaruh pesan hasil penyimpanan — kalimat yang dikembalikan procedure lewat `ErrMsg`.
 * Ia bukan isian dan bukan data: `D-68` menetapkan kontrak galat berbasis teks itu tidak
 * dibawa, dan penggantinya adalah pesan galat ber-`kode` yang sudah ditangani messageFor di
 * atas.
 */
export function DocumentObjectForm({ businesses, ...props }: Readonly<Props>) {
  return (
    <BusinessMappedForm
      {...props}
      schema={schema}
      textName="objek_dokumen"
      textLabel="Daftar Objek Dokumen"
      textMaxLength={MAX_DESCRIPTION_LENGTH}
      textAutoCompleteOff
      textOf={(row) => row.objek_dokumen}
      idOf={(row) => row.id}
      businessesOf={(row) => row.bisnis}
      // Judul yang sama untuk kedua modus, mengikuti layar lama. Beda modus tetap terlihat
      // dari ada-tidaknya baris "ID" di bawahnya.
      title="Memperbaharui Data"
      // Bisnis adalah GRID: `pyPageListProperty = TempDocObj.LIST_LBU_ID` dengan sel terikat
      // `.Note` berkelas ASM-FW-GISFW-Int-BUSINESS, kontrol `pxAutoComplete`.
      legend="ID Bisnis"
      emptyText="Belum ada bisnis yang dipilih. Objek dokumen ini tetap dapat disimpan."
      businessNames={businesses.map((b) => b.nama)}
      businessMaxLength={MAX_BUSINESS_NAME_LENGTH}
    />
  )
}
