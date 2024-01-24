import {getOptions, Options} from "./api";
import {useEffect, useState} from "react";

export const Settings = () => {

    const [options, setOptions] = useState<Options | null>(null);
    const [lastUpdateNSecondsAgo, setLastUpdateNSecondsAgo] = useState<number | null>(null);
    useEffect(() => {
        getOptions().then((options) => {
            setOptions(options);
            setLastUpdateNSecondsAgo(0);
        }).catch((err) => {
            console.error(err);
        })

        // 30 seconds
        const updateInterval = setInterval(() => {
            getOptions().then((options) => {
                setOptions(options);
                setLastUpdateNSecondsAgo(0);
            }).catch((err) => {
                console.error(err);
            })
        }, 10000);

        const tickInterval = setInterval(() => {
            setLastUpdateNSecondsAgo((lastUpdateNSecondsAgo) => {
                if(lastUpdateNSecondsAgo===null){
                    return null
                }
                    return lastUpdateNSecondsAgo + 1;
            });
        }, 1000);

        return () => {
            clearInterval(updateInterval);
            clearInterval(tickInterval);
        }

    }, []);

    const labelToValue = {
        "Email":options?.email,
        "Password":options?.password,
        "IMAP Server":options?.imap_server,
        "Strict Mail Parsing":options?.strict_mail_parsing,
        "IMAP Client Debug":options?.imap_client_debug,
        "Debug":options?.debug,
        "Limit to Mailboxes":options?.limit_to_mailboxes,
        "Skip Mailboxes":options?.skip_mailboxes,
        "DB Path":options?.db_path,
        "Max Pool Size":options?.max_pool_size
    }

    let updatedNSecondsAgoString = "Never"
    if(lastUpdateNSecondsAgo != null){
        updatedNSecondsAgoString = lastUpdateNSecondsAgo === 1 ? `${lastUpdateNSecondsAgo} second ago` : `${lastUpdateNSecondsAgo} seconds ago`
    }


    const keyValues = Object.entries(labelToValue).map(([key, value]) =>
        {

            const noVal = value === null || value === undefined

            const valueColor = noVal ? "text-gray-600 font-light" : "text-gray-800 font-medium"
            let valueText = noVal ? "Not Set" : value
            if(key==="Password"){
                valueText = "********"
            }
            // if type is array, join with comma
            if(Array.isArray(value) && !noVal){
                valueText = value.join(", ")
            }

        return (
            <div key={key} className="flex items-center space-x-2 border-b border-gray-200 py-2">
            <span className="text-gray-600">{key}</span>
            <span className={valueColor}>{valueText}</span>
        </div>)}
    )

    return (<div className="bg-white rounded-lg p-4 inline-block">
            <h2 className="text-xl font-semibold mb-4">Settings</h2>
        <div className="flex items-center space-x-2 border-b border-gray-200 py-2">
            <span className="text-gray-600">Last Updated</span>
            <span className="text-gray-800 font-medium">{updatedNSecondsAgoString}</span>
        </div>
            <div className="bg-white shadow-md rounded-lg p-4">
                {keyValues}
    </div>
    </div>
);
}