import { z } from 'zod'

import type { Business, SimasOnlineCauseOfLoss } from '@/api/types'
import { BusinessMappedForm } from '@/components/masterform/BusinessMappedForm'
import { businessListSchema } from '@/components/masterform/BusinessRowsField'

/**
 * Kedua batas harus sama dengan MaxDescriptionLength dan MaxBusinessNameLength di
 * `internal/mastercolsimasonline/mastercolsimasonline.go`.
 *
 * Kolomnya berlebar VARCHAR2(200) di POOLDATA.M_CAUSE_OF_LOSS_ONLINE dan
 * POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL; angka di bawah sengaja ditahan di separuhnya,
 * dan layar Pega sendiri tidak membatasi apa pun. Angkanya diperiksa di dua tempat
 * dengan sengaja: di sini supaya pengguna tahu
 * sebelum mengirim, dan di server karena API dapat ditembak tanpa melewati layar ini.
 * Server tetap yang berwenang.
 *
 * Bila angka ini berubah, berkas Go di atas harus ikut berubah — dan uji
 * `TestLengthLimitsAreMirroredInTheFrontend` yang menjaga pengingat itu tetap terlihat.
 */
const MAX_NAME_LENGTH = 100
const MAX_BUSINESS_NAME_LENGTH = 100

/**
 * Yang diperiksa di layar hanyalah PANJANG, dan itu disengaja.
 *
 * Layar Pega menandai seluruh isiannya `pyRequired=false` dan tidak punya satu pun
 * Validate rule, sehingga nama kosong maupun bisnis kembar sama-sama sah. Work Owner menetapkan 2026-09-21 layar disamakan dengan Pega.
 *
 * Batas panjang tetap ada karena ia bukan aturan bisnis melainkan penjaga terhadap lebar
 * kolom basis data — tanpa itu, nilai yang kepanjangan ditolak Oracle dengan ORA-12899
 * yang muncul sebagai galat 500 dan tidak dapat dibaca pengguna.
 */
const schema = z.object({
  nama: z
    .string()
    .trim()
    .max(MAX_NAME_LENGTH, `Nama Cause of loss paling panjang ${MAX_NAME_LENGTH} karakter.`),
  bisnis: businessListSchema(MAX_BUSINESS_NAME_LENGTH),
})

export type CauseOfLossFields = z.infer<typeof schema>

type Props = {
  /** Baris yang disunting; null berarti penambahan baru. */
  edited: SimasOnlineCauseOfLoss | null
  /** Saran bisnis, dibaca dari POOLDATA.BUSINESS milik entitas yang aktif. */
  businesses: Business[]
  /** Pemetaan bisnis baris yang disunting masih dimuat. */
  isLoadingBusinessMapping: boolean
  isSaving: boolean
  error: unknown
  onSave: (values: CauseOfLossFields) => void
  onCancel: () => void
}

/**
 * CauseOfLossForm adalah satu form untuk DUA mode — tambah dan ubah.
 *
 * Satu form, bukan dua, mengikuti sistem lama: `PageNewForSetSimasOnlineCOL` dan
 * `SetDataCauseofflossOnline` sama-sama mengisi satu halaman `TempCauseOfLoss` lalu
 * menandai modusnya lewat `pyLabel`. Isian yang dibandingkan pengguna karena itu berada
 * di tempat yang sama pada kedua mode.
 *
 * # Susunan isiannya disamakan PERSIS dengan Pega
 *
 * Diambil dari urutan kemunculannya di `Section/Online_BrowseCauseOfLoss-Section.xml`
 * (`D-13`: tata letak ditiru supaya pengguna tidak perlu belajar ulang):
 *
 *	baris 4301  ID Kerugian          M_COL_ID     read-only
 *	baris 4813  Nama Cause of loss   COL_DESC
 *	baris 5874  Bisnis               BISNISID     repeat grid, banyak baris
 *
 * Isian "ID Master Kerugian" (MST_COL_ID) yang dulu berada di antara keduanya DICABUT
 * 2026-09-23 atas keputusan Work Owner — kolomnya sudah tidak dipakai.
 *
 * Perhatikan label isian pertama adalah "ID Kerugian", bukan "ID". Itu sempat keliru pada
 * versi pertama modul ini dan dikoreksi setelah Work Owner memeriksanya pada 2026-09-21.
 *
 * Judulnya pun SAMA untuk kedua modus — "Memperbaharui Data Simas Online" — persis
 * seperti layar lama, yang menandai modusnya lewat `pyLabel` dan bukan lewat judul.
 */
export function CauseOfLossForm({ businesses, ...props }: Readonly<Props>) {
  return (
    <BusinessMappedForm
      {...props}
      schema={schema}
      textName="nama"
      textLabel="Nama Cause of loss"
      textMaxLength={MAX_NAME_LENGTH}
      textOf={(row) => row.nama}
      idOf={(row) => row.id}
      businessesOf={(row) => row.bisnis}
      idLabel="ID Kerugian"
      // Judul yang sama untuk kedua modus, mengikuti layar lama. Beda modus tetap terlihat
      // dari ada-tidaknya baris "ID Kerugian" di bawahnya.
      title="Memperbaharui Data Simas Online"
      // Bisnis adalah GRID: `pyPageListProperty = TempCauseOfLoss.BISNISID` berkelas
      // ASM-FW-GISFW-Int-BUSINESS, ber-`pyAllowFreeFormInput=true`.
      legend="Bisnis"
      emptyText="Belum ada bisnis yang dipilih. Cause of loss ini tetap dapat disimpan."
      businessNames={businesses.map((b) => b.nama)}
      businessMaxLength={MAX_BUSINESS_NAME_LENGTH}
    />
  )
}
