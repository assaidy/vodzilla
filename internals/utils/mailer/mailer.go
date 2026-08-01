package mailer

import (
	"context"
	"fmt"
	"strconv"

	"gopkg.in/gomail.v2"
)

type Mailer interface {
	Send(ctx context.Context, message Message) error
}

type Message struct {
	From        string
	To          []string
	Subject     string
	ContentType string
	Body        string
}

type impl struct {
	host     string
	port     int
	username string
	password string
	dialer   *gomail.Dialer
}

func New(host, port, username, password string) Mailer {
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("failed to parse port: %v", err))
	}

	return &impl{
		host:     host,
		port:     portNumber,
		username: username,
		password: password,
		dialer:   gomail.NewDialer(host, portNumber, username, password),
	}
}

func (me *impl) Send(ctx context.Context, message Message) error {
	m := gomail.NewMessage()
	m.SetHeader("From", message.From)
	m.SetHeader("To", message.To...)
	m.SetHeader("Subject", message.Subject)
	m.SetBody(message.ContentType, message.Body)

	errChan := make(chan error, 1)
	go func() {
		errChan <- me.dialer.DialAndSend(m)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
