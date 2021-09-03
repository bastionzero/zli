export interface EnvironmentDetails {
    id: string;
    organizationId: string;
    name: string;
    timeCreated: string;
    resources: EnvironmentResourceDetails[]
    isDefault: boolean;
}

export interface EnvironmentResourceDetails {
    id: string;
    environmentId: string;
    timeCreated: string;
    resourceId: string;
    resourceType: string;
}