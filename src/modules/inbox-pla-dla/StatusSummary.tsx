import {
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
} from "recharts";

import type { RingkasDaftar } from "./types";

type Props = {
  rows: RingkasDaftar[];
  active: string;
  isLoading: boolean;
  onSelect: (kode: string) => void;
};

/**
 * Palet irisan donat, berurutan mengikuti urutan baris tabel.
 *
 * Nilainya sama dengan donat Inbox Admin (`StatusRegister`) supaya kedua layar terbaca
 * sebagai satu sistem — keduanya memang grafik komposisi antrean yang sama sifatnya.
 */
const WARNA_IRISAN = [
  "#2f9fd0",
  "#f59e0b",
  "#10b981",
  "#8b5cf6",
  "#ef4444",
  "#64748b",
] as const;

/**
 * StatusSummary adalah panel atas layar: **donat** di kiri, tabel **"Status / Jumlah"** di
 * kanan — susunan `Section/InboxDLAReas_sect-Section.xml` apa adanya.
 *
 * # Tabelnya NAVIGASI, bukan hiasan
 *
 * Di Pega layar ini tidak punya bilah tab sama sekali: tabel inilah satu-satunya cara
 * berpindah daftar. Sel "Jumlah" ber-`pyFormat=pxLink` dengan aksi `refresh` yang membawa
 * `.CityID` — kode daftarnya.
 *
 * # Barisnya DAFTAR, bukan status klaim
 *
 * `Activity/GetLostAdjuster_act-Act.xml` menyusunnya satu per satu, dan `.CauseOfLoss`
 * diisi NAMA DAFTAR — `"NOT ANSWERED"`, `"NOT REPLIED FROM ASM"`, `"REPLIED FROM ASM"`,
 * dan seterusnya. Aliasnya menyesatkan seluruhnya; tidak satu pun berhubungan dengan
 * penyebab kerugian.
 *
 * # Kenapa barisnya digambar meski jumlahnya NOL
 *
 * Karena Pega menggambarnya. Tabel yang menghilangkan barisnya saat kosong membuat
 * pengguna menyimpulkan daftarnya tidak ada, bukan bahwa daftarnya sedang kosong.
 *
 * # Donatnya DIBAWA — koreksi atas keputusan 2026-10-09
 *
 * Versi sebelumnya sengaja tidak membawanya, dengan alasan enam angka tidak menjadi lebih
 * terbaca karena digambar sebagai irisan. Alasan itu dicabut atas permintaan Work Owner
 * (2026-10-11): `D-13` menetapkan tata letak Pega ditiru supaya petugas tidak perlu
 * belajar ulang, dan donat itu adalah hal pertama yang mereka lihat di layar lama.
 *
 * Biayanya pun ternyata nol: `recharts` sudah masuk bundel lewat Dashboard Claim dan Inbox
 * Admin, sehingga ia tidak menambah satu pustaka pun.
 *
 * Donat-nya `aria-hidden` dan irisan bernilai nol dibuang — Recharts tetap menggambar
 * irisan nol sebagai garis tipis yang dapat diklik, dan irisan tak terlihat yang menanggapi
 * klik membingungkan. Angkanya tidak hilang bagi pembaca layar: keenamnya tergambar sebagai
 * tabel tepat di sebelahnya.
 */
export function StatusSummary({ rows, active, isLoading, onSelect }: Props) {
  const irisan = rows
    .map((baris, index) => ({
      ...baris,
      warna: WARNA_IRISAN[index % WARNA_IRISAN.length] ?? WARNA_IRISAN[0],
    }))
    .filter((baris) => baris.jumlah > 0);

  return (
    <section
      className="grid gap-6 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut lg:grid-cols-2"
      aria-label="Status dan jumlah klaim per daftar"
    >
      <div className="flex min-h-56 items-center justify-center">
        {isLoading ? (
          <p className="text-sm text-slate-500">
            Menghitung jumlah tiap daftar…
          </p>
        ) : irisan.length === 0 ? (
          <p className="text-sm text-slate-500">
            Tidak ada daftar yang berisi.
          </p>
        ) : (
          <div className="h-64 w-full" aria-hidden="true">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={irisan}
                  dataKey="jumlah"
                  nameKey="status"
                  innerRadius="45%"
                  outerRadius="75%"
                  paddingAngle={irisan.length > 1 ? 1 : 0}
                  label={({ value }) => String(value ?? "")}
                  // Animasi dimatikan: ia membuat uji komponen harus menunggu tanpa alasan.
                  isAnimationActive={false}
                  // Irisan dikenali lewat INDEKS, bukan lewat isi objek yang dikirim
                  // Recharts — bentuk objek itu milik pustaka dan dapat berubah antarversi.
                  onClick={(_irisan, index) => {
                    const diklik = irisan[index];
                    if (diklik === undefined) return;
                    onSelect(diklik.kode);
                  }}
                >
                  {irisan.map((baris) => (
                    <Cell
                      key={baris.kode}
                      fill={baris.warna}
                      stroke={baris.kode === active ? "#0f172a" : "#ffffff"}
                      strokeWidth={baris.kode === active ? 3 : 1}
                      className="cursor-pointer"
                    />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    fontSize: "0.8125rem",
                    borderRadius: "0.5rem",
                  }}
                />
                <Legend
                  verticalAlign="bottom"
                  height={36}
                  wrapperStyle={{ fontSize: "0.75rem" }}
                  onClick={(entri) => {
                    const diklik = irisan.find(
                      (baris) => baris.status === entri.value,
                    );
                    if (diklik === undefined) return;
                    onSelect(diklik.kode);
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      <table className="w-full self-start border-collapse text-sm">
        <caption className="sr-only">
          Jumlah klaim pada setiap daftar. Pilih angkanya untuk membuka daftar
          itu.
        </caption>
        <thead>
          <tr className="border-b border-slate-300">
            <th
              scope="col"
              className="px-2 py-2 text-left font-semibold text-slate-700"
            >
              Status
            </th>
            <th
              scope="col"
              className="w-28 px-2 py-2 text-left font-semibold text-slate-700"
            >
              Jumlah
            </th>
          </tr>
        </thead>
        <tbody>
          {isLoading && (
            <tr>
              <td colSpan={2} className="px-2 py-3 text-slate-600">
                Menghitung jumlah tiap daftar…
              </td>
            </tr>
          )}

          {!isLoading &&
            rows.map((baris) => {
              const terbuka = baris.kode === active;
              return (
                <tr
                  key={baris.kode}
                  className={`border-b border-slate-200 last:border-b-0 ${
                    terbuka ? "bg-blue-50" : ""
                  }`}
                >
                  <th
                    scope="row"
                    className={`px-2 py-2 text-left font-normal ${
                      terbuka ? "font-semibold text-blue-900" : "text-slate-800"
                    }`}
                  >
                    {/*
                      Ikon map digambar seperti di layar lama, dan ia `aria-hidden`: ia
                      tidak menambah keterangan apa pun yang belum disebutkan nama
                      daftarnya.
                    */}
                    <span aria-hidden="true" className="mr-2">
                      📁
                    </span>
                    {baris.status}
                  </th>
                  <td className="px-2 py-2">
                    {/*
                      Angkanya TOMBOL, bukan tautan `<a>`.

                      Ia tidak menuju alamat mana pun — ia mengganti isi grid di bawahnya,
                      persis seperti aksi `refresh` di Pega. Menggambarnya sebagai `<a>`
                      tanpa `href` yang sah akan menjanjikan hal yang tidak ia lakukan:
                      membuka di tab baru, menyalin alamat, dan dibaca pembaca layar
                      sebagai tautan.
                    */}
                    <button
                      type="button"
                      onClick={() => {
                        onSelect(baris.kode);
                      }}
                      aria-label={`${baris.jumlah} klaim pada daftar ${baris.status}`}
                      aria-current={terbuka ? "true" : undefined}
                      className="rounded-kontrol px-1 font-semibold text-blue-700 underline-offset-2 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                    >
                      {baris.jumlah}
                    </button>
                  </td>
                </tr>
              );
            })}
        </tbody>
      </table>
    </section>
  );
}
