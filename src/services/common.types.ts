export interface TargetUser
{
    userName: string;
}

export enum TargetType {
    SSM = 'SSM',
    SSH = 'SSH',
    DYNAMIC = 'DYNAMIC',
    CLUSTER = 'CLUSTER'
}

export enum IdP {
    Google = 'Google',
    Microsoft = 'Microsoft'
}

export enum TargetStatus {
    NotActivated = 'NotActivated',
    Offline = 'Offline',
    Online = 'Online',
    Terminated = 'Terminated',
    Error = 'Error'
}

export interface TargetSummary
{
    id: string;
    name: string;
    environmentId: string;
    type: TargetType;
    agentVersion: string;
    status: TargetStatus;
    targetUsers: string[];
}

export interface ParsedTargetString
{
    type: TargetType;
    user: string;
    id: string;
    name: string;
    path: string;
    envId: string;
    envName: string;
}