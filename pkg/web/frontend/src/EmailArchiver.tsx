import {Fragment, ReactNode, useEffect, useState} from 'react';
import { QueryBuilderDnD } from '@react-querybuilder/dnd';
import * as ReactDnD from 'react-dnd';
import * as ReactDndHtml5Backend from 'react-dnd-html5-backend';
import type { RuleGroupType } from 'react-querybuilder';
import 'react-querybuilder/dist/query-builder.scss';
import { QueryBuilder,formatQuery } from 'react-querybuilder';
import {AttachmentMetaData, Email} from "./goGeneratedModels";
import {getEmails} from "./api";
import { toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import {asError,buttonClass} from "./utils";
import ReactModal from 'react-modal';
import {Modal} from "./Modal";
import { FixedSizeList as List } from 'react-window';
import {FileIcon, defaultStyles} from 'react-file-icon';

const Spinner = () => {
    return (
        <div className="border-4 border-gray-200 rounded-full w-8 h-8 border-t-4 border-t-blue-500 animate-spin"></div>
    );
};

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



const filterSerializer = (ruleGroup: RuleGroupType) :string=> {
    return encodeURIComponent(btoa(JSON.stringify(ruleGroup)));
};

const filterDeserializer = (query:string) : RuleGroupType => {
    return JSON.parse(atob(decodeURIComponent(query)));
};


const Cell = ({ content,onClick,extraClass }: { content: string|ReactNode,onClick?:()=>void,extraClass?:string }) => {

    let className = `flex flex-grow table-cell border p-2 truncate`;
    if(extraClass){
        className+=" "+extraClass;
    }
    let title = ""
    if(typeof content === "string"){
        title = content;
    }
    return <div title={title}  className={className} onClick={onClick} style={{width:0}}>{content}</div>

}

const HtmlCell = ({email}:{email:Email})=>{




    const [modalOpen,setModalOpen] = useState(false);

    let dangerousRawElement;
    if(email.html_content){
        dangerousRawElement = <div dangerouslySetInnerHTML={{__html:email.html_content}}></div>
    } else{
        dangerousRawElement = <div>No HTML Content</div>
    }
    const htmlModal = <Modal title={"HTML Content: "+email.subject} content={dangerousRawElement} open={modalOpen} closeReq={()=>{setModalOpen(false)}}/>
    let celContent=<div className={"text-center"}><button className={buttonClass('blue')} onClick={()=>{setModalOpen(true)}}>View HTML</button>
        {htmlModal}</div>

    if(!email.html_content){
        celContent = <div className={"text-center"}><button disabled={true} className={buttonClass('gray')} >No HTML</button></div>
    }

    return <Cell content={celContent}/>
}

const TextCell = ({email}:{email:Email})=>{
    const [modalOpen,setModalOpen] = useState(false);

    const textModal = <Modal title={"Text Content: "+email.subject} content={<pre>{email.text_content}</pre>} open={modalOpen} closeReq={()=>{setModalOpen(false)}}/>
    const textTrimmed = email.text_content?.substring(0,100);
    let cellContent = <div className={"text-center"} title={textTrimmed ? textTrimmed:""}><button className={buttonClass('blue')} onClick={()=>{setModalOpen(true)}}>View Text</button>
        {textModal}</div>;

    if(!email.text_content){
        cellContent = <div className={"text-center"}><button disabled={true} className={buttonClass('gray')} >No Text</button></div>
    }

    return <Cell content={cellContent}/>
}

const AttachmentCell = ({email}:{email:Email})=>{
    // the most common file type for missing extensions
    const commonMissingExtensions:{[key:string]:string} = {
        'image/jpg':'jpg',
        'image/gif':'gif',
        'image/png':'png',
        'image/jpeg':'jpeg',
    }

    const extensions:Record<number, string>= {}

    if (email?.attachments) {
        for(let i=0;i<email.attachments.length;i++){

            const attachment = email.attachments[i];
            let extension = attachment?.FileName.split('.').pop()?.replaceAll(/[^a-zA-Z0-9]/g, '');
            if (!extension) {
                extension = commonMissingExtensions[attachment?.FileType];
            }
            if (!extension) {
                extension = 'unknown';
            }
            extensions[i] = extension;
        }
    }

    const attachmentsIcons = Object.entries(extensions).map(([attachmentIndex,extension]:[string,string],index:number)=>{
        const attachment = email?.attachments?.[Number(attachmentIndex)];
        let title="";
        if(attachment?.FileName){
            title = attachment.FileName;
        }
        // @ts-ignore
        const fileIcon = <FileIcon extension={extension} {...defaultStyles[extension]} />
        return <div title={title} className={"max-w-6"} key={index}>
            {fileIcon}
        </div>
    });

    const attachmentModalContent = Object.entries(extensions).map(([attachmentIndex,extension]:[string,string],index:number)=>{
        const attachment = email?.attachments?.[Number(attachmentIndex)];
        // @ts-ignore
        const fileIcon = <FileIcon extension={extension} {...defaultStyles[extension]} />

        let fileSizeMessage = "";
        if(attachment?.FileSize){
            const fileSizeOrderOfMagnitude = Math.floor(Math.log(attachment?.FileSize)/Math.log(1024));
            const fileSizeNice = (attachment?.FileSize/Math.pow(1024,fileSizeOrderOfMagnitude)).toFixed(2);
            fileSizeMessage = fileSizeNice+" "+['B','KB','MB','GB','TB'][fileSizeOrderOfMagnitude];
        }


        return <div className={"flex flex-row"} key={index}>
            <div className="max-w-6">{fileIcon}</div>
            <div className={"ml-2"}>{attachment?.FileName}</div>
            <div className={"ml-2"}>{fileSizeMessage}</div>
        </div>
    })

    const [modalOpen,setModalOpen] = useState(false);
    const attachmentModal = <Modal title={"Attachments: "+email.subject} content={<div>{attachmentModalContent}</div>} open={modalOpen} closeReq={()=>{setModalOpen(false)}}/>


    let attachmentMessage = `View ${email?.attachments?.length} Attachments`;
    if(email?.attachments?.length === 1){
        attachmentMessage = `View Attachment`;
    }
    // todo get attachmentsIcons to play nicely with the button. Right now only icons will show if we include both icons and button
    let cellContent = <div className={"text-center"}>
        <div><button className={buttonClass('blue')} onClick={()=>{setModalOpen(true)}}>{attachmentMessage}</button></div>
        {attachmentModal}</div>;

    if(!email.attachments){
        cellContent = <div className={"text-center"}><button disabled={true} className={buttonClass('gray')} >No Attachments</button></div>
    }

    return <Cell content={cellContent}/>
}

const Row = ({ index, style, data,setQueryToSender }:{index:number,style:React.CSSProperties,data:Email[],setQueryToSender:(email:Email)=>void}) => {
    return (
        <div style={style} className="flex flex-row border-t border-gray-300">
            <Cell content={data[index].date} />
            <Cell content={data[index].subject}/>
            <Cell content={data[index].mailboxes?.join(",")}/>
            <Cell onClick={()=>{setQueryToSender(data[index])}}
                  extraClass="hover:bg-gray-100"
                  content={data[index].from_mailbox_1+"@"+data[index].from_host_1}  />
            <TextCell email={data[index]} />
            <AttachmentCell email={data[index]} />
            <HtmlCell email={data[index]} />
        </div>);
}

const VirtualizedTable = ({ emails,setQueryToSender } : { emails: Email[] ,setQueryToSender:(email:Email)=>void}) => {

    return (
        <div className="flex flex-col">
            <div className="flex sticky top-0 bg-white">
                <Cell content="Date"  extraClass="uppercase font-semibold text-sm border border-gray-300"/>
                <Cell content="Subject"  extraClass="uppercase font-semibold text-sm border border-gray-300"/>
                <Cell content="Mailboxes"  extraClass="uppercase font-semibold text-sm border border-gray-300"/>
                <Cell content="From"  extraClass="uppercase font-semibold text-sm border border-gray-300"/>
                <Cell content="Text Content"  extraClass="uppercase font-semibold text-sm border border-gray-300"/>
                <Cell content="Attachments"  extraClass="uppercase font-semibold text-sm border border-gray-300"/>
                <Cell content="Html Content" extraClass="uppercase font-semibold text-sm border border-gray-300"/>
            </div>
            <List
                height={600}
                itemCount={emails.length}
                itemSize={35}
                width={'100%'}
                itemData={emails}

            >
                {props => <Row {...props} setQueryToSender={setQueryToSender} />}
            </List>
        </div>
    );
};
export const EmailArchiver = ()=>{
    // const initialQuery: RuleGroupType = { combinator: 'and', rules: [
    //     { field: 'mailboxes', operator: 'contains', value: 'INBOX' },
    //     ] };


    // get 'filter' param from url:
    const urlParams = new URLSearchParams(window.location.search);
    const filterParam = urlParams.get('filter');
    let initialQuery: RuleGroupType;
    if(filterParam){
        initialQuery = filterDeserializer(filterParam);
    } else{
        initialQuery = { combinator: 'and', rules: [
                { field: 'sender_host_1', operator: '=', value: 'norwegian.com' },
            ] };
        // initialQuery = { combinator: 'and', rules: [
        //     { field: 'mailboxes', operator: 'contains', value: 'INBOX' },
        // ] };
    }
    const helpContent = <p>
        Use the filters to do discovery on the emails you want to archive. Once you have the filters set up, click the "Run Query" button to load the emails.
        <br/><br/>
        After you're done constructing the query, you can click "Archive All" to archive all the emails that match the query.
        <br/><br/>
        As a convenience, you can click on the sender to filter by sender.

        <h2 className="text-red-500 text-2xl text-center">HTML Warning</h2>
        <span>
            Clicking on 'View HTML' will load the HTML content of the email onto the page. There are two potential issues with this:
            <ol>
                <li className={"list decimal"}>This could load images from the internet, which could be used to track you.</li>
                <li className={"list decimal"}>This could load malicious javascript, which could be used to attack you (most email clients e.g. gmail block javascript)</li>
            </ol>
        </span>
    </p>

    const [query, setQuery] = useState(initialQuery);
    const [sql,setSql] = useState("");
    const [emails,setEmails] = useState<Email[]>([]);
    const [showHelpModal,setShowHelpModal] = useState(false);
    const [queryLoading,setQueryLoading] = useState(false);



    useEffect(()=>{
        ReactModal.setAppElement('#root');
    },[])

    const onRunQueryClick = async ()=>{
        try{
            setQueryLoading(true)
            const emails = await getEmails(sql)
            setEmails(emails)
            toast.success(`Loaded ${emails.length} emails`)
        }
        catch(e){
            toast.error(asError(e).message)
        } finally {
            setQueryLoading(false)
        }
    }

    useEffect(()=>{
        setSql(`SELECT  * FROM email WHERE ${formatQuery(query,
            {
                format:'sql'
            }
        )}`)
        const urlParams = new URLSearchParams(window.location.search);
        urlParams.set('filter',filterSerializer(query));
        window.history.replaceState({}, '', `${window.location.pathname}?${urlParams}`);
    },[query])

    // type script complains about the type of newQuery
    const handleQueryChange = (newQuery: RuleGroupType) => {
        setQuery(newQuery);
    };


    const setQueryToSender = (email:Email)=>{
        let rules = [...query.rules]

        rules = rules.filter(rule=>{
            console.log(rule)
            // cast to object of {field:string,operator:string,value:string}, this is the actual type.
            // the type script type is incorrect
            const ruleCasted = rule as {field:string,operator:string,value:string}
            return !["sender_name_1","sender_mailbox_1","sender_host_1"].includes(ruleCasted.field)
        })


        rules.push(
            { field: 'sender_mailbox_1', operator: '=', value: email.from_mailbox_1 },
            { field: 'sender_host_1', operator: '=', value: email.from_host_1 },
        )

        setQuery({...query,rules})

    }

    const archiveAll = async (emails:Email[])=>{
        toast.info("TODO")
    }

    const table = <VirtualizedTable emails={emails} setQueryToSender={setQueryToSender}/>

    let spinner:ReactNode = null;
    if(queryLoading){
        spinner = <Spinner/>
    }

    return (
        <Fragment>

            <QueryBuilderDnD dnd={{ ...ReactDnD, ...ReactDndHtml5Backend }}>
                <QueryBuilder fields={fields} query={query} onQueryChange={handleQueryChange}/>
            </QueryBuilderDnD>
            <div className="p-4">
                <div className="flex flex-col space-y-4">
                    <div className="flex justify-start items-center">
                        <div><pre className="bg-gray-100 p-4 rounded-md overflow-auto">{sql}</pre></div>
                        <button  className={buttonClass('blue')} onClick={onRunQueryClick}>Run Query</button>

                        <button className={buttonClass('blue')} onClick={()=>setShowHelpModal(true)}>
                            Help
                        </button>
                        <button className={buttonClass('blue')} onClick={()=>archiveAll(emails)}>
                            Archive All
                        </button>
                        {spinner}
                    </div>
                    {table}
                </div></div>
            <Modal title={"Email Archiver Help"} open={showHelpModal} closeReq={()=>setShowHelpModal(false)} content={helpContent}/>
        </Fragment>
    );
}