# Goals

The goal is to be able to filter and aggregate email by their metadata (e.g. sender), and bulk archive them. I am using gmail, but I'm trying to make it as extensible as possible. For example, instead of using the gmail API, I use IMAP.


# Current state
As of now, connection pooling of imap connections work all together to download all unique emails. Emails are parsed (including attachment metadata) and a unique id is used by hashing the contents. This way, emails in multiple folders (which is how gmail handles labels, by copying mail into multiple folders), can be correctly tagged with the proper labels/folders.


A preliminary frontend is a work in progress. The react frontend is served by the same webserver as the backend. So all you need to do is "go run cmd/main.go serve" to get a working frontend and backend.

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

# Data
Some data in email is array like. All data will be stored and queriable via a json query like interface, but for simplicity, the first piece of data is extracted from each array.

For example:

```json
{"Date":"2020-04-18T00:25:36+02:00","Subject":"Travel documents,  20 jun 2018,  Ref. XXXXX,  New York-EWR - Rome-Fiumicino","From":[{"PersonalName":"Norwegian.com","AtDomainList":"","MailboxName":"noreply","HostName":"norwegian.com"}],"Sender":[{"PersonalName":"Norwegian.com","AtDomainList":"","MailboxName":"noreply","HostName":"norwegian.com"}],"ReplyTo":[{"PersonalName":"Norwegian.com","AtDomainList":"","MailboxName":"noreply","HostName":"norwegian.com"}],"To":[{"PersonalName":"","AtDomainList":"","MailboxName":"shmuelkamensky","HostName":"gmail.com"}],"Cc": [{ "PersonalName":"SampleCC", "AtDomainList":"gmail.com", "MailboxName":"samplecc", "HostName":"gmail.com" }],"Bcc":[{"PersonalName":"SampleBCC","AtDomainList":"gmail.com","MailboxName":"samplebcc","HostName":"gmail.com"}],"InReplyTo":"","MessageId":"\u003c5ad67463.sogowmbs.wovks.79c7SMTPIN_ADDED_MISSING@mx.google.com\u003e"}
```
The following fields will be made available directly:

`from_name_1`: `Norwegian.com`
`from_mailbox_1`: `noreply`
`from_host_1`: `norwegian.com`
`sender_name_1`: `Norwegian.com`
`sender_mailbox_1`: `noreply`
`sender_host_1`: `norwegian.com`
`reply_to_name_1`: `Norwegian.com`
`reply_to_mailbox_1`: `noreply`
`reply_to_host_1`: `norwegian.com`
 
`to_name_1`: ``
`to_mailbox_1`: `shmuelkamensky`
`to_host_1`: `gmail.com`
`cc_name_1`: `SampleCC`
`cc_mailbox_1`: `samplecc`
`cc_host_1`: `gmail.com`
`bcc_name_1`: `SampleBCC`
`bcc_mailbox_1`: `samplebcc`
`bcc_host_1`: `gmail.com`
