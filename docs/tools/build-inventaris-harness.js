/* Membangun ulang TABEL inventaris di docs/Steering/22-INVENTARIS-HARNESS.md
   langsung dari direktori Harness/.

   Jalankan: node docs/tools/build-inventaris-harness.js "<root proyek>"

   Menulis tabel yang SAMA ke dua berkas: lampiran Steering dan bahan kerja
   ticketing. Hanya bagian di bawah penanda TABEL yang ditulis ulang; seluruh
   prosa di atasnya tidak disentuh, sehingga kedua berkas boleh punya pengantar
   sendiri tanpa tabelnya pernah berbeda. Tujuannya satu: angka dan daftar nama di lampiran itu
   TIDAK MUNGKIN menyimpang dari isi export, karena tidak diketik tangan.

   Direktori Harness/ hanya DIBACA. Tidak ada rule Pega yang diubah.          */

const fs = require('fs');
const path = require('path');

const ROOT = process.argv[2];
if (!ROOT) { console.error('Pemakaian: node build-inventaris-harness.js "<root proyek>"'); process.exit(1); }

const DIR = path.join(ROOT, 'Harness');
const OUT = [
  path.join(ROOT, 'docs', 'Steering', '22-INVENTARIS-HARNESS.md'),
  path.join(ROOT, 'docs', 'ticketing', 'INVENTARIS-HARNESS.md'),
];
const PENANDA = '## Inventaris';


/* ---------- usulan modul + tingkat keyakinan ----------

   PASTI   = bukti STRUKTURAL dari export (kelas harness). Tidak bergantung nama.
   KUAT    = dua bukti sejalan — topik nama DAN sidik jari berkas.
   DUGAAN  = satu bukti saja, atau bukti yang saling bertentangan.
   JANGGAL = sidik jarinya menyimpang dari pola mana pun; wajib ditinjau manusia.

   Catatan penting soal pemakaian nama berkas: nama dipakai untuk menduga TOPIK
   layar (rekening, sparepart, laporan), BUKAN untuk menduga POLA interaksinya.
   Dugaan pola diambil dari kelas harness dan sidik jari berkas. Memakai nama
   untuk menduga pola sudah diuji dan GAGAL (lihat prosa lampiran).             */

const RE_LAPORAN = /report|laporan|monitoring|tat|kpi|outstanding|studyclaim|dashboard/i;
const RE_MASTER  = /master|sparepart|bengkel|panel|rekening|supplier|recovery|causeofloss|dominanfactor|tolakklaim|xol|documenttype|typedocument|listdocument|listdettype|dettype|logins urvey/i;

function usul(b) {
  const kecilSepertiMaster = b.kb >= 250 && b.kb <= 290 && b.rd <= 15;

  /* 1. Bukti struktural mengalahkan segalanya. */
  if (b.kelas === 'Work (konteks klaim)') return ['U-4', 'PASTI'];

  /* 2. Sidik jari menyimpang dari pola mana pun. */
  if (b.rd === 0 && b.kb < 200) return ['?', 'JANGGAL'];

  /* 3. Bernama Inbox — pola diambil dari sidik jari, bukan dari namanya. */
  if (b.inbox) {
    if (kecilSepertiMaster)  return ['U-6', 'KUAT'];    // cetakan sama dengan layar master
    if (b.kb >= 900)         return ['U-3', 'KUAT'];    // layar kerja bervolume
    return ['U-3', 'DUGAAN'];                            // ukuran menengah, belum meyakinkan
  }

  /* 4. Topik dari nama, dikuatkan atau dilemahkan sidik jari. */
  if (RE_LAPORAN.test(b.nama)) return ['U-5', b.kb >= 400 ? 'KUAT' : 'DUGAAN'];
  if (RE_MASTER.test(b.nama))  return ['U-6', kecilSepertiMaster ? 'KUAT' : 'DUGAAN'];

  /* 5. Sisanya: layar transaksi klaim. */
  return ['U-4', 'DUGAAN'];
}

const baris = fs.readdirSync(DIR)
  .filter(f => f.endsWith('.xml'))
  .map(f => {
    const nama = f.replace(/-Harness\.xml$/, '');
    const isi  = fs.readFileSync(path.join(DIR, f), 'utf8');
    const m    = isi.match(/<pyClassName>([^<]*)<\/pyClassName>/);
    const raw  = m ? m[1] : '(tidak ada)';
    const kelas = raw.startsWith('ASM-FW') ? 'Work (konteks klaim)'
                : raw === '@baseclass'     ? '@baseclass'
                : raw === 'Data-Portal'    ? 'Data-Portal'
                : raw;
    const kb = Math.round(isi.length / 1024);
    const rd = (isi.match(/ReportDefinition/g) || []).length;
    const b  = { nama, kelas, kb, rd, inbox: /inbox/i.test(nama) };
    const [modul, yakin] = usul(b);
    return Object.assign(b, { modul, yakin });
  })
  .sort((a, b) => a.nama.localeCompare(b.nama));

const n = k => baris.filter(b => b.kelas === k).length;

const tabel = [
  PENANDA, '',
  '| Harness | Kelas | KB | RD | Bernama *Inbox* | **Usulan modul** | Keyakinan |',
  '|---|---|---:|---:|---|---|---|',
  ...baris.map(b => { const q = String.fromCharCode(96); return "| " + q + b.nama + q + " | " + b.kelas + " | " + b.kb + " | " + b.rd + " | " + (b.inbox ? "ya" : "—") + " | **" + b.modul + "** | " + b.yakin + " |"; }),
  '',
  `**Total: ${baris.length} harness** — ${n('Work (konteks klaim)')} berkelas Work · ` +
  `${n('@baseclass')} \`@baseclass\` · ${n('Data-Portal')} \`Data-Portal\` · ` +
  `${baris.filter(b => b.inbox).length} di antaranya bernama *Inbox*.`,
  '',
  '> Tabel di atas **dibangun ulang** oleh `docs/tools/build-inventaris-harness.js` langsung dari',
  "> direktori `Harness/`, termasuk kolom **Usulan modul** dan **Keyakinan** yang dihitung dari",
  "> aturan di `usul()`. Jangan menyuntingnya dengan tangan — koreksi Work Owner dicatat di",
  "> Decision Log, lalu aturannya disesuaikan di sini agar tabel tetap dapat dibangun ulang.",
  '',
].join('\n');

OUT.forEach(f => {
  const lama = fs.readFileSync(f, 'utf8');
  const i = lama.indexOf(PENANDA);
  if (i < 0) { console.error('Penanda "' + PENANDA + '" tidak ditemukan di ' + f); process.exit(1); }
  fs.writeFileSync(f, lama.slice(0, i) + tabel, 'utf8');
  console.log('MD   : ' + f);
});
console.log('Baris: ' + baris.length + ' harness · Work ' + n('Work (konteks klaim)') +
            ' · @baseclass ' + n('@baseclass') + ' · Data-Portal ' + n('Data-Portal'));
