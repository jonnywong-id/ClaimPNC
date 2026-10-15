import { useEffect, useMemo, useState } from "react";

import type { AutoClaimCompanySummary } from "@/api/types";
import { ErrorMessage } from "@/components/ErrorMessage";
import {
  ChevronIcon,
  CloseIcon,
  EmptyBoxIcon,
  SearchIcon,
  WarningIcon,
} from "@/components/Icon";

import { useAutoClaimSummary } from "./api";
import { CompanyBatches } from "./CompanyBatches";
import { loadMessage } from "./messages";

type Props = {
  /** Tab yang sedang terbuka. */
  source: string;
};

/**
 * Daftar perusahaan di KIRI, batch perusahaan terpilih di KANAN (master–detail).
 *
 * # Apa yang digantikannya
 *
 * Tabel perusahaan dengan baris melar (accordion, keputusan 2026-09-27). Work Owner
 * meminta bentuk ini pada 2026-09-28 dengan contoh tampilan "kategori di kiri, daftar di
 * kanan": daftar perusahaan tetap terlihat selama batch diperiksa, sehingga berpindah
 * perusahaan cukup satu klik — pada accordion, membuka satu baris mendorong perusahaan
 * berikutnya keluar layar.
 *
 * # Tiga keputusan kecil
 *
 * - **Perusahaan pertama langsung terpilih** saat tab dibuka, seperti contohnya. Panel
 *   kanan yang kosong sampai sesuatu diklik tidak memberi tahu apa pun.
 * - **Daftar kiri digulir, tidak dipaginasi**, seperti contohnya. Isinya puluhan baris dan
 *   dikirim server utuh; pencarian di atasnya yang mempersempit. Grid batch di kanan tetap
 *   dipaginasi server — di sanalah datanya dapat besar.
 * - **Pilihan dipertahankan saat mencari.** Kata kunci hanya menyaring daftar kiri; batch
 *   yang sedang diperiksa di kanan tidak ikut hilang hanya karena namanya tersaring keluar.
 *
 * # Penataan ulang (`D-82`, 2026-10-14)
 *
 * - Jumlah batch menjadi lencana; perusahaan terpilih berwarna biru.
 * - Kode yang belum terdaftar di Master Auto Claim diberi label kuning dan ditaruh PALING
 *   BAWAH — urutan server dipertahankan di dalam masing-masing kelompok.
 * - Di layar sempit (< lg) daftar kiri diganti satu tombol pemilih yang membuka lembar
 *   bawah berisi pencarian dan daftar yang sama.
 */
export function CompanyBrowser({ source }: Props) {
  const summary = useAutoClaimSummary(source);

  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<string | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  const company = useMemo(() => {
    const all = summary.data?.perusahaan ?? [];
    // Yang terdaftar di atas, yang belum terdaftar di bawah. `filter` dua kali, bukan
    // `sort`: urutan server di dalam tiap kelompok tidak boleh berubah.
    return [
      ...all.filter((c) => c.nama !== ""),
      ...all.filter((c) => c.nama === ""),
    ];
  }, [summary.data]);

  const visible = useMemo(() => {
    const word = query.trim().toLowerCase();
    if (word === "") return company;
    // Kode ikut dicari: petugas sering hanya ingat kode dari berkas unggahan, dan baris
    // tanpa master memang hanya punya kode.
    return company.filter(
      (c) =>
        c.nama.toLowerCase().includes(word) ||
        c.kode.toLowerCase().includes(word),
    );
  }, [company, query]);

  // Yang terpilih: pilihan pengguna bila masih ada di data, selain itu perusahaan pertama.
  // Dihitung saat menggambar, bukan disetel lewat efek, supaya tidak ada satu gambar pun
  // dengan panel kanan kosong di antara data tiba dan efek berjalan.
  const selected =
    company.find((c) => c.kode === picked) ??
    (picked === null ? company[0] : undefined);

  if (summary.isError) {
    const message = loadMessage(summary.error);
    return (
      <ErrorMessage
        title={message.title}
        description={message.description}
        tone={message.tone}
      />
    );
  }

  const listProps = {
    summaryReady: summary.isSuccess,
    loading: summary.isPending,
    total: company.length,
    visible,
    query,
    onQuery: setQuery,
    selected: selected?.kode,
  };

  return (
    <div className="grid gap-3.5 lg:grid-cols-[17rem_minmax(0,1fr)] 2xl:grid-cols-[19rem_minmax(0,1fr)]">
      {/* Layar lebar: daftar tetap di kiri. Layar sempit: disembunyikan, diganti tombol
          pemilih di bawah. */}
      <aside className="hidden flex-col overflow-hidden rounded-kartu border border-slate-200 bg-white shadow-lembut lg:sticky lg:top-4 lg:flex lg:max-h-[calc(100vh-6rem)] lg:self-start">
        <CompanyPanel
          {...listProps}
          inputId="cari-perusahaan"
          onPick={(kode) => setPicked(kode)}
        />
      </aside>

      {selected !== undefined && (
        <button
          type="button"
          aria-haspopup="dialog"
          onClick={() => setSheetOpen(true)}
          className="flex min-h-14 w-full items-center gap-3 rounded-kartu border border-slate-400 bg-white px-3 py-2 text-left lg:hidden"
        >
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm font-semibold text-slate-900">
              {selected.nama === "" ? selected.kode : selected.nama}
            </span>
            <span className="block truncate text-xs text-slate-500 tabular-nums">
              {selected.nama === "" ? "" : `${selected.kode} · `}
              {selected.jumlah_batch} batch
            </span>
          </span>
          <ChevronIcon className="h-4 w-4 shrink-0 rotate-90 text-slate-500" />
        </button>
      )}

      {/* Tanpa bingkai kartu: grid `DataTable` di dalamnya sudah berbingkai sendiri, dan
          kartu di dalam kartu di dalam kartu hanya menambah garis tanpa menambah arti. */}
      <section aria-label="Batch perusahaan terpilih" className="min-w-0">
        {selected === undefined ? (
          <div className="flex flex-col items-center gap-3 py-14 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
              <EmptyBoxIcon className="h-6 w-6" />
            </span>
            <p className="text-sm font-medium text-slate-700">
              {summary.isPending
                ? "Memuat…"
                : "Pilih perusahaan di sebelah kiri."}
            </p>
          </div>
        ) : (
          // `key` memasang ulang grid saat perusahaan berganti: halaman, kata kunci, dan
          // rincian yang terbuka milik perusahaan sebelumnya tidak boleh terbawa.
          <CompanyBatches
            key={selected.kode}
            source={source}
            company={selected.kode}
            companyName={selected.nama}
            batchCount={selected.jumlah_batch}
          />
        )}
      </section>

      {sheetOpen && (
        <CompanySheet onClose={() => setSheetOpen(false)}>
          <CompanyPanel
            {...listProps}
            inputId="cari-perusahaan-lembar"
            onClose={() => setSheetOpen(false)}
            onPick={(kode) => {
              setPicked(kode);
              setSheetOpen(false);
            }}
          />
        </CompanySheet>
      )}
    </div>
  );
}

type PanelProps = {
  summaryReady: boolean;
  loading: boolean;
  total: number;
  visible: AutoClaimCompanySummary[];
  query: string;
  onQuery: (query: string) => void;
  selected: string | undefined;
  onPick: (kode: string) => void;
  inputId: string;
  /** Diisi hanya pada lembar bawah: kepala panelnya membawa tombol Tutup. */
  onClose?: () => void;
};

/**
 * Isi panel perusahaan — kepala, pencarian, dan daftar.
 *
 * Dipakai DUA tempat dengan isi yang sama: kolom kiri di layar lebar dan lembar bawah di
 * layar sempit. Keduanya berbagi kata kunci dan pilihan, sehingga berpindah lebar layar
 * tidak menghilangkan apa pun.
 */
function CompanyPanel({
  summaryReady,
  loading,
  total,
  visible,
  query,
  onQuery,
  selected,
  onPick,
  inputId,
  onClose,
}: PanelProps) {
  const headingId = `${inputId}-judul`;
  return (
    <>
      <div className="flex items-center gap-2 px-4 pt-3.5 pb-2.5">
        <h2
          id={headingId}
          className="text-[0.6875rem] font-bold tracking-wider text-slate-600 uppercase"
        >
          Perusahaan
        </h2>
        <span className="ml-auto inline-flex h-[22px] items-center rounded-full bg-slate-100 px-2 text-xs font-bold text-slate-700 tabular-nums">
          {summaryReady ? total : "…"}
        </span>
        {onClose !== undefined && (
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup"
            className="-mr-2 flex h-11 w-11 items-center justify-center rounded-kontrol text-slate-600 hover:bg-slate-100"
          >
            <CloseIcon className="h-[18px] w-[18px]" />
          </button>
        )}
      </div>

      <div className="px-4 pb-3">
        <label htmlFor={inputId} className="sr-only">
          Cari nama perusahaan
        </label>
        <div className="relative">
          <span
            aria-hidden="true"
            className="pointer-events-none absolute inset-y-0 left-0 flex w-9 items-center justify-center text-slate-400"
          >
            <SearchIcon className="h-4 w-4" />
          </span>
          <input
            id={inputId}
            type="search"
            value={query}
            onChange={(event) => onQuery(event.target.value)}
            placeholder="Cari nama atau kode…"
            className={[
              "w-full rounded-kontrol border border-slate-400 bg-white py-2 pl-9 pr-3",
              "text-sm text-slate-900 placeholder:text-slate-500",
              "transition-[border-color,box-shadow] duration-150 ease-halus hover:border-slate-500",
              "focus:border-blue-500 focus:outline-none focus:ring-4 focus:ring-blue-500/15",
            ].join(" ")}
          />
        </div>
        {query.trim() !== "" && summaryReady && (
          <p className="mt-2 text-xs text-slate-600" role="status">
            {visible.length} dari {total} perusahaan cocok.
          </p>
        )}
      </div>

      {loading ? (
        <div
          className="space-y-2 px-4 pb-4"
          role="status"
          aria-label="Memuat daftar perusahaan"
        >
          {[0, 1, 2, 3].map((n) => (
            <div key={n} className="h-9 animate-pulse rounded bg-slate-100" />
          ))}
        </div>
      ) : visible.length === 0 ? (
        <p className="px-4 pb-4 text-sm text-slate-600">
          {query.trim() === ""
            ? "Belum ada batch klaim pada tab ini."
            : `Tidak ada perusahaan yang cocok dengan “${query.trim()}”.`}
        </p>
      ) : (
        <nav
          aria-label="Daftar perusahaan"
          className="min-h-0 flex-1 overflow-y-auto"
        >
          <ul>
            {visible.map((c) => (
              <li key={c.kode}>
                <CompanyItem
                  company={c}
                  active={selected === c.kode}
                  onPick={() => onPick(c.kode)}
                />
              </li>
            ))}
          </ul>
        </nav>
      )}
    </>
  );
}

/**
 * Satu baris daftar perusahaan.
 *
 * Tombol, bukan tautan: memilih perusahaan tidak berpindah halaman. Yang terpilih ditandai
 * GARIS di tepi kiri dan `aria-current`, bukan warna latar saja — pembedaan penting tidak
 * pernah hanya warna, dan pembaca layar tidak melihat warna.
 *
 * Lencana jumlah batch adalah anak TERAKHIR tombol; label "belum terdaftar" ada di bawah
 * nama, bukan di ujung baris.
 */
function CompanyItem({
  company,
  active,
  onPick,
}: {
  company: AutoClaimCompanySummary;
  active: boolean;
  onPick: () => void;
}) {
  const registered = company.nama !== "";
  let tone = "bg-amber-50/60 hover:bg-amber-50";
  if (active) tone = "bg-blue-50 shadow-[inset_3px_0_0_var(--color-blue-600)]";
  else if (registered) tone = "hover:bg-slate-50";
  return (
    <button
      type="button"
      onClick={onPick}
      aria-current={active ? "true" : undefined}
      className={[
        "flex min-h-14 w-full items-center gap-2.5 border-t border-slate-100 px-4 py-2.5 text-left",
        "transition-colors duration-150 ease-halus",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500/40",
        tone,
      ].join(" ")}
    >
      <span className="min-w-0 flex-1">
        <span
          className={[
            "block truncate text-sm leading-snug font-semibold",
            active ? "text-blue-700" : "text-slate-900",
          ].join(" ")}
        >
          {registered ? company.nama : company.kode}
        </span>
        {registered ? (
          <span className="mt-0.5 block truncate text-xs text-slate-500 tabular-nums">
            {company.kode}
          </span>
        ) : (
          <span className="mt-1 inline-flex items-center gap-1 rounded-md bg-amber-100 px-1.5 py-0.5 text-[0.6875rem] font-bold text-amber-900">
            <WarningIcon className="h-3 w-3" />
            Belum terdaftar di Master Auto Claim
          </span>
        )}
      </span>
      <span
        className={[
          "inline-flex h-[22px] min-w-7 shrink-0 items-center justify-center rounded-full px-1.5 text-xs font-bold tabular-nums",
          active ? "bg-blue-600 text-white" : "bg-slate-100 text-slate-700",
        ].join(" ")}
      >
        {company.jumlah_batch}
      </span>
    </button>
  );
}

/**
 * Lembar bawah pemilih perusahaan — hanya di layar sempit.
 *
 * Menutup dengan Escape, tombol Tutup, atau klik pada latar — pola yang sama dengan pop-up
 * rincian batch.
 */
function CompanySheet({
  onClose,
  children,
}: {
  onClose: () => void;
  children: React.ReactNode;
}) {
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 bg-slate-900/45 lg:hidden"
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="cari-perusahaan-lembar-judul"
        className="absolute inset-x-0 bottom-0 flex max-h-[80vh] flex-col overflow-hidden rounded-t-2xl bg-white shadow-terbang"
      >
        <div
          aria-hidden="true"
          className="mx-auto mt-2.5 mb-1 h-1 w-10 rounded-full bg-slate-300"
        />
        {children}
      </div>
    </div>
  );
}
