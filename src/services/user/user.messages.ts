export interface UserRegisterResponse
{
    userSessionId: string;
    mfaActionRequired: MfaActionRequired;
}