package main

import (
	"bytes"
	"html/template"

	"github.com/go-mail/mail/v2"
)

type mailer struct {
	dailer *mail.Dialer
	sender string
}

func newMailer(host string, port int, username string, password string, sender string) *mailer {
	dailer := mail.NewDialer(host, port, username, password)
	return &mailer{
		dailer: dailer,
		sender: sender,
	}
}

func (m *mailer) send(to string, tmpl *template.Template, data any) error {
	var subject bytes.Buffer
	err := tmpl.ExecuteTemplate(&subject, "subject", data)
	if err != nil {
		return err
	}
	var plainBody bytes.Buffer
	err = tmpl.ExecuteTemplate(&plainBody, "plainBody", data)
	if err != nil {
		return err
	}
	var htmlBody bytes.Buffer
	err = tmpl.ExecuteTemplate(&htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", to)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.SetBody("text/html", htmlBody.String())

	for i := 0; i < 3; i++ {
		err = m.dailer.DialAndSend(msg)
		if err == nil {
			break
		}
	}
	return err
}
