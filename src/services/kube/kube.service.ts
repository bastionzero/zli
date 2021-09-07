import { spawn } from 'child_process';
import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { GetKubeUnregisteredAgentYamlResponse, GetKubeUnregisteredAgentYamlRequest, GetUserInfoResponse, GetUserInfoRequest } from './kube.mesagges';
import { ClusterDetails, ClusterSummary } from './kube.types';

export class KubeService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/kube', logger);
    }

    public getKubeUnregisteredAgentYaml(
        clusterName: string,
        labels: string,
        namespace: string,
        environmentId: string,
    ): Promise<GetKubeUnregisteredAgentYamlResponse>
    {
        const request: GetKubeUnregisteredAgentYamlRequest = {
            clusterName: clusterName,
            labels: labels,
            namespace: namespace,
            environmentId: environmentId,
        };
        return this.Post('get-agent-yaml', request);
    }

    public GetUserInfoFromEmail(
        email: string
    ): Promise<GetUserInfoResponse>
    {
        const request: GetUserInfoRequest = {
            email: email,
        };

        return this.FormPostWithException('get-user', request);
    }

    public ListKubeClusters(): Promise<ClusterSummary[]> {
        return this.Post('list', {});
    }
}

export async function killDaemon(configService: ConfigService) {
    const kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if (kubeConfig['localPid'] != null) {
        // First try to kill the process
        spawn('pkill', ['-P', kubeConfig['localPid'].toString()]);

        // Update the config
        kubeConfig['localPid'] = null;
        configService.setKubeConfig(kubeConfig);
    }
}