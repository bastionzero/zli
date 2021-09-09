import { TargetType } from '../common.types';

export interface ConnectionSummary {
    id: string;
    timeCreated: number;
    serverId: string;
    sessionId: string;
    state: ConnectionState,
    serverType: TargetType,
    userName: string
}

export enum ConnectionState {
    Open = 'Open',
    Closed = 'Closed',
    Error = 'Error'
}

export interface ShellConnectionAuthDetails {
    connectionNodeId: string;
    authToken: string;
}

// TODO : Do we need both types? Stick with summary?
export interface ConnectionDetails
{
    id: string;
    timeCreated: number;
    targetId: string;
    sessionId: string;
    state: ConnectionState,
    serverType: TargetType,
    userName: string
}