package email

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
)

type Email struct {
	MessageId       string                      `json:"message_id,omitempty" db:"message_id"`
	Date            string                      `json:"date,omitempty" db:"date"`
	Subject         string                      `json:"subject,omitempty" db:"subject"`
	FromName1       string                      `json:"from_name_1,omitempty" db:"from_name_1"`
	FromMailbox1    string                      `json:"from_mailbox_1,omitempty" db:"from_mailbox_1"`
	FromHost1       string                      `json:"from_host_1,omitempty" db:"from_host_1"`
	SenderName1     string                      `json:"sender_name_1,omitempty" db:"sender_name_1"`
	SenderMailbox1  string                      `json:"sender_mailbox_1,omitempty" db:"sender_mailbox_1"`
	SenderHost1     string                      `json:"sender_host_1,omitempty" db:"sender_host_1"`
	ReplyToName1    string                      `json:"reply_to_name_1,omitempty" db:"reply_to_name_1"`
	ReplyToMailbox1 string                      `json:"reply_to_mailbox_1,omitempty" db:"reply_to_mailbox_1"`
	ReplyToHost1    string                      `json:"reply_to_host_1,omitempty" db:"reply_to_host_1"`
	ToName1         string                      `json:"to_name_1,omitempty" db:"to_name_1"`
	ToMailbox1      string                      `json:"to_mailbox_1,omitempty" db:"to_mailbox_1"`
	ToHost1         string                      `json:"to_host_1,omitempty" db:"to_host_1"`
	CcName1         string                      `json:"cc_name_1,omitempty" db:"cc_name_1"`
	CcMailbox1      string                      `json:"cc_mailbox_1,omitempty" db:"cc_mailbox_1"`
	CcHost1         string                      `json:"cc_host_1,omitempty" db:"cc_host_1"`
	BccName1        string                      `json:"bcc_name_1,omitempty" db:"bcc_name_1"`
	BccMailbox1     string                      `json:"bcc_mailbox_1,omitempty" db:"bcc_mailbox_1"`
	BccHost1        string                      `json:"bcc_host_1,omitempty" db:"bcc_host_1"`
	InReplyTo       string                      `json:"in_reply_to,omitempty" db:"in_reply_to"`
	Mailboxes       []string                    `json:"mailboxes,omitempty" db:"mailboxes"`
	ParseWarning    string                      `json:"parse_warning,omitempty" db:"parse_warning"`
	ParseError      string                      `json:"parse_error,omitempty" db:"parse_error"`
	OurId           string                      `json:"our_id,omitempty" db:"our_id"`
	Envelope        *imap.Envelope              `json:"envelope,omitempty" db:"envelope"`
	Flags           []string                    `json:"flags,omitempty" db:"flags"`
	UID             uint32                      `json:"uid,omitempty" db:"uid"`
	TextContent     string                      `json:"text_content,omitempty" db:"text_content"`
	HTMLContent     string                      `json:"html_content,omitempty" db:"html_content"`
	Attachments     []models.AttachmentMetaData `json:"attachments,omitempty" db:"attachments"`
	client          models.Client
}

// assumes the currently selected mailbox is the mailbox this email is in
func New(msg *imap.Message, client models.Client) models.Email {
	emailWrap := &Email{
		client: client,
	}
	return emailWrap.parseMessage(msg)
}

func NewFromDBRecord(rows *sqlx.Rows) (models.Email, error) {
	emailWrap := &Email{}

	rowData := make(map[string]interface{})
	err := rows.MapScan(rowData)
	if err != nil {
		return nil, utils.JoinErrors("error mapping row to email", err)
	}

	if !utils.IsInterfaceNil(rowData["message_id"]) {
		emailWrap.MessageId = rowData["message_id"].(string)
	}
	if !utils.IsInterfaceNil(rowData["date"]) {
		emailWrap.Date = rowData["date"].(string)
	}
	if !utils.IsInterfaceNil(rowData["subject"]) {
		emailWrap.Subject = rowData["subject"].(string)
	}
	if !utils.IsInterfaceNil(rowData["from_name_1"]) {
		emailWrap.FromName1 = rowData["from_name_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["from_mailbox_1"]) {
		emailWrap.FromMailbox1 = rowData["from_mailbox_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["from_host_1"]) {
		emailWrap.FromHost1 = rowData["from_host_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["sender_name_1"]) {
		emailWrap.SenderName1 = rowData["sender_name_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["sender_mailbox_1"]) {
		emailWrap.SenderMailbox1 = rowData["sender_mailbox_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["sender_host_1"]) {
		emailWrap.SenderHost1 = rowData["sender_host_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["reply_to_name_1"]) {
		emailWrap.ReplyToName1 = rowData["reply_to_name_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["reply_to_mailbox_1"]) {
		emailWrap.ReplyToMailbox1 = rowData["reply_to_mailbox_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["reply_to_host_1"]) {
		emailWrap.ReplyToHost1 = rowData["reply_to_host_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["to_name_1"]) {
		emailWrap.ToName1 = rowData["to_name_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["to_mailbox_1"]) {
		emailWrap.ToMailbox1 = rowData["to_mailbox_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["to_host_1"]) {
		emailWrap.ToHost1 = rowData["to_host_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["cc_name_1"]) {
		emailWrap.CcName1 = rowData["cc_name_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["cc_mailbox_1"]) {
		emailWrap.CcMailbox1 = rowData["cc_mailbox_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["cc_host_1"]) {
		emailWrap.CcHost1 = rowData["cc_host_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["bcc_name_1"]) {
		emailWrap.BccName1 = rowData["bcc_name_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["bcc_mailbox_1"]) {
		emailWrap.BccMailbox1 = rowData["bcc_mailbox_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["bcc_host_1"]) {
		emailWrap.BccHost1 = rowData["bcc_host_1"].(string)
	}
	if !utils.IsInterfaceNil(rowData["in_reply_to"]) {
		emailWrap.InReplyTo = rowData["in_reply_to"].(string)
	}
	if !utils.IsInterfaceNil(rowData["mailboxes"]) {
		emailWrap.Mailboxes = strings.Split(rowData["mailboxes"].(string), ",")
	}
	if !utils.IsInterfaceNil(rowData["parse_warning"]) {
		emailWrap.ParseWarning = rowData["parse_warning"].(string)
	}
	if !utils.IsInterfaceNil(rowData["parse_error"]) {
		emailWrap.ParseError = rowData["parse_error"].(string)
	}
	if !utils.IsInterfaceNil(rowData["our_id"]) {
		emailWrap.OurId = rowData["our_id"].(string)
	}
	if !utils.IsInterfaceNil(rowData["flags"]) {
		emailWrap.Flags = strings.Split(rowData["flags"].(string), ",")
	}
	if !utils.IsInterfaceNil(rowData["uid"]) {
		emailWrap.UID = rowData["uid"].(uint32)
	}
	if !utils.IsInterfaceNil(rowData["text_content"]) {
		emailWrap.TextContent = rowData["text_content"].(string)
	}
	if !utils.IsInterfaceNil(rowData["html_content"]) {
		emailWrap.HTMLContent = rowData["html_content"].(string)
	}
	emailWrap.Attachments = []models.AttachmentMetaData{}
	emailWrap.Mailboxes = []string{}

	if !utils.IsInterfaceNil(rowData["attachments"]) {
		err = json.Unmarshal([]byte(rowData["attachments"].(string)), &emailWrap.Attachments)
		if err != nil {
			return nil, utils.JoinErrors("error unmarshalling attachments", err)
		}
	}

	if !utils.IsInterfaceNil(rowData["mailboxes"]) {
		err = json.Unmarshal([]byte(rowData["mailboxes"].(string)), &emailWrap.Mailboxes)
		if err != nil {
			return nil, utils.JoinErrors("error unmarshalling mailboxes", err)
		}
	}

	return emailWrap, nil
}

func (emailWrap *Email) parseMessage(msg *imap.Message) *Email {
	// our id is a hash because message-id isn't reliable
	hashSources := []string{}

	email := &Email{
		Flags:    msg.Flags,
		Envelope: msg.Envelope,
	}

	if !utils.IsInterfaceNil(email.Envelope) {
		if !utils.IsInterfaceNil(email.Envelope.Date) {
			hashSources = append(hashSources, email.Envelope.Date.String())
			email.Date = email.Envelope.Date.String()
		}
		if !utils.IsInterfaceNil(email.Envelope.Subject) {
			hashSources = append(hashSources, email.Envelope.Subject)
			email.Subject = email.Envelope.Subject
		}
		if !utils.IsInterfaceNil(email.Envelope.From) {
			hashSources = append(hashSources, utils.MustJSON(email.Envelope.From))
			if len(email.Envelope.From) > 0 {
				email.FromName1 = email.Envelope.From[0].PersonalName
				email.FromMailbox1 = email.Envelope.From[0].MailboxName
				email.FromHost1 = email.Envelope.From[0].HostName
			}
		}
		if !utils.IsInterfaceNil(email.Envelope.To) {
			hashSources = append(hashSources, utils.MustJSON(email.Envelope.To))
			if len(email.Envelope.To) > 0 {
				email.ToName1 = email.Envelope.To[0].PersonalName
				email.ToMailbox1 = email.Envelope.To[0].MailboxName
				email.ToHost1 = email.Envelope.To[0].HostName
			}
		}
		if !utils.IsInterfaceNil(email.Envelope.Cc) {
			hashSources = append(hashSources, utils.MustJSON(email.Envelope.Cc))
			if len(email.Envelope.Cc) > 0 {
				email.CcName1 = email.Envelope.Cc[0].PersonalName
				email.CcMailbox1 = email.Envelope.Cc[0].MailboxName
				email.CcHost1 = email.Envelope.Cc[0].HostName
			}
		}
		if !utils.IsInterfaceNil(email.Envelope.Bcc) {
			hashSources = append(hashSources, utils.MustJSON(email.Envelope.Bcc))
			if len(email.Envelope.Bcc) > 0 {
				email.BccName1 = email.Envelope.Bcc[0].PersonalName
				email.BccMailbox1 = email.Envelope.Bcc[0].MailboxName
				email.BccHost1 = email.Envelope.Bcc[0].HostName
			}
		}
		if !utils.IsInterfaceNil(email.Envelope.ReplyTo) {
			hashSources = append(hashSources, utils.MustJSON(email.Envelope.ReplyTo))
			if len(email.Envelope.ReplyTo) > 0 {
				email.ReplyToName1 = email.Envelope.ReplyTo[0].PersonalName
				email.ReplyToMailbox1 = email.Envelope.ReplyTo[0].MailboxName
				email.ReplyToHost1 = email.Envelope.ReplyTo[0].HostName
			}
		}
		if !utils.IsInterfaceNil(email.Envelope.InReplyTo) {
			hashSources = append(hashSources, email.Envelope.InReplyTo)
			email.InReplyTo = email.Envelope.InReplyTo
		}
		if !utils.IsInterfaceNil(email.Envelope.MessageId) {
			hashSources = append(hashSources, email.Envelope.MessageId)
			email.MessageId = email.Envelope.MessageId
		}
	} else {
		// not much else we can do
		email.OurId = "nil-envelope;uid=" + strconv.Itoa(int(email.UID))
	}

	// NOTE: if needed in the future we can also hash the body, but I haven't seen any collisions yet,
	// 		 so it seems overkill

	if !strings.HasPrefix(email.OurId, "nil-envelope") {
		hasher := sha256.New()
		hasher.Write([]byte(strings.Join(hashSources, "")))
		email.OurId = hex.EncodeToString(hasher.Sum(nil))
	}

	r := msg.GetBody(models.SectionToFetch)
	if r == nil {
		errorMsg := "Server didn't returned message body"
		if emailWrap.client.Options().GetStrictMailParsing() {
			log.Fatal(errorMsg)
		} else {
			email.ParseError = errorMsg
			return email
		}
	}
	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to create mail reader: %v", err)
		if emailWrap.client.Options().GetStrictMailParsing() {
			log.Fatal(errorMessage, "\n")
		} else {
			email.ParseError = errorMessage
			return email
		}
	}

	for {

		// optimistic parsing, consume as much as possible
		// if we hit an error, we'll just set the parse error and move on, unless we're in strict mode.
		// We only save the last error we hit
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			if emailWrap.client.Options().GetStrictMailParsing() {
				log.Fatal("failed to parse next part ", err)
			} else {

				email.ParseError = err.Error()
			}
		}
		// sometime part is nil, not sure why, we'll consider that an error
		if part == nil {
			if emailWrap.client.Options().GetStrictMailParsing() {
				log.Fatal("part is nil")
			} else {
				email.ParseError = "received an empty message part from the mail parser"
				break
			}
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			// can be plain-text , HTML, or inline attachments
			contentType, params, err := h.ContentType()
			if err != nil {
				if emailWrap.client.Options().GetStrictMailParsing() {
					log.Fatal("failed to get content type", err)
				} else {
					email.ParseError = err.Error()
					break
				}
			}
			content, contentErr := io.ReadAll(part.Body)
			if contentErr != nil {
				if emailWrap.client.Options().GetStrictMailParsing() {
					log.Fatal("failed to read body", err)
				} else {
					email.ParseError = contentErr.Error()
				}
			}
			isHtml := strings.HasPrefix(contentType, "text/html")
			isText := strings.HasPrefix(contentType, "text/plain")

			if isHtml || isText {
				if isHtml && contentErr == nil {
					email.HTMLContent = string(content)
				}
				if isText && contentErr == nil {
					email.TextContent = string(content)
				}
			} else if strings.HasPrefix(contentType, "image/") {
				// image header looks like this:
				/*
					------_=_NextPart_e4d7791e-5588-4ad7-a5dc-070785def2b1
					Content-Type: image/png;
						name="image613224.png"
					Content-Transfer-Encoding: base64
					Content-ID: <image613224.png@0D025374.04F2C0BC>
					Content-Description: image613224.png
					Content-Disposition: inline;
						creation-date="Thu, 04 Jan 2024 15:46:38 +0000";
						filename=image613224.png;
						modification-date="Thu, 04 Jan 2024 15:46:38 +0000";
						size=1841
				*/

				attachment := models.AttachmentMetaData{}
				fileName := h.Get("Content-Description")
				size := 0
				if fileName == "" {
					// try and get name param
					for _, param := range []string{"name", "filename", "NAME", "FILENAME"} {
						if fileName != "" {
							break
						}
						if val, ok := params[param]; ok {
							fileName = val
						}
					}
				}

				if fileName == "" {
					fileName = h.Get("Content-ID")
				}

				if contentErr == nil {
					size = len(content)
				}

				attachment.Disposition = models.DispositionInline
				attachment.Encoding = h.Get("Content-Transfer-Encoding")
				attachment.FileType = contentType
				attachment.FileName = fileName
				attachment.FileSize = size
				email.Attachments = append(email.Attachments, attachment)

			} else {
				email.ParseWarning = fmt.Sprintf("unknown inline content type: %v\n", contentType)
				break
			}
		case *mail.AttachmentHeader:
			attachment := models.AttachmentMetaData{}

			contentType, _, err := h.ContentType()
			if err != nil {
				if emailWrap.client.Options().GetStrictMailParsing() {
					log.Fatal("failed to get content type", err)
				} else {
					email.ParseError = err.Error()
					break
				}
			}

			content, contentErr := io.ReadAll(part.Body)
			if contentErr != nil && emailWrap.client.Options().GetStrictMailParsing() {
				log.Fatal("failed to read body", err)
			}

			filename, _ := h.Filename()

			if filename == "" {
				filename = h.Get("Content-Description")
			}
			encoding := h.Get("Content-Transfer-Encoding")
			attachment.FileName = filename
			attachment.FileSize = len(content)
			attachment.Encoding = encoding
			attachment.FileType = contentType
			attachment.Disposition = models.DispositionAttachment
			email.Attachments = append(email.Attachments, attachment)
		}

	}
	return email
}

func (emailWrap *Email) GetParseWarning() string {
	return emailWrap.ParseWarning
}

func (emailWrap *Email) GetParseError() string {
	return emailWrap.ParseError
}

func (emailWrap *Email) GetOurID() string {
	return emailWrap.OurId
}

func (emailWrap *Email) GetEnvelope() *imap.Envelope {
	return emailWrap.Envelope
}

func (emailWrap *Email) GetFlags() []string {
	return emailWrap.Flags
}

func (emailWrap *Email) GetUID() uint32 {
	return emailWrap.UID
}

func (emailWrap *Email) GetTextContent() string {
	return emailWrap.TextContent
}

func (emailWrap *Email) GetHTMLContent() string {
	return emailWrap.HTMLContent
}

func (emailWrap *Email) GetAttachments() []models.AttachmentMetaData {
	return emailWrap.Attachments
}

func (emailWrap *Email) GetMessageId() string {
	return emailWrap.MessageId
}

func (emailWrap *Email) GetDate() string {
	return emailWrap.Date
}

func (emailWrap *Email) GetSubject() string {
	return emailWrap.Subject
}

func (emailWrap *Email) GetFromName1() string {
	return emailWrap.FromName1
}

func (emailWrap *Email) GetFromMailbox1() string {
	return emailWrap.FromMailbox1
}

func (emailWrap *Email) GetFromHost1() string {
	return emailWrap.FromHost1
}

func (emailWrap *Email) GetSenderName1() string {
	return emailWrap.SenderName1
}

func (emailWrap *Email) GetSenderMailbox1() string {
	return emailWrap.SenderMailbox1
}

func (emailWrap *Email) GetSenderHost1() string {
	return emailWrap.SenderHost1
}

func (emailWrap *Email) GetReplyToName1() string {
	return emailWrap.ReplyToName1
}

func (emailWrap *Email) GetReplyToMailbox1() string {
	return emailWrap.ReplyToMailbox1
}

func (emailWrap *Email) GetReplyToHost1() string {
	return emailWrap.ReplyToHost1
}

func (emailWrap *Email) GetToName1() string {
	return emailWrap.ToName1
}

func (emailWrap *Email) GetToMailbox1() string {
	return emailWrap.ToMailbox1
}

func (emailWrap *Email) GetToHost1() string {
	return emailWrap.ToHost1
}

func (emailWrap *Email) GetCcName1() string {
	return emailWrap.CcName1
}

func (emailWrap *Email) GetCcMailbox1() string {
	return emailWrap.CcMailbox1
}

func (emailWrap *Email) GetCcHost1() string {
	return emailWrap.CcHost1
}

func (emailWrap *Email) GetBccName1() string {
	return emailWrap.BccName1
}

func (emailWrap *Email) GetBccMailbox1() string {
	return emailWrap.BccMailbox1
}

func (emailWrap *Email) GetBccHost1() string {
	return emailWrap.BccHost1
}

func (emailWrap *Email) GetInReplyTo() string {
	return emailWrap.InReplyTo
}
