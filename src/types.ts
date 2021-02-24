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

export interface KeySplittingMessage<TPayload> {
    Payload: TPayload,
    Signature: string
}
export interface SynMessagePayload extends KeySplittingPayload {
    Nonce: string, 
    TargetId: string, 
    BZECert: string
}

export interface DataMessagePayload extends KeySplittingPayload {
    TargetId: string, 
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

export interface DataAckMessage extends KeySplittingMessage<DataAckPayload> { }

export interface SynMessage extends KeySplittingMessage<SynMessagePayload> { }

export interface SynAckMessage extends KeySplittingMessage<SynAckPayload> { }

export interface DataMessage extends KeySplittingMessage<DataMessagePayload> { }