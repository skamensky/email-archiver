package database

import (
	"database/sql"
	"fmt"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"os"
	"sync"
)

type DB struct {
	options models.Options
}

var dATABASE *DB
var mutex = &sync.Mutex{}

func GetDatabase() models.DB {
	if dATABASE != nil {
		return dATABASE
	} else {
		panic("database not initialized")
	}
}

func New(options models.Options) (models.DB, error) {
	if dATABASE != nil {
		return dATABASE, nil
	}
	db := &DB{
		options: options,
	}
	err := db.initDB()
	if err != nil {
		return nil, utils.JoinErrors("failed to initialize db", err)
	}

	dATABASE = db

	return dATABASE, nil
}

func (dbWrap *DB) getDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbWrap.options.DBPath())
	if err != nil {
		return nil, utils.JoinErrors("failed to open db", err)
	}
	return db, nil
}

func (dbWrap *DB) initDB() error {
	_, err := os.Stat(dbWrap.options.DBPath())
	if err == nil {
		fmt.Printf("%v already exists, skipping init\n", dbWrap.options.DBPath())
		return nil
	}

	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()
	_, err = db.Exec("DROP TABLE IF EXISTS email")
	if err != nil {
		return utils.JoinErrors("failed to drop table emails", err)
	}
	_, err = db.Exec("CREATE TABLE email (parse_warning text, parse_error text,our_id text primary key, envelope text, flags text, mailboxes text, text_content text, html_content text, attachments text)")
	if err != nil {
		return utils.JoinErrors("failed to create table email", err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS message_to_mailbox")
	if err != nil {
		return utils.JoinErrors("failed to drop table mailbox", err)
	}

	_, err = db.Exec("CREATE TABLE message_to_mailbox (mailbox_name text, our_id text, uid int, pending_sync integer, primary key (mailbox_name, uid))")
	if err != nil {
		return utils.JoinErrors("failed to create message_to_mailbox table", err)
	}
	// index on email_id so our updates are faster
	_, err = db.Exec("CREATE INDEX our_id_index ON message_to_mailbox (our_id)")
	if err != nil {
		return utils.JoinErrors("failed to create our_id_index index", err)
	}

	// essentially a list of uids
	_, err = db.Exec("DROP TABLE IF EXISTS message_staging")
	if err != nil {
		return utils.JoinErrors("failed to drop table message_staging", err)
	}
	_, err = db.Exec("CREATE TABLE message_staging(uid int, primary key (uid))")
	if err != nil {
		return utils.JoinErrors("failed to create message_staging table", err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS mailbox")
	if err != nil {
		return utils.JoinErrors("failed to drop table mailbox", err)
	}
	_, err = db.Exec("CREATE TABLE mailbox (name text primary key,uid_next int, uid_validity int)")
	return utils.JoinErrors("failed to create mailbox table", err)

}

func (dbWrap *DB) SetNextUID(mailbox models.Mailbox, nextUID uint32, uidValidity uint32) error {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO mailbox (name, uid_next,uid_validity) VALUES (?, ?, ? ) ON CONFLICT(name) DO UPDATE SET uid_next = ?, uid_validity = ?", mailbox.Name(), nextUID, uidValidity, nextUID, uidValidity)
	return utils.JoinErrors("failed to set next uid", err)
}

func (dbWrap *DB) GetNextUID(mailbox models.Mailbox) (models.MailboxRecord, error) {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return models.MailboxRecord{}, utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	var uidNext uint32
	var uidValidity uint32
	err = db.QueryRow("SELECT uid_next, uid_validity FROM folder WHERE name = ?", mailbox.Name()).Scan(&uidNext, &uidValidity)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.MailboxRecord{}, nil
		}
		return models.MailboxRecord{}, utils.JoinErrors("failed to get next uid", err)
	}
	return models.MailboxRecord{
		Name:        mailbox.Name(),
		UIDValidity: uidValidity,
		UIDNext:     uidNext,
	}, nil
}

func (dbWrap *DB) SetMessagesToSynced(mailbox models.Mailbox, uids []uint32) error {
	mutex.Lock()
	defer mutex.Unlock()

	err := dbWrap.truncateStaging()
	if err != nil {
		return utils.JoinErrors("failed to truncate staging", err)
	}
	err = dbWrap.addStagingMessages(uids)
	if err != nil {
		return utils.JoinErrors("failed to add staging messages", err)
	}

	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM message_to_mailbox WHERE mailbox_name = ? AND uid in (SELECT uid FROM message_staging)", mailbox.Name())
	if err != nil {
		return utils.JoinErrors("failed to delete from message_to_mailbox", err)
	}

	err = dbWrap.truncateStaging()
	if err != nil {
		return utils.JoinErrors("failed to truncate staging", err)
	}

	return nil
}

func (dbWrap *DB) AddToDB(emails []models.Email) error {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return utils.JoinErrors("failed to begin transaction", err)
	}

	insertEmailStmnt, err := tx.Prepare(`
		INSERT INTO email (parse_warning, parse_error, our_id, envelope, flags, text_content, html_content, attachments)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (our_id) DO NOTHING
	`)

	if err != nil {
		return utils.JoinErrors("failed to prepare insert statement", err)
	}
	defer insertEmailStmnt.Close()

	insertFolderStmnt, err := tx.Prepare(`
		INSERT INTO message_to_mailbox (mailbox_name, our_id, uid)
		VALUES (?, ?, ?)
		ON CONFLICT (mailbox_name, uid) DO NOTHING
	`)

	if err != nil {
		return utils.JoinErrors("failed to prepare insert statement", err)
	}

	// todo don't insert in a loop
	for _, email := range emails {
		_, err = insertEmailStmnt.Exec(email.ParseWarning(), email.ParseError(), email.OurID(), utils.MustJSON(email.Envelope()), utils.MustJSON(email.Flags()), email.TextContent(), email.HTMLContent(), utils.MustJSON(email.Attachments()))
		if err != nil {
			return utils.JoinErrors("failed to insert email", err)
		}
		_, err = insertFolderStmnt.Exec(email.Mailbox(), email.OurID(), email.UID())
		if err != nil {
			return utils.JoinErrors("failed to insert folder", err)
		}
	}

	err = tx.Commit()
	return utils.JoinErrors("failed to commit transaction", err)
}

func (dbWrap *DB) AggregateFolders() error {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}

	// update emails table, folders will be a json list of folders.
	_, err = db.Exec(`
		update email
		set mailboxes = subtable.mailboxes
		FROM
			(
				select json_group_array(mailbox_name) mailboxes, our_id
				FROM message_to_mailbox
				GROUP BY our_id
			) subtable
		WHERE email.our_id = subtable.our_id;
	`)

	return utils.JoinErrors("failed to update email table", err)

}

func (dbWrap *DB) GetMessagesPendingSync(mailbox models.Mailbox) ([]uint32, error) {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return nil, utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()
	pendingUIDs := []uint32{}
	rows, err := db.Query("SELECT uid FROM message_to_mailbox WHERE pending_sync = 1 AND mailbox_name = ?", mailbox.Name())
	if err != nil {
		return nil, utils.JoinErrors("failed to get messages pending sync", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uid uint32
		err = rows.Scan(&uid)
		if err != nil {
			return nil, utils.JoinErrors("failed to scan row", err)
		}
		pendingUIDs = append(pendingUIDs, uid)
	}
	return pendingUIDs, nil
}

func (dbWrap *DB) UpdateLocalMailboxState(mailbox models.Mailbox, newUids []uint32) error {
	mutex.Lock()
	defer mutex.Unlock()
	err := dbWrap.RemoveOrphanedEmailsFromMailbox(mailbox, newUids)
	if err != nil {
		return utils.JoinErrors("failed to remove orphaned emails", err)
	}
	return dbWrap.AddMissingEmailsToMailbox(mailbox, newUids)
}

// caller must hold mutex
func (dbWrap *DB) AddMissingEmailsToMailbox(mailbox models.Mailbox, newUids []uint32) error {
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	tx, err := db.Begin()

	if err != nil {
		return utils.JoinErrors("failed to begin transaction", err)
	}

	// if the email is already in the db, we don't need to add it again
	insertStmt, err := tx.Prepare("INSERT INTO message_to_mailbox (mailbox_name, uid, pending_sync) VALUES (?, ?, 1) ON CONFLICT DO NOTHING ")

	for _, uid := range newUids {
		_, err = insertStmt.Exec(mailbox.Name(), uid)
		if err != nil {
			return utils.JoinErrors("failed to insert uid", err)
		}
	}

	err = tx.Commit()
	return utils.JoinErrors("failed to commit transaction", err)
}

// caller must lock mutex
func (dbWrap *DB) RemoveOrphanedEmailsFromMailbox(mailbox models.Mailbox, newUids []uint32) error {

	err := dbWrap.truncateStaging()
	if err != nil {
		return utils.JoinErrors("failed to truncate staging events", err)
	}
	err = dbWrap.addStagingMessages(newUids)

	if err != nil {
		return utils.JoinErrors("failed to add mailbox message events", err)
	}

	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM message_to_mailbox WHERE mailbox_name = ? AND uid NOT IN (SELECT uid FROM message_staging)", mailbox.Name())
	if err != nil {
		return utils.JoinErrors("failed to remove orphaned emails using staging", err)
	}
	_, err = db.Exec("DELETE FROM message_to_mailbox WHERE mailbox_name = ? AND pending_sync = 1", mailbox.Name())
	if err != nil {
		return utils.JoinErrors("failed to remove orphaned emails pending syncs", err)
	}
	err = dbWrap.truncateStaging()
	if err != nil {
		return utils.JoinErrors("failed to truncate staging events", err)
	}
	return nil
}

// caller must hold mutex
func (dbWrap *DB) addStagingMessages(uids []uint32) error {
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	tx, err := db.Begin()

	if err != nil {
		return utils.JoinErrors("failed to begin transaction", err)
	}

	// if the email is already in the db, we don't need to add it again
	insertStmt, err := tx.Prepare("INSERT INTO message_staging (uid) VALUES (?) ON CONFLICT DO NOTHING ")

	for _, uid := range uids {
		_, err = insertStmt.Exec(uid)
		if err != nil {
			return utils.JoinErrors("failed to insert uid", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return utils.JoinErrors("failed to commit transaction", err)
	}
	return nil
}

func (dbWrap *DB) truncateStaging() error {
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	_, err = db.Exec("DELETE FROM message_staging")
	return utils.JoinErrors("failed to truncate message_staging", err)
}
