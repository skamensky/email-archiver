package options

import (
	"errors"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Options struct {
	Email             string `json:"email,omitempty"`
	Password          string `json:"password,omitempty"`
	ImapServer        string `json:"imap_server,omitempty"`
	StrictMailParsing bool   `json:"strict_mail_parsing,omitempty"`
	// WARNING: setting DEBUG to true creates a huge Debug.txt file
	ImapClientDebug bool `json:"imap_client_debug,omitempty"`
	Debug           bool `json:"debug,omitempty"`
	// both GetLimitToMailboxes and GetSkipMailboxes use % as a separator. E.g. "INBOX%Sent%Work Stuff" means ["INBOX", "Sent", "Work Stuff"]
	// only download emails from these mailboxes. if empty, GetLimitToMailboxes is defined as all mailboxes.
	// The final result is GetLimitToMailboxes - GetSkipMailboxes
	LimitToMailboxes []string `json:"limit_to_mailboxes,omitempty"`
	SkipMailboxes    []string `json:"skip_mailboxes,omitempty"`
	DBPath           string   `json:"db_path,omitempty"`
	MaxPoolSize      int      `json:"max_pool_size,omitempty"`
}

func New() (models.Options, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, utils.JoinErrors("Error loading .env file", err)
	}
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
			options.Email = value
		case "PASSWORD":
			options.Password = value
		case "IMAP_SERVER":
			options.ImapServer = value
			if !strings.Contains(value, ":") {
				options.ImapServer = value + ":993"
			}
		case "STRICT_MAIL_PARSING":
			options.StrictMailParsing, _ = strconv.ParseBool(value)
		case "IMAP_CLIENT_DEBUG":
			options.ImapClientDebug, _ = strconv.ParseBool(value)
		case models.DEBUG_ENVIRONMENT_KEY:
			options.Debug, _ = strconv.ParseBool(value)
		case "SKIP_MAILBOXES":
			// we use a % as a delimiter because it's not a valid character in most iamp server setup
			options.SkipMailboxes = strings.Split(value, "%")
		case "LIMIT_TO_MAILBOXES":
			options.LimitToMailboxes = strings.Split(value, "%")
		case "DB_PATH":
			// if the path isn't absolute, make it absolute relative to the current working directory
			if !filepath.IsAbs(value) {
				wd, err := os.Getwd()
				if err != nil {
					return nil, utils.JoinErrors("unable to get working directory", err)
				}
				value = filepath.Join(wd, value)
			}
			options.DBPath = value
		case "MAX_POOL_SIZE":
			maxPoolSize, err := strconv.Atoi(value)
			if err != nil {
				return nil, utils.JoinErrors("unable to parse MAX_POOL_SIZE", err)
			}
			if maxPoolSize < 1 {
				return nil, errors.New("MAX_POOL_SIZE must be greater than 0")
			}
			options.MaxPoolSize = maxPoolSize
		}
	}
	if options.Email == "" {
		return nil, errors.New("missing EMAIL")
	}
	if options.Password == "" {
		return nil, errors.New("missing PASSWORD")
	}
	if options.ImapServer == "" {
		return nil, errors.New("missing IMAP_SERVER")
	}
	if options.DBPath == "" {
		return nil, errors.New("missing DB_PATH")
	}
	if options.MaxPoolSize == 0 {
		options.MaxPoolSize = 3
	}
	return options, nil
}
func (options *Options) GetImapServer() string {
	return options.ImapServer
}
func (options *Options) GetEmail() string {
	return options.Email
}
func (options *Options) GetPassword() string {
	return options.Password
}

func (options *Options) GetStrictMailParsing() bool {
	return options.StrictMailParsing
}
func (options *Options) GetImapClientDebug() bool {
	return options.ImapClientDebug
}

func (options *Options) GetDebug() bool {
	return options.Debug
}

func (options *Options) GetLimitToMailboxes() []string {
	return options.LimitToMailboxes

}
func (options *Options) GetSkipMailboxes() []string {
	return options.SkipMailboxes
}
func (options *Options) GetDBPath() string {
	return options.DBPath
}

func (options *Options) GetMaxPoolSize() int {
	return options.MaxPoolSize
}
