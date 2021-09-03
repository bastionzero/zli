export interface EditPolicyRequest {
    id: string;
    name: string;
    type: string;
    subjects: Subject[];
    groups: Group[];
    context: string;
    policyMetadata: PolicyMetadata
}