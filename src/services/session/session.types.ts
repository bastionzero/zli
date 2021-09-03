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

export enum SessionState {
    Active = 'Active',
    Closed = 'Closed',
    Error = 'Error'
}