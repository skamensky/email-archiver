import {MailboxEvent,Options} from "./goGeneratedModels";
import React, { useMemo } from 'react';

export type MailboxEventMessage = {
    data: MailboxEvent
    id: number
}

export const Spinner = (
    {size = 'medium'}: { size?: 'small' | 'medium' | 'large' | 'xlarge' }
) => {

    if (size === 'small') {
        return (
            <div className="border-4 border-gray-200 rounded-full w-6 h-6 border-t-4 border-t-blue-500 animate-spin"></div>
        );
    }
    if (size === 'medium') {
        return (
            <div className="border-4 border-gray-200 rounded-full w-8 h-8 border-t-4 border-t-blue-500 animate-spin"></div>
        );
    }
    if (size === 'large') {
        return (
            <div className="border-4 border-gray-200 rounded-full w-12 h-12 border-t-4 border-t-blue-500 animate-spin"></div>
        );
    }

    if (size === 'xlarge') {
        return (
            <div className="border-4 border-gray-200 rounded-full w-16 h-16 border-t-4 border-t-blue-500 animate-spin"></div>
        );
    }

    throw new Error(`Invalid size ${size}`);

};


type Store = {
    options: Options
}

// @ts-ignore
window.__customStore = {
    options: {}
}

export const saveStore = (store: Store) => {
    // @ts-ignore
    window.__customStore = store;
}

export const loadStore = () : Store => {
    // @ts-ignore
    return window.__customStore;
}


export const ProgressBar = React.memo(({ progress, maxValue }: { progress: number, maxValue: number }) => {
    const { percentage, color, progressText } = useMemo(() => {
        const calculatedPercentage = Math.min(100, (progress / maxValue) * 100);
        const calculatedColor = progress === maxValue ? 'bg-green-500' : 'bg-blue-500';
        const progressText = `${progress}/${maxValue}`;
        return { percentage: calculatedPercentage, color: calculatedColor, progressText };
    }, [progress, maxValue]);

    return (
        <div className="border rounded-xl border-gray-300 overflow-hidden h-6 rounded-lg bg-gray-200">
            <div
                className={`${color} h-6 flex justify-center items-center text-black text-sm rounded-xl`}
                style={{ width: `${percentage}%` }}
            >
                {progressText}
            </div>
        </div>
    );
});


export type MailboxToSync ={
    [key: string]: MailboxEventMessage
};
