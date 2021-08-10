import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { SsmTunnelService } from '../ssm-tunnel/ssm-tunnel.service';
import { cleanExit } from './clean-exit.handler';
import { Dictionary, includes } from 'lodash';
import { PolicyQueryService } from '../http.service/http.service';
import { ParsedTargetString } from '../types';
import { VerbType } from '../http.service/http.service.types';
import { targetStringExample } from '../../src/utils';


export async function sshProxyHandler(configService: ConfigService, logger: Logger, sshTunnelParameters: SshTunnelParameters, keySplittingService: KeySplittingService, envMap: Dictionary<string>) {

    if(! sshTunnelParameters.parsedTarget) {
        logger.error('No targets matched your targetName/targetId or invalid target string, must follow syntax:');
        logger.error(targetStringExample);
        await cleanExit(1, logger);
    }
    const policyQueryService = new PolicyQueryService(configService, logger);
    const response = await policyQueryService.ListTargetUsers(sshTunnelParameters.parsedTarget.id, sshTunnelParameters.parsedTarget.type, {type: VerbType.Tunnel}, undefined);

    if(! response.allowed)
    {
        logger.error('You do not have sufficient permission to open a ssh tunnel to the target');
        await cleanExit(1, logger);
    }

    const allowedTargetUsers = response.allowedTargetUsers.map(u => u.userName);
    if(response.allowedTargetUsers && ! includes(allowedTargetUsers, sshTunnelParameters.parsedTarget.user)) {
        logger.error(`You do not have permission to tunnel as targetUser: ${sshTunnelParameters.parsedTarget.user}. Current allowed users for you: ${allowedTargetUsers}`);
        await cleanExit(1, logger);
    }

    const ssmTunnelService = new SsmTunnelService(logger, configService, keySplittingService, envMap['enableKeysplitting'] == 'true');
    ssmTunnelService.errors.subscribe(async errorMessage => {
        logger.error(errorMessage);
        await cleanExit(1, logger);
    });

    if( await ssmTunnelService.setupWebsocketTunnel(sshTunnelParameters.parsedTarget, sshTunnelParameters.port, sshTunnelParameters.identityFile)) {
        process.stdin.on('data', async (data) => {
            ssmTunnelService.sendData(data);
        });
    }

    configService.logoutDetected.subscribe(async () => {
        logger.error('\nLogged out by another zli instance. Terminating ssh tunnel\n');
        await ssmTunnelService.closeTunnel();
        await cleanExit(0, logger);
    });
}

export interface SshTunnelParameters {
    parsedTarget: ParsedTargetString;
    port: number;
    identityFile: string;
}