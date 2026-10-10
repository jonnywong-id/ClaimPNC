/**
 * Satu irisan cincin, sudah lengkap dengan warna dan nilainya.
 *
 * Bentuknya ditetapkan PEMANGGIL, bukan diturunkan di sini, supaya grafik dan tabel di
 * sebelahnya membaca daftar yang sama persis — dua sumber angka yang berdampingan dan
 * berselisih memaksa pembacanya memilih mana yang dipercaya.
 */
export type DonutSlice = {
  label: string
  value: number
  color: string
}

const SIZE = 260
const CENTER = SIZE / 2
const OUTER = 86
const INNER = 46

/** Jari-jari tempat angka dicetak — DI LUAR cincin, sebagaimana di layar lama. */
const LABEL = OUTER + 22

/**
 * Sudut mulai irisan pertama: pukul sembilan, bukan pukul dua belas.
 *
 * Bukan selera. Pada layar lama dengan dua irisan berimbang, "Answered" menempati PARUH
 * ATAS cincin dan angkanya tercetak di atasnya. Mulai dari pukul dua belas akan
 * menempatkannya di paruh KANAN, dan angkanya pindah ke samping — cincin yang sama
 * persis isinya, tetapi tidak dikenali lagi oleh petugas yang membukanya tiap hari.
 */
const START = 270

type Slice = DonutSlice & { start: number; end: number }

/**
 * Donat **"Answered / Not Answered"** di kepala layar Inbox Komunikasi Cabang.
 *
 * # Apa yang digantikan
 *
 * Grafik `Embed-Control-Mode-Chart` pada `Section/InboxKomunikasi-Section.xml` yang membaca
 * page `PietempCountKomunikasiCabang`, diisi `Activity/PNCCountKomunikasiCabang_Act-Act.xml`
 * langkah 10 dan 13 — dua baris, berlabel harfiah `"Answered"` dan `"Not Answered"`.
 *
 * # Kenapa ia KEMBALI digambar
 *
 * Versi pertama modul ini menggantinya dengan lencana angka di bilah tab, dengan alasan
 * kedua daftar menjadi tab dan angkanya harus tetap terbaca sekilas. Alasan itu benar, dan
 * lencananya tetap ada — tetapi ia menjawab "berapa", bukan "seberapa banding seberapa".
 * Perbandingan dua angka adalah hal yang paling cepat dibaca dari sebuah cincin dan paling
 * lambat dibaca dari dua angka yang terpisah oleh sebuah nama tab.
 *
 * # Ia TIDAK dapat diklik, dan itu disengaja
 *
 * Tabel di sebelahnya sudah menjadi navigasi — setiap barisnya membuka tabnya. Membuat
 * irisannya ikut dapat diklik menambah jalan kedua ke tempat yang sama, dan irisan tipis
 * bernilai satu adalah sasaran klik yang buruk. Alasan yang sama dipakai donat Inbox
 * Salvage.
 */
export function KomunikasiDonut({ slices: rows }: { slices: DonutSlice[] }) {
  // Baris bernilai nol tidak punya irisan yang dapat digambar, dan menyisakannya di legenda
  // berarti menggambar warna yang tidak muncul di mana pun pada grafiknya. Angkanya tetap
  // terbaca di tabel sebelahnya.
  const used = rows.filter((row) => row.value > 0)
  const total = used.reduce((sum, row) => sum + row.value, 0)

  if (total === 0) return null

  const slices: Slice[] = []
  let running = START
  for (const row of used) {
    const sweep = (row.value / total) * 360
    slices.push({ ...row, start: running, end: running + sweep })
    running += sweep
  }

  return (
    <div className="flex flex-col items-center gap-4">
      <svg
        viewBox={`0 0 ${SIZE} ${SIZE}`}
        className="h-64 w-64 shrink-0"
        role="img"
        aria-label={spokenSummary(slices, total)}
      >
        {/*
          Satu status yang memborong seluruh angka menjadi irisan 360°, dan busur SVG yang
          titik awal dan akhirnya berimpit tidak menggambar apa pun. Cincin utuh karena itu
          digambar sebagai lingkaran bergaris tebal, bukan sebagai busur.
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
              d={slicePath(slice.start, slice.end)}
              fill={slice.color}
              stroke="#ffffff"
              strokeWidth={1}
            />
          ))
        )}

        {slices.map((slice) => {
          const [x, y] = polar(LABEL, (slice.start + slice.end) / 2)
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

/** polar menerjemahkan sudut derajat menjadi titik, dengan 0° di atas. */
function polar(radius: number, degrees: number): [number, number] {
  const radian = ((degrees - 90) * Math.PI) / 180
  return [CENTER + radius * Math.cos(radian), CENTER + radius * Math.sin(radian)]
}

/** slicePath menyusun satu potongan cincin antara dua sudut. */
function slicePath(start: number, end: number): string {
  const [x1, y1] = polar(OUTER, start)
  const [x2, y2] = polar(OUTER, end)
  const [x3, y3] = polar(INNER, end)
  const [x4, y4] = polar(INNER, start)
  const large = end - start > 180 ? 1 : 0

  return [
    `M ${x1} ${y1}`,
    `A ${OUTER} ${OUTER} 0 ${large} 1 ${x2} ${y2}`,
    `L ${x3} ${y3}`,
    `A ${INNER} ${INNER} 0 ${large} 0 ${x4} ${y4}`,
    'Z',
  ].join(' ')
}

/**
 * spokenSummary menyusun bacaan grafik untuk pembaca layar.
 *
 * Grafik tanpa ini hanya terbaca sebagai gambar tanpa isi. Yang dibacakan angkanya, bukan
 * bentuknya — itulah yang dicari orang yang membukanya.
 */
function spokenSummary(slices: Slice[], total: number): string {
  const parts = slices
    .map((slice) => `${slice.label} ${slice.value.toLocaleString('id-ID')}`)
    .join(', ')

  return `Sebaran status register komunikasi, total ${total.toLocaleString('id-ID')}: ${parts}.`
}
