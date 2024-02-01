import {createContext, Fragment, ReactNode,  useContext, useEffect, useState} from 'react';
import type { RuleGroupType } from 'react-querybuilder';
import 'react-querybuilder/dist/query-builder.scss';
import { QueryBuilder,formatQuery } from 'react-querybuilder';
import {AttachmentMetaData, Email} from "./goGeneratedModels";
import {getEmails, getPersistedState, persistState, searchEmails} from "./api";
import { toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import {asError,buttonClass,useDebounce} from "./utils";
import ReactModal from 'react-modal';
import {Modal} from "./Modal";
import { FixedSizeList as List } from 'react-window';
import {FileIcon, defaultStyles} from 'react-file-icon';
import {Spinner} from "./common";
import {defaultPersistedState, PersistedState, SavedQuery} from './types';

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


const SaveQueryModal = () => {

    const {savedQueryModalState,setSavedQueryModalState,query,persistedState,setPersistedState,sql} = useEmailContext();

    const dateAsNumber = new Date().getTime();
    const saveQuery = async ()=>{

        if(savedQueryModalState.name.length==0){
            toast.error("Name cannot be empty");
            return;
        }

        setSavedQueryModalState({...savedQueryModalState,loading:true});
        try{
            const newSavedQuery:SavedQuery = {name:savedQueryModalState.name,query:query,date:dateAsNumber,sqlForDisplay:sql};
            console.log(query.rules[0])
            const newPersistedState = {...persistedState,queries:[...persistedState.queries,newSavedQuery]};
            await persistState(newPersistedState);
            setPersistedState(newPersistedState);
            toast.success("Saved Query");
            setSavedQueryModalState({...savedQueryModalState,loading:false,show:false,name:""});
        }catch(e){
            toast.error(asError(e).message);
            console.error(e);
            setSavedQueryModalState({...savedQueryModalState,loading:false});
        }
    }
    //export const Modal=({open,closeReq,title,content}:{open:boolean,closeReq:()=>void,title:string,content:ReactNode})=>{
    const modalContent = <div className={"flex flex-col"}>
        <div className={"flex flex-row justify-between"}>
            <div className={"text-lg"}>Save Query</div>
        </div>
        <div className={"flex flex-col"}>
            <div className={"text-lg"}>Name</div>
            <input className={"border p-2"} value={savedQueryModalState.name} onChange={(e)=>{setSavedQueryModalState({...savedQueryModalState,name:e.target.value})}}/>
        </div>
        <div className={"flex flex-col"}>
            <div className={"text-lg"}>Query</div>
            <div className={"border p-2 bg-gray-100"}><pre>{sql}</pre></div>
        </div>
        <div className={"flex flex-row justify-end"}>
            <button className={buttonClass('green')} onClick={saveQuery}>Save</button>
        </div>
    </div>

    return <Modal title={"Save Query"} content={modalContent} open={savedQueryModalState.show} closeReq={()=>{setSavedQueryModalState({...savedQueryModalState,show:false})}}/>
}


const Cell = ({ content,onClick,extraClass,title }: { content: string|ReactNode,onClick?:()=>void,extraClass?:string,title?:string }) => {

    let className = `flex flex-grow table-cell border p-2 truncate`;
    if(extraClass){
        className+=" "+extraClass;
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


    // the reason we treat text as html is when it's returned from search, we highlight the search terms using html
    let dangerousRawElement;
    if(email.text_content){
        //split by \r and \n and then join with <br/> to make it render like a text area
        const lines = email.text_content.split(/\r?\n/);
        const htmlLines = lines.map((line,i)=>{
            return <div dangerouslySetInnerHTML={{__html:line}} key={i}></div>
        })
        dangerousRawElement = <div className={"whitespace-pre-wrap"}>{htmlLines}</div>

    } else{
        dangerousRawElement = <div>No Text Content</div>
    }


    const textModal = <Modal title={"Text Content: "+email.subject} content={dangerousRawElement} open={modalOpen} closeReq={()=>{setModalOpen(false)}}/>
    const textTrimmed = extractTextFromHTML(email.text_content)?.substring(0,300);



    let cellContent = <div className={"text-center"} title={textTrimmed ? textTrimmed:""}><button className={buttonClass('blue')} onClick={()=>{setModalOpen(true)}}>View Text</button>
        {textModal}</div>;

    if(!email.text_content){
        cellContent = <div className={"text-center"}><button disabled={true} className={buttonClass('gray')} >No Text</button></div>
    }

    return <Cell content={cellContent} title={textTrimmed}/>
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
    const attachmentNamesAsList = email?.attachments?.map((attachment:AttachmentMetaData)=>attachment?.FileName).join(", ").substring(0,300)


    return <Cell content={cellContent} title={attachmentNamesAsList}/>
}


const extractTextFromHTML = (html?:string)=>{

    if (!html) {
        return "";
    }

    // html=<span class="bg-yellow-200 text-black>hello</span>
    // should return hello
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    return doc.body.textContent || "";
}

const Row = ({ index, style, data }:{index:number,style:React.CSSProperties,data:Email[]}) => {

    const {setQueryToSender} = useEmailContext();
    // for search formatting
    const fromRawHTML = `${data[index].from_mailbox_1}@${data[index].from_host_1}`
    const fromSpan = <span dangerouslySetInnerHTML={{__html:fromRawHTML}}></span>
    // const fromNameAsRawHtml = <> dangerouslySetInnerHTML={{__html:data[index].from_name_1}}</>
    const subjectAsRawHtml = <span dangerouslySetInnerHTML={{__html:data[index].subject||""}}></span>


    return (
        <div style={style} className="flex flex-row border-t border-gray-300">
            <Cell content={data[index].date} title={data[index].date} />
            <Cell content={subjectAsRawHtml} title={extractTextFromHTML(data[index].subject)} />
            <Cell content={data[index].mailboxes?.join(",")}/>
            <Cell onClick={()=>{setQueryToSender(data[index])}}
                  extraClass="hover:bg-gray-100"
                  content={fromSpan}  title={extractTextFromHTML(data[index].from_name_1)+": " +extractTextFromHTML(data[index].from_mailbox_1)+"@"+extractTextFromHTML(data[index].from_host_1)} />
            <TextCell email={data[index]} />
            <AttachmentCell email={data[index]} />
            <HtmlCell email={data[index]} />
        </div>);
}

const VirtualizedTable = ({ emails } : { emails: Email[]}) => {

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
                {props => <Row {...props} />}
            </List>
        </div>
    );
};

type SavedQueryModalState = {
    name:string,
    query:RuleGroupType,
    show:boolean,
    loading:boolean,
}

type ViewQueriesModalState = {
    show:boolean,
}

interface EmailContextType {
    searchQuery: string;
    setSearchQuery: (query: string) => void;
    emails: Email[];
    setEmails: (emails: Email[]) => void;
    sql: string;
    setSql: (sql: string) => void;
    showHelpModal: boolean;
    setShowHelpModal: (show: boolean) => void;
    queryLoading: boolean;
    setQueryLoading: (loading: boolean) => void;
    query: RuleGroupType;
    setQuery: (query: RuleGroupType) => void;
    setQueryToSender:(email:Email)=>void,
    runQuery: () => void;
    savedQueryModalState: SavedQueryModalState;
    setSavedQueryModalState: (state: SavedQueryModalState) => void;
    persistedState: PersistedState;
    setPersistedState: (state: PersistedState) => void;
    viewQueriesModalState: ViewQueriesModalState;
    setViewQueriesModalState: (state: ViewQueriesModalState) => void;
}


const EmailContext = createContext<EmailContextType | undefined>(undefined);

export const useEmailContext = () => {
    const context = useContext(EmailContext);
    if (context === undefined) {
        throw new Error('useEmailContext must be used within an EmailProvider');
    }
    return context;
};


export const EmailProvider = ({ children }: { children: React.ReactNode }) => {

    const initialQuery = { combinator: 'and', rules: [
            { field: 'from_host_1', operator: '=', value: 'norwegian.com' },
        ] };

    const emailArchiveFilterKey= 'emailArchiveFilter'

    const filter = localStorage.getItem(emailArchiveFilterKey);
    const [query, setQuery] = useState<RuleGroupType>(filter ? JSON.parse(filter) : initialQuery);
    const [emails, setEmails] = useState<Email[]>([]);
    const [queryLoading, setQueryLoading] = useState(false);
    const [searchQuery, setSearchQuery] = useState<string>('');
    const [sql, setSql] = useState('');
    const [showHelpModal, setShowHelpModal] = useState(false);
    const debouncedSearchQuery = useDebounce(searchQuery, 500);
    const [persistedState,setPersistedState]=useState<PersistedState>(defaultPersistedState);
    const [savedQueryModalState,setSavedQueryModalState]=useState<SavedQueryModalState>({name:'',query:initialQuery,show:false,loading:false});
    const [viewQueriesModalState,setViewQueriesModalState]=useState<ViewQueriesModalState>({show:false});

    const setQueryAndSave = (query: RuleGroupType) => {
        localStorage.setItem(emailArchiveFilterKey, JSON.stringify(query));
        setQuery(query);
    };


    useEffect(()=>{
        async function doSearch(){
            if(debouncedSearchQuery){
                try{
                    const emails=await searchEmails(debouncedSearchQuery);
                    setEmails(emails);
                    toast.success(`Loaded ${emails.length} emails (via search)`)
                } catch(e){
                    console.error(e);
                    toast.error(`Error searching emails: ${asError(e).message}`);
                }
            }
        }
        doSearch();
    },[debouncedSearchQuery])


    useEffect(()=>{
        ReactModal.setAppElement('#root');
        const loadSavedQueries=async()=>{
            try{
                const persistedState = await getPersistedState();
                setPersistedState(persistedState);
            } catch(e){
                    console.error(e);
                    toast.error(`Error loading saved queries: ${asError(e).message}`);
            }
        }
        loadSavedQueries();

    },[])

    const runQuery = async ()=>{
        try{
            setQueryLoading(true)
            const emails = await getEmails(sql)
            setEmails(emails)
            toast.success(`Loaded ${emails.length} emails (via query)`)
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
    },[query])

    const setQueryToSender = (email:Email)=> {
        let rules = [...query.rules]

        rules = rules.filter(rule => {
            console.log(rule)
            // cast to object of {field:string,operator:string,value:string}, this is the actual type.
            // the type script type is incorrect
            const ruleCasted = rule as { field: string, operator: string, value: string }
            return !["from_name_1", "from_mailbox_1", "from_host_1"].includes(ruleCasted.field)
        })

        rules.push({
            field: "from_name_1",
            operator: "=",
            value: email.from_name_1
        })

        rules.push({
            field: "from_host_1",
            operator: "=",
            value: email.from_host_1
        })

        const newQuery = Object.assign({}, query);
        newQuery.rules = rules;
        setQueryAndSave(newQuery);
    }

    return (
        <EmailContext.Provider
            value={{
                emails,
                setEmails,
                queryLoading,
                setQueryLoading,
                query,
                setQuery: setQueryAndSave,
                searchQuery,
                setSearchQuery,
                sql,
                setSql,
                showHelpModal,
                setShowHelpModal,
                setQueryToSender,
                runQuery,
                savedQueryModalState,
                setSavedQueryModalState,
                persistedState,
                setPersistedState,
                viewQueriesModalState,
                setViewQueriesModalState
            }}
        >
            {children}
        </EmailContext.Provider>
    );
};

const ViewQueriesModal = ()=> {
    const {persistedState, viewQueriesModalState,setPersistedState,setViewQueriesModalState,runQuery,setQuery,setSql} = useEmailContext();

    const {queries} = persistedState;

    const deleteQuery = async (query:SavedQuery)=>{
        try{
            const newPersistedState = {...persistedState};
            newPersistedState.queries = newPersistedState.queries.filter(q=>q.name!==query.name);
            await persistState(newPersistedState);
            setPersistedState(newPersistedState);
            toast.success(`Deleted query ${query.name}`)

            if (newPersistedState.queries.length === 0){
                setViewQueriesModalState({...viewQueriesModalState,show:false});
            }

        } catch(e){
            console.error(e);
            toast.error(`Error deleting query ${query.name}: ${asError(e).message}`)
        }
    }


    // export type SavedQuery = {
    //     name: string
    //     query: RuleGroupType
    //     date: number
    //     sqlForDisplay: string
    // }


    // show saved query. Each query has a name, a query, and a date. Sort by date. Date is an epoch number in ms. Convert to date string for display.
    // each saved query should have header style name, date, delete button and then the query itself (not editable, gray background)
    const runSavedQuery = (query:SavedQuery)=>{
        setQuery(query.query);

        console.log(query.sqlForDisplay)
        setSql(query.sqlForDisplay)
        runQuery();
        setViewQueriesModalState({...viewQueriesModalState,show:false});
    }

    let modalContent = <div>
        <div className="flex justify-between">
            <h2 className="text-2xl">Saved Queries</h2>
        </div>
        <div className="flex flex-col space-y-4 mt-4"> {/* Added space between sections */}
            {queries
                .sort((a, b) => b.date - a.date) // Sort by date descending
                .map((query, i) => {
                    const formattedDate = new Date(query.date).toLocaleString("en-US"); // Format date
                    return (
                        <div key={i} className="border-2 rounded-lg p-4 bg-white"> {/* Added border and padding */}
                            <div className="flex justify-between items-center mb-4"> {/* Adjusted margin */}
                                <div>
                                    <h3 className="font-semibold">{query.name}</h3>
                                    <p className="text-sm">{formattedDate}</p>
                                </div>
                                <button className="bg-red-500 hover:bg-red-700 text-white font-bold py-1 px-2 rounded" onClick={()=>deleteQuery(query)}>Delete</button>
                                {/*run query button*/}
                                <button className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-1 px-2 rounded" onClick={()=>{runSavedQuery(query)}}>Run</button>
                            </div>
                            <div className="bg-gray-200 p-2 rounded">
                                <pre>{query.sqlForDisplay}</pre>
                            </div>
                        </div>
                    )
                })}
        </div>
    </div>


    if (queries.length === 0){
        modalContent = <div>
            <div className="flex justify-between">
                <h2 className="text-2xl">Saved Queries</h2>
            </div>
            <div className="flex flex-col space-y-4 mt-4"> {/* Added space between sections */}
                <p>No saved queries. In order to save a query, build the query you want and then press "Save Query".</p>
            </div>
        </div>
    }



    return (
        <Modal
            open={viewQueriesModalState.show}
            closeReq={() => setViewQueriesModalState({show: false})}
            title="Saved Queries"
            content={modalContent}
        />
    );


}


export const EmailArchiver = ()=>{
    const {persistedState,savedQueryModalState,setSavedQueryModalState,runQuery,emails, queryLoading, query, setQuery, searchQuery, setSearchQuery, sql, showHelpModal, setShowHelpModal,setViewQueriesModalState} = useEmailContext();

    const archiveAll = async (emails:Email[])=>{
        toast.info("TODO")
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



    let spinner:ReactNode = null;
    if(queryLoading){
        spinner = <Spinner/>
    }

    return (
        <Fragment>
            <div className="flex p-2">
                <div className="flex-1 text-center">
                     <QueryBuilder fields={fields} query={query} onQueryChange={setQuery}/>
                    </div>
                <div>
                    <button  style={{background:"#ccdbf1", color:"black"}}   className={buttonClass('#ccdbf1')} onClick={runQuery}>Run Query</button>
                    <button  style={{background:"#ccdbf1", color:"black"}}   className={buttonClass('#ccdbf1')} onClick={()=>setSavedQueryModalState({...savedQueryModalState,show:true})}>Save Query</button>
                    <button  style={{background:"#ccdbf1", color:"black"}}   className={buttonClass('#ccdbf1')} onClick={()=>setViewQueriesModalState({show:true})}>View Saved Queries</button>
                </div>
                <div className="space-y-4 flex-1 text-center p-2">
                <input className="border border-gray-300 p-2 rounded-md" value={searchQuery} onChange={(e)=>setSearchQuery(e.target.value)} placeholder="Search"/>
                </div>
            </div>
            <div className="p-4">
                <div className="flex flex-col ">
                    <div className="flex justify-start items-center">
                        <div><pre className="bg-gray-100 p-4 rounded-md overflow-auto">{sql}</pre></div>
                        <button className={buttonClass('blue')} onClick={()=>setShowHelpModal(true)}>
                            Help
                        </button>
                        <button className={buttonClass('blue')} onClick={()=>archiveAll(emails)}>
                            Archive All
                        </button>
                        {spinner}
                    </div>
                    <VirtualizedTable emails={emails}/>
                </div></div>
            <Modal title={"Email Archiver Help"} open={showHelpModal} closeReq={()=>setShowHelpModal(false)} content={helpContent}/>
            <SaveQueryModal/>
            <ViewQueriesModal/>
        </Fragment>
    );
}