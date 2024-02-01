package client

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap"
	goImapClient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Client struct {
	*goImapClient.Client
	parent         *ClientConnPool
	currentMailbox models.Mailbox
	lastPing       time.Time
	id             int
}

func newClient(ops models.Options, id int, clientPool *ClientConnPool) (*Client, error) {
	// from wiki: The imap.CharsetReader variable can be set by end users to parse charsets other than us-ascii and utf-8.
	// For instance, go-message's charset.Reader (which supports all common encodings) can be used:
	imap.CharsetReader = charset.Reader

	clientWrapper := &Client{
		id:     id,
		parent: clientPool,
	}

	imapClient, err := goImapClient.DialTLS(ops.GetImapServer(), &tls.Config{})

	if err != nil {
		return nil, utils.JoinErrors("failed to dial imap server", err)
	} else {
		clientWrapper.Client = imapClient
	}

	utils.DebugPrintln(fmt.Sprintf("client %d: connected to imap server", id))

	if ops.GetImapClientDebug() {
		debugDir := "imap_debug"
		if _, err := os.Stat(debugDir); os.IsNotExist(err) {
			err = os.Mkdir(debugDir, 0755)
			if err != nil {
				return nil, utils.JoinErrors("failed to create debug dir", err)
			}
		}

		debugFile := filepath.Join(debugDir, "client_"+strconv.Itoa(id)+".log")
		debugFileHandle, err := os.OpenFile(debugFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		err = utils.JoinErrors("unable to truncate debug file", os.Truncate(debugFile, 0))
		if err != nil {
			return nil, err
		}

		imapClient.SetDebug(debugFileHandle)
	}
	if err := imapClient.Login(ops.GetEmail(), ops.GetPassword()); err != nil {
		return nil, utils.JoinErrors("failed to login", err)
	}

	clientWrapper.lastPing = time.Now()
	return clientWrapper, nil
}

func (clientWrap *Client) ListMailboxInfos() ([]*imap.MailboxInfo, error) {
	mailboxInfoChan := make(chan *imap.MailboxInfo)
	mailboxInfos := []*imap.MailboxInfo{}

	doneMailboxList := make(chan error, 1)
	go func() {
		doneMailboxList <- clientWrap.List("", "*", mailboxInfoChan)
	}()

	for m := range mailboxInfoChan {
		// note, we cannot use the imapClient.Select() method here, as it will deadlock since iterating over mailboxes
		// locks an internal imap checkoutMut, and the imapClient.Select() method also locks the same checkoutMut
		mailboxInfos = append(mailboxInfos, m)
	}

	if err := <-doneMailboxList; err != nil {
		return nil, err
	}
	clientWrap.lastPing = time.Now()
	return mailboxInfos, nil
}

func (clientWrap *Client) DownloadMailbox(mBox models.Mailbox) error {
	_, err := clientWrap.Client.Select(mBox.Name(), true)
	if err != nil {
		return utils.JoinErrors("failed to select mailbox", err)
	}
	mBox.SetClient(clientWrap)
	err = mBox.DownloadEmails()
	if err != nil {
		clientWrap.Statuses() <- models.MailboxEvent{
			Mailbox:   mBox.Name(),
			EventType: models.MailboxDownloadError,
			Error:     err.Error(),
		}
		return utils.JoinErrors("failed to download emails from inbox", err)
	}
	clientWrap.lastPing = time.Now()
	return nil
}

func (clientWrap *Client) Logout() error {

	err := utils.JoinErrors("failed to logout", clientWrap.Client.Logout())
	var termError error
	if err != nil {
		termError = utils.JoinErrors("failed to terminate tcp connection", clientWrap.Client.Terminate())
		err = utils.JoinErrors(fmt.Sprintf("logout error: %v, tcp terminate error:", err), termError)
	}

	return err
}

/*
selects a mailbox, and sets the currentMailbox to the selected mailbox
read only ensures that upon fetching the email is not marked as read
*/
func (clientWrap *Client) Select(mailboxName string, readOnly bool) error {

	var mbox models.Mailbox
	if _, ok := clientWrap.parent.mailboxesCache[mailboxName]; !ok {
		// refresh the cache
		_, err := clientWrap.ListMailboxInfos()
		if err != nil {
			return utils.JoinErrors("could not list mailboxes", err)
		}
	}
	mbox = clientWrap.parent.mailboxesCache[mailboxName]
	_, err := clientWrap.Client.Select(mailboxName, readOnly)
	if err != nil {
		return utils.JoinErrors(fmt.Sprintf("could not select mailbox %v", mailboxName), err)
	}
	clientWrap.currentMailbox = mbox
	clientWrap.lastPing = time.Now()
	return nil
}

func (clientWrap *Client) RawSelect(mailboxName string, readOnly bool) (*imap.MailboxStatus, error) {
	status, err := clientWrap.Client.Select(mailboxName, readOnly)
	if err != nil {
		return nil, utils.JoinErrors(fmt.Sprintf("could not select mailbox %v", mailboxName), err)
	}
	clientWrap.lastPing = time.Now()
	return status, nil
}

func (clientWrap *Client) CurrentMailbox() models.Mailbox {
	return clientWrap.currentMailbox
}

func (clientWrap *Client) Statuses() chan<- models.MailboxEvent {
	return clientWrap.parent.statuses
}

func (clientWrap *Client) Options() models.Options {
	return clientWrap.parent.options
}

func (clientWrap *Client) UidFetch(uids []uint32, items []imap.FetchItem, ch chan *imap.Message) error {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	err := clientWrap.Client.UidFetch(seqset, items, ch)
	if err != nil {
		return utils.JoinErrors("failed to fetch", err)
	}
	clientWrap.lastPing = time.Now()
	return nil
}

func (clientWrap *Client) ListAllUids(mailbox models.Mailbox) ([]uint32, error) {
	err := clientWrap.Select(mailbox.Name(), true)
	if err != nil {
		return nil, err
	}

	uids, err := clientWrap.Client.UidSearch(imap.NewSearchCriteria())
	if err != nil {
		return nil, utils.JoinErrors("failed to search mailbox", err)
	}
	clientWrap.lastPing = time.Now()
	return uids, nil
}
func (clientWrap *Client) MoveToMailbox(fromMailbox models.Mailbox, toMailbox models.Mailbox, uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}
	err := clientWrap.Select(fromMailbox.Name(), true)
	if err != nil {
		return utils.JoinErrors("failed to select mailbox", err)
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	err = clientWrap.UidMove(seqset, toMailbox.Name())
	if err != nil {
		return utils.JoinErrors("failed to move emails", err)
	}
	clientWrap.lastPing = time.Now()
	return nil
}
func (clientWrap *Client) CopyToMailbox(fromMailbox models.Mailbox, toMailbox models.Mailbox, uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}
	err := clientWrap.Select(fromMailbox.Name(), true)
	if err != nil {
		return utils.JoinErrors("failed to select mailbox", err)
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	err = clientWrap.UidCopy(seqset, toMailbox.Name())
	if err != nil {
		return utils.JoinErrors("failed to copy emails", err)
	}
	clientWrap.lastPing = time.Now()
	return nil
}

func (clientWrap *Client) LastPing() time.Time {
	return clientWrap.lastPing
}

func (clientWrap *Client) Id() int {
	return clientWrap.id
}
