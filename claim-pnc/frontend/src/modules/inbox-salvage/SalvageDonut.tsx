import type { StatusCount } from './types'

/**
 * Warna irisan, DITETAPKAN menurut urutan baris pencacah.
 *
 * Bukan diacak dan bukan diturunkan dari namanya: urutan baris pencacah tetap, sehingga
 * warna sebuah status tidak berubah antarpembukaan layar. Petugas yang membuka layar ini
 * setiap hari mengenali irisannya dari warnanya sebelum membaca legendanya.
 */
const PALETTE = [
  '#3b9fd8',
  '#e8a33d',
  '#8e5bd0',
  '#3eab6b',
  '#2ec4b6',
  '#e24b4b',
  '#7fc24b',
  '#2b3f5c',
  '#c2538f',
  '#8a8f98',
]

const SIZE = 260
const CENTER = SIZE / 2
const OUTER = 86
const INNER = 46

/** Jari-jari tempat angka dicetak — DI LUAR cincin, sebagaimana di layar lama. */
const LABEL = OUTER + 22

type Slice = {
  label: string
  value: number
  color: string
  start: number
  end: number
}

/**
 * Donat **"Status Salvage"** di kepala layar Inbox Salvage.
 *
 * # Apa yang digantikan
 *
 * Grafik pada `Section/InboxSalvageASM-Section.xml` yang membaca page `PieTempALLSalvage`,
 * diisi `Activity/GCNMCountSalvage_act-Act.xml` berdampingan dengan tabel ringkasnya.
 *
 * # SATU sumber angka, bukan dua
 *
 * Di Pega, grafik dan tabel diisi DUA page berbeda — `PieTempALLSalvage` dan
 * `TempALLSalvage` — dengan daftar label yang tidak sama. Grafiknya memuat irisan
 * "Waive Salvage" yang tidak ada satu pun barisnya di tabel, sementara tabelnya memuat
 * "Tidak Ada Salvage" dan "Salvage Buyback" yang tidak ada irisannya di grafik.
 *
 * Di sini keduanya dibaca dari SATU daftar yang sama. Grafik dan tabel yang berdampingan
 * karena itu tidak dapat berselisih — dan selisih itu bukan hal yang layak direplikasi:
 * ia membuat pembacanya harus memilih mana yang dipercaya.
 *
 * # Ia TIDAK dapat diklik, dan itu disengaja
 *
 * Tabel di sebelahnya sudah menjadi navigasi — setiap barisnya membuka daftarnya. Membuat
 * irisannya ikut dapat diklik menambah jalan kedua ke tempat yang sama, dan irisan tipis
 * bernilai satu atau dua adalah sasaran klik yang buruk. Yang digambar di sini bacaan
 * sekilas atas sebarannya; perpindahan daftar tetap lewat tabelnya.
 */
export function SalvageDonut({ rows }: { rows: StatusCount[] }) {
  // Baris bernilai nol tidak punya irisan yang dapat digambar, dan menyisakannya di
  // legenda berarti menggambar warna yang tidak muncul di mana pun pada grafiknya.
  // Angkanya tetap terbaca di tabel sebelahnya.
  const dipakai = rows.filter((row) => row.jumlah > 0)
  const total = dipakai.reduce((sum, row) => sum + row.jumlah, 0)

  if (total === 0) return null

  const slices: Slice[] = []
  let berjalan = 0
  for (const [index, row] of dipakai.entries()) {
    const sapuan = (row.jumlah / total) * 360
    slices.push({
      label: row.status_salvage,
      value: row.jumlah,
      color: PALETTE[index % PALETTE.length] ?? '#8a8f98',
      start: berjalan,
      end: berjalan + sapuan,
    })
    berjalan += sapuan
  }

  return (
    <div className="flex flex-col items-center gap-4">
      <svg
        viewBox={`0 0 ${SIZE} ${SIZE}`}
        className="h-64 w-64 shrink-0"
        role="img"
        aria-label={ringkasanLisan(slices, total)}
      >
        {/*
          Satu status yang memborong seluruh angka menjadi irisan 360°, dan busur SVG
          yang titik awal dan akhirnya berimpit tidak menggambar apa pun. Cincin utuh
          karena itu digambar sebagai lingkaran bergaris tebal, bukan sebagai busur.
        */}
        {slices.length === 1 ? (
          <circle
            cx={CENTER}
            cy={CENTER}
            r={(OUTER + INNER) / 2}
            fill="none"
            stroke={slices[0]?.color}
            strokeWidth={OUTER - INNER}
          />
        ) : (
          slices.map((slice) => (
            <path
              key={slice.label}
              d={jalurIrisan(slice.start, slice.end)}
              fill={slice.color}
              stroke="#ffffff"
              strokeWidth={1}
            />
          ))
        )}

        {slices.map((slice) => {
          const [x, y] = kutub(LABEL, (slice.start + slice.end) / 2)
          return (
            <text
              key={`angka-${slice.label}`}
              x={x}
              y={y}
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-slate-700 text-[11px]"
            >
              {slice.value.toLocaleString('id-ID')}
            </text>
          )
        })}
      </svg>

      {/* Legenda di bawah grafik, sebagaimana di layar lama. */}
      <ul className="flex flex-wrap justify-center gap-x-4 gap-y-1.5 text-xs text-slate-600">
        {slices.map((slice) => (
          <li key={`legenda-${slice.label}`} className="flex items-center gap-1.5">
            <span
              aria-hidden="true"
              className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: slice.color }}
            />
            {slice.label}
          </li>
        ))}
      </ul>
    </div>
  )
}

/** kutub menerjemahkan sudut derajat menjadi titik, dengan 0° di atas. */
function kutub(radius: number, derajat: number): [number, number] {
  const radian = ((derajat - 90) * Math.PI) / 180
  return [CENTER + radius * Math.cos(radian), CENTER + radius * Math.sin(radian)]
}

/** jalurIrisan menyusun satu potongan cincin antara dua sudut. */
function jalurIrisan(start: number, end: number): string {
  const [x1, y1] = kutub(OUTER, start)
  const [x2, y2] = kutub(OUTER, end)
  const [x3, y3] = kutub(INNER, end)
  const [x4, y4] = kutub(INNER, start)
  const besar = end - start > 180 ? 1 : 0

  return [
    `M ${x1} ${y1}`,
    `A ${OUTER} ${OUTER} 0 ${besar} 1 ${x2} ${y2}`,
    `L ${x3} ${y3}`,
    `A ${INNER} ${INNER} 0 ${besar} 0 ${x4} ${y4}`,
    'Z',
  ].join(' ')
}

/**
 * ringkasanLisan menyusun bacaan grafik untuk pembaca layar.
 *
 * Grafik tanpa ini hanya terbaca sebagai gambar tanpa isi. Yang dibacakan angkanya, bukan
 * bentuknya — itulah yang dicari orang yang membukanya.
 */
function ringkasanLisan(slices: Slice[], total: number): string {
  const bagian = slices
    .map((slice) => `${slice.label} ${slice.value.toLocaleString('id-ID')}`)
    .join(', ')

  return `Sebaran status salvage, total ${total.toLocaleString('id-ID')}: ${bagian}.`
}
