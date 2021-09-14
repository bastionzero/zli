import { Group, PolicyMetadata, Subject } from './policy.types';

export interface AddPolicyRequest {
    name: string;
    type: string;
    subjects: Subject[];
    groups: Group[];
    context: string;
    policyMetadata: PolicyMetadata
}

export interface EditPolicyRequest extends AddPolicyRequest {
    id: string;
}