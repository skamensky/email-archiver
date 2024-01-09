package main

import (
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/mapstructure"
)

type Disposition string

const (
	DispositionAttachment Disposition = "attachment"
	DispositionInline     Disposition = "inline"
	DispositionUnknown    Disposition = ""
)

type AttachmentMetaData struct {
	FileName    string
	FileType    string
	FileSubType string
	FileSize    int
	Encoding    string
	Disposition Disposition
}

type Email struct {
	Mailbox      string
	ParseWarning string
	ParseError   string
	OurID        string
	Envelope     *imap.Envelope
	Flags        []string
	UID          uint32
	TextContent  string
	HTMLContent  string
	Attachments  []AttachmentMetaData
}

type Options struct {
	Email             string
	Password          string
	ImapServer        string
	StrictMailParsing bool
	// WARNING: setting DEBUG to true creates a huge debug.txt file
	Debug         bool
	SkipMailboxes []string
}

type Client struct {
	*client.Client
	Options *Options
}

func joinErrors(message string, err error) error {
	return errors.Join(errors.New(message), err)
}

func setup() *Client {
	// from wiki: The imap.CharsetReader variable can be set by end users to parse charsets other than us-ascii and utf-8.
	// For instance, go-message's charset.Reader (which supports all common encodings) can be used:
	imap.CharsetReader = charset.Reader

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	options, err := setupOptions()
	if err != nil {
		log.Fatal("failed to setup options", err)
	}

	imapClient, err := setupImapClient(options)

	if err != nil {
		log.Fatal("failed to setup imap client", err)
	}

	err = initDB()
	if err != nil {
		log.Fatal("failed to init db", err)
	}

	return imapClient

}

func setupOptions() (*Options, error) {
	options := &Options{}
	for _, enivronVal := range os.Environ() {
		kv := strings.Split(enivronVal, "=")
		if len(kv) != 2 {
			continue
		}
		key := kv[0]
		value := kv[1]
		switch strings.ToUpper(key) {
		case "EMAIL":
			options.Email = value
		case "PASSWORD":
			options.Password = value
		case "IMAP_SERVER":
			options.ImapServer = value
			if !strings.Contains(value, ":") {
				options.ImapServer = value + ":993"
			}
		case "STRICT_MAIL_PARSING":
			options.StrictMailParsing, _ = strconv.ParseBool(value)
		case "DEBUG":
			options.Debug, _ = strconv.ParseBool(value)
		case "SKIP_MAILBOXES":
			// we use a % as a delimiter because it's not a valid character in most iamp server setup
			options.SkipMailboxes = strings.Split(value, "%")
		}
	}

	if options.Email == "" {
		return nil, errors.New("missing EMAIL")
	}
	if options.Password == "" {
		return nil, errors.New("missing PASSWORD")
	}
	if options.ImapServer == "" {
		return nil, errors.New("missing IMAP_SERVER")
	}

	return options, nil
}

func setupImapClient(options *Options) (*Client, error) {
	imapClient, err := client.DialTLS(options.ImapServer, &tls.Config{})

	if err != nil {
		log.Fatal("failed to dial imap server", err)
	}

	if options.Debug {
		debugFile := "debug.txt"
		debugFileHandle, err := os.OpenFile(debugFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		os.Truncate(debugFile, 0)
		if err != nil {
			log.Fatal(err)
		}
		imapClient.SetDebug(debugFileHandle)
	}
	if err := imapClient.Login(options.Email, options.Password); err != nil {
		log.Fatal("failed to login", err)
	}
	return &Client{
		imapClient,
		options,
	}, nil

}

func downloadEachMailbox(imapClient *Client) error {
	mailboxes := make(chan *imap.MailboxInfo)
	doneMailboxList := make(chan error, 1)
	go func() {
		doneMailboxList <- imapClient.List("", "*", mailboxes)
	}()

	mailboxNames := []string{}
	// we need to close the mailbox channel before we can select a mailbox, so we extract the mailbox names first
	for m := range mailboxes {
		toUse := true
		for _, v := range m.Attributes {
			if v == "\\Noselect" {
				toUse = false
			}
		}
		for _, v := range imapClient.Options.SkipMailboxes {
			if v == m.Name {
				toUse = false
			}
		}
		if toUse {
			mailboxNames = append(mailboxNames, m.Name)
		}
	}

	if err := <-doneMailboxList; err != nil {
		return joinErrors("failed to list mailboxes", err)
	}

	for _, m := range mailboxNames {
		err := downloadEmailsFromInbox(imapClient, m)
		if err != nil {
			return joinErrors("failed to download emails from inbox", err)
		}
	}
	return nil
}

func main() {

	imapClient := setup()
	defer imapClient.Logout()

	if err := downloadEachMailbox(imapClient); err != nil {
		log.Fatal("failed to download each mailbox", err)
	}
	if err := aggregateFolders(); err != nil {
		log.Fatal("failed to aggregate folders", err)
	}
	fmt.Println("Downloaded all emails")

}

func downloadEmailsFromInbox(imapClient *Client, mailBoxName string) error {
	emails := []Email{}

	// read only ensures that upon fetching the email is not marked as read
	readOnlyMailbox, err := imapClient.Select(mailBoxName, true)
	if err != nil {
		return joinErrors("failed to select mailbox", err)
	}
	seqset := new(imap.SeqSet)

	startingUID, err := getNextUID(mailBoxName)
	if err != nil {
		return joinErrors("failed to get next uid", err)
	}

	if startingUID == readOnlyMailbox.UidNext {
		fmt.Printf("Already caught up with %s mailbox\n", mailBoxName)
		return nil
	}

	numberOfEmailsToFetch := readOnlyMailbox.UidNext - startingUID
	seqset.AddRange(startingUID, readOnlyMailbox.UidNext-1)
	messages := make(chan *imap.Message)
	done := make(chan error, 1)
	section := &imap.BodySectionName{}

	go func() {
		items := []imap.FetchItem{
			section.FetchItem(),
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
		}

		done <- imapClient.Fetch(seqset, items, messages)
	}()

	fmt.Println()
	bar := progressbar.Default(int64(numberOfEmailsToFetch), "Downloading emails from "+mailBoxName+" mailbox")

	for msg := range messages {
		bar.Add(1)

		// our id is a hash because message-id isn't reliable
		hashSources := []string{}

		email := Email{
			Flags:    msg.Flags,
			UID:      msg.Uid,
			Envelope: msg.Envelope,
			Mailbox:  mailBoxName,
		}

		if !isInterfaceNil(email.Envelope) {
			if !isInterfaceNil(email.Envelope.Date) {
				hashSources = append(hashSources, email.Envelope.Date.String())
			}
			if !isInterfaceNil(email.Envelope.Subject) {
				hashSources = append(hashSources, email.Envelope.Subject)
			}
			if !isInterfaceNil(email.Envelope.From) {
				hashSources = append(hashSources, MustJSON(email.Envelope.From))
			}
			if !isInterfaceNil(email.Envelope.To) {
				hashSources = append(hashSources, MustJSON(email.Envelope.To))
			}
			if !isInterfaceNil(email.Envelope.Cc) {
				hashSources = append(hashSources, MustJSON(email.Envelope.Cc))
			}
			if !isInterfaceNil(email.Envelope.Bcc) {
				hashSources = append(hashSources, MustJSON(email.Envelope.Bcc))
			}
			if !isInterfaceNil(email.Envelope.ReplyTo) {
				hashSources = append(hashSources, MustJSON(email.Envelope.ReplyTo))
			}
			if !isInterfaceNil(email.Envelope.InReplyTo) {
				hashSources = append(hashSources, email.Envelope.InReplyTo)
			}
			if !isInterfaceNil(email.Envelope.MessageId) {
				hashSources = append(hashSources, email.Envelope.MessageId)
			}
		} else {
			// not much else we can do
			email.OurID = "nil-envelope;uid=" + strconv.Itoa(int(email.UID))
		}

		// NOTE: if needed in the future we can also hash the body, but I haven't seen any collisions yet,
		// 		 so it seems overkill
		hasher := sha256.New()
		hasher.Write([]byte(strings.Join(hashSources, "")))
		email.OurID = hex.EncodeToString(hasher.Sum(nil))

		r := msg.GetBody(section)
		if r == nil {
			errorMsg := "Server didn't returned message body"
			if imapClient.Options.StrictMailParsing {
				log.Fatal(errorMsg)
			} else {
				email.ParseError = errorMsg
				emails = append(emails, email)
				continue
			}
		}
		// Create a new mail reader
		mr, err := mail.CreateReader(r)
		if err != nil {
			errorMessage := fmt.Sprintf("failed to create mail reader: %v", err)
			if imapClient.Options.StrictMailParsing {
				log.Fatal(errorMessage, "\n")
			} else {
				email.ParseError = errorMessage
				emails = append(emails, email)
				continue
			}
		}

		for {

			// optimistic parsing, consume as much as possible
			// if we hit an error, we'll just set the parse error and move on, unless we're in strict mode
			// we only save the last error we hit
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				if imapClient.Options.StrictMailParsing {
					log.Fatal("failed to parse next part ", err)
				} else {

					email.ParseError = err.Error()
				}
			}
			// sometime part is nil, not sure why, we'll consider that an error
			if part == nil {
				if imapClient.Options.StrictMailParsing {
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
					if imapClient.Options.StrictMailParsing {
						log.Fatal("failed to get content type", err)
					} else {
						email.ParseError = err.Error()
						break
					}
				}
				content, contentErr := io.ReadAll(part.Body)
				if contentErr != nil {
					if imapClient.Options.StrictMailParsing {
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

					attachment := AttachmentMetaData{}
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

					attachment.Disposition = DispositionInline
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
				attachment := AttachmentMetaData{}

				contentType, _, err := h.ContentType()
				if err != nil {
					if imapClient.Options.StrictMailParsing {
						log.Fatal("failed to get content type", err)
					} else {
						email.ParseError = err.Error()
						break
					}
				}

				content, contentErr := io.ReadAll(part.Body)
				if contentErr != nil && imapClient.Options.StrictMailParsing {
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
				attachment.Disposition = DispositionAttachment
				email.Attachments = append(email.Attachments, attachment)

			}

		}
		emails = append(emails, email)
	}
	if err := <-done; err != nil {
		return joinErrors("failed to fetch mail", err)
	}

	if err := addToDB(emails); err != nil {
		return joinErrors("failed to add emails to db", err)
	}

	if err := setNextUID(mailBoxName, readOnlyMailbox.UidNext); err != nil {
		return joinErrors("failed to set next uid", err)
	}

	return nil
}

func initDB() error {
	_, err := os.Stat("./data.db")
	if err == nil {
		fmt.Printf("data.db already exists, skipping init\n")
		return nil
	}

	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		return joinErrors("failed to open db", err)
	}
	defer db.Close()
	_, err = db.Exec("DROP TABLE IF EXISTS email")
	if err != nil {
		return joinErrors("failed to drop table emails", err)
	}
	_, err = db.Exec("CREATE TABLE email (parse_warning text, parse_error text,our_id text primary key, envelope text, flags text, folders text, uid int, text_content text, html_content text, attachments text)")
	if err != nil {
		return joinErrors("failed to create table email", err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS message_to_folder")
	if err != nil {
		return joinErrors("failed to drop table folders", err)
	}

	_, err = db.Exec("CREATE TABLE message_to_folder (folder_name text, email_id text, primary key (folder_name, email_id))")
	if err != nil {
		return joinErrors("failed to create message_to_folder table", err)
	}
	// index on email_id so our updates are faster
	_, err = db.Exec("CREATE INDEX email_id_index ON message_to_folder (email_id)")
	if err != nil {
		return joinErrors("failed to create email_id_index index", err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS folder")
	if err != nil {
		return joinErrors("failed to drop table folders", err)
	}
	_, err = db.Exec("CREATE TABLE folder (name text primary key,uid_next int)")
	if err != nil {
		return joinErrors("failed to create folder table", err)
	}

	return nil
}

func setNextUID(mailboxName string, nextUID uint32) error {
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		return joinErrors("failed to open db", err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO folder (name, uid_next) VALUES (?, ?) ON CONFLICT(name) DO UPDATE SET uid_next = ?", mailboxName, nextUID, nextUID)
	if err != nil {
		return joinErrors("failed to set next uid", err)
	}
	return nil
}

func getNextUID(mailboxName string) (uint32, error) {
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		return 0, joinErrors("failed to open db", err)
	}
	defer db.Close()

	var uid uint32
	err = db.QueryRow("SELECT uid_next FROM folder WHERE name = ?", mailboxName).Scan(&uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return 1, nil
		}
		return 0, joinErrors("failed to get next uid", err)
	}
	return uid, nil
}

func addToDB(emails []Email) error {
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		return joinErrors("failed to open db", err)
	}
	defer db.Close()

	insertEmailStmnt, err := db.Prepare(`
		INSERT INTO email (parse_warning, parse_error, our_id, envelope, flags, uid, text_content, html_content, attachments)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (our_id) DO NOTHING
	`)

	if err != nil {
		return joinErrors("failed to prepare insert statement", err)
	}
	defer insertEmailStmnt.Close()

	insertFolderStmnt, err := db.Prepare(`
		INSERT INTO message_to_folder (folder_name, email_id)
		VALUES (?, ?)
		ON CONFLICT (folder_name, email_id) DO NOTHING
		`)

	if err != nil {
		return joinErrors("failed to prepare insert statement", err)
	}

	// todo don't insert in a loop
	for _, email := range emails {
		_, err = insertEmailStmnt.Exec(email.ParseWarning, email.ParseError, email.OurID, MustJSON(email.Envelope), MustJSON(email.Flags), email.UID, email.TextContent, email.HTMLContent, MustJSON(email.Attachments))
		if err != nil {
			return joinErrors("failed to insert email", err)
		}
		_, err = insertFolderStmnt.Exec(email.Mailbox, email.OurID)
		if err != nil {
			return joinErrors("failed to insert folder", err)
		}
	}

	return nil
}

func aggregateFolders() error {

	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		return joinErrors("failed to open db", err)
	}

	// update emails table, folders will be a json list of folders.
	_, err = db.Exec(`
			update email
			set folders = subtable.folders
			FROM
			(
				select json_group_array(folder_name) folders, email_id
				FROM message_to_folder
				GROUP BY email_id
			) subtable
			WHERE email.our_id = subtable.email_id;
	`)
	if err != nil {
		return joinErrors("failed to update email table", err)
	}

	return nil

}

func MustJSON(i interface{}) string {
	if reflect.TypeOf(i).Kind() == reflect.Struct {
		var mapInterface map[string]interface{}
		err := mapstructure.Decode(i, &mapInterface)
		if err != nil {
			log.Fatal("failed to decode interface to map ", err)
		}

		json, err := json.Marshal(mapInterface)
		if err != nil {
			log.Fatal("failed to marshal map to json", err)
		}
		return string(json)
	} else {
		json, err := json.Marshal(i)
		if err != nil {
			log.Fatal("failed to marshal interface to json", err)
		}
		return string(json)
	}
}

func isInterfaceNil(i interface{}) bool {
	return reflect.DeepEqual(i, reflect.Zero(reflect.TypeOf(i)).Interface())
}
