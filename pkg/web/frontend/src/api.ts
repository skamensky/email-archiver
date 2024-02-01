import {Options,Email,MailboxRecord} from "./goGeneratedModels";
import {defaultPersistedState, PersistedState} from "./types";

const server = 'http://localhost:8080';


export const getOptions = async (): Promise<Options> => {
    const response = await fetch(`${server}/api/options`, {

    })
    const json = await response.json();

    if (json.error) {
        console.error(json.error)
        throw new Error(json.error);
    }
    return new Options(json);
}

export const getEmails = async (sqlQuery:string):Promise<Email[]> => {
    const response = await fetch(`${server}/api/emails`, {
        method:'POST',
        body:JSON.stringify({sqlQuery})
    })

    const json = await response.json();
    if (json.error) {
        console.error(json.error)
        throw new Error(json.error);
    }
    return json.emails.map((e:any) => new Email(e));
}

export const searchEmails = async (searchQuery:string):Promise<Email[]> => {
    const response = await fetch(`${server}/api/search`, {
        method:'POST',
        body:JSON.stringify({searchQuery})
    })

    const json = await response.json();
    if (json.error) {
        console.error(json.error)
        throw new Error(json.error);
    }

    return json.emails.map((e:any) => new Email(e));
}

export const getMailboxRecords = async ():Promise<MailboxRecord[]> => {
    const response = await fetch(`${server}/api/mailboxes`, {
        method:'GET',
    })

    const json = await response.json();
    if (json.error) {
        console.error(json.error)
        throw new Error(json.error);
    }
    return json.mailboxes.map((e:any) => new MailboxRecord(e));
}

export const syncMailboxes = async (mailboxes: string[]):Promise<void> => {
    const response = await fetch(`${server}/api/sync`, {
        method:'POST',
        body:JSON.stringify({mailboxes})
    })

    const json = await response.json();
    if (json.error) {
        console.error(json.error)
        throw new Error(json.error);
    }
    return;
}


export const persistState = async (state:PersistedState):Promise<void> => {
    const response = await fetch(`${server}/api/set_frontend_state`, {
        method:'POST',
        body:JSON.stringify({state:JSON.stringify(state)})
    })

    const json = await response.json();
    if (json.error) {
        console.error(json.error)
        throw new Error(json.error);
    }
    return;
}

export const getPersistedState = async ():Promise<PersistedState> => {
    const response = await fetch(`${server}/api/get_frontend_state`, {
        method:'GET',
    })

    const json = await response.json();
    if (json.error) {
        console.error(json.error)

        if(json.error.includes('no rows in result set')){
            try{
                await persistState(defaultPersistedState);
                return defaultPersistedState;
            }
            catch(e){
                console.error(e);
                throw new Error(`Error creating initial state: ${e}`);
            }
        } else{
            throw new Error(json.error);
        }
    }

    return JSON.parse(json.state) as PersistedState;

}