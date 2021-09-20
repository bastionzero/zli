import { spawn } from 'child_process';
import { ConfigService } from '../config/config.service';
import { HttpService } from '../http/http.service';
import { Logger } from '../logger/logger.service';
import { GetKubeUnregisteredAgentYamlResponse, GetKubeUnregisteredAgentYamlRequest, GetUserInfoResponse, GetUserInfoRequest } from './kube.mesagges';
import { ClusterSummary } from './kube.types';

export interface KubeConfig {
    keyPath: string,
    certPath: string,
    token: string,
    localHost: string,
    localPort: number,
    localPid: number,
    assumeRole: string,
    assumeCluster: string,
}

export class KubeService extends HttpService
{
    constructor(configService: ConfigService, logger: Logger)
    {
        super(configService, 'api/v1/kube', logger);
    }

    public getKubeUnregisteredAgentYaml(
        clusterName: string,
        labels: { [index: string ]: string },
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

        return this.Post('get-user', request);
    }

    public ListKubeClusters(): Promise<ClusterSummary[]> {
        return this.Get('list', {});
    }
}

export async function killDaemon(configService: ConfigService) {
    const kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if ( kubeConfig['localPid'] != null) {
        // First try to kill the process
        if (process.platform === 'win32') {
            spawn('taskkill', ['/F', '/T', '/PID', kubeConfig['localPid'].toString()]);
        } else if (process.platform === 'linux') {
            spawn('pkill', ['-s', kubeConfig['localPid'].toString()]);
        } else {
            spawn('kill', ['-9', kubeConfig['localPid'].toString()]);
        }

        // Update the config
        kubeConfig['localPid'] = null;
        configService.setKubeConfig(kubeConfig);

        return true;
    } else {
        return false;
    }
}

export function getDefaultKubeConfig(): KubeConfig {
    return {
        keyPath: null,
        certPath: null,
        token: null,
        localHost: null,
        localPort: null,
        localPid: null,
        assumeRole: null,
        assumeCluster: null,
    };
}