import React, {useEffect, useState} from 'react';
import './App.css';
import {BrowserRouter as Router, Link, Route, Routes} from "react-router-dom";
import {Sync} from "./Sync";
import {Settings} from "./Settings";
import {EmailArchiver,EmailProvider} from "./EmailArchiver";
import {toast, ToastContainer} from 'react-toastify';
import {LayoutPlayground} from "./LayoutPlayground";
import useWebSocket from 'react-use-websocket';
import {MailboxEventType, Options} from "./goGeneratedModels";
import {MailboxEventMessage, MailboxToSync} from "./common";
import {getOptions} from "./api";


function App() {

    const [options, setOptions] = useState<Options >();
    const [mailboxToSycState, setMailboxToSyncState] = useState<MailboxToSync>({} as MailboxToSync);

    const { sendMessage, lastMessage, readyState } = useWebSocket('ws://localhost:8080/ws',
        {
            shouldReconnect: (closeEvent) => true,
        });

    useEffect(() => {
        if (lastMessage !== null) {
            //@ts-ignore
            try{
                const lastMailboxEvent = JSON.parse(lastMessage.data) as MailboxEventMessage;
                if(lastMailboxEvent){
                    if(lastMailboxEvent.data.EventType === MailboxEventType.MailboxSyncWarning){
                        toast.warn(`Warning: ${lastMailboxEvent.data.Mailbox} - ${lastMailboxEvent.data.Warning}`,{ delay:4000 })
                    }
                    if (lastMailboxEvent.data.EventType === MailboxEventType.MailboxDownloadError){
                        toast.error(`Error: ${lastMailboxEvent.data.Mailbox} - ${lastMailboxEvent.data.Error}`,{ delay:4000 })
                    }
                    setMailboxToSyncState(prevState => ({...prevState,[lastMailboxEvent.data.Mailbox]:lastMailboxEvent}));
                }
            }
            catch(err){
                console.error(err);
            }

        }
    }, [lastMessage, setMailboxToSyncState]);


    useEffect(() => {
        async function getOptionsAndSet(){
            try{
                const options = await getOptions();
                setOptions(options);
            }
            catch(err){
                console.error(err);
            }
        }
        getOptionsAndSet();
    }, []);


    return (
        <EmailProvider>
            {/* we use context provider for email archiver instead of state so that state persists across route changes*/}
        <Router>
            <div className="flex flex-col h-screen">
                <nav className="flex bg-blue-500 text-white p-4 space-x-4">
                    <Link className="hover:bg-blue-700 px-3 py-2 rounded-md" to="/">Folders</Link>
                    <Link className="hover:bg-blue-700 px-3 py-2 rounded-md" to="/settings">Settings</Link>
                    <Link className="hover:bg-blue-700 px-3 py-2 rounded-md" to="/emails">Emails</Link>
                </nav>
                <div className="flex-grow">
                    <Routes>
                        <Route path="/" element={<Sync mailboxToSycState={mailboxToSycState} options={options} resetSyncState={()=>setMailboxToSyncState({})} />} />
                        <Route path="/settings" element={<Settings />}   />
                        <Route path="/emails" element={
                                <EmailArchiver />
                        } />
                        <Route path="/layout-playground" element={<LayoutPlayground />} />

                        <Route path="*" element={<div>Not Found</div>} />
                    </Routes>
                </div>
            </div>
            <ToastContainer pauseOnFocusLoss={false}  />
        </Router>
        </EmailProvider>
    );
}

export default App;
