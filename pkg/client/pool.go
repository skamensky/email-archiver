package client

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/skamensky/email-archiver/pkg/database"
	"github.com/skamensky/email-archiver/pkg/mailbox"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"sync"
	"time"
)

type ClientConnPool struct {
	pool            chan models.Client
	poolMap         map[int]models.Client
	checkoutMut     sync.Mutex
	mailboxCacheMut sync.Mutex
	options         models.Options
	statusesHandler func(models.MailboxEvent)
	statuses        chan models.MailboxEvent
	mailboxesCache  map[string]models.Mailbox
	nextId          int
}

func NewClientConnPool(options models.Options, statusHandler func(models.MailboxEvent)) models.ClientPool {
	pool := &ClientConnPool{
		pool:           make(chan models.Client, options.MaxPoolSize()),
		options:        options,
		mailboxesCache: make(map[string]models.Mailbox),
		statuses:       make(chan models.MailboxEvent),
		nextId:         1,
		poolMap:        make(map[int]models.Client),
	}
	if statusHandler == nil {
		pool.statusesHandler = func(event models.MailboxEvent) {}
	} else {
		pool.statusesHandler = statusHandler
	}
	return pool
}

func (clientPool *ClientConnPool) Get() (models.Client, error) {
	clientPool.checkoutMut.Lock()

	ttl := 10 * time.Minute
	nextId := clientPool.nextId
	select {
	case client := <-clientPool.pool:
		defer clientPool.checkoutMut.Unlock()
		if time.Since(client.LastPing()) > ttl {
			utils.DebugPrintln(fmt.Sprintf("client %d is stale, logging out", nextId))
			client.Logout()
			delete(clientPool.poolMap, nextId)
			var err error
			client, err = newClient(clientPool.options, clientPool.statusesHandler, nextId, clientPool)
			clientPool.poolMap[nextId] = client
			clientPool.nextId++
			if err != nil {
				return nil, err
			}
		}
		return client, nil
	default:

		// lazy create new connection if we haven't reached max pool size
		if len(clientPool.poolMap) < clientPool.options.MaxPoolSize() {
			defer clientPool.checkoutMut.Unlock()
			client, err := newClient(clientPool.options, clientPool.statusesHandler, nextId, clientPool)
			if err != nil {
				return nil, err
			}
			clientPool.poolMap[nextId] = client
			clientPool.nextId++
			utils.DebugPrintln(fmt.Sprintf("Created new connection. Total active connections: %d", len(clientPool.poolMap)))
			return client, nil
		}
	}

	// Pool is full and all connections are in use. Unlock because multiple goroutines can be waiting for a connection
	clientPool.checkoutMut.Unlock()
	conn := <-clientPool.pool
	return conn, nil
}

func (clientPool *ClientConnPool) Put(client models.Client) {

	clientPool.checkoutMut.Lock()
	defer clientPool.checkoutMut.Unlock()
	select {
	case clientPool.pool <- client:
	default:
		// Pool is full, close the connection
		utils.DebugPrintln(fmt.Sprintf("Closing connection, current pool size: %d", len(clientPool.poolMap)))
		delete(clientPool.poolMap, client.Id())
		client.Logout()
	}
}

func (clientPool *ClientConnPool) Close() {
	clientPool.checkoutMut.Lock()
	defer clientPool.checkoutMut.Unlock()
	defer close(clientPool.statuses)
	close(clientPool.pool)
	for conn := range clientPool.pool {
		conn.Logout()
		delete(clientPool.poolMap, conn.Id())
	}
}

func (clientPool *ClientConnPool) Statuses() chan models.MailboxEvent {
	return clientPool.statuses
}

func (clientPool *ClientConnPool) SetMailboxCache(mailbox models.Mailbox) {
	clientPool.mailboxCacheMut.Lock()
	defer clientPool.mailboxCacheMut.Unlock()
	clientPool.mailboxesCache[mailbox.Name()] = mailbox
}

func (clientPool *ClientConnPool) ListMailboxes() ([]models.Mailbox, error) {
	// TODO, allow for a configurable way of refreshing the mailbox cache
	if clientPool.mailboxesCache == nil || len(clientPool.mailboxesCache) == 0 {
		err := clientPool.HydrateMailboxCache()
		if err != nil {
			return nil, utils.JoinErrors("failed to hydrate mailbox cache", err)
		}
	}

	mboxes := make([]models.Mailbox, 0, len(clientPool.mailboxesCache))
	for _, mbox := range clientPool.mailboxesCache {
		mboxes = append(mboxes, mbox)
	}
	return mboxes, nil
}

func (clientPool *ClientConnPool) HydrateMailboxCache() error {
	/*

		steps:
		1. use a single connection to List all allMailboxes
		2. for each mailbox, use a single connection to fetch the mailbox status

	*/

	type selectResult struct {
		status *imap.MailboxStatus
		err    error
	}

	// TODO: deal with a hypothetical race condition where all checked out clients call this function at the same time
	listClient, err := clientPool.Get()
	if err != nil {
		return utils.JoinErrors("failed to get client from pool", err)
	}
	allMailboxes, err := listClient.ListMailboxInfos()

	if err != nil {
		return utils.JoinErrors("failed to list allMailboxes", err)
	}

	mailboxNameToInfo := map[string]*imap.MailboxInfo{}

	for _, mboxInfo := range allMailboxes {
		if utils.NewSet(mboxInfo.Attributes).Contains("\\Noselect") {
			continue
		}
		mailboxNameToInfo[mboxInfo.Name] = mboxInfo
	}
	statusChan := make(chan *selectResult, len(mailboxNameToInfo))

	clientPool.Put(listClient)
	for _, mboxInfo := range mailboxNameToInfo {
		go func(mboxName string, statChan chan *selectResult) {
			client, err := clientPool.Get()
			if err != nil {
				statChan <- &selectResult{nil, err}
				return
			}

			defer clientPool.Put(client)

			mboxStatus, err := client.RawSelect(mboxName, true)
			if err != nil {
				statChan <- &selectResult{nil, err}
				return
			} else {
				statChan <- &selectResult{mboxStatus, nil}
			}
		}(mboxInfo.Name, statusChan)
	}
	for i := 0; i < len(mailboxNameToInfo); i++ {
		res := <-statusChan
		if res.err != nil {
			return utils.JoinErrors("failed to get mailbox status", err)
		}
		mbox := mailbox.New(res.status, mailboxNameToInfo[res.status.Name])
		clientPool.SetMailboxCache(mbox)
	}

	return nil

}

func (clientPool *ClientConnPool) SyncMailboxMessageStates() error {

	utils.DebugPrintln("Syncing message states for all mailboxes")
	mailboxes, err := clientPool.ListMailboxes()
	if err != nil {
		return utils.JoinErrors("failed to list mailboxes", err)
	}
	errChan := make(chan error, len(mailboxes))

	for _, m := range mailboxes {

		go func(mbox models.Mailbox, pool *ClientConnPool, errChan chan error) {
			client, err := pool.Get()
			defer clientPool.Put(client)
			utils.DebugPrintln("Syncing message states for mailbox: " + mbox.Name())
			if err != nil {
				errChan <- utils.JoinErrors(fmt.Sprintf("failed to get client for mailbox %s", mbox.Name()), err)
				return
			}
			_, err = client.RawSelect(mbox.Name(), true)
			if err != nil {
				errChan <- utils.JoinErrors(fmt.Sprintf("failed to select mailbox %s", mbox.Name()), err)
				return
			}
			mbox.SetClient(client)
			err = mbox.SyncToLocalState()
			if err != nil {
				errChan <- utils.JoinErrors(fmt.Sprintf("failed to sync mailbox %s", mbox.Name()), err)
				return
			}
			errChan <- nil
		}(m, clientPool, errChan)
	}

	for i := 0; i < len(mailboxes); i++ {
		err := <-errChan
		if err != nil {
			return utils.JoinErrors("failed to sync mailbox", err)
		}
	}

	return nil
}

func (pool *ClientConnPool) DownloadAllMailboxes() error {
	mailboxes, err := pool.ListMailboxes()
	if err != nil {
		return utils.JoinErrors("failed to list mailboxes", err)
	}

	err = pool.SyncMailboxMessageStates()
	if err != nil {
		return utils.JoinErrors("failed to sync mailbox message states", err)
	}

	type mailboxDownloadResult struct {
		mbox models.Mailbox
		err  error
	}

	mailboxNameToInfo := map[string]models.Mailbox{}
	skipMailboxes := utils.NewSet(pool.options.SkipMailboxes())
	limitToMailboxes := utils.NewSet(pool.options.LimitToMailboxes())
	allMailboxes := utils.NewSet([]string{})
	unselectableMailboxes := utils.NewSet([]string{})

	if err != nil {
		return utils.JoinErrors("failed to list mailboxes", err)
	}

	for _, m := range mailboxes {
		if m.HasAttribute("\\Noselect") {
			unselectableMailboxes.Add(m.Name())
		}
		allMailboxes.Add(m.Name())
		mailboxNameToInfo[m.Name()] = m
	}
	finalMailboxes := allMailboxes.Minus(unselectableMailboxes)
	if len(limitToMailboxes) > 0 {
		finalMailboxes = finalMailboxes.Intersection(limitToMailboxes)
	}
	finalMailboxes = finalMailboxes.Minus(skipMailboxes)

	resultChan := make(chan mailboxDownloadResult, len(finalMailboxes))

	for _, mbName := range finalMailboxes.ToSlice() {
		go func(mbox models.Mailbox, pool *ClientConnPool, resChan chan mailboxDownloadResult) {
			client, err := pool.Get()
			if err != nil {
				resultChan <- mailboxDownloadResult{
					mbox: mbox,
					err:  utils.JoinErrors("failed to get client from pool", err),
				}
				return
			}
			defer pool.Put(client)
			mbox.SetClient(client)
			utils.DebugPrintln(fmt.Sprintf("downloading mailbox %s", mbox.Name()))
			err = mbox.DownloadEmails()
			resultChan <- mailboxDownloadResult{
				mbox: mbox,
				err:  err,
			}
		}(mailboxNameToInfo[mbName], pool, resultChan)
	}

	for i := 0; i < len(finalMailboxes); i++ {
		result := <-resultChan
		if result.err != nil {
			pool.Statuses() <- models.MailboxEvent{
				Mailbox:   result.mbox.Name(),
				EventType: models.MailboxDownloadError,
				Error:     result.err.Error(),
			}
			return utils.JoinErrors("failed to download emails from inbox", result.err)
		}
	}

	err = database.GetDatabase().AggregateFolders()
	if err != nil {
		return utils.JoinErrors("failed to aggregate folders", err)
	}
	return nil
}
