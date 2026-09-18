import type { ReactNode } from 'react'

/** Satu kolom tabel. */
export type KolomTabel<T> = {
  /** Pengenal kolom, dipakai sebagai key React. */
  kunci: string
  /** Judul yang tampil di kepala kolom. */
  judul: string
  /** Isi sel untuk satu baris. */
  isi: (baris: T) => ReactNode
  /**
   * Kelas Tailwind tambahan untuk sel dan kepala kolom — dipakai mengatur lebar atau
   * perataan. Contoh: `'w-24'`, `'text-right'`.
   */
  kelas?: string
}

type Props<T> = {
  kolom: KolomTabel<T>[]
  baris: T[]
  /** Nilai unik per baris; dipakai React untuk melacak baris antar-render. */
  kunciBaris: (baris: T) => string
  /** Keterangan di atas tabel — sebutkan sumber datanya bila ada. */
  keterangan?: string
  /** Pesan saat belum ada satu baris pun. */
  pesanKosong?: string
  /** Lebar minimum tabel sebelum digulirkan mendatar. */
  lebarMinimum?: string
}

/**
 * TabelData adalah SATU-SATUNYA tabel yang boleh dipakai layar di folder `modules/`.
 *
 * # Kenapa komponen ini ada
 *
 * Sistem lama memuat 269 section, dan **268 di antaranya memakai pola grid yang sama**.
 * Di Pega pola itu diulang di setiap section; di sini ia dibangun sekali dan dipakai
 * ratusan kali. Itulah *leverage* yang dimaksud `06-MODULE-BREAKDOWN.md` §4 ketika
 * menyebut `U-2` sebagai investasi terpenting di frontend — dan sekaligus *locality*:
 * perbaikan di satu tempat memperbaiki seluruh layar.
 *
 * Tanpa komponen ini, 74 layar akan berisi 74 tafsir berbeda tentang bagaimana sebuah
 * tabel terlihat, bagaimana ia berperilaku saat kosong, dan bagaimana ia bersikap di
 * lebar layar tablet. Persis kegagalan yang `D-23` minta dicegah.
 *
 * # Lingkup yang sengaja dibatasi
 *
 * Ia menampilkan baris yang SUDAH disiapkan pemanggil. Ia tidak menyaring, tidak
 * mengurutkan, dan tidak memaginasi. Ketiganya sengaja tidak ada di sini karena
 * `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 menetapkan penyaringan dan paginasi berjalan
 * di sisi server — memasangnya di komponen ini akan mengajak layar berikutnya
 * mengambil seluruh baris lalu memotongnya di peramban, tepat yang dilarang.
 *
 * Master status progres berjumlah sedikit sehingga seluruh barisnya memang dimuat
 * sekaligus. Layar bervolume besar nanti menambah paginasi server-side sebagai properti
 * baru di sini, bukan dengan membuat tabel keduanya.
 */
export function TabelData<T>({
  kolom,
  baris,
  kunciBaris,
  keterangan,
  pesanKosong = 'Belum ada data.',
  lebarMinimum = 'min-w-[32rem]',
}: Props<T>) {
  return (
    // Pembungkus bergulir mendatar: surveyor memakai tablet dan ponsel (D-12), dan
    // tabel yang memaksa seluruh halaman melebar membuat layarnya tidak dapat dibaca.
    <div className="overflow-x-auto">
      <table className={`w-full border-collapse text-sm ${lebarMinimum}`}>
        {keterangan && (
          <caption className="mb-2 text-left text-xs uppercase tracking-wide text-slate-500">
            {keterangan}
          </caption>
        )}
        <thead>
          <tr className="border-b border-slate-200 text-left text-slate-600">
            {kolom.map((k) => (
              <th key={k.kunci} scope="col" className={`py-2 pr-4 font-medium ${k.kelas ?? ''}`}>
                {k.judul}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {baris.length === 0 ? (
            <tr>
              {/* Tabel kosong tetap punya baris, bukan tabel yang hilang: pengguna
                  perlu tahu daftarnya memang kosong — bukan mengira layarnya gagal. */}
              <td colSpan={kolom.length} className="py-6 text-center text-slate-500">
                {pesanKosong}
              </td>
            </tr>
          ) : (
            baris.map((b) => (
              <tr key={kunciBaris(b)} className="border-b border-slate-100">
                {kolom.map((k) => (
                  <td key={k.kunci} className={`py-2 pr-4 text-slate-900 ${k.kelas ?? ''}`}>
                    {k.isi(b)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
