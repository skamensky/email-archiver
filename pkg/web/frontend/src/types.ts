import {RuleGroupType} from "react-querybuilder";

export type SavedQuery = {
    name: string
    query: RuleGroupType
    date: number
    sqlForDisplay: string
}

export const defaultPersistedState = {queries:[]};


export type PersistedState = {
    queries: SavedQuery[]
}

