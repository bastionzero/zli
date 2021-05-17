import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import {
    DynamicAccessConfigService,
    EnvironmentService,
    SshTargetService,
    SsmTargetService
} from '../http.service/http.service';
import { TargetSummary, TargetType } from '../types';
import { MixpanelService } from '../mixpanel.service/mixpanel.service';
import { version } from '../../package.json';
import { oauthMiddleware } from '../middlewares/oauth-middleware';
import { LoggerConfigService } from '../logger-config.service/logger-config.service';
import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';


export function fetchDataMiddleware(configService: ConfigService, logger: Logger) {
    // Greedy fetch of some data that we use frequently
    const ssmTargetService = new SsmTargetService(configService, logger);
    const sshTargetService = new SshTargetService(configService, logger);
    const dynamicConfigService = new DynamicAccessConfigService(configService, logger);
    const envService = new EnvironmentService(configService, logger);

    const dynamicConfigs = dynamicConfigService.ListDynamicAccessConfigs()
        .then(result =>
            result.map<TargetSummary>((config, _index, _array) => {
                return {type: TargetType.DYNAMIC, id: config.id, name: config.name, environmentId: config.environmentId, agentVersion: 'N/A', status: undefined};
            })
        );

    // We will to show existing dynamic access targets for file transfer
    // UX to be more pleasant as people cannot file transfer to configs
    // only the DATs they produce from the config
    const ssmTargets = ssmTargetService.ListSsmTargets(true)
        .then(result =>
            result.map<TargetSummary>((ssm, _index, _array) => {
                return {type: TargetType.SSM, id: ssm.id, name: ssm.name, environmentId: ssm.environmentId, agentVersion: ssm.agentVersion, status: ssm.status};
            })
        );


    const sshTargets = sshTargetService.ListSshTargets()
        .then(result =>
            result.map<TargetSummary>((ssh, _index, _array) => {
                return {type: TargetType.SSH, id: ssh.id, name: ssh.alias, environmentId: ssh.environmentId, agentVersion: 'N/A', status: undefined};
            })
        );

    const envs = envService.ListEnvironments();

    return {
        dynamicConfigs: dynamicConfigs,
        ssmTargets: ssmTargets,
        sshTargets: sshTargets,
        envs: envs
    };
}

export function mixedPanelTrackingMiddleware(configService: ConfigService, argv: any) {
    // Mixpanel tracking
    const mixedPanelService = new MixpanelService(configService);

    // Only captures args, not options at the moment. Capturing configName flag
    // does not matter as that is handled by which mixpanel token is used
    // TODO: capture options and flags
    mixedPanelService.TrackCliCall(
        'CliCommand',
        {
            'cli-version': version,
            'command': argv._[0],
            args: argv._.slice(1)
        }
    );

    return mixedPanelService;
}

export async function oAuthMiddleware(configService: ConfigService, logger: Logger) {
    // OAuth
    await oauthMiddleware(configService, logger);
    const me = configService.me(); // if you have logged in, this should be set
    const sessionId = configService.sessionId();
    logger.info(`Logged in as: ${me.email}, bzero-id:${me.id}, session-id:${sessionId}`);
}

export async function initMiddleware(argv: any) {
    // Configure our logger
    const loggerConfigService = new LoggerConfigService(<string> argv.configName);
    // isTTY detects whether the process is being run with a text terminal ("TTY") attached
    // This way we detect whether we should connect logger.error to stderr in order
    // to be able to print error messages to the user (e.g. ssh-proxy mode)
    const logger = new Logger(loggerConfigService, !!argv.debug, !!argv.silent, !!process.stdout.isTTY);
    // Config init
    const configService = new ConfigService(<string>argv.configName, logger);

    // KeySplittingService init
    const keySplittingService = new KeySplittingService(configService, logger);
    await keySplittingService.init();

    return {
        loggingConfigService: loggerConfigService,
        logger: logger,
        configService: configService,
        keySplittingService: keySplittingService
    };
}