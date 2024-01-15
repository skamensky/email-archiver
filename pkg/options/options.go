package options

import (
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Options struct {
	email             string
	password          string
	imapServer        string
	strictMailParsing bool
	// WARNING: setting DEBUG to true creates a huge debug.txt file
	imapClientDebug bool
	debug           bool
	// both LimitToMailboxes and SkipMailboxes use % as a separator. E.g. "INBOX%Sent%Work Stuff" means ["INBOX", "Sent", "Work Stuff"]
	// only download emails from these mailboxes. if empty, LimitToMailboxes is defined as all mailboxes.
	// The final result is LimitToMailboxes - SkipMailboxes
	limitToMailboxes []string
	skipMailboxes    []string
	dBPath           string
	maxPoolSize      int
}

func New() (models.Options, error) {
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
			options.email = value
		case "PASSWORD":
			options.password = value
		case "IMAP_SERVER":
			options.imapServer = value
			if !strings.Contains(value, ":") {
				options.imapServer = value + ":993"
			}
		case "STRICT_MAIL_PARSING":
			options.strictMailParsing, _ = strconv.ParseBool(value)
		case "IMAP_CLIENT_DEBUG":
			options.imapClientDebug, _ = strconv.ParseBool(value)
		case models.DEBUG_ENVIRONMENT_KEY:
			options.debug, _ = strconv.ParseBool(value)
		case "SKIP_MAILBOXES":
			// we use a % as a delimiter because it's not a valid character in most iamp server setup
			options.skipMailboxes = strings.Split(value, "%")
		case "LIMIT_TO_MAILBOXES":
			options.limitToMailboxes = strings.Split(value, "%")
		case "DB_PATH":
			// if the path isn't absolute, make it absolute relative to the current working directory
			if !filepath.IsAbs(value) {
				wd, err := os.Getwd()
				if err != nil {
					return nil, utils.JoinErrors("unable to get working directory", err)
				}
				value = filepath.Join(wd, value)
			}
			options.dBPath = value
		case "MAX_POOL_SIZE":
			maxPoolSize, err := strconv.Atoi(value)
			if err != nil {
				return nil, utils.JoinErrors("unable to parse MAX_POOL_SIZE", err)
			}
			if maxPoolSize < 1 {
				return nil, errors.New("MAX_POOL_SIZE must be greater than 0")
			}
			options.maxPoolSize = maxPoolSize
		}
	}

	if options.email == "" {
		return nil, errors.New("missing EMAIL")
	}
	if options.password == "" {
		return nil, errors.New("missing PASSWORD")
	}
	if options.imapServer == "" {
		return nil, errors.New("missing IMAP_SERVER")
	}
	if options.dBPath == "" {
		return nil, errors.New("missing DB_PATH")
	}
	if options.maxPoolSize == 0 {
		options.maxPoolSize = 3
	}
	return options, nil
}
func (options *Options) ImapServer() string {
	return options.imapServer
}
func (options *Options) Email() string {
	return options.email
}
func (options *Options) Password() string {
	return options.password
}

func (options *Options) StrictMailParsing() bool {
	return options.strictMailParsing
}
func (options *Options) ImapClientDebug() bool {
	return options.imapClientDebug
}

func (options *Options) Debug() bool {
	return options.debug
}

func (options *Options) LimitToMailboxes() []string {
	return options.limitToMailboxes

}
func (options *Options) SkipMailboxes() []string {
	return options.skipMailboxes
}
func (options *Options) DBPath() string {
	return options.dBPath
}

func (options *Options) MaxPoolSize() int {
	return options.maxPoolSize
}
