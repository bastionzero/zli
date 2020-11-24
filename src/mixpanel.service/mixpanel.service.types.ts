export interface MixpanelMetadata
{
    distinct_id: string,
    client_type: string, // "CLI"
}

export interface TrackNewConnection extends MixpanelMetadata
{
    ConnectionType: string
}