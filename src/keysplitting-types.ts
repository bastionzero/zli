export interface BZECert {
    InitialIdToken: string,
    CurrentIdToken: string,
    ClientPublicKey: string,
    Rand: string,
    SignatureOnRand: string
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
    BZECert: BZECert
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

export interface SynMessageWrapper {
    SynPayload: SynMessage
}

export interface DataMessageWrapper {
    DataPayload: DataMessage
}

export interface DataAckMessageWrapper {
    DataAckPayload: DataAckMessage
}

export interface SynAckMessageWrapper {
    SynAckPayload: SynAckMessage
}