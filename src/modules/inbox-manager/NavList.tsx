import { useState } from "react";

import { SearchIcon } from "@/components/Icon";

import type { Counter, Tab } from "./types";

type Props = {
  tabs: Tab[];
  counters: Counter[];

  /** Kode bagian yang sedang terbuka. */
  active: string;
  onSelect: (code: string) => void;
};

/**
 * Daftar kiri Inbox Manager — tiga dashboard dan sembilan antrean dalam SATU daftar.
 *
 * # Apa yang digantikan, dan atas dasar apa
 *
 * Bilah tab empat butir dengan bilah antrean di dalam "Approval Master". Rancangan layar Work
 * Owner 2026-10-14 mengganti keduanya dengan daftar ini: "Approval Master" menjadi JUDUL
 * kelompok antrean, bukan butir yang dapat dibuka — dan Penolakan Klaim ikut sejajar dengan
 * delapan lainnya, padahal di Pega ia di luar bilah tab dan hanya terbuka dari pencacahnya.
 *
 * # "—" berbeda dari "0"
 *
 * "0" adalah pencacah yang terbaca dan memang kosong. "—" berarti tidak ada angka: tab yang
 * di Pega tidak punya pencacah (Produktivitas Klaim, Klaim), atau sumber antrean yang sedang
 * tidak terbaca (view INVALID) — menuliskannya "0" akan membuat antreannya tampak kosong.
 *
 * # Angkanya kembali digambar
 *
 * Lencana pencacah sempat dicabut 2026-10-08; rancangan layar 2026-10-14 menggambarnya lagi di
 * setiap butir, dan itu yang berlaku. Sumbernya tetap pencacah Pega
 * (`Activity/CountDashbroardManager`). Butir tanpa pencacah — Produktivitas dan Klaim, yang di
 * Pega diberi angka penanda "1" — dan pencacah yang sumbernya tidak terbaca digambar "—",
 * bukan nol.
 */
export function NavList({ tabs, counters, active, onSelect }: Props) {
  const [query, setQuery] = useState("");

  /*
    Kelompok antrean bernama seperti di Pega — "Approval Master", induk kesembilan antrean
    pada pohon pencacah `Activity/CountDashbroardManager` (permintaan Work Owner 2026-10-14).
    Namanya diambil dari tab induk yang dikirim server, bukan ditulis ulang di sini.
  */
  const queueGroup =
    tabs.find((tab) => tab.jenis === "ringkasan")?.nama ?? "Approval Master";

  const groups = [
    { name: "Dashboard", items: tabs.filter((tab) => tab.jenis === "dashboard") },
    { name: queueGroup, items: tabs.filter((tab) => tab.jenis === "antrean") },
  ];

  const needle = query.trim().toLowerCase();
  const matches = (tab: Tab) => tab.nama.toLowerCase().includes(needle);
  const total = groups.reduce(
    (acc, group) => acc + group.items.filter(matches).length,
    0,
  );

  function countOf(code: string): number | null {
    const counter = counters.find((item) => item.tab === code);
    if (!counter || counter.tidak_tersedia) return null;
    return counter.jumlah;
  }

  return (
    <section className="overflow-hidden rounded-kartu border border-slate-200 bg-white shadow-lembut lg:sticky lg:top-4">
      <div className="p-3">
        <label className="sr-only" htmlFor="inbox-manager-cari">
          Cari dashboard atau antrean
        </label>
        <div className="relative">
          <SearchIcon
            aria-hidden="true"
            className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400"
          />
          <input
            id="inbox-manager-cari"
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Cari dashboard atau antrean…"
            className="w-full rounded-kontrol border border-slate-300 bg-white py-2 pr-3 pl-9 text-sm placeholder:text-slate-400 hover:border-slate-400 focus:border-blue-500 focus:ring-4 focus:ring-blue-500/15 focus:outline-none"
          />
        </div>
      </div>

      <div role="tablist" aria-orientation="vertical" aria-label="Bagian Inbox Manager">
        {groups.map((group) => {
          const items = group.items.filter(matches);
          if (items.length === 0) return null;

          return (
            <div key={group.name}>
              <div className="flex items-center gap-2 border-t border-slate-100 bg-slate-50/80 px-4 py-2">
                <span className="text-[10px] font-semibold tracking-wide text-slate-500 uppercase">
                  {group.name}
                </span>
                <span className="ml-auto inline-flex h-5 min-w-[1.5rem] items-center justify-center rounded-full bg-slate-100 px-2 text-[11px] font-semibold text-slate-700">
                  {group.items.length}
                </span>
              </div>
              <ul className="divide-y divide-slate-100">
                {items.map((tab) => {
                  const selected = tab.kode === active;
                  const count = countOf(tab.kode);
                  return (
                    <li key={tab.kode}>
                      <button
                        type="button"
                        role="tab"
                        aria-selected={selected}
                        onClick={() => onSelect(tab.kode)}
                        className={[
                          "flex w-full items-start gap-3 border-l-[3px] px-4 py-2.5 text-left",
                          "transition-colors duration-150 ease-halus focus:outline-none focus-visible:bg-blue-50",
                          selected
                            ? "border-l-blue-600 bg-blue-50 pl-[13px]"
                            : "border-l-transparent hover:bg-slate-50",
                        ].join(" ")}
                      >
                        {/*
                          Hanya nama — keterangan tab TIDAK digambar di butirnya (Work Owner,
                          2026-10-14). Keterangannya sudah ada di kepala kartu yang terbuka.
                        */}
                        <span
                          className={`min-w-0 flex-1 text-sm leading-snug font-medium ${selected ? "text-blue-700" : "text-slate-800"}`}
                        >
                          {tab.nama}
                        </span>
                        <span
                          className={[
                            "mt-0.5 inline-flex h-5 min-w-[1.5rem] shrink-0 items-center justify-center rounded-full px-2 text-[11px] font-semibold tabular-nums",
                            selected
                              ? "bg-blue-600 text-white"
                              : count
                                ? "bg-slate-100 text-slate-700"
                                : "bg-slate-50 text-slate-400",
                          ].join(" ")}
                        >
                          {count === null ? "—" : count}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          );
        })}
      </div>

      {total === 0 ? (
        <p className="px-4 py-6 text-center text-xs text-slate-500">
          Tidak ada yang cocok.
        </p>
      ) : null}
    </section>
  );
}
