package utils

import (
	"context"
	"fmt"
	"strconv"

	"gopkg.in/gomail.v2"
)

type Mailer struct {
	host     string
	port     int
	username string
	password string
	dialer   *gomail.Dialer
}

func NewMailer(host, port, username, password string) *Mailer {
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("failed to parse port: %v", err))
	}

	return &Mailer{host: host, port: portNumber, username: username, password: password}
}

func (me *Mailer) SendEmail(ctx context.Context, message MailerMessage) error {
	m := gomail.NewMessage()
	m.SetHeader("From", message.From)
	m.SetHeader("To", message.To...)
	m.SetHeader("Subject", message.Subject)
	m.SetBody(message.ContentType, message.Body)

	errChan := make(chan error, 1)
	go func() {
		errChan <- gomail.NewDialer(
			me.host,
			me.port,
			me.username,
			me.password,
		).DialAndSend(m)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

type MailerMessage struct {
	From        string
	To          []string
	Subject     string
	ContentType string
	Body        string
}
