package notifier

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"

	log "github.com/sirupsen/logrus"
)

type NotifyTemplate struct {
	Name string
	URL  string
}

func SendEmail(msg []byte) error {
	addr := fmt.Sprintf("%s:%s", os.Getenv("SMTP_ADDRESS"), os.Getenv("SMTP_TLS_PORT"))
	auth := smtp.PlainAuth("", os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_ADDRESS"))
	from := os.Getenv("SMTP_USERNAME")
	recipient := os.Getenv("USER_EMAIL")

	err := smtp.SendMail(addr, auth, from, []string{recipient}, msg)
	if err != nil {
		return err
	}

	log.Infof("Successfully notified %s", recipient)

	return nil
}

func ParseTemplate(filename string, data interface{}) (string, error) {
	t, err := template.ParseFiles(filename)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
