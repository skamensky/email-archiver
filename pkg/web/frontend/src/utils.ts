import {useEffect, useState} from "react";

export const asError=(e:any):Error => {
    if (e instanceof Error) return e;
    return new Error(e);
}

export const buttonClass=(color:string):string => {
    return `ml-4 bg-${color}-500 hover:bg-${color}-700 text-white font-bold p-1 rounded text-sm`;
}

export const buttonClassDisabled=():string => {
    return `ml-4 bg-gray-500 text-white font-bold p-1 rounded text-sm opacity-50 cursor-not-allowed`;
}

export function useDebounce<T>(value:T, delay:number):T {
    const [debouncedValue, setDebouncedValue] = useState<T>(value);

    useEffect(() => {
        const handler = setTimeout(() => {
            setDebouncedValue(value);
        }, delay);

        return () => {
            clearTimeout(handler);
        };
    }, [value, delay]);

    return debouncedValue;
}
