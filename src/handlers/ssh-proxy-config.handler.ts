import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';


export function sshProxyConfigHandler(configService: ConfigService, logger: Logger, processName: string) {
    let prefix = 'bzero-';
    let keyPath = configService.sshKeyPath();
    let configName = configService.getConfigName();
    let configNameArg = '';
    if(configName != 'prod') {
        prefix = `${configName}-${prefix}`;
        configNameArg = `--configName=${configName}`;
    }

    logger.info(`
Add the following lines to your ssh config (~/.ssh/config) file:

host ${prefix}*
IdentityFile ${keyPath}
ProxyCommand ${processName} ssh-proxy ${configNameArg} -s %h %r %p ${keyPath}


Then you can use native ssh to connect to any of your ssm targets using the following syntax:

ssh <user>@${prefix}<ssm-target-id-or-name>
`);
}