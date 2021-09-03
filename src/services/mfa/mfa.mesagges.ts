export interface MfaTokenRequest
{
    token: string;
}

export interface MfaResetResponse
{
    mfaSecretUrl: string;
}

export interface MfaResetRequest
{
    forceSetup: boolean;
}

export interface MfaClearRequest
{
    userId: string;
}