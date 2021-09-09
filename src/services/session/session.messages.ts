import { ConnectionsToOpen, SessionDetails } from './session.types';

export interface ListSessionsResponse {
    sessions: SessionDetails[];
}

export interface CreateSessionRequest {
    displayName?: string;
    connectionsToOpen: ConnectionsToOpen[];
}

export interface CreateSessionResponse {
    sessionId: string;
}

export interface CloseSessionResponse {
}

export interface CloseSessionRequest {
    sessionId: string;
}