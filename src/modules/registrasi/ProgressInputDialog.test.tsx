import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useSelectedPortal } from "@/app/portal";
import { useSession } from "@/app/session";

import { ProgressTab } from "./EstimateTabs";

/** Uji modal Input Progress Claim. Seluruh data KARANGAN (`D-69`). */

const BASE = "/api/registrasi/klaim/klaim-1";
let calls: { url: string; method: string; body: unknown }[] = [];

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

const FORM = {
  posisi: [
    {
      id: "1",
      kasus_id: "K",
      posisi: "REGISTER",
      status: "Done",
      terbuka: false,
    },
    {
      id: "2",
      kasus_id: "K",
      posisi: "SURVEY",
      status: "On Progress",
      terbuka: true,
    },
  ],
  status_progress_1: [
    { id: "004", nama: "SURVEY" },
    { id: "008", nama: "WAITING LOD" },
  ],
  status: ["SUDAH BERHASIL DIHUBUNGKAN"],
  tipe_dokumen: ["ALL"],
  email_progress: "",
  status_wajib_email: ["003", "008", "011"],
  status_tipe_dokumen: "003",
};

function install(post: () => Response) {
  vi.stubGlobal("fetch", (url: string, init?: RequestInit) => {
    const method = init?.method ?? "GET";
    calls.push({ url, method, body: init?.body });
    if (url === `${BASE}/progres` && method === "GET")
      return Promise.resolve(json(200, { progres: [], komunikasi: [] }));
    if (url === `${BASE}/progres/form`) return Promise.resolve(json(200, FORM));
    if (url.startsWith("/api/registrasi/progres/status-perlu?status1=004"))
      return Promise.resolve(json(200, [{ id: "59", nama: "HASIL SURVEY" }]));
    if (url.startsWith("/api/registrasi/progres/status-perlu"))
      return Promise.resolve(json(200, []));
    if (url === `${BASE}/progres` && method === "POST")
      return Promise.resolve(post());
    return Promise.resolve(json(404, { kode: "x", pesan: "x" }));
  });
}

function wrap() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ProgressTab claimID="klaim-1" />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  calls = [];
  useSession.setState({
    token: "token-uji",
    user: null,
    validUntil: "2026-12-31T00:00:00Z",
  });
  useSelectedPortal.getState().select("ASM");
});

afterEach(() => {
  vi.unstubAllGlobals();
  useSelectedPortal.getState().clear();
});

describe("ProgressInputDialog", () => {
  it("mengirim isian ke posisi yang belum Done", async () => {
    install(() => json(200, { peringatan: [] }));
    wrap();
    await userEvent.click(
      await screen.findByRole("button", { name: "Input Progress Claim" }),
    );
    const posisi = await screen.findByLabelText("Posisi");
    expect(posisi).toHaveValue("2");
    expect(
      screen.queryByRole("option", { name: /REGISTER/ }),
    ).not.toBeInTheDocument();

    await userEvent.selectOptions(
      screen.getByLabelText("Status Progress 1"),
      "004",
    );
    expect(
      screen.queryByLabelText("Email Progress Klaim"),
    ).not.toBeInTheDocument();
    await waitFor(() =>
      expect(calls.some((c) => c.url.includes("status1=004"))).toBe(true),
    );
    await userEvent.type(
      screen.getByLabelText("STATUS PERLU 1"),
      "HASIL SURVEY",
    );
    await userEvent.selectOptions(
      screen.getByLabelText("Status"),
      "SUDAH BERHASIL DIHUBUNGKAN",
    );
    await userEvent.type(screen.getByLabelText("Remarks"), "catatan");
    await userEvent.click(screen.getByRole("button", { name: "Submit" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    const post = calls.find((c) => c.method === "POST")!;
    expect(JSON.parse(post.body as string)).toEqual({
      posisi_id: "2",
      status_progress_1: "004",
      status_perlu: ["59"],
      status: "SUDAH BERHASIL DIHUBUNGKAN",
      remarks: "catatan",
      tgl_next_follow_up: "",
      tipe_dokumen: "",
      email_progress: "",
    });
  });

  it("menampilkan pesan Pega dari server dan isian wajib email", async () => {
    install(() =>
      json(422, {
        kode: "validasi_gagal",
        pesan: "x",
        rincian: [
          {
            kode: "progres",
            field: "status_perlu",
            pesan: "Status Progress ini Telah di Inputkan pada hari ini",
          },
        ],
      }),
    );
    wrap();
    await userEvent.click(
      await screen.findByRole("button", { name: "Input Progress Claim" }),
    );
    await userEvent.selectOptions(
      await screen.findByLabelText("Status Progress 1"),
      "008",
    );
    expect(screen.getByLabelText("Email Progress Klaim")).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("STATUS PERLU 1"), "TIDAK ADA");
    expect(
      await screen.findByText("STATUS PROGRESS 2 TIDAK DITEMUKAN"),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Submit" }));
    expect(
      await screen.findByText("Progress tidak dapat disimpan"),
    ).toBeInTheDocument();
  });
});
