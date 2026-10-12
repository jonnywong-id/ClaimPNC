import { ambil } from './dokumen'

/**
 * Padanan When rule Pega yang menentukan **apa yang digambar** di popup rincian.
 *
 * # Kenapa ini ada
 *
 * Popup rincian tidak menggambar hal yang sama untuk setiap klaim. Dua tab hanya muncul pada
 * lini tertentu, dan tabel objek berganti kolom menurut lini bisnisnya. Pega mengaturnya lewat
 * When rule yang dipasang sebagai `pyContainerVisibleWhen` pada wadah tab maupun grid.
 *
 * Berkas ini menyalin aturan itu, bukan mendekatinya.
 *
 * # Kondisinya dibaca dari `pyParametersParamValue`, BUKAN `pyConditionString`
 *
 * Ini perlu ditulis karena sekali membuat saya salah. `pyConditionString` pada berkas When
 * adalah **label tampilan yang usang**: enam When berbeda membawa teks yang sama persis
 * ("Kode Bisnis = 24/18/17/10/07/06"), dan `IsFire` bahkan hanya berisi teks pengganti
 * "[Double click to add condition]".
 *
 * Membaca label itu membuat saya menyimpulkan dua aturan "kosong di export" dan menggantinya
 * dengan tebakan berbasis data. Yang benar ada di `pyParametersParamValue`, dan seluruh enam
 * aturan punya kondisi yang jelas.
 *
 * # Dari mana nilainya dibaca
 *
 * Ketiga properti berada di snapshot polis di dalam dokumen JSON klaim. `IsTravel` di Pega
 * memeriksa DUA jalur yang berbeda untuk nilai yang sama, dan keduanya ikut diperiksa di sini.
 */

/** Membaca satu nilai polis dari dokumen; kosong bila jalurnya tidak ada. */
function nilaiPolis(dokumen: Record<string, unknown>, ekor: string): string {
  for (const jalur of [`Policy.Quotation.${ekor}`, `ClaimData.PolicyData.Quotation.${ekor}`]) {
    const hasil = ambil(dokumen, jalur)
    if (hasil.ada && hasil.nilai.trim() !== '') return hasil.nilai.trim()
  }
  return ''
}

const panel = (d: Record<string, unknown>) => nilaiPolis(d, 'GroupPanel')
const jenis = (d: Record<string, unknown>) => nilaiPolis(d, 'BusinessType')
const kode = (d: Record<string, unknown>) => nilaiPolis(d, 'BusinessCode')

/** `IsPA` — `Policy.Quotation.GroupPanel = "002"`. Sama dengan `isPA_PNC`. */
export const isPA = (d: Record<string, unknown>) => panel(d) === '002'

/** `IsTravel` — `Policy.Quotation.GroupPanel = "005"`. */
export const isTravel = (d: Record<string, unknown>) => panel(d) === '005'

/** `IsAneka` — `Policy.Quotation.BusinessCode = "10140"`. */
export const isAneka = (d: Record<string, unknown>) => kode(d) === '10140'

/** `IsMarineCargo` — `BusinessType = "MarineCargo"` ATAU `GroupPanel = "004"`. */
export const isMarineCargo = (d: Record<string, unknown>) =>
  jenis(d) === 'MarineCargo' || panel(d) === '004'

/** `IsFire` — `BusinessType = "Fire"` ATAU `GroupPanel = "006"`. */
export const isFire = (d: Record<string, unknown>) => jenis(d) === 'Fire' || panel(d) === '006'

/** `IsHE` — `BusinessType = "HE"` ATAU `BusinessType = "ContractorsPM"`. */
export const isHE = (d: Record<string, unknown>) =>
  jenis(d) === 'HE' || jenis(d) === 'ContractorsPM'

/**
 * Nama penjaga yang dipakai spesifikasi grid, dipetakan ke fungsinya.
 *
 * Dipisahkan dari spesifikasi supaya yang tertulis di sana adalah **nama When rule Pega** apa
 * adanya — sehingga penelusuran balik ke export tetap mungkin tanpa membaca kode ini.
 */
export const PENJAGA: Record<string, (d: Record<string, unknown>) => boolean> = {
  IsPA: isPA,
  isPA_PNC: isPA,
  IsTravel: isTravel,
  IsAneka: isAneka,
  IsMarineCargo: isMarineCargo,
  IsFire: isFire,
  IsHE: isHE,
}

/**
 * Menilai satu ungkapan penjaga seperti yang tertulis di `pyContainerVisibleWhen`.
 *
 * Bentuk yang didukung persis yang muncul di export: satu nama, atau beberapa nama yang
 * disambung `||`. Tidak ada bentuk lain di sembilan section popup, dan menambah bentuk yang
 * tidak dipakai hanya menambah hal yang bisa salah.
 *
 * Nama yang tidak dikenal mengembalikan **false**, bukan true: lebih baik satu bagian tidak
 * tergambar dan ketahuan, daripada tergambar untuk lini yang salah dan tidak ketahuan.
 */
export function penjagaTerpenuhi(ungkapan: string, dokumen: Record<string, unknown>): boolean {
  return ungkapan
    .split('||')
    .map((bagian) => bagian.trim())
    .filter(Boolean)
    .some((nama) => PENJAGA[nama]?.(dokumen) ?? false)
}
