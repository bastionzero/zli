import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { AddPolicyRequest, EditPolicyRequest } from './policy.messages';
import { PolicySummary } from './policy.types';

export class PolicyService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/Policy', logger);
    }

    public ListAllPolicies(): Promise<PolicySummary[]>
    {
        return this.Post('list', {});
    }

    public EditPolicy(
        policy: PolicySummary
    ): Promise<void> {
        const request: EditPolicyRequest = {
            id: policy.id,
            name: policy.name,
            type: policy.type,
            subjects: policy.subjects,
            groups: policy.groups,
            context: JSON.stringify(policy.context),
            policyMetadata: policy.metadata
        };
        return this.Post('edit', request);
    }

    public AddPolicy(req : AddPolicyRequest) : Promise<PolicySummary>
    {
        return this.Post<AddPolicyRequest, PolicySummary>('add', req);
    }
}