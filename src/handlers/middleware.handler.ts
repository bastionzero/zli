import { Logger } from '../services/logger/logger.service';
import { ConfigService } from '../services/config/config.service';
import { version } from '../../package.json';
import { oauthMiddleware } from '../middlewares/oauth-middleware';
import { LoggerConfigService } from '../services/logger/logger-config.service';
import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';
import { TargetSummary, TargetType } from '../services/common.types';
import { DynamicAccessConfigService } from '../services/dynamic-access-config/dynamic-access-config.service';
import { EnvironmentService } from '../services/environment/environment.service';
import { KubeService } from '../services/kube/kube.service';
import { ClusterDetails } from '../services/kube/kube.types';
import { MixpanelService } from '../services/mixpanel/mixpanel.service';
import { SsmTargetService } from '../services/ssm-target/ssm-target.service';


export function fetchDataMiddleware(configService: ConfigService, logger: Logger) {
    // Greedy fetch of some data that we use frequently
    const ssmTargetService = new SsmTargetService(configService, logger);
    const kubeService = new KubeService(configService, logger);
    const dynamicConfigService = new DynamicAccessConfigService(configService, logger);
    const envService = new EnvironmentService(configService, logger);

    const dynamicConfigs = new Promise<TargetSummary[]>( async (res, _) => {
        try
        {
            const response = await dynamicConfigService.ListDynamicAccessConfigs();
            const results = response.map<TargetSummary>((config, _index, _array) => {
                return {type: TargetType.DYNAMIC, id: config.id, name: config.name, environmentId: config.environmentId, agentVersion: 'N/A', status: undefined, targetUsers: undefined};
            });

            res(results);
        } catch (e: any) {
            logger.error(`Failed to fetch dynamic access configs: ${e}`);
            return res([]);
        }
    });

    // We will to show existing dynamic access targets for file transfer
    // UX to be more pleasant as people cannot file transfer to configs
    // only the DATs they produce from the config
    const ssmTargets = new Promise<TargetSummary[]>( async (res, _) => {
        try
        {
            const response = await ssmTargetService.ListSsmTargets(true);
            const results = response.map<TargetSummary>((ssm, _index, _array) => {
                return {type: TargetType.SSM, id: ssm.id, name: ssm.name, environmentId: ssm.environmentId, agentVersion: ssm.agentVersion, status: ssm.status, targetUsers: undefined};
            });

            res(results);
        } catch (e: any) {
            logger.error(`Failed to fetch ssm targets: ${e}`);
            return res([]);
        }
    });

    const clusterTargets = kubeService.ListKubeClusters()
        .then(result =>
            result.map<ClusterDetails>((cluster, _index, _array) => {
                return { id: cluster.id, name: cluster.clusterName, status: cluster.status, environmentId: cluster.environmentId, targetUsers: cluster.validUsers, agentVersion: cluster.agentVersion, lastAgentUpdate: cluster.lastAgentUpdate};
            })
        );

    const envs = envService.ListEnvironments();

    return {
        dynamicConfigs: dynamicConfigs,
        ssmTargets: ssmTargets,
        clusterTargets: clusterTargets,
        envs: envs
    };
}

export function mixpanelTrackingMiddleware(configService: ConfigService, argv: any) {
    // Mixpanel tracking
    const mixpanelService = new MixpanelService(configService);

    // Only captures args, not options at the moment. Capturing configName flag
    // does not matter as that is handled by which mixpanel token is used
    // TODO: capture options and flags
    mixpanelService.TrackCliCall(
        'CliCommand',
        {
            'cli-version': version,
            'command': argv._[0],
            args: argv._.slice(1)
        }
    );

    return mixpanelService;
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