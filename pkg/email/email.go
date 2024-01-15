package email

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"io"
	"log"
	"strconv"
	"strings"
)

type Email struct {
	mailbox      string
	parseWarning string
	parseError   string
	ourId        string
	envelope     *imap.Envelope
	flags        []string
	uID          uint32
	textContent  string
	hTMLContent  string
	attachments  []models.AttachmentMetaData
	client       models.Client
}

// assumes the currently selected mailbox is the mailbox this email is in
func New(msg *imap.Message, client models.Client) models.Email {
	emailWrap := &Email{
		client: client,
	}
	return emailWrap.parseMessage(msg)
}

func (emailWrap *Email) parseMessage(msg *imap.Message) *Email {
	// our id is a hash because message-id isn't reliable
	hashSources := []string{}

	email := &Email{
		flags:    msg.Flags,
		envelope: msg.Envelope,
		mailbox:  emailWrap.client.CurrentMailbox().Name(),
	}

	if !utils.IsInterfaceNil(email.envelope) {
		if !utils.IsInterfaceNil(email.envelope.Date) {
			hashSources = append(hashSources, email.envelope.Date.String())
		}
		if !utils.IsInterfaceNil(email.envelope.Subject) {
			hashSources = append(hashSources, email.envelope.Subject)
		}
		if !utils.IsInterfaceNil(email.envelope.From) {
			hashSources = append(hashSources, utils.MustJSON(email.envelope.From))
		}
		if !utils.IsInterfaceNil(email.envelope.To) {
			hashSources = append(hashSources, utils.MustJSON(email.envelope.To))
		}
		if !utils.IsInterfaceNil(email.envelope.Cc) {
			hashSources = append(hashSources, utils.MustJSON(email.envelope.Cc))
		}
		if !utils.IsInterfaceNil(email.envelope.Bcc) {
			hashSources = append(hashSources, utils.MustJSON(email.envelope.Bcc))
		}
		if !utils.IsInterfaceNil(email.envelope.ReplyTo) {
			hashSources = append(hashSources, utils.MustJSON(email.envelope.ReplyTo))
		}
		if !utils.IsInterfaceNil(email.envelope.InReplyTo) {
			hashSources = append(hashSources, email.envelope.InReplyTo)
		}
		if !utils.IsInterfaceNil(email.envelope.MessageId) {
			hashSources = append(hashSources, email.envelope.MessageId)
		}
	} else {
		// not much else we can do
		email.ourId = "nil-envelope;uid=" + strconv.Itoa(int(email.uID))
	}

	// NOTE: if needed in the future we can also hash the body, but I haven't seen any collisions yet,
	// 		 so it seems overkill
	hasher := sha256.New()
	hasher.Write([]byte(strings.Join(hashSources, "")))
	email.ourId = hex.EncodeToString(hasher.Sum(nil))

	r := msg.GetBody(models.SectionToFetch)
	if r == nil {
		errorMsg := "Server didn't returned message body"
		if emailWrap.client.Options().StrictMailParsing() {
			log.Fatal(errorMsg)
		} else {
			email.parseError = errorMsg
			return email
		}
	}
	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to create mail reader: %v", err)
		if emailWrap.client.Options().StrictMailParsing() {
			log.Fatal(errorMessage, "\n")
		} else {
			email.parseError = errorMessage
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
			if emailWrap.client.Options().StrictMailParsing() {
				log.Fatal("failed to parse next part ", err)
			} else {

				email.parseError = err.Error()
			}
		}
		// sometime part is nil, not sure why, we'll consider that an error
		if part == nil {
			if emailWrap.client.Options().StrictMailParsing() {
				log.Fatal("part is nil")
			} else {
				email.parseError = "received an empty message part from the mail parser"
				break
			}
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			// can be plain-text , HTML, or inline attachments
			contentType, params, err := h.ContentType()
			if err != nil {
				if emailWrap.client.Options().StrictMailParsing() {
					log.Fatal("failed to get content type", err)
				} else {
					email.parseError = err.Error()
					break
				}
			}
			content, contentErr := io.ReadAll(part.Body)
			if contentErr != nil {
				if emailWrap.client.Options().StrictMailParsing() {
					log.Fatal("failed to read body", err)
				} else {
					email.parseError = contentErr.Error()
				}
			}
			isHtml := strings.HasPrefix(contentType, "text/html")
			isText := strings.HasPrefix(contentType, "text/plain")

			if isHtml || isText {
				if isHtml && contentErr == nil {
					email.hTMLContent = string(content)
				}
				if isText && contentErr == nil {
					email.textContent = string(content)
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
				email.attachments = append(email.attachments, attachment)

			} else {
				email.parseWarning = fmt.Sprintf("unknown inline content type: %v\n", contentType)
				break
			}
		case *mail.AttachmentHeader:
			attachment := models.AttachmentMetaData{}

			contentType, _, err := h.ContentType()
			if err != nil {
				if emailWrap.client.Options().StrictMailParsing() {
					log.Fatal("failed to get content type", err)
				} else {
					email.parseError = err.Error()
					break
				}
			}

			content, contentErr := io.ReadAll(part.Body)
			if contentErr != nil && emailWrap.client.Options().StrictMailParsing() {
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
			email.attachments = append(email.attachments, attachment)
		}

	}
	return email
}

func (e *Email) Mailbox() string {
	return e.mailbox
}

func (e *Email) ParseWarning() string {
	return e.parseWarning
}

func (e *Email) ParseError() string {
	return e.parseError
}

func (e *Email) OurID() string {
	return e.ourId
}

func (e *Email) Envelope() *imap.Envelope {
	return e.envelope
}

func (e *Email) Flags() []string {
	return e.flags
}

func (e *Email) UID() uint32 {
	return e.uID
}

func (e *Email) TextContent() string {
	return e.textContent
}

func (e *Email) HTMLContent() string {
	return e.hTMLContent
}

func (e *Email) Attachments() []models.AttachmentMetaData {
	return e.attachments
}
