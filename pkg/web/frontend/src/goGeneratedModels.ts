/* Do not change, this code is generated from Golang structs */


export enum MailboxEventType {
    MailboxSyncQueued = "MailboxSyncQueued",
    MailboxDownloadStarted = "MailboxDownloadStarted",
    MailboxDownloadCompleted = "MailboxDownloadCompleted",
    MailboxDownloadSkipped = "MailboxDownloadSkipped",
    MailboxDownloadError = "MailboxDownloadError",
    MailboxDownloadProgress = "MailboxDownloadProgress",
    MailboxSyncWarning = "MailboxSyncWarning",
}
export class Options {
    email?: string;
    password?: string;
    imap_server?: string;
    strict_mail_parsing?: boolean;
    imap_client_debug?: boolean;
    debug?: boolean;
    limit_to_mailboxes?: string[];
    skip_mailboxes?: string[];
    db_path?: string;
    max_pool_size?: number;

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.email = source["email"];
        this.password = source["password"];
        this.imap_server = source["imap_server"];
        this.strict_mail_parsing = source["strict_mail_parsing"];
        this.imap_client_debug = source["imap_client_debug"];
        this.debug = source["debug"];
        this.limit_to_mailboxes = source["limit_to_mailboxes"];
        this.skip_mailboxes = source["skip_mailboxes"];
        this.db_path = source["db_path"];
        this.max_pool_size = source["max_pool_size"];
    }
}
export class AttachmentMetaData {
    FileName: string;
    FileType: string;
    FileSubType: string;
    FileSize: number;
    Encoding: string;
    Disposition: string;

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.FileName = source["FileName"];
        this.FileType = source["FileType"];
        this.FileSubType = source["FileSubType"];
        this.FileSize = source["FileSize"];
        this.Encoding = source["Encoding"];
        this.Disposition = source["Disposition"];
    }
}
export class Address {
    PersonalName: string;
    AtDomainList: string;
    MailboxName: string;
    HostName: string;

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.PersonalName = source["PersonalName"];
        this.AtDomainList = source["AtDomainList"];
        this.MailboxName = source["MailboxName"];
        this.HostName = source["HostName"];
    }
}
export class Time {


    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);

    }
}
export class Envelope {
    Date: Time;
    Subject: string;
    From: Address[];
    Sender: Address[];
    ReplyTo: Address[];
    To: Address[];
    Cc: Address[];
    Bcc: Address[];
    InReplyTo: string;
    MessageId: string;

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.Date = this.convertValues(source["Date"], Time);
        this.Subject = source["Subject"];
        this.From = this.convertValues(source["From"], Address);
        this.Sender = this.convertValues(source["Sender"], Address);
        this.ReplyTo = this.convertValues(source["ReplyTo"], Address);
        this.To = this.convertValues(source["To"], Address);
        this.Cc = this.convertValues(source["Cc"], Address);
        this.Bcc = this.convertValues(source["Bcc"], Address);
        this.InReplyTo = source["InReplyTo"];
        this.MessageId = source["MessageId"];
    }

	convertValues(a: any, classs: any, asMap: boolean = false): any {
	    if (!a) {
	        return a;
	    }
	    if (a.slice) {
	        return (a as any[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                a[key] = new classs(a[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class Email {
    message_id?: string;
    date?: string;
    subject?: string;
    from_name_1?: string;
    from_mailbox_1?: string;
    from_host_1?: string;
    sender_name_1?: string;
    sender_mailbox_1?: string;
    sender_host_1?: string;
    reply_to_name_1?: string;
    reply_to_mailbox_1?: string;
    reply_to_host_1?: string;
    to_name_1?: string;
    to_mailbox_1?: string;
    to_host_1?: string;
    cc_name_1?: string;
    cc_mailbox_1?: string;
    cc_host_1?: string;
    bcc_name_1?: string;
    bcc_mailbox_1?: string;
    bcc_host_1?: string;
    in_reply_to?: string;
    mailboxes?: string[];
    parse_warning?: string;
    parse_error?: string;
    our_id?: string;
    envelope?: Envelope;
    flags?: string[];
    uid?: number;
    text_content?: string;
    html_content?: string;
    attachments?: AttachmentMetaData[];

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.message_id = source["message_id"];
        this.date = source["date"];
        this.subject = source["subject"];
        this.from_name_1 = source["from_name_1"];
        this.from_mailbox_1 = source["from_mailbox_1"];
        this.from_host_1 = source["from_host_1"];
        this.sender_name_1 = source["sender_name_1"];
        this.sender_mailbox_1 = source["sender_mailbox_1"];
        this.sender_host_1 = source["sender_host_1"];
        this.reply_to_name_1 = source["reply_to_name_1"];
        this.reply_to_mailbox_1 = source["reply_to_mailbox_1"];
        this.reply_to_host_1 = source["reply_to_host_1"];
        this.to_name_1 = source["to_name_1"];
        this.to_mailbox_1 = source["to_mailbox_1"];
        this.to_host_1 = source["to_host_1"];
        this.cc_name_1 = source["cc_name_1"];
        this.cc_mailbox_1 = source["cc_mailbox_1"];
        this.cc_host_1 = source["cc_host_1"];
        this.bcc_name_1 = source["bcc_name_1"];
        this.bcc_mailbox_1 = source["bcc_mailbox_1"];
        this.bcc_host_1 = source["bcc_host_1"];
        this.in_reply_to = source["in_reply_to"];
        this.mailboxes = source["mailboxes"];
        this.parse_warning = source["parse_warning"];
        this.parse_error = source["parse_error"];
        this.our_id = source["our_id"];
        this.envelope = this.convertValues(source["envelope"], Envelope);
        this.flags = source["flags"];
        this.uid = source["uid"];
        this.text_content = source["text_content"];
        this.html_content = source["html_content"];
        this.attachments = this.convertValues(source["attachments"], AttachmentMetaData);
    }

	convertValues(a: any, classs: any, asMap: boolean = false): any {
	    if (!a) {
	        return a;
	    }
	    if (a.slice) {
	        return (a as any[]).map(elem => this.convertValues(elem, classs));
	    } else if ("object" === typeof a) {
	        if (asMap) {
	            for (const key of Object.keys(a)) {
	                a[key] = new classs(a[key]);
	            }
	            return a;
	        }
	        return new classs(a);
	    }
	    return a;
	}
}
export class MailboxRecord {
    name: string;
    last_synced: number;
    attributes: string[];
    num_emails: number;

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.name = source["name"];
        this.last_synced = source["last_synced"];
        this.attributes = source["attributes"];
        this.num_emails = source["num_emails"];
    }
}
export class MailboxEvent {
    Mailbox: string;
    TotalToDownload: number;
    TotalDownloaded: number;
    Error: string;
    Warning: string;
    EventType: MailboxEventType;

    constructor(source: any = {}) {
        if ('string' === typeof source) source = JSON.parse(source);
        this.Mailbox = source["Mailbox"];
        this.TotalToDownload = source["TotalToDownload"];
        this.TotalDownloaded = source["TotalDownloaded"];
        this.Error = source["Error"];
        this.Warning = source["Warning"];
        this.EventType = source["EventType"];
    }
}