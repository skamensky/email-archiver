const server = 'http://localhost:8080';


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
export const getOptions = async (): Promise<Options> => {
    const response = await fetch(`${server}/api/options`, {

    })
    const json = await response.json();
    console.log(json);
    return new Options(json);
}