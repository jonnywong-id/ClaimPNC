import { useState } from "react";

import { NetworkError } from "@/api/client";
import { ErrorMessage } from "@/components/ErrorMessage";
import { SearchIcon } from "@/components/Icon";

import { useOutstandingSummary } from "./api";
import type {
  DocumentStatusCode,
  DocumentStatusCount,
  OutstandingFilter,
} from "./types";

/** Id panel isi tab — satu panel yang isinya berganti mengikuti tab aktif. */
export const PANEL_ID = "panel-status-dokumen";

/** Id tombol tab, dirujuk panel lewat `aria-labelledby`. */
export function tabId(kode: string): string {
  return `tab-status-dokumen-${kode}`;
}

type Props = {
  /** Penyaring layar yang sedang berlaku; jumlah tab menghormati semuanya kecuali status. */
  filter: OutstandingFilter;

  /** Status yang sedang dipilih; teks kosong berarti seluruh status (tab "ALL Case"). */
  selected: DocumentStatusCode | "";
  onSelect: (status: DocumentStatusCode | "") => void;
};

/**
 * Deret tab status dokumen My Inbox.
 *
 * # Apa yang digantikannya
 *
 * `Section/InboxRegister_Section-Section.xml` menampilkan status dokumen lewat TIGA hal
 * sekaligus: sembilan tab bertuliskan tebal (`:11438`, `:11626`, `:11841`, `:12023`,
 * `:12170`, `:12632`, `:12825`, `:12971`, `:13184`), sebuah donut (`:4151` —
 * `pyType=pie`, `pySubType=doughnut`), dan tabel jumlahnya.
 *
 * Ketiganya diganti **satu deret tab** atas keputusan Work Owner 2026-10-05: donut dan
 * tabel hierarkinya tidak dibawa. Angka yang dulu dibaca dari keduanya kini hidup sebagai
 * lencana pada tabnya sendiri — satu tempat, bukan tiga, dan tidak ada lagi dua sajian
 * yang bisa berbeda tanpa ada yang menyadarinya.
 *
 * Perilakunya mengikuti `SourceTab` pada Inbox Auto Claim: `role="tablist"` dan berpindah
 * dengan panah. Yang ditiru POLANYA; modulnya tidak disentuh.
 *
 * # Daftar di samping, bukan deret tab di atas
 *
 * Sejak 2026-10-14, atas permintaan pengguna, tabnya digambar sebagai PANEL DAFTAR
 * VERTIKAL di kiri grid, bukan deret bergaris bawah di atasnya. Sembilan judul panjang dalam satu deret
 * tidak muat dan harus digulir menyamping, sehingga tab di ujung kanan tidak terlihat;
 * dalam daftar, semuanya terlihat sekaligus dan jumlahnya rata kanan, mudah dibandingkan.
 * Semantiknya tidak berubah: tetap `tablist`, kini `aria-orientation="vertical"`.
 *
 * Komponen ini sengaja HANYA menggambar panelnya. Tata letak dua kolom diatur halaman
 * dengan flex, supaya bila panel tidak digambar (respons tanpa status) grid otomatis
 * mengisi lebar penuh, tanpa kolom kosong di kirinya.
 *
 * # Tab tanpa lencana bukan tab bernilai nol
 *
 * Enam dari sembilan tab belum dapat dihitung dari `T_CLAIMLIST_ADMIN`:
 *
 *     Loss Adjuster · Internal Surveyor   Work-SurveyClaim — case type LAIN
 *     Temporary Close · Deadline          Resolved-Completed — di LUAR himpunan ini
 *     Communication                       tabel komunikasi
 *     TKA                                 kolom TKA_1 TIDAK ADA di tabel ini
 *
 * Keenamnya tetap DIGAMBAR supaya petugas Pega mengenali layarnya, tetapi tanpa lencana
 * dan tidak dapat ditekan. Memberinya angka nol akan menyatakan "tidak ada satu pun",
 * padahal yang benar "belum dihitung".
 */
export function DocumentStatusTab({ filter, selected, onSelect }: Props) {
  const summary = useOutstandingSummary(filter);

  if (summary.isError) {
    return (
      <div className={ASIDE_CLASS}>
        <ErrorMessage
          title="Status dokumen tidak dapat dimuat"
          description={
            summary.error instanceof NetworkError
              ? "Server Claim PNC tidak dapat dihubungi."
              : "Daftar klaim di bawah tetap dapat dipakai. Coba muat ulang halaman ini."
          }
          tone="gangguan"
        />
      </div>
    );
  }

  if (summary.isPending) {
    return (
      <div
        className={`${ASIDE_CLASS} h-96 animate-pulse rounded-kartu bg-slate-100`}
        aria-label="Memuat status dokumen"
      />
    );
  }

  // `?? []` bukan kerapian: respons yang cacat atau versi backend yang lebih tua akan
  // menjatuhkan SELURUH layar bila dibaca membabi buta — dan yang hilang hanyalah deret
  // tab ini, sementara daftar klaim di bawahnya tetap berguna.
  const status = summary.data.status ?? [];
  const total = summary.data.total ?? 0;

  if (status.length === 0) return null;

  return (
    <aside
      className={`${ASIDE_CLASS} rounded-kartu border border-slate-200 bg-white p-3 shadow-lembut`}
    >
      <StatusTabList
        status={status}
        total={total}
        selected={selected}
        onSelect={onSelect}
      />
    </aside>
  );
}

/**
 * Lebar kolom panel. Di bawah `2xl` panelnya menumpuk di atas grid selebar penuh (alasannya
 * di `OutstandingPage`); di atasnya ia kolom tetap 16rem — cukup untuk judul terpanjang
 * ("Deadline To Temporary Close") dalam dua baris tanpa memotong lencananya.
 */
const ASIDE_CLASS = "w-full 2xl:w-64 2xl:shrink-0";

/**
 * Daftar tabnya sendiri.
 *
 * # Kenapa `role="tablist"`, bukan sekadar deretan tombol
 *
 * Dengan peran yang benar, pembaca layar mengumumkan "tab 2 dari 9" dan panah
 * atas/bawah berpindah tab — perilaku yang diharapkan pengguna papan ketik dari sesuatu
 * yang terlihat seperti tab. Deretan tombol biasa terbaca sebagai sembilan tombol lepas
 * tanpa hubungan. Panah kiri/kanan tetap diterima: di bawah `2xl` daftarnya berkolom, dan
 * pengguna yang terbiasa dengan deret tab lama tidak kehilangan kebiasaannya.
 *
 * # Kotak cari hanya menyaring JUDUL di panel ini
 *
 * Ia tidak menyentuh grid maupun kueri ke server. Tab yang sedang aktif tetap aktif meski
 * tersembunyi oleh pencarian.
 */
function StatusTabList({
  status,
  total,
  selected,
  onSelect,
}: {
  status: DocumentStatusCount[];
  total: number;
  selected: DocumentStatusCode | "";
  onSelect: (status: DocumentStatusCode | "") => void;
}) {
  const [cari, setCari] = useState("");
  const kata = cari.trim().toLowerCase();
  const tampil =
    kata === ""
      ? status
      : status.filter((s) => s.judul.toLowerCase().includes(kata));

  // Hanya tab yang dapat dipilih — dan sedang terlihat — yang ikut navigasi panah.
  // Memasukkan yang mati akan membuat panah berhenti di tab yang tidak dapat dibuka, dan
  // pengguna papan ketik kehilangan cara meneruskannya.
  const dapatDipilih = tampil.filter((s) => s.dapat_dipilih);

  /** Tab "ALL Case" berarti TANPA penyaring; kode selainnya menyaring satu status. */
  function kodeTerpilih(s: DocumentStatusCount): DocumentStatusCode | "" {
    return s.kode === "semua" ? "" : s.kode;
  }

  function aktif(s: DocumentStatusCount): boolean {
    return kodeTerpilih(s) === selected;
  }

  function pindah(arah: -1 | 1) {
    if (dapatDipilih.length === 0) return;
    const sekarang = dapatDipilih.findIndex((s) => aktif(s));
    // Tab aktif yang tersembunyi oleh pencarian: panah membawa ke ujung daftar yang
    // terlihat, bukan diam.
    const tujuan =
      sekarang < 0
        ? arah === 1
          ? 0
          : dapatDipilih.length - 1
        : // Berputar di ujungnya, mengikuti perilaku tab yang lazim.
          (sekarang + arah + dapatDipilih.length) % dapatDipilih.length;
    const berikut = dapatDipilih[tujuan];
    if (berikut !== undefined) onSelect(kodeTerpilih(berikut));
  }

  return (
    <>
      <label htmlFor="status-dokumen-cari" className="sr-only">
        Cari status dokumen
      </label>
      <div className="relative">
        <span
          aria-hidden="true"
          className="pointer-events-none absolute inset-y-0 left-0 flex w-9 items-center justify-center text-slate-400"
        >
          <SearchIcon className="h-4 w-4" />
        </span>
        <input
          id="status-dokumen-cari"
          type="search"
          value={cari}
          onChange={(e) => setCari(e.target.value)}
          placeholder="Cari status…"
          className={[
            "w-full rounded-kontrol border border-slate-300 bg-white py-2 pl-9 pr-3",
            "text-sm text-slate-900 placeholder:text-slate-400",
            "transition-[border-color,box-shadow] duration-150 ease-halus",
            "hover:border-slate-400",
            "focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15",
          ].join(" ")}
        />
      </div>

      <div
        role="tablist"
        aria-label="Status dokumen"
        aria-orientation="vertical"
        className="mt-3 grid gap-0.5 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-1"
      >
        {tampil.map((s) => {
          const dipilih = aktif(s);

          // "ALL Case" memakai total, bukan jumlahnya sendiri: ia keadaan tanpa penyaring,
          // dan angkanya wajib cocok dengan total paginasi grid di sampingnya.
          const jumlah = s.kode === "semua" ? total : s.jumlah;

          return (
            <button
              key={s.kode}
              id={tabId(s.kode)}
              role="tab"
              type="button"
              disabled={!s.dapat_dipilih}
              aria-selected={dipilih}
              aria-controls={dipilih ? PANEL_ID : undefined}
              // Hanya tab yang aktif yang dapat di-Tab-kan; panah memindahkan yang lain.
              // Itu pola papan ketik baku untuk tablist, dan tanpa itu pengguna papan
              // ketik harus menekan Tab melewati sembilan tab sebelum sampai ke isinya.
              tabIndex={dipilih ? 0 : -1}
              /*
                Lencananya diberi nama, bukan dibiarkan terbaca sebagai angka telanjang.
                Tanpa ini pembaca layar menyebut "Complete documents0" — angkanya menempel
                tanpa jeda, dan "0" sendirian tidak menyatakan nol apa.
              */
              aria-label={
                !s.dapat_dipilih
                  ? `${s.judul}, belum tersedia`
                  : jumlah === null
                    ? s.judul
                    : `${s.judul}, ${jumlah} klaim`
              }
              title={
                s.dapat_dipilih
                  ? undefined
                  : "Sumber datanya belum dimigrasikan dari Pega."
              }
              onClick={() => onSelect(kodeTerpilih(s))}
              onKeyDown={(event) => {
                if (event.key === "ArrowDown" || event.key === "ArrowRight") {
                  event.preventDefault();
                  pindah(1);
                }
                if (event.key === "ArrowUp" || event.key === "ArrowLeft") {
                  event.preventDefault();
                  pindah(-1);
                }
              }}
              className={[
                "flex w-full items-center justify-between gap-3 rounded-kontrol border-l-[3px] px-3 py-2.5 text-left text-sm",
                "transition-[color,background-color,border-color] duration-150 ease-halus",
                "focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20",
                !s.dapat_dipilih
                  ? "cursor-not-allowed border-transparent text-slate-400"
                  : dipilih
                    ? "border-blue-600 bg-blue-50 font-semibold text-blue-700"
                    : "border-transparent text-slate-700 hover:bg-slate-50 hover:text-slate-900",
              ].join(" ")}
            >
              <span className="min-w-0">{s.judul}</span>
              {/*
                Lencana hanya digambar bila jumlahnya BENAR-BENAR dihitung — konvensi yang
                sama dengan tab "Data rejected" pada Inbox Laporan Klaim.
              */}
              {jumlah !== null && (
                <span
                  aria-hidden="true"
                  className={[
                    "shrink-0 tabular-nums",
                    dipilih
                      ? "font-semibold text-blue-700"
                      : "font-medium text-slate-600",
                  ].join(" ")}
                >
                  {jumlah}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {tampil.length === 0 && (
        <p className="mt-3 px-3 text-sm text-slate-500" role="status">
          Tidak ada status yang cocok.
        </p>
      )}
    </>
  );
}
