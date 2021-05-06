import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';
import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';
import { SsmTunnelService } from '../ssm-tunnel/ssm-tunnel.service';
import { cleanExit } from './clean-exit.handler';
import { Dictionary } from 'lodash';
import { PolicyQueryService } from '../http.service/http.service';
import { ParsedTargetString } from '../types';
import _ from 'lodash';
import { VerbType } from '../http.service/http.service.types';


export async function sshProxyHandler(configService: ConfigService, logger: Logger, sshTunnelParameters: SshTunnelParameters, keySplittingService: KeySplittingService, envMap: Dictionary<string>) {

    const policyQueryService = new PolicyQueryService(configService, logger);
    const response = await policyQueryService.ListTargetUsers(sshTunnelParameters.parsedTarget.id, sshTunnelParameters.parsedTarget.type, {type: VerbType.Tunnel}, undefined);

    if(! response.allowed)
    {
        const errorMessage = 'You do not have sufficient permission to open a ssh tunnel to the target';
        logger.error(errorMessage);
        process.stderr.write(`\n${errorMessage}\n`);
        await cleanExit(1, logger);
    }

    const allowedTargetUsers = response.allowedTargetUsers.map(u => u.userName);
    if(response.allowedTargetUsers && ! _.includes(allowedTargetUsers, sshTunnelParameters.parsedTarget.user)) {
        const errorMessage = `You do not have permission to tunnel as targetUser: ${sshTunnelParameters.parsedTarget.user}. Current allowed users for you: ${allowedTargetUsers}`;
        logger.error(errorMessage);
        process.stderr.write(`\n${errorMessage}\n`);

        await cleanExit(1, logger);
    }

    const ssmTunnelService = new SsmTunnelService(logger, configService, keySplittingService, envMap['enableKeysplitting'] == 'true');
    ssmTunnelService.errors.subscribe(async errorMessage => {
        process.stderr.write(`\n${errorMessage}\n`);
        await cleanExit(1, logger);
    });

    if( await ssmTunnelService.setupWebsocketTunnel(sshTunnelParameters.parsedTarget, sshTunnelParameters.port, sshTunnelParameters.identityFile)) {
        process.stdin.on('data', async (data) => {
            ssmTunnelService.sendData(data);
        });
    }

    configService.logoutDetected.subscribe(async () => {
        logger.debug('Logged out by another zli instance. Terminating ssh tunnel');
        process.stderr.write(`\nLogged out by another zli instance. Terminating ssh tunnel...\n`);
        await ssmTunnelService.closeTunnel();
        await cleanExit(0, logger);
    });
}

export interface SshTunnelParameters {
    parsedTarget: ParsedTargetString;
    port: number;
    identityFile: string;
}