import { ConnectionState } from './http.service/http.service.types';

export enum TargetType {
    SSM = 'SSM',
    SSH = 'SSH',
    DYNAMIC = 'DYNAMIC',
    CLUSTER = 'CLUSTER'
}

export enum SessionState {
    Active = 'Active',
    Closed = 'Closed',
    Error = 'Error'
}

export enum IdP {
    Google = 'Google',
    Microsoft = 'Microsoft'
}

export enum SsmTargetStatus {
    NotActivated = 'NotActivated',
    Offline = 'Offline',
    Online = 'Online',
    Terminated = 'Terminated',
    Error = 'Error'
}

export enum KubeClusterStatus {
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
    status: SsmTargetStatus;
}

export interface ClusterSummary
{
    id: string;
    name: string;
    status: KubeClusterStatus;
    environmentId: string;
    validUsers: string[];
    lastAgentUpdate: Date;
    agentVersion: string;
}

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