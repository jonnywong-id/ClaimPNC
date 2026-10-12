package memory

import "claim-pnc/internal/masterpasalai"

// SampleClause adalah baris contoh untuk pengembangan dan pengujian.
//
// # Seluruhnya KARANGAN
//
// Tidak satu pun berasal dari basis data mana pun. Isi `POOLDATA.MST_PASAL_AI` belum pernah
// dilihat siapa pun di tim ini; yang diterima adalah kuerinya, bukan datanya.
//
// `WP_ID` di sini dibuat berurutan dan berpadding tiga digit. **Bentuk aslinya tidak
// diketahui** — DDL-nya belum ada (`R-08`). Padding dipakai supaya urutan teks pada contoh
// ini sama dengan urutan angkanya; pada data sungguhan, `ORDER BY WP_ID` adalah pengurutan
// teks dan "10" akan mendahului "9".
//
// Bentuknya menyerupai wording polis asuransi umum supaya layarnya dapat dinilai dengan
// teks sepanjang yang wajar, terutama kolom **Kejadian** yang digambar `pxTextArea` dan
// paling lebar (283 px berbanding 48 dan 47).
//
// # Kenapa 28 baris, bukan segenggam
//
// Ukuran halamannya **25** (`masterpasalai.PageSize`), dan paginasinya dikerjakan di sisi
// server. Dengan kurang dari 26 baris, halaman kedua tidak pernah terbentuk — dan cacat
// paginasi adalah yang paling mudah lolos justru karena datanya terlalu sedikit untuk
// memunculkannya.
//
// 28 memberi halaman pertama yang penuh dan halaman kedua yang berisi tiga baris, sehingga
// kedua keadaan itu terlihat sekaligus.
func SampleClause() []masterpasalai.Clause {
	return []masterpasalai.Clause{
		{ID: "001", Number: "1", Paragraph: "1", Event: "Kebakaran yang timbul dari hubungan arus pendek pada instalasi listrik di dalam bangunan yang dipertanggungkan."},
		{ID: "002", Number: "1", Paragraph: "2", Event: "Kebakaran yang menjalar dari bangunan di sekitarnya, sepanjang bangunan tertanggung tidak menjadi sumber apinya."},
		{ID: "003", Number: "1", Paragraph: "3", Event: "Kerusakan akibat air yang dipakai memadamkan kebakaran, termasuk oleh petugas pemadam."},
		{ID: "004", Number: "2", Paragraph: "1", Event: "Petir yang menyambar langsung bangunan atau isi yang dipertanggungkan."},
		{ID: "005", Number: "2", Paragraph: "2", Event: "Kerusakan perangkat elektronik akibat induksi petir, sepanjang tersambung pada instalasi tetap."},
		{ID: "006", Number: "3", Paragraph: "1", Event: "Ledakan yang bersumber dari ketel uap, tabung gas, atau bejana bertekanan milik tertanggung."},
		{ID: "007", Number: "3", Paragraph: "2", Event: "Ledakan yang bersumber dari luar lokasi risiko dan merambat ke bangunan tertanggung."},
		{ID: "008", Number: "4", Paragraph: "1", Event: "Kejatuhan pesawat terbang atau benda yang terlepas darinya."},
		{ID: "009", Number: "5", Paragraph: "1", Event: "Asap yang berasal dari kebakaran pada objek yang dipertanggungkan."},
		{ID: "010", Number: "6", Paragraph: "1", Event: "Kerusuhan dan pemogokan yang disertai perbuatan jahat terhadap harta benda tertanggung."},
		{ID: "011", Number: "6", Paragraph: "2", Event: "Penghalangan kerja yang mengakibatkan kerusakan langsung pada bangunan."},
		{ID: "012", Number: "7", Paragraph: "1", Event: "Perbuatan jahat pihak ketiga, tidak termasuk pencurian dan percobaan pencurian."},
		{ID: "013", Number: "8", Paragraph: "1", Event: "Tertabrak kendaraan bermotor yang bukan milik dan bukan dalam penguasaan tertanggung."},
		{ID: "014", Number: "9", Paragraph: "1", Event: "Angin topan, badai, dan banjir yang terjadi di lokasi risiko yang tercantum pada polis."},
		{ID: "015", Number: "9", Paragraph: "2", Event: "Tanah longsor yang terjadi tanpa didahului pekerjaan penggalian oleh tertanggung."},
		{ID: "016", Number: "10", Paragraph: "1", Event: "Kerusakan mesin akibat kesalahan pengoperasian oleh tenaga kerja yang berwenang."},
		{ID: "017", Number: "10", Paragraph: "2", Event: "Kerusakan mesin akibat benda asing yang masuk ke dalam mesin saat beroperasi."},
		{ID: "018", Number: "11", Paragraph: "1", Event: "Kecelakaan diri yang mengakibatkan cacat tetap sebagian pada tertanggung."},
		{ID: "019", Number: "11", Paragraph: "2", Event: "Kecelakaan diri yang mengakibatkan meninggal dunia dalam 180 hari sejak kejadian."},
		{ID: "020", Number: "12", Paragraph: "1", Event: "Biaya pengobatan dan perawatan akibat kecelakaan, sebatas nilai yang tercantum pada ikhtisar polis."},
		{ID: "021", Number: "13", Paragraph: "1", Event: "Kerugian pengangkutan barang akibat kapal kandas, tenggelam, atau terbakar."},
		{ID: "022", Number: "13", Paragraph: "2", Event: "Kerugian pengangkutan akibat pembuangan barang ke laut untuk menyelamatkan kapal."},
		{ID: "023", Number: "14", Paragraph: "1", Event: "Kerusakan kemasan yang mengakibatkan kerugian pada barang di dalamnya selama pengangkutan."},
		{ID: "024", Number: "15", Paragraph: "1", Event: "Pembatalan perjalanan karena sakit yang dibuktikan surat keterangan dokter."},
		{ID: "025", Number: "15", Paragraph: "2", Event: "Keterlambatan keberangkatan lebih dari enam jam yang dinyatakan pihak pengangkut."},
		{ID: "026", Number: "16", Paragraph: "1", Event: "Kehilangan bagasi tercatat selama berada dalam penguasaan pihak pengangkut."},
		{ID: "027", Number: "17", Paragraph: "1", Event: "Tanggung gugat kepada pihak ketiga atas cedera badan yang timbul di lokasi usaha tertanggung."},
		{ID: "028", Number: "18", Paragraph: "1", Event: "Kegagalan debitur memenuhi kewajiban pembayaran sesuai perjanjian kredit yang dijamin."},
	}
}
