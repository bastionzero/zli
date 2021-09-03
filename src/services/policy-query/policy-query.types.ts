export interface Verb
{
    type: VerbType;
}

export enum VerbType {
    Shell = 'Shell',
    FileTransfer = 'FileTransfer',
    Tunnel = 'Tunnel'
}