import type { AutoClaimTab } from "@/api/types";

/** Id panel isi tab — satu panel yang isinya berganti mengikuti tab aktif. */
export const PANEL_ID = "panel-jenis-klaim";

/** Id tombol tab, dirujuk panel lewat `aria-labelledby`. */
export function tabId(kode: string): string {
  return `tab-jenis-klaim-${kode}`;
}

type Props = {
  tab: AutoClaimTab[];
  selected: string;
  onSelect: (source: string) => void;
};

/**
 * Tab ANEKA / Asuransi Kredit / Travel.
 *
 * # Ketiganya satu harness, tiga tabel
 *
 * `InboxAutoClaim/InboxAutoClaim-Harness.xml` memuat ketiga tab ini (`pyCaption`), dan
 * masing-masing membaca TABEL yang berbeda:
 *
 *	ANEKA            TMP_BATCH_AUTO_CLAIM    kolom INISIALID
 *	Asuransi Kredit  TMP_BATCH_CLAIM_KREDIT  kolom AGENID
 *	Travel           TMP_BATCH_AUTO_TRAVEL   kolom INISIALID
 *
 * Label yang dipakai di sini disalin dari harness, bukan diturunkan dari nama rule:
 * rule-nya bernama `*_AutoClaim` sementara yang dilihat pengguna adalah **ANEKA**.
 *
 * # Kenapa `role="tablist"`, bukan sekadar deretan tombol
 *
 * Dengan peran yang benar, pembaca layar mengumumkan "tab 2 dari 3" dan panah kiri/kanan
 * berpindah tab — perilaku yang diharapkan pengguna papan ketik dari sesuatu yang terlihat
 * seperti tab. Deretan tombol biasa terbaca sebagai tiga tombol lepas tanpa hubungan.
 */
export function SourceTab({ tab, selected, onSelect }: Props) {
  if (tab.length === 0) return null;

  function pindah(arah: -1 | 1) {
    const sekarang = tab.findIndex((t) => t.kode === selected);
    if (sekarang < 0) return;
    // Berputar di ujungnya, mengikuti perilaku tab yang lazim.
    const tujuan = (sekarang + arah + tab.length) % tab.length;
    const berikut = tab[tujuan];
    if (berikut !== undefined) onSelect(berikut.kode);
  }

  return (
    // Di layar sempit tab berubah menjadi deretan pil yang dapat digulir mendatar,
    // mengikuti desain ponsel (`D-82`); di layar lebar tetap tab bergaris bawah.
    <div
      role="tablist"
      aria-label="Jenis klaim"
      className="flex gap-1.5 overflow-x-auto px-3 pt-3 sm:gap-1 sm:border-b sm:border-slate-200 sm:px-4 sm:pt-0"
    >
      {tab.map((t) => {
        const aktif = t.kode === selected;
        return (
          <button
            key={t.kode}
            id={tabId(t.kode)}
            role="tab"
            type="button"
            aria-selected={aktif}
            aria-controls={aktif ? PANEL_ID : undefined}
            // Hanya tab yang aktif yang dapat di-Tab-kan; panah memindahkan yang lain.
            // Itu pola papan ketik baku untuk tablist, dan tanpa itu pengguna papan ketik
            // harus menekan Tab melewati setiap tab sebelum sampai ke isinya.
            tabIndex={aktif ? 0 : -1}
            onClick={() => onSelect(t.kode)}
            onKeyDown={(event) => {
              if (event.key === "ArrowRight") {
                event.preventDefault();
                pindah(1);
              }
              if (event.key === "ArrowLeft") {
                event.preventDefault();
                pindah(-1);
              }
            }}
            className={[
              "shrink-0 text-sm font-semibold whitespace-nowrap",
              "transition-[color,border-color,background-color] duration-150 ease-halus",
              "focus:outline-none focus-visible:ring-4 focus-visible:ring-blue-500/20",
              // Ponsel: pil. Layar lebar: garis bawah 3px pada tab aktif.
              "rounded-full border px-3.5 py-2",
              "sm:-mb-px sm:rounded-none sm:border-0 sm:border-b-[3px] sm:px-4 sm:py-3",
              aktif
                ? "border-slate-800 bg-slate-800 text-white sm:border-blue-600 sm:bg-transparent sm:text-blue-700"
                : "border-slate-300 bg-white text-slate-700 hover:text-slate-900 sm:border-transparent sm:bg-transparent sm:text-slate-600 sm:hover:border-slate-300",
            ].join(" ")}
          >
            {t.label}
          </button>
        );
      })}
    </div>
  );
}
