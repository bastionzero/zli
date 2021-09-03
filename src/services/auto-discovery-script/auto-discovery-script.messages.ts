export interface GetAutodiscoveryScriptResponse {
    autodiscoveryScript: string;
}

export interface GetAutodiscoveryScriptRequest {
    targetNameScript: string;
    apiUrl: string;
    envId: string;
    agentVersion: string;
}