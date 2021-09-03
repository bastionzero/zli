export interface UserSummary
{
    id: string;
    organizationId: string;
    fullName: string;
    email: string;
    isAdmin: boolean;
    timeCreated: Date;
    lastLogin: Date;
}