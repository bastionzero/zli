import { Group, PolicyMetadata, Subject } from './policy.types';

export interface EditPolicyRequest {
    id: string;
    name: string;
    type: string;
    subjects: Subject[];
    groups: Group[];
    context: string;
    policyMetadata: PolicyMetadata
}