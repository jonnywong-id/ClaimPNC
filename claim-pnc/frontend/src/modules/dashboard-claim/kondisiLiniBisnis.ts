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
 * Ketiga properti dulu digali dari snapshot polis di dalam dokumen JSON klaim. Sejak migrasi
 * lepas dari JSON (2026-10-12), keduanya datang dari KEPALA rincian — server membacanya dari
 * T_CLAIMLIST_ADMIN.GROUPPANEL_1 dan T_CLAIM_PNC.POLIS_JENIS_BISNIS.
 */

/**
 * Lini bisnis klaim, dibawa KEPALA rincian — bukan lagi digali dari dokumen JSON.
 *
 * Dua kode inilah yang dibandingkan rule Pega (`IsPA`, `IsHE`, `IsFire`, …). Nama bisnis
 * TIDAK dipakai: ia teks bebas, dan membandingkannya berarti menebak.
 */
export type LiniBisnisKlaim = {
  group_panel: string
  jenis_bisnis: string
}

const panel = (d: LiniBisnisKlaim) => d.group_panel.trim()
const jenis = (d: LiniBisnisKlaim) => d.jenis_bisnis.trim()
/**
 * `IsAneka` membandingkan BusinessCode, yang TIDAK ada di kepala rincian.
 *
 * Dikembalikan kosong, sehingga `IsAneka` selalu salah. Itu dinyatakan, bukan ditebak dari
 * kolom lain — dan akibatnya terbatas: varian Aneka pada grid objek tidak terpilih, sedangkan
 * varian lain tetap bekerja.
 *
 * BELUM DIPUTUSKAN — pertanyaan terbuka: kolom mana di `T_CLAIM_PNC` yang sepadan dengan
 * `BusinessCode` (`BUSINESSCODE` ada, tetapi kesepadanannya belum diukur).
 */
const kode = (_d: LiniBisnisKlaim) => ''

/** `IsPA` — `Policy.Quotation.GroupPanel = "002"`. Sama dengan `isPA_PNC`. */
export const isPA = (d: LiniBisnisKlaim) => panel(d) === '002'

/** `IsTravel` — `Policy.Quotation.GroupPanel = "005"`. */
export const isTravel = (d: LiniBisnisKlaim) => panel(d) === '005'

/** `IsAneka` — `Policy.Quotation.BusinessCode = "10140"`. */
export const isAneka = (d: LiniBisnisKlaim) => kode(d) === '10140'

/** `IsMarineCargo` — `BusinessType = "MarineCargo"` ATAU `GroupPanel = "004"`. */
export const isMarineCargo = (d: LiniBisnisKlaim) =>
  jenis(d) === 'MarineCargo' || panel(d) === '004'

/** `IsFire` — `BusinessType = "Fire"` ATAU `GroupPanel = "006"`. */
export const isFire = (d: LiniBisnisKlaim) => jenis(d) === 'Fire' || panel(d) === '006'

/** `IsHE` — `BusinessType = "HE"` ATAU `BusinessType = "ContractorsPM"`. */
export const isHE = (d: LiniBisnisKlaim) => jenis(d) === 'HE' || jenis(d) === 'ContractorsPM'

/**
 * Nama penjaga yang dipakai spesifikasi grid, dipetakan ke fungsinya.
 *
 * Dipisahkan dari spesifikasi supaya yang tertulis di sana adalah **nama When rule Pega** apa
 * adanya — sehingga penelusuran balik ke export tetap mungkin tanpa membaca kode ini.
 */
export const PENJAGA: Record<string, (d: LiniBisnisKlaim) => boolean> = {
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
export function penjagaTerpenuhi(ungkapan: string, lini: LiniBisnisKlaim): boolean {
  return ungkapan
    .split('||')
    .map((bagian) => bagian.trim())
    .filter(Boolean)
    .some((nama) => PENJAGA[nama]?.(lini) ?? false)
}
