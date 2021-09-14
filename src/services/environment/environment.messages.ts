export interface CreateEnvironmentRequest {
    name: string;
    description?: string;
    offlineCleanupTimeoutHours: number;
}

export interface CreateEnvironmentResponse {
    id: string;
}