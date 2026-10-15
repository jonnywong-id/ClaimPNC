import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { KlaimView } from "./DashboardViews";
import { OutstandingView } from "./OutstandingView";
import type { Panel, Tab } from "./types";

/** Uji tampilan dashboard — rancangan layar Work Owner 2026-10-14. Data KARANGAN (`D-69`). */

const KEPUTUSAN = {
  dapat_diputuskan: false,
  dapat_massal: false,
  alasan_wajib_saat_menolak: false,
};

const TAB_OS: Tab = {
  kode: "1",
  nama: "Outstanding",
  keterangan: "Klaim berjalan.",
  jenis: "dashboard",
  keputusan: KEPUTUSAN,
  punya_penyaring_periode: false,
};

function picPanel(names: [string, number][]): Panel {
  return {
    kunci: "pic",
    judul: "",
    kolom: [
      { kunci: "pic", judul: "PIC" },
      { kunci: "jumlah", judul: "OS" },
    ],
    baris: [
      { pic: { teks: "ALL" }, jumlah: { jumlah: names.reduce((a, [, n]) => a + n, 0) } },
      ...names.map(([pic, n]) => ({ pic: { teks: pic }, jumlah: { jumlah: n } })),
    ],
    penyaring_baris: "pic",
  };
}

const COB: Panel = {
  kunci: "grup_bisnis",
  judul: "",
  kolom: [
    { kunci: "grup_bisnis", judul: "COB" },
    { kunci: "jumlah", judul: "OS" },
  ],
  baris: [{ grup_bisnis: { teks: "FIRE" }, jumlah: { jumlah: 3 } }],
};

const SUMMARY: Panel = {
  kunci: "kategori_os",
  judul: "",
  kolom: [
    { kunci: "kategori_dol", judul: "Kategori/DOL" },
    { kunci: "reinsurer", judul: "Reinsurer" },
    { kunci: "2025", judul: "2025" },
    { kunci: "2026", judul: "2026" },
  ],
  baris: [
    { kategori_dol: { teks: "SURVEY" }, reinsurer: { teks: "LEADER" }, "2025": { jumlah: 4 }, "2026": {} },
    { kategori_dol: { teks: "SURVEY" }, reinsurer: { teks: "MEMBER" }, "2025": { jumlah: 1 }, "2026": { jumlah: 2 } },
  ],
};

function renderOutstanding(panels: Panel[], onSelectPIC = vi.fn()) {
  render(
    <OutstandingView
      tab={TAB_OS}
      panels={panels}
      filters={[]}
      isLoading={false}
      failure={null}
      selectedPIC=""
      onSelectPIC={onSelectPIC}
      draftFilter={{ reinsurer: "", kategori_os: "", pic: "" }}
      onDraftChange={vi.fn()}
      onApplyFilter={vi.fn()}
      onClearFilter={vi.fn()}
    />,
  );
}

/** Nama PIC pada tabel Per PIC, berurutan, tanpa baris ALL. */
function picNames(): string[] {
  const table = screen.getByRole("columnheader", { name: "PIC" }).closest("table")!;
  return within(table)
    .getAllByRole("row")
    .slice(1)
    .map((row) => within(row).getAllByRole("cell")[0]!.textContent ?? "")
    .filter((name) => name !== "ALL");
}

describe("Outstanding", () => {
  const many: [string, number][] = [
    ["ANDI", 1], ["BUDI", 9], ["CITRA", 4], ["DEDI", 7], ["EKA", 2], ["FAJAR", 8],
    ["GITA", 3], ["HADI", 6], ["INDAH", 5], ["JONI", 10], ["KIKI", 11],
  ];

  it("mengurutkan PIC terbanyak, menampilkan sembilan, lalu sisanya lewat tautan", async () => {
    renderOutstanding([picPanel(many), COB, SUMMARY]);

    expect(screen.getByText("66 OS")).toBeInTheDocument();
    expect(picNames()).toEqual([
      "KIKI", "JONI", "BUDI", "FAJAR", "DEDI", "HADI", "INDAH", "CITRA", "GITA",
    ]);

    await userEvent.click(screen.getByRole("button", { name: "Lihat 2 PIC lainnya" }));
    expect(picNames()).toHaveLength(11);
    expect(screen.getByText("Seluruh PIC sudah tampil · total 66")).toBeInTheDocument();
  });

  it("kembali ke urutan server saat urutan terbanyak dimatikan", async () => {
    renderOutstanding([picPanel(many), COB, SUMMARY]);

    const sort = screen.getByRole("button", { name: "OS — terbanyak" });
    expect(sort).toHaveAttribute("aria-pressed", "true");
    await userEvent.click(sort);
    expect(sort).toHaveAttribute("aria-pressed", "false");
    expect(picNames().slice(0, 3)).toEqual(["ANDI", "BUDI", "CITRA"]);
  });

  it("menjadikan nama PIC dan ALL tautan penyaring", async () => {
    const onSelect = vi.fn();
    renderOutstanding([picPanel([["ANDI", 2]]), COB, SUMMARY], onSelect);

    await userEvent.click(screen.getByRole("button", { name: "ANDI" }));
    expect(onSelect).toHaveBeenLastCalledWith("ANDI");
    await userEvent.click(screen.getByRole("button", { name: "ALL" }));
    expect(onSelect).toHaveBeenLastCalledWith("ALL");
    expect(screen.getByRole("button", { name: "ALL" })).toHaveAttribute("aria-current", "true");
  });

  it("menulis kategori sekali per kelompok dan menjumlahkan kolom Total", () => {
    renderOutstanding([picPanel([["ANDI", 2]]), COB, SUMMARY]);

    expect(screen.getByText("Tahun = tanggal klaim dibuat")).toBeInTheDocument();
    const table = screen.getByRole("columnheader", { name: "Kategori/DOL" }).closest("table")!;
    const rows = within(table).getAllByRole("row").slice(1);

    const first = within(rows[0]!).getAllByRole("cell").map((cell) => cell.textContent);
    const second = within(rows[1]!).getAllByRole("cell").map((cell) => cell.textContent);
    expect(first).toEqual(["SURVEY", "LEADER", "4", "—", "4"]);
    // Nama kategori kedua hanya untuk pembaca layar.
    expect(second).toEqual(["SURVEY", "MEMBER", "1", "2", "3"]);
    expect(within(rows[1]!).getAllByRole("cell")[0]!.querySelector(".sr-only")).not.toBeNull();
  });
});

describe("Klaim", () => {
  const TAB_KLAIM: Tab = {
    ...TAB_OS,
    kode: "3",
    nama: "Klaim",
    punya_penyaring_periode: true,
    label_terapkan_periode: "Lihat Data",
  };

  const BISNIS: Panel = {
    kunci: "bisnis",
    judul: "Total Klaim Bisnis",
    kolom: [
      { kunci: "nama_bisnis", judul: "COB" },
      { kunci: "total_klaim", judul: "Total Klaim" },
      { kunci: "nilai_klaim", judul: "NILAI Klaim (Rp)", uang: true },
      { kunci: "jumlah_akseptasi", judul: "CASE Akseptasi" },
      { kunci: "nilai_akseptasi", judul: "NILAI Akseptasi (Rp)", uang: true },
      { kunci: "jumlah_ditolak", judul: "CASE Reject" },
      { kunci: "nilai_ditolak", judul: "NILAI Reject (Rp)", uang: true },
    ],
    baris: [
      {
        nama_bisnis: { teks: "FIRE" },
        total_klaim: { jumlah: 1234 },
        nilai_klaim: { nilai: "987654321.5" },
        jumlah_akseptasi: {},
        nilai_akseptasi: { nilai: "123456789" },
        jumlah_ditolak: { jumlah: 42 },
        nilai_ditolak: { nilai: "-1500000" },
      },
    ],
  };

  it("mengelompokkan CASE dan NILAI di bawah tahapannya, dengan Rupiah utuh", () => {
    render(
      <KlaimView
        tab={TAB_KLAIM}
        panels={[BISNIS]}
        isLoading={false}
        failure={null}
        period={null}
        onPeriodChange={vi.fn()}
      />,
    );

    expect(screen.getByRole("columnheader", { name: "Akseptasi" })).toHaveAttribute("colspan", "2");
    expect(screen.getByRole("columnheader", { name: "Reject" })).toBeInTheDocument();
    expect(screen.getAllByRole("columnheader", { name: "CASE" })).toHaveLength(2);
    expect(screen.getAllByRole("columnheader", { name: "NILAI (Rp)" })).toHaveLength(2);

    // Rupiah TIDAK dibulatkan ke miliar.
    expect(screen.getByText("987.654.321,5")).toBeInTheDocument();
    expect(screen.getByText("1.234")).toBeInTheDocument();
    expect(screen.getByText("−1.500.000")).toHaveClass("text-red-700");

    // Pencacah nol (sel kosong dari server) ditulis 0.
    const row = screen.getByText("FIRE").closest("tr")!;
    expect(within(row).getAllByRole("cell")[3]).toHaveTextContent(/^0$/);

    expect(
      screen.getByText("Tanpa penyaring periode, angka mencakup seluruh periode."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Lihat Data" })).toBeInTheDocument();
  });
});
