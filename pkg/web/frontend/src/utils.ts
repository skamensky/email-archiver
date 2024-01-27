export const asError=(e:any):Error => {
    if (e instanceof Error) return e;
    return new Error(e);
}

export const buttonClass=(color:string):string => {
    return `ml-4 bg-${color}-500 hover:bg-${color}-700 text-white font-bold py-1 px-1 rounded text-sm`;
}

