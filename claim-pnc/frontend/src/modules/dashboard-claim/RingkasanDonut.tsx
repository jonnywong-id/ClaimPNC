import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'

import type { Kartu, Tile } from './types'

/**
 * Donut komposisi keempat tile — padanan grafik pada layar lama.
 *
 * `Section/DashboardClaim_Section-Section.xml` menggambar donut berdampingan dengan daftar
 * "Status Register / Jumlah". Keduanya menampilkan ANGKA YANG SAMA; yang berbeda hanya
 * bentuknya.
 *
 * # Irisannya dapat diklik, sama seperti kartunya
 *
 * Di layar lama, menekan angka pada daftar membuka telusurnya. Donut di sini mengikuti
 * perilaku itu: menekan irisan memilih tile yang sama dengan menekan kartunya.
 *
 * # Angka yang sama WAJIB tetap ada sebagai teks
 *
 * Donut-nya ber-`aria-hidden`, dan itu bukan kelalaian: pembaca layar tidak boleh kehilangan
 * data. Keempat angkanya sudah tergambar sebagai kartu di atas donut ini, sehingga yang
 * disembunyikan hanyalah penggambaran ulangnya — bukan datanya.
 */
export function RingkasanDonut({
  kartu,
  terpilih,
  onPilih,
}: {
  kartu: Kartu[]
  terpilih: Tile | null
  onPilih: (tile: Tile) => void
}) {
  const total = kartu.reduce((jumlah, item) => jumlah + item.jumlah, 0)

  // Panel tetap digambar saat totalnya nol, dengan pesan yang menyebutkan sebabnya.
  // Mengembalikan null akan membuat layar melompat setiap kali penyaring mempersempit
  // hasilnya sampai habis — dan pengguna tidak tahu apakah panelnya hilang atau datanya.
  if (total === 0) {
    return (
      <section className="mb-6 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
        <h2 className="text-sm font-medium text-slate-700">Komposisi</h2>
        <p className="mt-6 pb-6 text-center text-sm text-slate-500">
          Tidak ada pekerjaan yang cocok dengan penyaring yang sedang berlaku.
        </p>
      </section>
    )
  }

  // Irisan bernilai nol dibuang: Recharts tetap menggambarnya sebagai garis tipis yang dapat
  // diklik, dan irisan tak terlihat yang menanggapi klik membingungkan.
  const data = kartu.filter((item) => item.jumlah > 0)

  return (
    <section className="mb-6 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
      <h2 className="text-sm font-medium text-slate-700">Komposisi</h2>

      <div className="h-64" aria-hidden="true">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              dataKey="jumlah"
              nameKey="judul"
              innerRadius="55%"
              outerRadius="80%"
              paddingAngle={1}
              // Animasi dimatikan: ia membuat uji komponen harus menunggu tanpa alasan.
              isAnimationActive={false}
              // Irisan dikenali lewat INDEKS, bukan lewat isi objek yang dikirim Recharts —
              // bentuk objek itu milik pustaka dan dapat berubah antar versi.
              onClick={(_irisan, index) => {
                const diklik = data[index]
                if (diklik === undefined) return
                onPilih(diklik.tile)
              }}
            >
              {data.map((item, index) => (
                <Cell
                  key={item.tile}
                  fill={warnaIrisan(index)}
                  stroke={item.tile === terpilih ? '#0f172a' : '#ffffff'}
                  strokeWidth={item.tile === terpilih ? 3 : 1}
                  className="cursor-pointer"
                />
              ))}
            </Pie>

            {/*
              Tipe `nilai` dan `nama` datang dari Recharts sebagai nilai yang boleh apa saja,
              termasuk undefined. Dinormalkan di sini alih-alih dipaksa dengan `as`: tooltip
              yang menerima bentuk tak terduga lebih baik menampilkan teks kosong daripada
              menjatuhkan seluruh grafik.
            */}
            <Tooltip
              formatter={(nilai, nama) => {
                const angka = typeof nilai === 'number' ? nilai : Number(nilai ?? 0)
                return [`${angka.toLocaleString('id-ID')} (${persen(angka, total)})`, String(nama ?? '')]
              }}
              contentStyle={{ fontSize: '0.8125rem', borderRadius: '0.5rem' }}
            />
            <Legend
              verticalAlign="bottom"
              height={36}
              wrapperStyle={{ fontSize: '0.75rem' }}
              onClick={(entri) => {
                const diklik = data.find((item) => item.judul === entri.value)
                if (diklik === undefined) return
                onPilih(diklik.tile)
              }}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </section>
  )
}

/** Persentase satu irisan, dibulatkan dua desimal seperti layar lama. */
function persen(nilai: number, total: number): string {
  if (total === 0) return '0%'
  return `${((nilai / total) * 100).toFixed(2)}%`
}

/**
 * Palet irisan.
 *
 * Nilainya sama dengan donut My Inbox supaya kedua layar terbaca sebagai satu sistem —
 * bukan karena kebetulan, melainkan karena keduanya memang grafik komposisi yang sama
 * sifatnya.
 */
const PALET = [
  '#2563eb', // blue-600
  '#d97706', // amber-600
  '#7c3aed', // violet-600
  '#0d9488', // teal-600
] as const

function warnaIrisan(index: number): string {
  // Modulo menjamin indeksnya di dalam jangkauan, tetapi TypeScript tidak dapat
  // membuktikannya. Yang dipakai `?? PALET[0]`, bukan `as string`.
  return PALET[index % PALET.length] ?? PALET[0]
}
