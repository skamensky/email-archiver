import React from 'react';
import './App.css';
import { BrowserRouter as Router, Route, Link, Routes } from "react-router-dom";
// import {Home} from "./Home";
import {Settings} from "./Settings";
import {EmailArchiver} from "./EmailArchiver";

const LocalStateViewer = () => <div className="p-4">
    <h1 className="text-2xl"> Local State Viewer</h1>
    <h1 className="text-2xl">Sync Now</h1>

</div>;


function App() {

    return (
        <Router>
            <div className="flex flex-col h-screen">
                <nav className="flex bg-blue-500 text-white p-4 space-x-4">
                    <Link className="hover:bg-blue-700 px-3 py-2 rounded-md" to="/">Home</Link>
                    <Link className="hover:bg-blue-700 px-3 py-2 rounded-md" to="/settings">Settings</Link>
                    <Link className="hover:bg-blue-700 px-3 py-2 rounded-md" to="/emails">Emails</Link>
                </nav>
                <div className="flex-grow">
                    <Routes>
                        <Route path="/" element={<LocalStateViewer />} />
                        <Route path="/settings" element={<Settings />}   />
                        <Route path="/emails" element={<EmailArchiver />} />
                    </Routes>
                </div>
            </div>
        </Router>
    );
}

export default App;
