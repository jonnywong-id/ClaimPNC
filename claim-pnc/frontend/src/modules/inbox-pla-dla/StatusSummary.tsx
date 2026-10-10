import type { RingkasDaftar } from './types'

type Props = {
  rows: RingkasDaftar[]
  active: string
  isLoading: boolean
  onSelect: (kode: string) => void
}

/**
 * StatusSummary adalah tabel **"Status / Jumlah"** di samping daftar.
 *
 * # Ia NAVIGASI, bukan hiasan
 *
 * Di Pega layar ini tidak punya bilah tab sama sekali: tabel inilah satu-satunya cara
 * berpindah daftar. Sel "Jumlah" ber-`pyFormat=pxLink` dengan aksi `refresh` yang membawa
 * `.CityID` — kode daftarnya (`Section/InboxDLAReas_sect-Section.xml`).
 *
 * # Barisnya DAFTAR, bukan status klaim
 *
 * `Activity/GetLostAdjuster_act-Act.xml` menyusunnya satu per satu, dan `.CauseOfLoss`
 * diisi NAMA DAFTAR — `"NOT ANSWERED"`, `"NOT REPLIED FROM ASM"`, `"REPLIED FROM ASM"`,
 * dan seterusnya. Aliasnya menyesatkan seluruhnya; tidak satu pun berhubungan dengan
 * penyebab kerugian.
 *
 * Versi sebelumnya menggambar cacahan STATUS KLAIM di dalam daftar yang sedang terbuka —
 * hal yang tidak ada di layar Pega mana pun — dan menggambarnya sebagai chip, bukan
 * tabel. Keduanya dikoreksi 2026-10-09 atas permintaan Work Owner.
 *
 * # Kenapa barisnya digambar meski jumlahnya NOL
 *
 * Karena Pega menggambarnya. Tangkapan layar yang menyertai permintaan memperlihatkan
 * keenam barisnya dengan angka nol seluruhnya — dan itu memang bentuk yang benar: tabel
 * yang menghilangkan barisnya saat kosong membuat pengguna menyimpulkan daftarnya tidak
 * ada, bukan bahwa daftarnya sedang kosong.
 *
 * Versi sebelumnya `return null` saat tak ada baris, sehingga pada keadaan di tangkapan
 * layar itu tabelnya **tidak tergambar sama sekali**.
 *
 * # Kenapa TANPA bagan
 *
 * Pega menggambar bagan batang di sampingnya. Ia tidak dibawa: bagannya menggambarkan hal
 * yang sama dengan enam angka di sebelahnya, dan enam angka tidak menjadi lebih terbaca
 * karena digambar sebagai batang. Ia juga menambah satu pustaka bagan ke dalam bundel.
 */
export function StatusSummary({ rows, active, isLoading, onSelect }: Props) {
  return (
    <section
      className="rounded-kartu border border-slate-200 bg-white shadow-lembut"
      aria-label="Status dan jumlah klaim per daftar"
    >
      <table className="w-full border-collapse text-sm">
        <caption className="sr-only">
          Jumlah klaim pada setiap daftar. Pilih angkanya untuk membuka daftar itu.
        </caption>
        <thead>
          <tr className="border-b border-slate-200">
            <th
              scope="col"
              className="px-4 py-2.5 text-left font-semibold text-slate-700"
            >
              Status
            </th>
            <th
              scope="col"
              className="px-4 py-2.5 text-left font-semibold text-slate-700"
            >
              Jumlah
            </th>
          </tr>
        </thead>
        <tbody>
          {isLoading && (
            <tr>
              <td colSpan={2} className="px-4 py-3 text-slate-600">
                Menghitung jumlah tiap daftar…
              </td>
            </tr>
          )}

          {!isLoading &&
            rows.map((baris) => {
              const terbuka = baris.kode === active
              return (
                <tr
                  key={baris.kode}
                  className={`border-b border-slate-100 last:border-b-0 ${
                    terbuka ? 'bg-blue-50' : ''
                  }`}
                >
                  <th
                    scope="row"
                    className={`px-4 py-2.5 text-left font-normal ${
                      terbuka ? 'font-semibold text-blue-900' : 'text-slate-800'
                    }`}
                  >
                    {baris.status}
                  </th>
                  <td className="px-4 py-2.5">
                    {/*
                      Angkanya TOMBOL, bukan tautan `<a>`.

                      Ia tidak menuju alamat mana pun — ia mengganti isi grid di sebelahnya,
                      persis seperti aksi `refresh` di Pega. Menggambarnya sebagai `<a>`
                      tanpa `href` yang sah akan menjanjikan hal yang tidak ia lakukan:
                      membuka di tab baru, menyalin alamat, dan dibaca pembaca layar
                      sebagai tautan.
                    */}
                    <button
                      type="button"
                      onClick={() => { onSelect(baris.kode) }}
                      aria-label={`${baris.jumlah} klaim pada daftar ${baris.status}`}
                      aria-current={terbuka ? 'true' : undefined}
                      className="rounded-kontrol px-1 font-semibold text-blue-700 underline-offset-2 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                    >
                      {baris.jumlah}
                    </button>
                  </td>
                </tr>
              )
            })}
        </tbody>
      </table>
    </section>
  )
}
