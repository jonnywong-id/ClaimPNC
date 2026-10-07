// Package smtpsend mengirim surel lewat SMTP untuk modul yang memberi tahu petugas: alamat
// server, kredensial, penerima, dan langkah percakapan SMTP-nya.
//
// Isi surelnya tetap milik modul pemakai. Yang tinggal di sini hanyalah pengirimannya, yang
// sebelumnya tersalin sama persis di tiga paket notification.
package smtpsend

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// DefaultTimeout dipakai bila Config.Timeout tidak diisi.
const DefaultTimeout = 20 * time.Second

// Config adalah alamat server surel beserta pengirim dan penerimanya. Seluruhnya dari
// konfigurasi lingkungan, bukan dari kode (`D-15`).
type Config struct {
	Host string
	Port int

	// User dan Password boleh kosong untuk relay internal tanpa autentikasi.
	User     string
	Password string

	From string

	// To adalah penerima; alamat kosong dibuang.
	To []string

	// Timeout membatasi seluruh percakapan; kosong berarti DefaultTimeout.
	Timeout time.Duration
}

// Complete menyatakan konfigurasi cukup untuk mengirim: server, port, pengirim, dan
// sedikitnya satu penerima.
func (c Config) Complete() bool {
	return strings.TrimSpace(c.Host) != "" &&
		c.Port > 0 &&
		strings.TrimSpace(c.From) != "" &&
		len(c.Recipients()) > 0
}

// Recipients adalah penerima yang tidak kosong, sudah dipangkas.
func (c Config) Recipients() []string {
	result := make([]string, 0, len(c.To))
	for _, address := range c.To {
		if trimmed := strings.TrimSpace(address); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Send mengirim message kepada to. prefix mengawali setiap pesan galat, misalnya
// "masterrekening/notification".
//
// STARTTLS dipakai bila server menawarkannya. Ia tidak dipaksakan karena relay internal
// sering belum memasang sertifikat; yang dipaksakan justru sebaliknya — kredensial hanya
// dikirim setelah sambungan terenkripsi. Tenggang dipasang pada sambungan, bukan hanya pada
// pemutarnya: server yang menerima koneksi lalu diam adalah kegagalan yang paling sering
// menggantung proses.
func (c Config) Send(ctx context.Context, to []string, message []byte, prefix string) error {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	address := net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
	dialer := &net.Dialer{Timeout: timeout}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("%s: menghubungi server surel: %w", prefix, err)
	}
	_ = connection.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(connection, c.Host)
	if err != nil {
		_ = connection.Close()
		return fmt.Errorf("%s: memulai percakapan SMTP: %w", prefix, err)
	}
	defer func() { _ = client.Close() }()

	secured := false
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(nil); err != nil {
			return fmt.Errorf("%s: menegakkan TLS: %w", prefix, err)
		}
		secured = true
	}
	if c.User != "" {
		if !secured {
			return errors.New(prefix + ": server surel tidak mendukung STARTTLS; " +
				"kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		if err := client.Auth(smtp.PlainAuth("", c.User, c.Password, c.Host)); err != nil {
			return fmt.Errorf("%s: autentikasi SMTP ditolak: %w", prefix, err)
		}
	}

	if err := client.Mail(c.From); err != nil {
		return fmt.Errorf("%s: server menolak alamat pengirim: %w", prefix, err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("%s: server menolak alamat tujuan: %w", prefix, err)
		}
	}

	write, err := client.Data()
	if err != nil {
		return fmt.Errorf("%s: membuka badan surel: %w", prefix, err)
	}
	if _, err := write.Write(message); err != nil {
		_ = write.Close()
		return fmt.Errorf("%s: menulis badan surel: %w", prefix, err)
	}
	if err := write.Close(); err != nil {
		return fmt.Errorf("%s: menutup badan surel: %w", prefix, err)
	}
	return client.Quit()
}
