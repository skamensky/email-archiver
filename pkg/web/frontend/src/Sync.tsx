import {MailboxEventType, MailboxRecord, Options} from "./goGeneratedModels";
import {useEffect, useState} from "react";
import {getMailboxRecords, syncMailboxes} from "./api";
import {toast} from "react-toastify";
import {MailboxToSync, ProgressBar, Spinner} from "./common";
import {buttonClass, buttonClassDisabled} from "./utils";
import {sync} from "framer-motion";

const SyncCell = (props: {queued:boolean,mailboxRecord: MailboxRecord,relistMailboxes:()=>void,allSyncing:boolean,doneSyncingDuringBulkSync:boolean})=>{

    const [isSyncing, setIsSyncing] = useState<boolean>(false);
    let buttonText = isSyncing ? "Syncing..." : "Sync Now";
    let _buttonClass = isSyncing ? buttonClassDisabled() : buttonClass('blue');
    _buttonClass += " p-2";

    const checkmarkSvg = <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24"
                                         stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M5 13l4 4L19 7"/>
    </svg>

    const sync = async () => {
        setIsSyncing(true);
        try{
            await syncMailboxes([props.mailboxRecord.name])
            toast.success(`Synced ${props.mailboxRecord.name} successfully`);
            props.relistMailboxes();
        }
        catch(err){
            // @ts-ignore
            toast.error(err.message);
            console.error(err);
        } finally{
            setIsSyncing(false);
        }
    }

    const showSyncing = props.allSyncing || isSyncing;


    const className = cellClass(175)

    //cell flex-1 text-center truncate border bg-gray-100 border-gray-300 p-1 min-w-[175px]
    if (props.allSyncing){
        if (props.doneSyncingDuringBulkSync){
            return <div className={className}>
                {checkmarkSvg}
            </div>
        }
        else if (props.queued){
            return  <div className={className+' text-blue-500'}>
                Queued...
            </div>
        } else{
            return <div className={className}>
                <Spinner size={'medium'} />
            </div>
        }
    } else{
        if (showSyncing){
            return <div className={className}>
                <Spinner size={'medium'} />
            </div>
        } else{
            return <div className={className}>
                <button onClick={sync} disabled={showSyncing} className={_buttonClass}>{buttonText}</button>
            </div>
        }
    }

}



export const Sync=({mailboxToSycState,options,resetSyncState}:{mailboxToSycState?:MailboxToSync,options?:Options,resetSyncState:()=>void})=>{

    const [mailboxRecords, setMailboxRecords] = useState<MailboxRecord[]>([]);
    const [loading, setLoading] = useState<boolean>(false);
    const [allSyncing, setAllSyncing] = useState<boolean>(false);

    useEffect(() => {
        relistMaiboxes();
        resetSyncState();
    }, []);


    const syncAll = async () => {
        setAllSyncing(true);
        try{
            await syncMailboxes(mailboxRecords.map((record)=>{return record.name}));
            //relistMaiboxes also calls resetSyncState
            // await relistMaiboxes();
            toast.success(`Synced all mailboxes successfully`);
        }
        catch(err){
            // @ts-ignore
            toast.error(err.message);
            console.error(err);
        } finally{
            setAllSyncing(false);
        }
    }

    const relistMaiboxes = async()=>{
        setLoading(true);
        try{
            const records = await getMailboxRecords();
            setMailboxRecords(records.sort((a, b) => {
                // if the name is inbox, it goes first, otherwise sort by name
                if (a.name.toLowerCase() === "inbox") {
                    return -1;
                }
                if (b.name.toLowerCase() === "inbox") {
                    return 1;
                }
                return a.name.localeCompare(b.name);
            }));

            resetSyncState();
        } catch (err){
            //@ts-ignore
            toast.error(err.message);
            //@ts-ignore
            console.error(err)
        } finally {
            setLoading(false);
        }

    }

    let spinner: JSX.Element | null = null;
    if (loading) {
        spinner = <Spinner/>;
    }

    const rows=mailboxRecords.map((record) => {
        let timestampMessage:string = "";
        if(record.last_synced){
            const timestamp = new Date(Number(record.last_synced*1000));
            const now = new Date();
            const diff = now.getTime() - timestamp.getTime();
            const days = Math.floor(diff / (1000 * 60 * 60 * 24));
            const hours = Math.floor(diff / (1000 * 60 * 60));
            const minutes = Math.floor(diff / (1000 * 60));
            if(days > 0){
                timestampMessage = `${days} days ago`;
            }else if(hours > 0){
                timestampMessage = `${hours} hours ago`;
            }
            else if(minutes==1){
                timestampMessage = `${minutes} minute ago`;
            }
            else if(minutes > 0){
                timestampMessage = `${minutes} minutes ago`;
            }else{
                timestampMessage = "Just Now";
            }
        } else{
            timestampMessage = "Never Synced";
        }


        let lastEvent = mailboxToSycState?.[record.name];
        let progressBar;
        let syncCell ;

        if(lastEvent){
            if(lastEvent.data.EventType==MailboxEventType.MailboxDownloadProgress && lastEvent.data.TotalToDownload !=lastEvent.data.TotalDownloaded){
                progressBar = <ProgressBar progress={lastEvent.data.TotalDownloaded} maxValue={lastEvent.data.TotalToDownload} />
                syncCell = <SyncCell queued={false} mailboxRecord={record} relistMailboxes={relistMaiboxes} allSyncing={allSyncing} doneSyncingDuringBulkSync={false} />
            }
            if(lastEvent?.data.EventType==MailboxEventType.MailboxSyncQueued){
                syncCell = <SyncCell queued={true} mailboxRecord={record} relistMailboxes={relistMaiboxes} allSyncing={allSyncing} doneSyncingDuringBulkSync={false} />
            } else if(lastEvent?.data.EventType==MailboxEventType.MailboxDownloadCompleted){
                syncCell = <SyncCell queued={false} mailboxRecord={record} relistMailboxes={relistMaiboxes} allSyncing={allSyncing} doneSyncingDuringBulkSync={true} />
            } else{
                syncCell = <div className={cellClass(175)+" text-gray-500"} >Processing...</div>
            }
        } else{
            syncCell = <SyncCell queued={false} mailboxRecord={record} relistMailboxes={relistMaiboxes} allSyncing={allSyncing} doneSyncingDuringBulkSync={false} />
        }
        if(options?.skip_mailboxes?.includes(record.name)){
            syncCell = <div className={cellClass(175)+" text-gray-500"} >Skipped</div>
        }
        // syncCell = <div className={cellClass(175)}>Skipped</div>


        const attributesNice = record.attributes?.map(a=>a.replace("\\","")).join(", ");
            return  <div key={record.name}  className="row flex border-b border-gray-300">
            <div title={record.name} className={cellClass(300)}>{record.name}</div>
            <div className={cellClass(175)}>{timestampMessage}</div>
            <div className={cellClass(175)}>{record.num_emails}</div>
            <div title={attributesNice} className={cellClass(225)}>{attributesNice}</div>
            {syncCell}
            <div className={cellClass(175)}>{progressBar}</div>
        </div>
    })

    // todo show number of messages in mailbox
    return <> <div className={"flex"}>
        <h1 className="text-2xl p-2">Refresh Mailbox Data</h1>
        <div className={"p-2"} ><button onClick={relistMaiboxes} className={buttonClass('blue')}>Refresh</button>{spinner}</div>
        <div className={"p-2"}><button onClick={syncAll} className={buttonClass('blue')}>Sync All</button></div>


    </div>
        <div className="table flex flex-col m-1">

            <div className="row header flex ">
                <div className={headerClass(300)}>Mailbox</div>
                <div className={headerClass(175)}>Last Sync</div>
                <div className={headerClass(175)}>Number Of Emails</div>
                <div className={headerClass(225)}>Attributes</div>
                <div className={headerClass(175)}>Sync Now</div>
                <div className={headerClass(175)}>Sync Progress</div>
            </div>
            {rows}
    </div>
    </>;
}

const cellClass=(size:number)=>{
    return `cell flex-1 truncate border border-gray-200 p-1 min-w-[${size}px]`
}

const headerClass=(size:number)=>{
    return cellClass(size)+" font-bold"
}