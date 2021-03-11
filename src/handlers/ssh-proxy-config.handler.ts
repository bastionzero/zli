import { ConfigService } from '../config.service/config.service';
import { Logger } from '../logger.service/logger';


export function sshProxyConfigHandler(configService: ConfigService, logger: Logger, processName: string) {
    let keyPath = configService.sshKeyPath();
    let configName = configService.getConfigName();
    let configNameArg = '';
    if(configName != 'prod') {
        configNameArg = `--configName=${configName}`;
    }

    logger.info(`
Add the following lines to your ssh config (~/.ssh/config) file:

host bzero-*
IdentityFile ${keyPath}
ProxyCommand ${processName} ssh-proxy ${configNameArg} -s %h %r %p ${keyPath}


Then you can use native ssh to connect to any of your ssm targets using the following syntax:

ssh <user>@bzero-<ssm-target-id-or-name>
`);
}