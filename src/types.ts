export enum TargetType {
    SSM = 'SSM',
    SSH = 'SSH',
    DYNAMIC = 'DYNAMIC'
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

export interface KeySplittingPayload {
    Type: string, 
    Action: string
}

export interface KeySplittingMessage {
    Payload: KeySplittingPayload,
    Signature: string
}
export interface SynMessagePayload extends KeySplittingPayload {
    Nonce: string, 
    TargetID: string, 
    BZECert: string
}

export interface DataMessagePayload extends KeySplittingPayload {
    TargetID: string, 
    HPointer: string,
    Payload: string, 
    BZECert: string
}
export interface SynAckPayload extends KeySplittingPayload {
    HPointer: string,
    Nonce: string, 
    TargetPublicKey: string
}

export interface DataAckPayload extends KeySplittingPayload {
    HPointer: string, 
    Payload: string, 
    TargetPublicKey: string
}

export interface DataAckMessage extends KeySplittingMessage { }

export interface SynMessage extends KeySplittingMessage { }

export interface SynAckMessage extends KeySplittingMessage { }

export interface DataMessage extends KeySplittingMessage { }