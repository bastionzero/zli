import { MfaActionRequired } from '../mfa/mfa.types';

export interface UserRegisterResponse
{
    userSessionId: string;
    mfaActionRequired: MfaActionRequired;
}