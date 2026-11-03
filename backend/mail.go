package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxRecipients = 10

type mailConfig struct {
	Host, Port, Username, Password, FromName string
	DefaultRecipients                        []string
}

func loadMailConfig() mailConfig {
	return mailConfig{
		Host: env("SMTP_HOST", "smtp.hostinger.com"), Port: env("SMTP_PORT", "465"),
		Username: os.Getenv("SMTP_USERNAME"), Password: os.Getenv("SMTP_PASSWORD"),
		FromName: env("SMTP_FROM_NAME", "Cells"),
		DefaultRecipients: strings.FieldsFunc(os.Getenv("NOTIFY_RECIPIENTS"), func(r rune) bool {
			return r == ',' || r == ';' || r == '\n' || r == '\r'
		}),
	}
}

func (c mailConfig) validate() error {
	if strings.TrimSpace(c.Password) == "" {
		return errors.New("Isi SMTP_PASSWORD pada .env server, lalu buat ulang container backend.")
	}
	if c.Port != "465" && c.Port != "587" {
		return errors.New("SMTP_PORT harus 465 (SSL/TLS) atau 587 (STARTTLS).")
	}
	if strings.TrimSpace(c.Host) == "" || strings.ContainsAny(c.Host, " /:\r\n\t") {
		return errors.New("SMTP_HOST tidak valid.")
	}
	if _, err := normalizeRecipients([]string{c.Username}); err != nil {
		return errors.New("SMTP_USERNAME harus berupa alamat email mailbox yang valid.")
	}
	if hasControl(c.FromName) || utf8.RuneCountInString(c.FromName) > 100 {
		return errors.New("SMTP_FROM_NAME tidak valid (maksimal 100 karakter).")
	}
	return nil
}

func hasControl(value string) bool {
	return strings.ContainsFunc(value, unicode.IsControl)
}

func normalizeRecipients(values []string) ([]string, error) {
	if len(values) > maxRecipients {
		return nil, fmt.Errorf("Maksimal %d alamat email per broadcast.", maxRecipients)
	}
	result := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		address, err := mail.ParseAddress(value)
		// Keep mailbox input unambiguous, ASCII-only, and safe for SMTP envelopes.
		if err != nil || address.Address != value || len(value) > 254 || hasControl(value) || strings.ContainsFunc(value, func(r rune) bool { return r > 127 }) {
			return nil, errors.New("Gunakan alamat email valid tanpa nama tampilan, misalnya user@example.com.")
		}
		key := strings.ToLower(value)
		if !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	return result, nil
}

func validateEmailContent(subject, body string, formats ...string) error {
	limit := 10000
	if len(formats) > 0 && formats[0] == "html" {
		limit = 100000
	}
	if subject == "" || utf8.RuneCountInString(subject) > 160 || hasControl(subject) {
		return errors.New("Subjek harus 1–160 karakter tanpa baris baru.")
	}
	if strings.TrimSpace(body) == "" || utf8.RuneCountInString(body) > limit || strings.ContainsRune(body, 0) {
		return fmt.Errorf("Pesan harus 1–%d karakter dan tidak boleh berisi karakter NUL.", limit)
	}
	return nil
}

func emailMessage(c mailConfig, subject, body string, formats ...string) ([]byte, error) {
	contentType := "text/plain"
	if len(formats) > 0 && formats[0] == "html" {
		contentType = "text/html"
	}
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	from := (&mail.Address{Name: c.FromName, Address: c.Username}).String()
	domain := c.Username[strings.LastIndex(c.Username, "@")+1:]
	body = strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\r", "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.ReplaceAll(body, "\n", "\r\n")))
	var message strings.Builder
	fmt.Fprintf(&message, "From: %s\r\nTo: undisclosed-recipients:;\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@%s>\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n",
		from, mime.QEncoding.Encode("UTF-8", subject), time.Now().Format(time.RFC1123Z), token, domain, contentType)
	for len(encoded) > 76 {
		message.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	message.WriteString(encoded + "\r\n")
	return []byte(message.String()), nil
}

var errDeliveryUnknown = errors.New("delivery status unknown")

func (c mailConfig) send(ctx context.Context, recipients []string, subject, body string, formats ...string) error {
	if len(formats) > 0 {
		if err := validateNotifyFormat(formats[0]); err != nil {
			return err
		}
	}
	if err := c.validate(); err != nil {
		return err
	}
	validated, err := normalizeRecipients(recipients)
	if err != nil || len(validated) == 0 {
		return errors.New("Daftar penerima tidak valid atau kosong.")
	}
	if err := validateEmailContent(subject, body, formats...); err != nil {
		return err
	}
	message, err := emailMessage(c, subject, body, formats...)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(c.Host, c.Port))
	if err != nil {
		return err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	tlsConfig := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	var transport net.Conn = conn
	if c.Port == "465" {
		secure := tls.Client(conn, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return err
		}
		transport = secure
	}
	client, err := smtp.NewClient(transport, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if c.Port == "587" {
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if err := client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
		return err
	}
	return submitEmail(client, c.Username, validated, message)
}

// All recipients share one SMTP transaction. Abort before DATA if any RCPT is
// rejected; no recipient sees the other addresses in the message headers.
func submitEmail(client *smtp.Client, from string, recipients []string, message []byte) error {
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return errDeliveryUnknown
	}
	if err := writer.Close(); err != nil {
		// No automatic retry: the remote server may already have accepted DATA.
		return errDeliveryUnknown
	}
	// A successful final DATA reply is the acceptance boundary. Closing the
	// connection later does not turn an accepted message into a failed send.
	return nil
}
