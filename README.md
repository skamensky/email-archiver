# Goals

The goal is to be able to filter and aggregate email by their metadata (e.g. sender), and bulk archive them. I am using gmail, but I'm trying to make it as extensible as possible. For example, instead of using the gmail API, I use IMAP.


# Current state
As of now, connection pooling of imap connections work all together to download all unique emails. Emails are parsed (including attachment metadata) and a unique id is used by hashing the contents. This way, emails in multiple folders (which is how gmail handles labels, by copying mail into multiple folders), can be correctly tagged with the proper labels/folders.

# Options

All options are read from environment variables. This program supports `.env`.


- `EMAIL`=`john@example.com`
- `PASSWORD`=`your email imap password`
- `IMAP_SERVER`=`imap.gmail.com:993` IMAP_SERVER server also works without a port in which case it will use the default port 993
- `IMAP_CLIENT_DEBUG`=`false`
 whether or not to output all imap client data to a file (warning, this is a lot of data)
- `DEBUG`=`false` whether or not to output all of this programs debug statements to stdout
- `SKIP_MAILBOXES`=`[Gmail]/All Mail%[Gmail]/Important` which folders to skip when downloading emails. Separated by a `%` since that is an invalid character for a folder name
- `LIMIT_TO_MAILBOXES`=`Euro Trip 2018` which folders to limit the download to. Separated by a `%` since that is an invalid character for a folder name. If used in conjunction with `SKIP_MAILBOXES`, the final result is  `limit folders - skip folders + `, if `SKIP_MAILBOXES` is not set, then it is just `limit folders`.
- `DB_PATH`=`data.db` sqlite database path. If it's not a full path, it will be relative to the current working directory
- `MAX_POOL_SIZE`=`10` how many imap connections to use at once


# Run
`go run cmd/main.go download`

# TODO
Let's add an interface and a builtin webserver!