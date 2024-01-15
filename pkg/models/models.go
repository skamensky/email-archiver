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
	Name        string
	UIDValidity uint32
	UIDNext     uint32
}

type MailboxEventType int

const (
	MailboxDownloadStarted MailboxEventType = iota
	MailboxDownloadCompleted
	MailboxDownloadSkipped
	MailboxDownloadError
	MailboxDownloadProgress
	MailboxSyncWarning
)

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
	Mailbox() string
	ParseWarning() string
	ParseError() string
	OurID() string
	Envelope() *imap.Envelope
	Flags() []string
	UID() uint32
	TextContent() string
	HTMLContent() string
	Attachments() []AttachmentMetaData
}

type Mailbox interface {
	SyncUidValidity() error
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
	ImapServer() string
	Email() string
	Password() string
	StrictMailParsing() bool
	ImapClientDebug() bool
	Debug() bool
	LimitToMailboxes() []string
	SkipMailboxes() []string
	DBPath() string
	MaxPoolSize() int
}

type ClientPool interface {
	Get() (Client, error)
	Put(Client)
	ListMailboxes() ([]Mailbox, error)
	DownloadAllMailboxes() error
	SyncMailboxMessageStates() error
	Close()
}

type Client interface {
	Logout() error
	Statuses() chan<- MailboxEvent
	CurrentMailbox() Mailbox
	Options() Options
	Fetch(*imap.SeqSet, []imap.FetchItem, chan *imap.Message) error
	ListAllUids(Mailbox) ([]uint32, error)
	ListMailboxInfos() ([]*imap.MailboxInfo, error)
	CopyToMailbox(fromMailbox Mailbox, toMailbox Mailbox, uids []uint32) error
	MoveToMailbox(fromMailbox Mailbox, toMailbox Mailbox, uids []uint32) error
	DownloadMailbox(Mailbox) error
	LastPing() time.Time
	RawSelect(mailboxName string, readOnly bool) (*imap.MailboxStatus, error)
	Id() int
}

type DB interface {
	SetNextUID(mailbox Mailbox, nextUID uint32, uidValidity uint32) error
	GetNextUID(Mailbox) (MailboxRecord, error)
	AddToDB([]Email) error
	AggregateFolders() error
	UpdateLocalMailboxState(Mailbox, []uint32) error
	GetMessagesPendingSync(Mailbox) ([]uint32, error)
	SetMessagesToSynced(mailbox Mailbox, uids []uint32) error
}
