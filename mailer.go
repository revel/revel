package revel

import (
  "encoding/base64"
  "mime/multipart"
  "path/filepath"
  "crypto/tls"
  "io/ioutil"
  "net/smtp"
  "runtime"
  "strings"
  "reflect"
  "errors"
  "bytes"
  "net"
  "fmt"
  "io"
  "os"
)

const CRLF = "\r\n"

type Mailer struct {
  to, cc, bcc []string            // these are here in so that in the future defaults can be set on the mailer
  address, from, username string
  port int
  tls, debug, concurrent bool
}

// an encapsulation of a sending mail's config
// this is in an effort to make it thread safe
type MailConfig struct {
  to, cc, bcc []string
  renderargs map[string]interface{}
  from, template string
  attachments map[string][]byte
}

type H map[string]interface{}

func (m *Mailer) new_config(template_name string, mail_args map[string]interface{}) (MailConfig, error) {
  mail_config := MailConfig{}
  mail_config.renderargs = mail_args
  mail_config.template = template_name
  ok := true

  mail_config.from, ok = Config.String("mail.from") 
  if !ok {
    ERROR.Println("mail.from not set")
  }

  if mail_args["from"] != nil {
    mail_config.from = reflect.ValueOf(mail_args["from"]).String()
  } else {
    mail_config.from = Config.StringDefault("mail.from", Config.StringDefault("mail.username", ""))
  }
  if mail_config.renderargs["to"] != nil {
    mail_config.to = makeSAFI(mail_args["to"])
  }
  if mail_config.renderargs["cc"] != nil {
    mail_config.cc = makeSAFI(mail_args["cc"])
  }
  if mail_config.renderargs["bcc"] != nil {
    mail_config.bcc = makeSAFI(mail_args["bcc"])
  }

  if mail_config.renderargs["attachments"] != nil {
    attachement_paths := makeSAFI(mail_args["attachments"])
    for _, file := range attachement_paths {
      mail_config.attachments = make(map[string][]byte)

      b, err := ioutil.ReadFile(file)
      if err != nil {
        ERROR.Println(err)
        return mail_config, err
      }

      _, fileName := filepath.Split(file)
      mail_config.attachments[fileName] = b
    }
  }

  if ok {
    return mail_config, nil
  } else {
    return mail_config, errors.New("There was a problem with your config please check the logs")
  }
}

func (m *Mailer) Address() string {
  if m.address == "" {
    ok := false
    m.address, ok = Config.String("mail.address")
    if !ok {
      ERROR.Println("mail address not set")
    }
  }
  return m.address
}

func (m *Mailer) Username() string {
  if m.username == "" {
    ok := false
    m.username, ok = Config.String("mail.username")
    if !ok {
      ERROR.Println("mail username not set")
    }
  }
  return m.username
}

func (m *Mailer) Port() int {
  if m.port == 0 {
    ok := false
    m.port, ok = Config.Int("mail.port")
    if !ok {
      ERROR.Println("mail port not set")
    }
  }
  return m.port
}

func (m *Mailer) isTLS() bool {
  return Config.BoolDefault("mail.tls", false)
}

func (m *Mailer) isDebug() bool {
  return Config.BoolDefault("mail.debug", false)
}

func (m *Mailer) isConcurrent() bool {
  return Config.BoolDefault("mail.concurrent", false)
}

func (m *Mailer) getClient() (*smtp.Client, error) {
  var c *smtp.Client
  if m.isTLS() == true {
    conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", m.Address(), m.Port()), nil)
    if err != nil {
      return nil, err
    }
    c, err = smtp.NewClient(conn, m.Address())
    if err != nil {
      return nil, err
    }
  } else {
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", m.Address(), m.Port()))
    if err != nil {
      return nil, err
    }
    c, err = smtp.NewClient(conn, m.Address())
    if err != nil {
      return nil, err
    }
  }
  return c, nil
}

func (m *Mailer) Send(mail_args map[string]interface{}) error {
  pc, _, _, _ := runtime.Caller(1)
  names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
  template :=  names[len(names)-2] + "/" + names[len(names)-1]

  mail_config, err := m.new_config(template, mail_args)
  if err != nil {
    return err
  }

  if m.isDebug() {
    return m.sendDebug(mail_config)
  }else{
    if m.isConcurrent() {
      go m.send(mail_config)
      return nil
    }else{
      return m.send(mail_config)
    }
  }
}

func (m *Mailer) sendDebug(mail_config MailConfig) error {
  mail, err := m.renderMail(mail_config, nil)
  if err != nil {
    return err
  }
  fmt.Println(string(mail))
  return nil
}

func (m *Mailer) send(mail_config MailConfig) error {
  c, err := m.getClient()
  if err != nil {
    return err
  }

  if ok, _ := c.Extension("STARTTLS"); ok {
    if err = c.StartTLS(nil); err != nil {
      return err
    }
  }

  if err = c.Auth(smtp.PlainAuth(mail_config.from, m.Username(), m.getPassword(), m.Address())); err != nil {
    return err
  }

  if err = c.Mail(m.Username()); err != nil {
    return err
  }

  if len(mail_config.to) + len(mail_config.cc) + len(mail_config.bcc) == 0 {
    return fmt.Errorf("Cannot send email without recipients")
  }

  recipients := append(mail_config.to, append(mail_config.cc, mail_config.bcc...)...)
  for _, addr := range recipients {
    if err = c.Rcpt(addr); err != nil {
      return err
    }
  }
  w, err := c.Data()
  if err != nil {
    return err
  }

  mail, err := m.renderMail(mail_config, w)
  if err != nil {
    return err
  }

  _, err = w.Write(mail)
  if err != nil {
    return err
  }
  err = w.Close()
  if err != nil {
    return err
  }

  return c.Quit()
}

func (m *Mailer) renderMail(mail_config MailConfig, w io.WriteCloser) ([]byte, error) {
  multi := newMulti(w)

  body, err := m.renderBody(mail_config, w)
  if err != nil {
    return nil, err
  }

  mail := []string{
    "Subject: " + reflect.ValueOf(mail_config.renderargs["subject"]).String(),
    "From: " + mail_config.from,
    "To: " + strings.Join(mail_config.to, ","),
    "Bcc: " + strings.Join(mail_config.bcc, ","),
    "Cc: " + strings.Join(mail_config.cc, ","),
    "MIME-Version: 1.0",
    "Content-Type: multipart/mixed; boundary=" + multi.Boundary(),
    "Content-Transfer-Encoding: 7bit",
    CRLF,
    "--" + multi.Boundary(),
    body,
    m.renderAttachments(mail_config, multi.Boundary()),
    "--" + multi.Boundary() + "--",
    CRLF,
  }

  return []byte(strings.Join(mail, CRLF)), nil
}

func (m *Mailer) renderBody(mail_config MailConfig, w io.WriteCloser) (string, error) {
  multi := newMulti(w)

  body := bytes.NewBuffer(nil)

  body.WriteString("Mime-Version: 1.0" + CRLF)
  body.WriteString("Content-Type: multipart/alternative; boundary=" + multi.Boundary() + "; charset=UTF-8" + CRLF)
  body.WriteString("Content-Transfer-Encoding: 7bit" + CRLF + CRLF)

  template_count := 0
  contents := map[string]string{"plain": m.renderTemplate(mail_config, "txt"), "html": m.renderTemplate(mail_config, "html")}
  for k, v := range contents {
    if v != "" {
      body.WriteString("--" + multi.Boundary() + CRLF + "Content-Type: text/" + k + "; charset=UTF-8" + CRLF + "Content-Transfer-Encoding: quoted-printable" + CRLF + CRLF + v + CRLF + CRLF)
      template_count++
    }
  }

  body.WriteString("--" + multi.Boundary() + "--")

  if template_count == 0 {
    return "", fmt.Errorf("No valid mail templates were found with the names %s.[html|txt]", mail_config.template)
  }

  return body.String(), nil
}

func (m *Mailer) renderAttachments(mail_config MailConfig, boundary string) string {
  body := bytes.NewBuffer(nil)

  if len(mail_config.attachments) > 0 {
    body.WriteString(CRLF)
    for k, v := range mail_config.attachments {
      body.WriteString("--" + boundary + CRLF)
      body.WriteString("Content-Type: application/octet-stream"+CRLF)
      body.WriteString("Content-Transfer-Encoding: base64"+CRLF)
      body.WriteString("Content-Disposition: attachment; filename=\"" + k + "\"" + CRLF + CRLF)

      b := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
      base64.StdEncoding.Encode(b, v)
      body.Write(b)
      body.WriteString(CRLF)
    }
  }

  return body.String()
}

func (m *Mailer) renderTemplate(mail_config MailConfig, mime string) string {
  var body bytes.Buffer
  template, err := MainTemplateLoader.Template(mail_config.template + "." + mime)
  if template == nil || err != nil {
    if m.isDebug() {
      ERROR.Println(err)
    }
    return ""
  } else {
    template.Render(&body, mail_config.renderargs)
  }
  return body.String()
}

// look for the env variable if not check the
// config
func (m *Mailer) getPassword() string {
  if os.Getenv("REVEL_EMAIL_PW") != "" {
    return os.Getenv("REVEL_EMAIL_PW")
  }else if password, ok := Config.String("mail.password"); ok {
    return password
  }

  ERROR.Println("mail password not set")
  return ""
}

// this just simplifies debugging, so if we are not 
//writing to a mail.Data() then just write to am empty writer
func newMulti(w io.WriteCloser) *multipart.Writer {
  if w != nil {
    return multipart.NewWriter(w)
  }else{
    return multipart.NewWriter(bytes.NewBufferString(""))
  }
}

// make string array from interface this is used for
// extracting options from the mail args
func makeSAFI(intfc interface{}) []string{
  result := []string{}
  slicev := reflect.ValueOf(intfc)
  for i := 0; i < slicev.Len(); i++ {
    result = append(result, slicev.Index(i).String())
  }
  return result
}

