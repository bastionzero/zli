export interface MixpanelMetadata
{
    distinct_id: string,
    client_type: string, // "CLI"
    UserSessionId: string
}

export interface TrackNewConnection extends MixpanelMetadata
{
    ConnectionType: string
}