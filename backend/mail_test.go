package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecipientValidation(t *testing.T) {
	got, err := normalizeRecipients([]string{" user@example.com ", "user@example.com", "other@example.com"})
	if err != nil || !reflect.DeepEqual(got, []string{"user@example.com", "other@example.com"}) {
		t.Fatalf("normalization: %v, %v", got, err)
	}
	for _, value := range []string{"", "invalid", "Name <a@example.com>", "a@example.com\r\nBcc: b@example.com", "a@example.com,b@example.com", "é@example.com"} {
		if _, err := normalizeRecipients([]string{value}); err == nil {
			t.Errorf("accepted invalid recipient %q", value)
		}
	}
	if _, err := normalizeRecipients(make([]string, maxRecipients+1)); err == nil {
		t.Fatal("accepted too many recipients")
	}
}

func TestEmailValidationAndEncoding(t *testing.T) {
	for _, subject := range []string{"", "hello\r\nBcc: bad@example.com", strings.Repeat("a", 161)} {
		if validateEmailContent(subject, "body") == nil {
			t.Fatalf("accepted invalid subject %q", subject)
		}
	}
	if validateEmailContent("subject", strings.Repeat("a", 10001)) == nil || validateEmailContent("subject", " ") == nil {
		t.Fatal("accepted invalid body")
	}
	c := mailConfig{Username: "sender@example.com", FromName: "Cells"}
	subject, body := "Pengingat 日本語 🌱", "Hello!\nBaris kedua.\n.Titik"
	raw, err := emailMessage(c, subject, body)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := (&mime.WordDecoder{}).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil || decoded != subject {
		t.Fatalf("subject did not round-trip: %s, %v", decoded, err)
	}
	content, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, msg.Body))
	if err != nil || string(content) != strings.ReplaceAll(body, "\n", "\r\n") {
		t.Fatalf("body did not round-trip: %q, %v", content, err)
	}
	if msg.Header.Get("To") != "undisclosed-recipients:;" || msg.Header.Get("Bcc") != "" || msg.Header.Get("Message-ID") == "" {
		t.Fatal("missing message ID or recipient privacy")
	}
}

func TestMailConfig(t *testing.T) {
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_USERNAME", "sender@example.com")
	c := loadMailConfig()
	if c.validate() == nil {
		t.Fatal("missing password accepted")
	}
	c.Password = "test-only"
	for _, port := range []string{"465", "587"} {
		c.Port = port
		if err := c.validate(); err != nil {
			t.Fatal(err)
		}
	}
	c.Port = "25"
	if c.validate() == nil {
		t.Fatal("unencrypted SMTP port accepted")
	}
}

func TestMailDefaultsHaveNoPersonalAddresses(t *testing.T) {
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("NOTIFY_RECIPIENTS", "")
	t.Setenv("SMTP_PASSWORD", "test-only")
	c := loadMailConfig()
	if c.Username != "" || len(c.DefaultRecipients) != 0 {
		t.Fatal("email addresses must come from explicit configuration")
	}
	if c.validate() == nil {
		t.Fatal("mail must be unavailable without an explicitly configured sender")
	}
}

func TestHTMLMailContentType(t *testing.T) {
	raw, err := emailMessage(mailConfig{Username: "sender@example.com", FromName: "Cells"}, "Reminder", "<table><tr><td>Task</td></tr></table>", "html")
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Header.Get("Content-Type") != "text/html; charset=UTF-8" {
		t.Fatal("HTML email sent as plain text")
	}
	body, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, msg.Body))
	if err != nil || !strings.Contains(string(body), "<table>") {
		t.Fatal("HTML body did not round-trip", err)
	}
}

func TestSMTPTransaction(t *testing.T) {
	for _, scenario := range []string{"accepted", "reject-recipient", "lost-data-reply"} {
		t.Run(scenario, func(t *testing.T) {
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			clientConn.SetDeadline(time.Now().Add(3 * time.Second))
			commands := make(chan []string, 1)
			go func() {
				defer serverConn.Close()
				seen := []string{}
				defer func() { commands <- seen }()
				reader := textproto.NewReader(bufio.NewReader(serverConn))
				fmt.Fprint(serverConn, "220 test SMTP\r\n")
				for {
					line, err := reader.ReadLine()
					if err != nil {
						return
					}
					seen = append(seen, line)
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"), strings.HasPrefix(line, "MAIL"):
						fmt.Fprint(serverConn, "250 OK\r\n")
					case strings.HasPrefix(line, "RCPT"):
						if scenario == "reject-recipient" && strings.Contains(line, "second@") {
							fmt.Fprint(serverConn, "550 rejected\r\n")
						} else {
							fmt.Fprint(serverConn, "250 OK\r\n")
						}
					case line == "DATA":
						fmt.Fprint(serverConn, "354 send data\r\n")
						data, err := reader.ReadDotBytes()
						if err != nil {
							return
						}
						seen = append(seen, string(data))
						if scenario == "lost-data-reply" {
							return
						}
						fmt.Fprint(serverConn, "250 queued\r\n")
						return
					default:
						return
					}
				}
			}()
			client, err := smtp.NewClient(clientConn, "localhost")
			if err != nil {
				t.Fatal(err)
			}
			err = submitEmail(client, "sender@example.com", []string{"first@example.com", "second@example.com"}, []byte("Subject: Test\r\n\r\nHello\r\n"))
			client.Close()
			seen := <-commands
			switch scenario {
			case "accepted":
				if err != nil {
					t.Fatal(err)
				}
			case "reject-recipient":
				if err == nil || strings.Contains(strings.Join(seen, "\n"), "DATA") {
					t.Fatal("sent DATA after recipient rejection")
				}
			case "lost-data-reply":
				if !errors.Is(err, errDeliveryUnknown) {
					t.Fatalf("expected unknown delivery, got %v", err)
				}
			}
		})
	}
}

func TestNotifyRequiresAuthenticationAndCSRF(t *testing.T) {
	app := &App{}
	for _, tc := range []struct{ method, path string }{{"GET", "/api/notify"}, {"PUT", "/api/notify"}, {"POST", "/api/notify/test"}, {"POST", "/api/notify/broadcast"}} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		r.Header.Set(csrfHeader, csrfHeaderVal)
		w := httptest.NewRecorder()
		app.routes().ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
		if tc.method != "GET" {
			r.Header.Del(csrfHeader)
			w = httptest.NewRecorder()
			app.routes().ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("missing CSRF: %d", w.Code)
			}
		}
	}
}
