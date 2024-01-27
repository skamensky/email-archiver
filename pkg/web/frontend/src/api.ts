import {Options,Email} from "./goGeneratedModels";

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