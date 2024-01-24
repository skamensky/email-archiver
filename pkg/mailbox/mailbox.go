package mailbox

import (
	"fmt"
	"github.com/emersion/go-imap"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
	"github.com/skamensky/email-archiver/pkg/database"
	"github.com/skamensky/email-archiver/pkg/email"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"strings"
)

type Disposition string

const (
	DispositionAttachment Disposition = "attachment"
	DispositionInline     Disposition = "inline"
	DispositionUnknown    Disposition = ""
)

type Mailbox struct {
	client      models.Client
	name        string
	uidValidity uint32
	uidNext     uint32
	attributes  utils.Set[string]
}

func New(mailboxStatus *imap.MailboxStatus, mailboxInfo *imap.MailboxInfo) models.Mailbox {
	return &Mailbox{
		name:        mailboxStatus.Name,
		uidValidity: mailboxStatus.UidValidity,
		uidNext:     mailboxStatus.UidNext,
		attributes:  utils.NewSet(mailboxInfo.Attributes),
	}
}

func (mailboxWrap *Mailbox) HasAttribute(attribute string) bool {
	return mailboxWrap.attributes.Contains(attribute)
}

func (mailboxWrap *Mailbox) MailboxRecord() models.MailboxRecord {

	return models.MailboxRecord{
		Name:        mailboxWrap.name,
		UIDValidity: mailboxWrap.uidValidity,
		UIDNext:     mailboxWrap.uidNext,
	}
}

func (mailboxWrap *Mailbox) SetClient(client models.Client) {
	mailboxWrap.client = client
}

func (mailboxWrap *Mailbox) SetMailboxRecord(mailboxRecord models.MailboxRecord) {
	mailboxWrap.uidValidity = mailboxRecord.UIDValidity
	mailboxWrap.uidNext = mailboxRecord.UIDNext
}

func (mailboxWrap *Mailbox) Client() models.Client {
	return mailboxWrap.client
}

func (mailboxWrap *Mailbox) Name() string {
	return mailboxWrap.name
}

func (mailboxWrap *Mailbox) SyncUidValidity() error {
	// TODO: implement
	return nil
}

/*
assumes correct mailbox is selected
*/
func (mailboxWrap *Mailbox) SyncToLocalState() error {

	uidsRemote, err := mailboxWrap.Client().ListAllUids(mailboxWrap)
	if err != nil {
		return utils.JoinErrors("could not list all uids", err)
	}
	db := database.GetDatabase()
	return utils.JoinErrors("could not sync to local state", db.UpdateLocalMailboxState(mailboxWrap, uidsRemote))

}

func (mailboxWrap *Mailbox) addMailboxEvent(eventType models.MailboxEvent) {
	eventType.Mailbox = mailboxWrap.Name()
	mailboxWrap.Client().Statuses() <- eventType
}

func (mailboxWrap *Mailbox) DownloadEmails() error {
	// relies on the mailbox being synced to local state

	mailboxWrap.addMailboxEvent(
		models.MailboxEvent{
			EventType: models.MailboxDownloadStarted,
		})

	doneChan := make(chan error, 1)
	utils.DebugPrintln("syncing to local state")
	err := mailboxWrap.SyncToLocalState()
	if err != nil {
		return utils.JoinErrors("could not sync to local state", err)
	}

	utils.DebugPrintln("getting messages pending sync")
	uidsToFetch, err := database.GetDatabase().GetMessagesPendingSync(mailboxWrap)

	if err != nil {
		return utils.JoinErrors("could not get messages pending sync", err)
	}

	if len(uidsToFetch) == 0 {
		mailboxWrap.addMailboxEvent(
			models.MailboxEvent{
				EventType:       models.MailboxDownloadCompleted,
				TotalDownloaded: 0,
				TotalToDownload: 0,
			})
		return nil
	}

	emails := []models.Email{}
	messages := make(chan *imap.Message)

	utils.DebugPrintln("fetching messages")
	go func() {
		seqset := new(imap.SeqSet)
		// NOTE: we used to use uidValidity+nextUID. But relying on the uidValidity does not get us moved emails and I've seen other issues with it being unreliable
		// so we just fetch all messages and then compare to what we have locally
		seqset.AddNum(uidsToFetch...)

		items := []imap.FetchItem{
			models.SectionToFetch.FetchItem(),
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
		}
		doneChan <- mailboxWrap.Client().Fetch(seqset, items, messages)
	}()

	utils.DebugPrintln("parsing messages")
	mailboxWrap.addMailboxEvent(
		models.MailboxEvent{
			EventType:       models.MailboxDownloadStarted,
			TotalToDownload: len(uidsToFetch),
		})

	fmt.Println()

	bar := progressbar.Default(int64(len(uidsToFetch)), "Downloading emails from "+mailboxWrap.Name()+" mailbox")
	messagesProcessed := 0
	for msg := range messages {
		messagesProcessed++
		bar.Add(1)

		mailboxWrap.addMailboxEvent(
			models.MailboxEvent{
				EventType:       models.MailboxDownloadProgress,
				TotalDownloaded: messagesProcessed,
			})

		emailParsed := email.New(msg, mailboxWrap.Client())

		if emailParsed.GetParseWarning() != "" || emailParsed.GetParseError() != "" {
			warnings := []string{}
			if emailParsed.GetParseWarning() != "" {
				warnings = append(warnings, "parse warning: "+emailParsed.GetParseWarning())
			}
			if emailParsed.GetParseError() != "" {
				warnings = append(warnings, "parse error: "+emailParsed.GetParseError())
			}
			warning := strings.Join(warnings, ", ")
			mailboxWrap.addMailboxEvent(
				models.MailboxEvent{
					EventType: models.MailboxSyncWarning,
					Warning:   warning,
				})
		}

		emails = append(emails, emailParsed)
	}

	if err := <-doneChan; err != nil {
		return utils.JoinErrors("failed to fetch", err)
	}

	err = database.GetDatabase().SetNextUID(mailboxWrap, mailboxWrap.uidNext, mailboxWrap.uidValidity)
	if err != nil {
		return utils.JoinErrors("failed to set next uid", err)
	}

	utils.DebugPrintln("adding to db")
	if err := database.GetDatabase().AddToDB(emails); err != nil {
		return utils.JoinErrors("failed to add to db", err)
	}

	utils.DebugPrintln("setting messages to synced in db")
	err = database.GetDatabase().SetMessagesToSynced(mailboxWrap, uidsToFetch)
	if err != nil {
		return utils.JoinErrors("failed to set messages to synced", err)
	}

	mailboxWrap.addMailboxEvent(
		models.MailboxEvent{
			EventType:       models.MailboxDownloadCompleted,
			TotalDownloaded: messagesProcessed,
			TotalToDownload: len(uidsToFetch),
		})

	return nil
}
