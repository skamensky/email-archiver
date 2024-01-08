package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/textproto"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/mapstructure"
	"github.com/sony/sonyflake"
)

const januaryFirst2024 = 1704060000

var sf *sonyflake.Sonyflake

func nextSonyflake() uint64 {
	if id, err := sf.NextID(); err != nil {
		panic(err)
	} else {
		return id
	}
}

func initIdGenerator() error {
	settings := sonyflake.Settings{
		StartTime: time.Unix(januaryFirst2024, 0),
	}
	settings.MachineID = func() (uint16, error) {
		// if this ever gets scaled to multiple machines, this should be set externally
		return 1, nil
	}
	sf = sonyflake.NewSonyflake(settings)
	_, err := sf.NextID()
	return err
}

type Disposition string

const (
	DispositionAttachment Disposition = "attachment"
	DispositionInline     Disposition = "inline"
	DispositionUnknown    Disposition = ""
)

type AttachmentMetaData struct {
	EmailUID    uint64
	FileName    string
	FileType    string
	FileSubType string
	FileSize    uint32
	Encoding    string
	Disposition Disposition
}

type Email struct {
	OurID       uint64
	Envelope    imap.Envelope
	Flags       []string
	Labels      []string
	UID         uint32
	TextContent string
	HTMLContent string
	Attachments []AttachmentMetaData
}

// Envelope
// Date      time.Time
// Subject   string
// From      []Address
// Sender    []Address
// ReplyTo   []Address
// To        []Address
// Cc        []Address
// Bcc       []Address
// InReplyTo string
// MessageID string

type Data struct {
	UID           string
	Path          string
	BodyStructure string
	Subject       string
}

func main() {
	err := initIdGenerator()
	if err != nil {
		log.Fatalf("failed to init id generator: %v", err)
	}

	password := ""
	email := "shmuelkamensky@gmail.com"
	imapServer := "imap.gmail.com:993"

	// create a new file:
	f, err := os.Create("log.txt")
	f.Truncate(0)

	c, err := imapclient.DialTLS(imapServer, &imapclient.Options{
		DebugWriter: f,
	})
	if err != nil {
		log.Fatalf("failed to dial IMAP server: %v", err)
	}
	defer c.Close()

	loginCommand := c.Login(email, password)
	err = loginCommand.Wait()
	if err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	mailboxes, err := c.List("", "%", nil).Collect()
	if err != nil {
		log.Fatalf("failed to list mailboxes: %v", err)
	}
	log.Printf("Found %v mailboxes", len(mailboxes))
	for _, mbox := range mailboxes {
		log.Printf(" - %v", mbox.Mailbox)
	}

	selectedMbox, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		log.Fatalf("failed to select INBOX: %v", err)
	}
	log.Printf("INBOX contains %v messages", selectedMbox.NumMessages)

	numMessagesToFetch := uint32(100)
	if selectedMbox.NumMessages > 0 {
		seqSet := new(imap.SeqSet)
		seqSet.AddRange(selectedMbox.NumMessages-numMessagesToFetch-1, selectedMbox.NumMessages)

		fetchOptions := &imap.FetchOptions{
			Flags:    true,
			UID:      true,
			Envelope: true,
			BodyStructure: &imap.FetchItemBodyStructure{
				Extended: true,
			},
			BodySection: []*imap.FetchItemBodySection{
				{
					Specifier: imap.PartSpecifierHeader,
				},
				{
					Specifier: imap.PartSpecifierText,
				},
			},
		}
		emails := []Email{}
		messages, err := c.Fetch(*seqSet, fetchOptions).Collect()
		if err != nil {
			log.Fatalf("failed to fetch messages: %v", err)
		}

		sequenceSet := new(imap.SeqSet)

		//open new sqlite db
		db, err := sql.Open("sqlite3", "./data.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		// drop table if exists
		_, err = db.Exec("DROP TABLE IF EXISTS data")
		if err != nil {
			log.Fatal(err)
		}
		// create table
		_, err = db.Exec("CREATE TABLE data (our_id int, emailData TEXT, subject TEXT)")
		if err != nil {
			log.Fatal(err)
		}
		// prepare statement
		stmt, err := db.Prepare("INSERT INTO data(our_id, emailData, subject) values(?,?,?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		fmt.Println("messages: ", len(messages))
		for _, msg := range messages {

			sequenceSet.AddNum(msg.SeqNum)
			flags := []string{}
			for _, flag := range msg.Flags {
				flags = append(flags, string(flag))
			}
			email := Email{
				OurID:    nextSonyflake(),
				Envelope: *msg.Envelope,
				Flags:    flags,
				UID:      msg.UID,
			}

			textualNodes := [][]int{}

			msg.BodyStructure.Walk(func(path []int, bs imap.BodyStructure) (walkChildren bool) {

				_, multiOk := bs.(*imap.BodyStructureMultiPart)
				if multiOk {
					// children will be walked automatically, no need to address container directly
					return true
				}

				part, ok := bs.(*imap.BodyStructureSinglePart)

				if !ok {
					return true
				}

				if strings.HasPrefix(strings.ToLower(part.MediaType()), "text/") {
					// handled elsewhere
					textualNodes = append(textualNodes, path)
					return true
				}

				attachment := extractMetadata(part)

				if !structIsEmpty(attachment) {
					attachment.EmailUID = email.OurID
					email.Attachments = append(email.Attachments, attachment)
				}
				return true
			})

			for _, path := range textualNodes {
				fmt.Println("text node path: ", path)
			}
			count := 0
			for section, byteArray := range msg.BodySection {
				reader := bytes.NewReader(byteArray)
				bufioReader := bufio.NewReader(reader)
				if section.Specifier == imap.PartSpecifierHeader {
					fmt.Println("Header is at ", count)
					headers, err := textproto.ReadHeader(bufioReader)
					if err != nil {
						log.Fatalf("failed to read header: %v", err)
					}
					contentType := headers.Get("Content-Type")
					fmt.Println("Content-Type: ", contentType)

				} else if section.Specifier == imap.PartSpecifierText {
					fmt.Println("Text is at ", count)
				} else {
					fmt.Println("section.Specifier: ", section.Specifier, " is at ", count)
				}
				count++

			}

			emails = append(emails, email)

		}
		fmt.Println("Inserting emails into db")

		for _, email := range emails {
			rawData := map[string]interface{}{}
			err = mapstructure.Decode(email, &rawData)
			if err != nil {
				log.Fatalf("failed to decode email: %v", err)
			}
			jsonData, err := json.MarshalIndent(rawData, "", "  ")
			if err != nil {
				log.Fatalf("failed to marshal email: %v", err)
			}
			stmt.Exec(email.OurID, string(jsonData), email.Envelope.Subject)
		}
		if err := c.Logout().Wait(); err != nil {
			log.Fatalf("failed to logout: %v", err)
		}

		log.Println("Done!")
	}
}

func extractMetadata(bs *imap.BodyStructureSinglePart) AttachmentMetaData {
	possibleFileNameParams := []string{"filename", "name", "FILENAME", "NAME"}
	bsInternal := *bs
	attachment := AttachmentMetaData{}

	if bsInternal.Disposition() != nil {
		params := bsInternal.Disposition().Params
		attachment.FileName = bs.Filename()
		attachment.FileSize = bs.Size
		attachment.Disposition = DispositionUnknown
		attachment.FileType = bsInternal.Type
		attachment.FileSubType = bsInternal.Subtype
		attachment.Encoding = bsInternal.Encoding

		// try and resolve filename. TODO add Extended support
		if attachment.FileName == "" {
			fileNameParam := ""
			for _, param := range possibleFileNameParams {
				if val, ok := params[param]; ok {
					fileNameParam = val
					break
				}
			}

			if fileNameParam != "" {
				attachment.FileName = bsInternal.Disposition().Params[fileNameParam]
			}
		}

		if attachment.FileName == "" {
			attachment.FileName = bsInternal.Description
		}

		if bsInternal.Disposition() != nil {
			dispositionLower := strings.ToLower(bsInternal.Disposition().Value)
			switch dispositionLower {
			case "attachment":
				attachment.Disposition = DispositionAttachment
			case "inline":
				attachment.Disposition = DispositionInline
			}
		}
	}
	return attachment
}

func structIsEmpty(i interface{}) bool {
	return i == nil || reflect.DeepEqual(i, reflect.Zero(reflect.TypeOf(i)).Interface())
}
