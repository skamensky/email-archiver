package mailbox

import (
	"fmt"
	"github.com/emersion/go-imap"
	_ "github.com/mattn/go-sqlite3"
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
	client        models.Client
	name          string
	mailboxRecord models.MailboxRecord
	attributes    utils.Set[string]
}

func New(mailboxStatus *imap.MailboxStatus, mailboxInfo *imap.MailboxInfo) models.Mailbox {
	return &Mailbox{
		name: mailboxStatus.Name,
		mailboxRecord: models.MailboxRecord{
			Name:       mailboxStatus.Name,
			Attributes: mailboxInfo.Attributes,
		},
		attributes: utils.NewSet(mailboxInfo.Attributes),
	}
}

func (mailboxWrap *Mailbox) HasAttribute(attribute string) bool {
	return mailboxWrap.attributes.Contains(attribute)
}

func (mailboxWrap *Mailbox) MailboxRecord() models.MailboxRecord {
	return mailboxWrap.mailboxRecord
}

func (mailboxWrap *Mailbox) SetClient(client models.Client) {
	mailboxWrap.client = client
}

func (mailboxWrap *Mailbox) SetMailboxRecord(mailboxRecord models.MailboxRecord) {
	mailboxWrap.mailboxRecord = mailboxRecord
}

func (mailboxWrap *Mailbox) Client() models.Client {
	return mailboxWrap.client
}

func (mailboxWrap *Mailbox) Name() string {
	return mailboxWrap.name
}

/*
assumes correct mailbox is selected
*/
func (mailboxWrap *Mailbox) SyncToLocalState() error {

	allUids, err := mailboxWrap.Client().ListAllUids(mailboxWrap)

	if err != nil {
		return utils.JoinErrors("could not list all uids", err)
	}
	db := database.GetDatabase()
	return utils.JoinErrors("could not sync to local state", db.UpdateLocalMailboxState(mailboxWrap, allUids))

}

func (mailboxWrap *Mailbox) addMailboxEvent(eventType models.MailboxEvent) {
	eventType.Mailbox = mailboxWrap.Name()
	mailboxWrap.Client().Statuses() <- eventType
}

// mailbox should be selected
// relies on the mailbox being synced to local state
// caller should have already run:
// mailboxWrap.SyncToLocalState()
func (mailboxWrap *Mailbox) DownloadEmails() error {

	// sanity check that the correct mailbox is selected.
	//This is the result of a nasty bug which cause downloading emails and associating them with the wrong mailbox
	if mailboxWrap.Client().CurrentMailbox().Name() != mailboxWrap.Name() {
		return fmt.Errorf("Attempted to download emails from mailbox %s, but mailbox %s is selected", mailboxWrap.Name(), mailboxWrap.Client().CurrentMailbox().Name())
	}

	mailboxWrap.addMailboxEvent(
		models.MailboxEvent{
			EventType: models.MailboxDownloadStarted,
		})

	doneChan := make(chan error, 1)
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
		// mark as synced
		err = database.GetDatabase().SaveMailboxRecord(mailboxWrap.mailboxRecord)
		return utils.JoinErrors("failed to set next uid", err)
	}

	emails := []models.Email{}
	messages := make(chan *imap.Message)

	go func() {

		items := []imap.FetchItem{
			models.SectionToFetch.FetchItem(),
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchUid,
		}
		// NOTE: we used to use uidValidity+nextUID. But relying on the uidValidity does not get us moved emails and I've seen other issues with it being unreliable
		// so we just fetch all messages and then compare to what we have locally
		doneChan <- mailboxWrap.Client().UidFetch(uidsToFetch, items, messages)

	}()

	messagesProcessed := 0
	for msg := range messages {
		messagesProcessed++

		mailboxWrap.addMailboxEvent(
			models.MailboxEvent{
				EventType:       models.MailboxDownloadProgress,
				TotalDownloaded: messagesProcessed,
				TotalToDownload: len(uidsToFetch),
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
	if err := database.GetDatabase().AddEmails(mailboxWrap.Name(), emails); err != nil {
		return utils.JoinErrors("failed to add to db", err)
	}

	err = database.GetDatabase().SaveMailboxRecord(mailboxWrap.mailboxRecord)
	if err != nil {
		return utils.JoinErrors("failed to set next uid", err)
	}

	if messagesProcessed < len(uidsToFetch) {
		// I haven't gotten to the bottom of why this happens.
		mailboxWrap.addMailboxEvent(
			models.MailboxEvent{
				EventType:       models.MailboxSyncWarning,
				Warning:         fmt.Sprintf("tried to fetch %d messages but only got %d", len(uidsToFetch), messagesProcessed),
				TotalDownloaded: messagesProcessed,
				TotalToDownload: len(uidsToFetch),
			},
		)
	}

	mailboxWrap.addMailboxEvent(
		models.MailboxEvent{
			EventType:       models.MailboxDownloadCompleted,
			TotalDownloaded: messagesProcessed,
			TotalToDownload: len(uidsToFetch),
		})

	return nil
}
