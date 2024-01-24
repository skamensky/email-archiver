import { useState } from 'react';
import { QueryBuilderDnD } from '@react-querybuilder/dnd';
import * as ReactDnD from 'react-dnd';
import * as ReactDndHtml5Backend from 'react-dnd-html5-backend';
import type { RuleGroupType } from 'react-querybuilder';
import { QueryBuilder } from 'react-querybuilder';
import 'react-querybuilder/dist/query-builder.scss';

// our_id text primary key,parse_warning text,parse_error text,envelope text,flags text,mailboxes text,text_content text,html_content text,attachments text,message_id text,date text,subject text,from_name_1 text,from_mailbox_1 text,from_host_1 text,sender_name_1 text,sender_mailbox_1 text,sender_host_1 text,reply_to_name_1 text,reply_to_mailbox_1 text,reply_to_host_1 text,to_name_1 text,to_mailbox_1 text,to_host_1 text,cc_name_1 text,cc_mailbox_1 text,cc_host_1 text,bcc_name_1 text,bcc_mailbox_1 text,bcc_host_1 text,in_reply_to text);

const fields = [
    { name: 'our_id', label: 'Our ID'},
    { name: 'flags', label: 'Flags'},
    { name: 'mailboxes', label: 'Mailboxes'},
    { name: 'text_content', label: 'Text Content'},
    { name: 'html_content', label: 'Html Content'},
    { name: 'attachments', label: 'Attachments'},
    { name: 'date', label: 'Date'},
    { name: 'subject', label: 'Subject'},

    { name: 'from_name_1', label: 'From Name'},
    { name: 'from_mailbox_1', label: 'From Mailbox'},
    { name: 'from_host_1', label: 'From Host'},

    { name: 'sender_name_1', label: 'Sender Name'},
    { name: 'sender_mailbox_1', label: 'Sender Mailbox'},
    { name: 'sender_host_1', label: 'Sender Host'},

    { name: 'reply_to_name_1', label: 'Reply To Name'},
    { name: 'reply_to_mailbox_1', label: 'Reply To Mailbox'},
    { name: 'reply_to_host_1', label: 'Reply To Host'},

    { name: 'to_name_1', label: 'To Name'},
    { name: 'to_mailbox_1', label: 'To Mailbox'},
    { name: 'to_host_1', label: 'To Host'},

    { name: 'cc_name_1', label: 'Cc Name'},
    { name: 'cc_mailbox_1', label: 'Cc Mailbox'},
    { name: 'cc_host_1', label: 'Cc Host'},

    { name: 'bcc_name_1', label: 'Bcc Name'},
    { name: 'bcc_mailbox_1', label: 'Bcc Mailbox'},
    { name: 'bcc_host_1', label: 'Bcc Host'},


    { name: 'in_reply_to', label: 'In Reply To'},

]

export const EmailArchiver = ()=>{
    const initialQuery: RuleGroupType = { combinator: 'and', rules: [
        { field: 'mailboxes', operator: 'contains', value: 'INBOX' },
        ] };

        const [query, setQuery] = useState(initialQuery);

        // type script complains about the type of newQuery
        const handleQueryChange = (newQuery: RuleGroupType) => {
            setQuery(newQuery);
            console.log(newQuery)
        };

        return (
            <QueryBuilderDnD dnd={{ ...ReactDnD, ...ReactDndHtml5Backend }}>
                <QueryBuilder fields={fields} query={query} onQueryChange={handleQueryChange} />
            </QueryBuilderDnD>
        );
}