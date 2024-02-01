package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/skamensky/email-archiver/pkg/email"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"os"
	"strings"
	"sync"
	"time"
)

type DB struct {
	options models.Options
}

var dATABASE *DB
var mutex = &sync.Mutex{}

var tableToColumns = map[string][]string{}

func GetDatabase() models.DB {
	if dATABASE != nil {
		return dATABASE
	} else {
		panic("database not initialized")
	}
}

// gets columns in order of definition
func (dbWrap *DB) getTableColumns(tableName string) ([]string, error) {
	if tableToColumns == nil {
		tableToColumns = map[string][]string{}
	}
	//select name from pragma_table_info('tablName') order by cid

	//check if we already have the columns
	if columns, ok := tableToColumns[tableName]; ok {
		return columns, nil
	}

	db, err := dbWrap.getDB()

	if err != nil {
		return nil, utils.JoinErrors("failed to open db", err)
	}

	rows, err := db.Query(fmt.Sprintf("select name from pragma_table_info('%v') order by cid", tableName))
	if err != nil {
		return nil, utils.JoinErrors("failed to get columns", err)
	}
	defer rows.Close()
	columns := []string{}
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, utils.JoinErrors("failed to scan column name", err)
		}
		columns = append(columns, name)
	}
	err = rows.Err()
	if err != nil {
		return nil, utils.JoinErrors("failed to get columns", err)
	}
	tableToColumns[tableName] = columns
	return columns, nil
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

func (dbWrap *DB) getDB() (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite3", dbWrap.options.GetDBPath())

	if err != nil {
		return nil, utils.JoinErrors("failed to open db", err)
	}
	return db, nil
}

func (dbWrap *DB) initDB() error {
	_, err := os.Stat(dbWrap.options.GetDBPath())
	if err == nil {
		return nil
	}

	utils.DebugPrintln("DB", dbWrap.options.GetDBPath(), "does not exist, initializing DB")

	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()
	_, err = db.Exec("DROP TABLE IF EXISTS email")
	if err != nil {
		return utils.JoinErrors("failed to drop table emails", err)
	}
	_, err = db.Exec(`
		CREATE TABLE email (
			our_id text primary key,
			parse_warning text,
			parse_error text,
			envelope text,
			flags text,
			mailboxes text,
			text_content text,
			html_content text,
			attachments text,
			message_id text,
			date text,
			subject text,
			from_name_1 text,
			from_mailbox_1 text,
			from_host_1 text,
			sender_name_1 text,
			sender_mailbox_1 text,
			sender_host_1 text,
			reply_to_name_1 text,
			reply_to_mailbox_1 text,
			reply_to_host_1 text,
			to_name_1 text,
			to_mailbox_1 text,
			to_host_1 text,
			cc_name_1 text,
			cc_mailbox_1 text,
			cc_host_1 text,
			bcc_name_1 text,
			bcc_mailbox_1 text,
			bcc_host_1 text,
			in_reply_to text
		);`)
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
	_, err = db.Exec("CREATE TABLE message_staging(uid int,mailbox_name text,primary key (mailbox_name, uid))")
	if err != nil {
		return utils.JoinErrors("failed to create message_staging table", err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS email_fts")
	if err != nil {
		return utils.JoinErrors("failed to drop table email_fts", err)
	}

	_, err = db.Exec("CREATE VIRTUAL TABLE email_fts USING fts5(our_id unindexed, text_content, subject, from_name_1, from_mailbox_1, from_host_1, content=email);")
	if err != nil {
		return utils.JoinErrors("failed to create email_fts table", err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS mailbox")
	if err != nil {
		return utils.JoinErrors("failed to drop table mailbox", err)
	}

	_, err = db.Exec("create table persisted_frontend_state (  state text, id text primary key default 'state_id' );")
	if err != nil {
		return utils.JoinErrors("failed to create persisted_frontend_state table", err)
	}

	_, err = db.Exec("CREATE TABLE mailbox (name text primary key,attributes text,last_synced int, num_emails int)")
	return utils.JoinErrors("failed to create mailbox table", err)

}

func (dbWrap *DB) SaveMailboxRecord(mailbox models.MailboxRecord) error {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	// now utc:
	now := time.Now().UTC().Unix()
	attributesAsJson, err := json.Marshal(mailbox.Attributes)

	if err != nil {
		return utils.JoinErrors("failed to marshal attributes", err)
	}

	_, err = db.Exec("INSERT INTO mailbox (name,attributes,last_synced) VALUES (?, ?, ? ) ON CONFLICT(name) DO UPDATE SET attributes = ?, last_synced = ?", mailbox.Name, string(attributesAsJson), now, string(attributesAsJson), now)
	return utils.JoinErrors("failed to insert mailbox record", err)
}

func (dbWrap *DB) GetMailboxRecord(mailbox models.Mailbox) (models.MailboxRecord, error) {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return models.MailboxRecord{}, utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	var lastSynced sql.NullInt64
	var attributes sql.NullString

	err = db.QueryRow("SELECT attributes,last_synced FROM folder WHERE name = ?", mailbox.Name()).Scan(&lastSynced, &attributes)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.MailboxRecord{}, nil
		}
		return models.MailboxRecord{}, utils.JoinErrors("failed to query mailbox record", err)
	}

	if !lastSynced.Valid {
		lastSynced.Int64 = 0
	}
	attributesAsList := []string{}

	if attributes.Valid {
		err = json.Unmarshal([]byte(attributes.String), &attributesAsList)
		if err != nil {
			return models.MailboxRecord{}, utils.JoinErrors("failed to unmarshal attributes", err)
		}
	}

	return models.MailboxRecord{
		Name:       mailbox.Name(),
		Attributes: attributesAsList,
		LastSynced: lastSynced.Int64,
	}, nil
}

func (dbWrap *DB) GetAllMailboxRecords() ([]models.MailboxRecord, error) {
	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return nil, utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, attributes, last_synced,num_emails FROM mailbox")
	if err != nil {
		return nil, utils.JoinErrors("failed to get all mailbox records", err)
	}

	defer rows.Close()
	var records []models.MailboxRecord
	for rows.Next() {
		var name string
		var attributes sql.NullString
		var lastSynced sql.NullInt64
		var numEmails sql.NullInt64
		err = rows.Scan(&name, &attributes, &lastSynced, &numEmails)
		if err != nil {
			return nil, utils.JoinErrors("failed to scan mailbox record", err)
		}

		if !lastSynced.Valid {
			lastSynced.Int64 = 0
		}

		attributesAsList := []string{}
		if attributes.Valid {
			err = json.Unmarshal([]byte(attributes.String), &attributesAsList)
			if err != nil {
				return nil, utils.JoinErrors("failed to unmarshal attributes", err)
			}
		}

		records = append(records, models.MailboxRecord{
			Name:       name,
			Attributes: attributesAsList,
			LastSynced: lastSynced.Int64,
			NumEmails:  int(numEmails.Int64),
		})
	}
	return records, nil
}

func (dbWrap *DB) AddEmails(mailbox string, emails []models.Email) error {

	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	tx, err := db.Beginx()
	if err != nil {
		return utils.JoinErrors("failed to begin transaction", err)
	}

	insertEmailStmnt, err := tx.Prepare(`INSERT INTO email (our_id,parse_warning ,parse_error ,envelope ,flags ,text_content ,html_content ,attachments ,message_id ,date ,subject ,from_name_1 ,from_mailbox_1 ,from_host_1 ,sender_name_1 ,sender_mailbox_1 ,sender_host_1 ,reply_to_name_1 ,reply_to_mailbox_1 ,reply_to_host_1 ,to_name_1 ,to_mailbox_1 ,to_host_1 ,cc_name_1 ,cc_mailbox_1 ,cc_host_1 ,bcc_name_1 ,bcc_mailbox_1 ,bcc_host_1 ,in_reply_to )
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		
		ON CONFLICT (our_id) DO NOTHING
	`)

	if err != nil {
		return utils.JoinErrors("failed to prepare insert statement", err)
	}
	defer insertEmailStmnt.Close()

	insertFolderStmnt, err := tx.Prepare(`
		INSERT INTO message_to_mailbox (mailbox_name, our_id, uid,pending_sync)
		VALUES (?, ?, ?, 0)
		ON CONFLICT (mailbox_name, uid) DO UPDATE SET our_id = excluded.our_id, pending_sync = 0
	`)

	if err != nil {
		return utils.JoinErrors("failed to prepare insert statement", err)
	}

	for _, mail := range emails {
		_, err = insertEmailStmnt.Exec(mail.GetOurID(), mail.GetParseWarning(), mail.GetParseError(), utils.MustJSON(mail.GetEnvelope()), utils.MustJSON(mail.GetFlags()), mail.GetTextContent(), mail.GetHTMLContent(), utils.MustJSON(mail.GetAttachments()), mail.GetMessageId(), mail.GetDate(), mail.GetSubject(), mail.GetFromName1(), mail.GetFromMailbox1(), mail.GetFromHost1(), mail.GetSenderName1(), mail.GetSenderMailbox1(), mail.GetSenderHost1(), mail.GetReplyToName1(), mail.GetReplyToMailbox1(), mail.GetReplyToHost1(), mail.GetToName1(), mail.GetToMailbox1(), mail.GetToHost1(), mail.GetCcName1(), mail.GetCcMailbox1(), mail.GetCcHost1(), mail.GetBccName1(), mail.GetBccMailbox1(), mail.GetBccHost1(), mail.GetInReplyTo())
		if err != nil {
			return utils.JoinErrors("failed to insert email", err)
		}
		_, err = insertFolderStmnt.Exec(mailbox, mail.GetOurID(), mail.GetUID())
		if err != nil {
			return utils.JoinErrors("failed to insert folder", err)
		}
	}

	err = tx.Commit()

	return utils.JoinErrors("failed to commit transaction", err)
}

func (dbWrap *DB) AggregateFolders() error {

	// TODO add a column 'numberOfMessages' to the mailbox table and update it here.

	mutex.Lock()
	defer mutex.Unlock()
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}

	_, err = db.Exec(`
		UPDATE mailbox
		SET num_emails = (
		SELECT COUNT(*)
		FROM message_to_mailbox
		WHERE message_to_mailbox.mailbox_name = mailbox.name
		)
	`)

	if err != nil {
		return utils.JoinErrors("failed to update mailbox table", err)
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

// useful for debugging, let's keep it around.
func (dbWrap *DB) debugPrintMessageToMailboxTable(mailboxName string, querier sqlx.Queryer) {
	vals, err := querier.Query("SELECT uid,pending_sync FROM message_to_mailbox WHERE mailbox_name = ?", mailboxName)
	if err != nil {
		utils.PanicIfError(err)
	}
	type uidAndPendingSync struct {
		Uid         uint32 `json:"uid"`
		PendingSync int    `json:"pending_sync"`
	}
	idsInDB := []uidAndPendingSync{}
	for vals.Next() {
		var uid uint32
		var pendingSync int
		utils.PanicIfError(vals.Scan(&uid, &pendingSync))
		idsInDB = append(idsInDB, uidAndPendingSync{Uid: uid, PendingSync: pendingSync})
	}
	utils.PanicIfError(vals.Close())
	utils.DebugPrintln("idsInDB", utils.MustJSON(idsInDB))

}

func (dbWrap *DB) UpdateLocalMailboxState(mailbox models.Mailbox, allUids []uint32) error {
	mutex.Lock()
	defer mutex.Unlock()

	err := dbWrap.RemoveOrphanedEmailsFromMailbox(mailbox, allUids)
	if err != nil {
		return utils.JoinErrors("failed to remove orphaned emails", err)
	}

	err = dbWrap.AddMissingEmailsToMailbox(mailbox, allUids)

	return err
}

// caller must hold mutex
func (dbWrap *DB) AddMissingEmailsToMailbox(mailbox models.Mailbox, allUids []uint32) error {
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	tx, err := db.Begin()

	if err != nil {
		return utils.JoinErrors("failed to begin transaction", err)
	}

	// if the email is already in the db, we don't need to add it again, the last value of pending_sync will be preserved (usually 0)
	insertStmt, err := tx.Prepare("INSERT INTO message_to_mailbox (mailbox_name, uid, pending_sync) VALUES (?, ?, 1) ON CONFLICT DO NOTHING ")

	for _, uid := range allUids {
		_, err = insertStmt.Exec(mailbox.Name(), uid)
		if err != nil {
			return utils.JoinErrors("failed to insert uid", err)
		}
	}

	err = tx.Commit()
	return utils.JoinErrors("failed to commit transaction", err)
}

// caller must lock mutex
func (dbWrap *DB) RemoveOrphanedEmailsFromMailbox(mailbox models.Mailbox, allUids []uint32) error {

	err := dbWrap.clearStaging(mailbox)
	if err != nil {
		return utils.JoinErrors("failed to truncate staging events", err)
	}
	err = dbWrap.addStagingMessages(allUids, mailbox)

	if err != nil {
		return utils.JoinErrors("failed to add mailbox message events", err)
	}

	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	// after this runs, message_to_mailbox will not contain uids that are not in the mailbox.
	_, err = db.Exec("DELETE FROM message_to_mailbox WHERE mailbox_name = ? AND uid NOT IN (SELECT uid FROM message_staging where mailbox_name = ?)", mailbox.Name(), mailbox.Name())
	if err != nil {
		return utils.JoinErrors("failed to remove orphaned emails using staging", err)
	}
	return nil
}

// caller must hold mutex
func (dbWrap *DB) addStagingMessages(uids []uint32, mailbox models.Mailbox) error {
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
	insertStmt, err := tx.Prepare("INSERT INTO message_staging (uid,mailbox_name) VALUES (?,?) ON CONFLICT DO NOTHING ")

	for _, uid := range uids {
		_, err = insertStmt.Exec(uid, mailbox.Name())
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

func (dbWrap *DB) clearStaging(mailbox models.Mailbox) error {
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}
	_, err = db.Exec("DELETE FROM message_staging WHERE mailbox_name = ?", mailbox.Name())
	return utils.JoinErrors("failed to truncate message_staging", err)
}

func (dbWrap *DB) GetEmails(sqlQuery string, params ...interface{}) ([]models.Email, error) {
	db, err := dbWrap.getDB()
	if err != nil {
		return nil, utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()
	utils.DebugPrintln(fmt.Sprintf("executing query: %s", sqlQuery))
	emails := []models.Email{}

	rows, err := db.Queryx(sqlQuery, params...)

	if err != nil {
		return nil, utils.JoinErrors("failed to execute query", err)
	}
	defer rows.Close()

	for rows.Next() {
		mail, err := email.NewFromDBRecord(rows)
		if err != nil {
			utils.DebugPrintln("failed to create email from db record")
			return nil, utils.JoinErrors("failed to create email from db record", err)
		}
		emails = append(emails, mail)
	}
	return emails, nil
}

func (dbWrap *DB) FullTextSearch(searchTerm string) ([]models.Email, error) {
	allEmailColumns, err := dbWrap.getTableColumns("email")
	if err != nil {
		return nil, utils.JoinErrors("failed to get email table columns", err)
	}

	searchFieldsInOrder, err := dbWrap.getTableColumns("email_fts")
	if err != nil {
		return nil, utils.JoinErrors("failed to get email_fts table columns", err)
	}

	emailColumnsToSelect := utils.NewSet(allEmailColumns).Minus(utils.NewSet(searchFieldsInOrder)).ToSlice()
	emailColumnsToSelectEmaiLPrepended := []string{}
	for _, col := range emailColumnsToSelect {
		emailColumnsToSelectEmaiLPrepended = append(emailColumnsToSelectEmaiLPrepended, fmt.Sprintf("email.%s AS %s", col, col))
	}

	emailFtsColumns := []string{"email_fts.our_id as our_id"}
	for index, col := range searchFieldsInOrder[1:] {
		// the reason we are generating this is because I was already hit by a bug from misnumbering the columns

		//highlight(email_fts, 1, '<span class="bg-yellow-200 text-black">', '</span>') as text_content,
		emailFtsColumns = append(emailFtsColumns, fmt.Sprintf(`highlight(email_fts, %d, '<span class="bg-yellow-200 text-black">', '</span>') as %s`, index+1, col))
	}

	// the goal of this sql sqlQuery is to match select * from email but filter on matched results using full text search
	// and replace the matched fields with highlighted versions
	sqlQuery := fmt.Sprintf(`
		SELECT
			%s
		FROM email_fts
		JOIN email ON email.our_id = email_fts.our_id
		WHERE email_fts MATCH ?
		`, strings.Join(append(emailFtsColumns, emailColumnsToSelectEmaiLPrepended...), ",\n\t\t\t"))

	return dbWrap.GetEmails(sqlQuery, searchTerm)
}

func (dbWrap *DB) UpdateFTS() error {
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}

	// the reason we drop and recreate the table is because whenever I tried deleting I got '[SQLITE_CORRUPT_VTAB] Content in the virtual table is corrupt (database disk image is malformed)

	_, err = db.Exec("DROP TABLE IF EXISTS email_fts")
	if err != nil {
		return utils.JoinErrors("failed to drop email_fts", err)
	}

	_, err = db.Exec("CREATE VIRTUAL TABLE email_fts USING fts5(our_id unindexed, text_content, subject, from_name_1, from_mailbox_1, from_host_1, content=email)")

	if err != nil {
		return utils.JoinErrors("failed to recreate email_fts", err)
	}

	_, err = db.Exec("INSERT INTO email_fts(our_id,text_content, subject, from_name_1, from_mailbox_1, from_host_1) SELECT our_id,text_content, subject, from_name_1, from_mailbox_1, from_host_1 FROM email")
	if err != nil {
		return utils.JoinErrors("failed to insert into email_fts", err)
	}

	return nil
}
func (dbWrap *DB) GetFrontendState() (string, error) {
	db, err := dbWrap.getDB()
	if err != nil {
		return "", utils.JoinErrors("failed to open db", err)
	}
	defer db.Close()

	var frontendState string

	err = db.QueryRow("SELECT state FROM persisted_frontend_state").Scan(&frontendState)

	if err != nil {
		return "", utils.JoinErrors("failed to get frontend state", err)
	}

	return frontendState, nil
}

func (dbWrap *DB) SetFrontendState(state string) error {
	db, err := dbWrap.getDB()
	if err != nil {
		return utils.JoinErrors("failed to open db", err)
	}

	_, err = db.Exec("INSERT INTO persisted_frontend_state(state,id) VALUES(?,'state_id') ON CONFLICT(id) DO UPDATE SET state = ?", state, state)

	if err != nil {
		return utils.JoinErrors("failed to set frontend state", err)
	}

	defer db.Close()
	return nil
}
