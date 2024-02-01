package models

import (
	"github.com/emersion/go-imap"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Disposition string

const (
	DispositionAttachment Disposition = "attachment"
	DispositionInline     Disposition = "inline"
	DispositionUnknown    Disposition = ""
)

type MailboxRecord struct {
	Name       string   `json:"name"`
	LastSynced int64    `json:"last_synced"`
	Attributes []string `json:"attributes"`
	NumEmails  int      `json:"num_emails"`
}

type MailboxEventType string

func (m MailboxEventType) TSName() string {
	return string(m)
}

const (
	MailboxSyncQueued        MailboxEventType = "MailboxSyncQueued"
	MailboxDownloadStarted   MailboxEventType = "MailboxDownloadStarted"
	MailboxDownloadCompleted MailboxEventType = "MailboxDownloadCompleted"
	MailboxDownloadSkipped   MailboxEventType = "MailboxDownloadSkipped"
	MailboxDownloadError     MailboxEventType = "MailboxDownloadError"
	MailboxDownloadProgress  MailboxEventType = "MailboxDownloadProgress"
	MailboxSyncWarning       MailboxEventType = "MailboxSyncWarning"
)

// unfortunately, this is needed for typescriptify. We must manually update this list to stay in sync with MailboxEventType
var AllEventTypes = []MailboxEventType{
	MailboxSyncQueued,
	MailboxDownloadStarted,
	MailboxDownloadCompleted,
	MailboxDownloadSkipped,
	MailboxDownloadError,
	MailboxDownloadProgress,
	MailboxSyncWarning,
}

// used by both email.go and mailbox.go, which led to a circular dependency.
// TODO: find a better place for this
var SectionToFetch = &imap.BodySectionName{}

// shared by options and utils which would cause circular dependency
const DEBUG_ENVIRONMENT_KEY = "DEBUG"

type MailboxEvent struct {
	Mailbox         string
	TotalToDownload int
	TotalDownloaded int
	Error           string
	Warning         string
	EventType       MailboxEventType
}
type MailboxDiff struct {
	NewEmails     []uint32
	DeletedEmails []uint32
}

type AttachmentMetaData struct {
	FileName    string
	FileType    string
	FileSubType string
	FileSize    int
	Encoding    string
	Disposition Disposition
}

type Email interface {
	GetParseWarning() string
	GetParseError() string
	GetOurID() string
	GetEnvelope() *imap.Envelope
	GetFlags() []string
	GetUID() uint32
	GetTextContent() string
	GetHTMLContent() string
	GetAttachments() []AttachmentMetaData
	GetMessageId() string
	GetDate() string
	GetSubject() string
	GetFromName1() string
	GetFromMailbox1() string
	GetFromHost1() string
	GetSenderName1() string
	GetSenderMailbox1() string
	GetSenderHost1() string
	GetReplyToName1() string
	GetReplyToMailbox1() string
	GetReplyToHost1() string
	GetToName1() string
	GetToMailbox1() string
	GetToHost1() string
	GetCcName1() string
	GetCcMailbox1() string
	GetCcHost1() string
	GetBccName1() string
	GetBccMailbox1() string
	GetBccHost1() string
	GetInReplyTo() string
}

type Mailbox interface {
	DownloadEmails() error
	Name() string
	Client() Client
	SetClient(Client)
	SyncToLocalState() error
	HasAttribute(string) bool
	MailboxRecord() MailboxRecord
	SetMailboxRecord(MailboxRecord)
}

type Options interface {
	GetImapServer() string
	GetEmail() string
	GetPassword() string
	GetStrictMailParsing() bool
	GetImapClientDebug() bool
	GetDebug() bool
	GetLimitToMailboxes() []string
	GetSkipMailboxes() []string
	GetDBPath() string
	GetMaxPoolSize() int
}

type ClientPool interface {
	Get() (Client, error)
	Put(Client)
	ListMailboxes() ([]Mailbox, error)
	DownloadMailboxes([]Mailbox) error
	SyncMailboxMessageStates([]Mailbox) error
	Close()
	SetEventHandler(func(*MailboxEvent))
}

type Client interface {
	Logout() error
	Statuses() chan<- MailboxEvent
	CurrentMailbox() Mailbox
	Options() Options
	UidFetch([]uint32, []imap.FetchItem, chan *imap.Message) error
	ListAllUids(Mailbox) ([]uint32, error)
	ListMailboxInfos() ([]*imap.MailboxInfo, error)
	CopyToMailbox(fromMailbox Mailbox, toMailbox Mailbox, uids []uint32) error
	MoveToMailbox(fromMailbox Mailbox, toMailbox Mailbox, uids []uint32) error
	DownloadMailbox(Mailbox) error
	LastPing() time.Time
	RawSelect(mailboxName string, readOnly bool) (*imap.MailboxStatus, error)
	Select(mailboxName string, readOnly bool) error
	Id() int
}

type DB interface {
	SaveMailboxRecord(MailboxRecord) error
	GetAllMailboxRecords() ([]MailboxRecord, error)
	AddEmails(mailbox string, emails []Email) error
	AggregateFolders() error
	UpdateLocalMailboxState(Mailbox, []uint32) error
	GetMessagesPendingSync(Mailbox) ([]uint32, error)
	GetEmails(sqlQuery string, params ...interface{}) ([]Email, error)
	UpdateFTS() error
	FullTextSearch(string) ([]Email, error)
	SetFrontendState(string) error
	GetFrontendState() (string, error)
	// todo: allow for options to be set and retrieved in DB in addition to env vars
	//GetOptions() (Options, error)
	//SetOptions(Options) error

	// todo: allow for saving and loading queries
}
