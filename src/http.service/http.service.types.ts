import { ReadStream } from "fs";
import { SessionState, TargetType } from "../types";

export interface CreateSessionRequest {
    displayName?: string;
    connectionsToOpen: ConnectionsToOpen[];
}

export interface CreateSessionResponse {
    sessionId: string;
}

export interface CloseSessionRequest {
    sessionId: string;
}

export interface CloseSessionResponse {
}

export interface ListSessionsRequest {
}

export interface ListSessionsResponse {
    sessions: SessionDetails[];
}

export interface SessionDetails {
    id: string;
    displayName: string;
    timeCreated: number;
    state: SessionState,
    connections: ConnectionSummary[]
}

export interface ConnectionsToOpen {
      serverId: string;
      connectionType: TargetType,
      count: number
}

export enum ConnectionState {
    Open = "Open",
    Closed = "Closed",
    Error = "Error"
}

export interface CreateConnectionRequest {
    sessionId: string;
    serverId: string;
    serverType: TargetType;
    username: string;
}

export interface CreateConnectionResponse {
    connectionId: string;
}

export interface CloseConnectionRequest {
    connectionId: string;
}

export interface CloseConnectionResponse {
}

export interface ConnectionSummary {
    id: string;
    timeCreated: number;
    serverId: string;
    sessionId: string;
    state: ConnectionState,
    serverType: TargetType
}

export interface SsmTargetSummary {
    id: string;
    name: string;
    status: SsmTargetStatus;
    environmentId?: string;
}

export enum SsmTargetStatus {
    Online = "Online",
    Offline = "Offline"
}

export enum AuthenticationType {
    Password = "Password",
    PrivateKey = "PrivateKey",
    UseExisting = "UseExisting"
}

export interface SshTargetSummary {
    id: string;
    alias: string;
    host: string;
    userName: string;
    port: number;
    authenticationType: AuthenticationType;
    color: string;
    vpnId?: string;
    environmentId?: string;
}

export interface EnvironmentDetails {
    id: string;
    organizationId: string;
    name: string;
    timeCreated: string;
    resources: EnvironmentResourceDetails[]
    isDefault: boolean;
}

export interface EnvironmentResourceDetails {
    id: string;
    environmentId: string;
    timeCreated: string;
    resourceId: string;
    resourceType: string;
}

export interface ListEnvironmentsRequest {
}

export interface UploadFileRequest {
    targetId: string,
    targetType: TargetType,
    targetFilePath: string,
    file: ReadStream,
    targetUser?: string
};

export interface UploadFileResponse {
}


export interface DownloadFileRequest {
    targetId: string,
    targetType: TargetType,
    filePath: string,
    targetUser?: string
    downloadedFileName?: string
};

export interface DownloadFileResponse {
}

export enum FileServiceErrors {
    UnknownError = "UnknownError",
    ScpException = "ScpException"
}

export interface ScpErrorMsg {
    reason: string;
}