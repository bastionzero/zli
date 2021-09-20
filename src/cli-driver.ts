import {
    disambiguateTarget,
    isGuid,
    targetStringExample
} from './utils';
import { ConfigService } from './services/config/config.service';
import { checkVersionMiddleware } from './middlewares/check-version-middleware';
import { Logger } from './services/logger/logger.service';
import { LoggerConfigService } from './services/logger/logger-config.service';
import { KeySplittingService } from '../webshell-common-ts/keysplitting.service/keysplitting.service';
import { cleanExit } from './handlers/clean-exit.handler';

// Handlers
import { initMiddleware, oAuthMiddleware, fetchDataMiddleware, mixpanelTrackingMiddleware } from './handlers/middleware.handler';
import { sshProxyConfigHandler } from './handlers/ssh-proxy-config.handler';
import { sshProxyHandler, SshTunnelParameters } from './handlers/ssh-proxy/ssh-proxy.handler';
import { loginHandler } from './handlers/login/login.handler';
import { connectHandler } from './handlers/connect/connect.handler';
import { listTargetsHandler } from './handlers/list-targets/list-targets.handler';
import { configHandler } from './handlers/config.handler';
import { logoutHandler } from './handlers/logout.handler';
import { startKubeDaemonHandler } from './handlers/tunnel/start-kube-daemon.handler';
import { autoDiscoveryScriptHandler } from './handlers/autodiscovery-script/autodiscovery-script-handler';
import { listConnectionsHandler } from './handlers/list-connections/list-connections.handler';
import { listUsersHandler } from './handlers/user/list-users.handler';
import { attachHandler } from './handlers/attach/attach.handler';
import { closeConnectionHandler } from './handlers/close-connection/close-connection.handler';
import { generateKubeconfigHandler } from './handlers/generate-kube/generate-kubeconfig.handler';
import { addTargetUserHandler } from './handlers/target-user/add-target-user.handler';
import { deleteTargetUserHandler } from './handlers/target-user/delete-target-user.handler';
import { addUserToPolicyHandler } from './handlers/user/add-user-policy.handler';
import { deleteUserFromPolicyHandler } from './handlers/user/delete-user-policy.handler';
import { generateKubeYamlHandler } from './handlers/generate-kube/generate-kube-yaml.handler';
import { disconnectHandler } from './handlers/disconnect/disconnect.handler';
import { kubeStatusHandler } from './handlers/tunnel/status.handler';
import { bctlHandler } from './handlers/bctl.handler';
import { listPoliciesHandler } from './handlers/policy/list-policies.handler';
import { listTargetUsersHandler } from './handlers/target-user/list-target-users.handler';
import { fetchGroupsHandler } from './handlers/group/fetch-groups.handler';
import { generateBashHandler } from './handlers/generate-bash/generate-bash.handler';

// 3rd Party Modules
import { Dictionary, includes } from 'lodash';
import yargs from 'yargs';
import { describeClusterHandler } from './handlers/describe-cluster/describe-cluster.handler';
import { deleteGroupFromPolicyHandler } from './handlers/group/delete-group-policy.handler';
import { addGroupToPolicyHandler } from './handlers/group/add-group-policy.handler';
import { loginCmdBuilder } from './handlers/login/login.command-builder';
import { connectCmdBuilder } from './handlers/connect/connect.command-builder';
import { tunnelCmdBuilder } from './handlers/tunnel/tunnel.command-builder';
import { policyCmdBuilder } from './handlers/policy/policy.command-builder';
import { describeClusterCmdBuilder } from './handlers/describe-cluster/describe-cluster.command-builder';
import { disconnectCmdBuilder } from './handlers/disconnect/disconnect.command-builder';
import { attachCmdBuilder } from './handlers/attach/attach.command-builder';
import { closeConnectionCmdBuilder } from './handlers/close-connection/close-connection.command-builder';
import { listTargetsCmdBuilder } from './handlers/list-targets/list-targets.command-builder';
import { listConnectionsCmdBuilder } from './handlers/list-connections/list-connections.command-builder';
import { userCmdBuilder } from './handlers/user/user.command-builder';
import { groupCmdBuilder } from './handlers/group/group.command-builder';
import { targetUserCmdBuilder } from './handlers/target-user/target-user.command-builder';
import { sshProxyCmdBuilder } from './handlers/ssh-proxy/ssh-proxy.command-builder';
import { autoDiscoveryScriptCommandBuilder } from './handlers/autodiscovery-script/autodiscovery-script.command-builder';
import { generateKubeCmdBuilder } from './handlers/generate-kube/generate-kube.command-builder';
import { generateBashCmdBuilder } from './handlers/generate-bash/generate-bash.command-builder';
import { TargetSummary, TargetType, TargetStatus } from './services/common.types';
import { EnvironmentDetails } from './services/environment/environment.types';
import { MixpanelService } from './services/mixpanel/mixpanel.service';
import { PolicyType } from './services/policy/policy.types';
import { ClusterDetails } from './services/kube/kube.types';

export class CliDriver
{
    private configService: ConfigService;
    private keySplittingService: KeySplittingService
    private loggerConfigService: LoggerConfigService;
    private logger: Logger;

    private mixpanelService: MixpanelService;

    private ssmTargets: Promise<TargetSummary[]>;
    private dynamicConfigs: Promise<TargetSummary[]>;
    private clusterTargets: Promise<ClusterDetails[]>;
    private envs: Promise<EnvironmentDetails[]>;

    // use the following to shortcut middleware according to command
    private oauthCommands: string[] = [
        'kube',
        'ssh-proxy-config',
        'connect',
        'tunnel',
        'user',
        'targetUser',
        'describe-cluster',
        'disconnect',
        'attach-to-connection',
        'close',
        'list-targets',
        'lt',
        'list-clusters',
        'lk',
        'list-connections',
        'lc',
        'copy',
        'ssh-proxy',
        'autodiscovery-script',
        'generate',
        'policy',
        'group',
        'generate-bash'
    ];

    private mixpanelCommands: string[] = [
        'kube',
        'ssh-proxy-config',
        'connect',
        'tunnel',
        'user',
        'targetUser',
        'describe-cluster',
        'disconnect',
        'attach-to-connection',
        'close',
        'list-targets',
        'lt',
        'list-clusters',
        'lk',
        'list-connections',
        'lc',
        'copy',
        'ssh-proxy',
        'autodiscovery-script',
        'generate',
        'policy',
        'group',
        'generate-bash'
    ];

    private fetchCommands: string[] = [
        'connect',
        'tunnel',
        'user',
        'targetUser',
        'describe-cluster',
        'disconnect',
        'attach-to-connection',
        'close',
        'list-targets',
        'lt',
        'list-clusters',
        'lk',
        'list-connections',
        'lc',
        'copy',
        'ssh-proxy',
        'autodiscovery-script',
        'generate',
        'policy',
        'group',
        'generate-bash'
    ];

    private adminOnlyCommands: string[] = [
        'group',
        'user',
        'targetUser',
        'policy'
    ];

    // available options for TargetType autogenerated from enum
    private targetTypeChoices: string[] = Object.keys(TargetType).map(tt => tt.toLowerCase());
    private targetStatusChoices: string[] = Object.keys(TargetStatus).map(s => s.toLowerCase());

    // available options for PolicyType autogenerated from enum
    private policyTypeChoices: string[] = Object.keys(PolicyType).map(s => s.toLowerCase());

    // Mapping from env vars to options if they exist
    private envMap: Dictionary<string> = {
        'configName': process.env.ZLI_CONFIG_NAME || 'prod',
        'enableKeysplitting': process.env.ZLI_ENABLE_KEYSPLITTING || 'true'
    };

    public start()
    {
        // @ts-ignore TS2589
        yargs(process.argv.slice(2))
            .scriptName('zli')
            .usage('$0 <cmd> [args]')
            .wrap(null)
            .middleware(async (argv) => {
                const initResponse = await initMiddleware(argv);
                this.loggerConfigService = initResponse.loggingConfigService;
                this.logger = initResponse.logger;
                this.configService = initResponse.configService;
                this.keySplittingService = initResponse.keySplittingService;
            })
            .middleware(async (argv) => {
                if(!includes(this.oauthCommands, argv._[0]))
                    return;
                await checkVersionMiddleware(this.configService, this.logger);
            })
            .middleware(async (argv) => {
                if(!includes(this.oauthCommands, argv._[0]))
                    return;
                await oAuthMiddleware(this.configService, this.logger);
            })
            .middleware(async (argv) => {
                if(includes(this.adminOnlyCommands, argv._[0]) && !this.configService.me().isAdmin){
                    this.logger.error(`This is an admin restricted command. Please login as an admin to perform it.`);
                    await cleanExit(1, this.logger);
                }
            })
            .middleware(async (argv) => {
                if(!includes(this.mixpanelCommands, argv._[0]))
                    return;
                this.mixpanelService = mixpanelTrackingMiddleware(this.configService, argv);
            })
            .middleware((argv) => {
                if(!includes(this.fetchCommands, argv._[0]))
                    return;

                const fetchDataResponse = fetchDataMiddleware(this.configService, this.logger);
                this.dynamicConfigs = fetchDataResponse.dynamicConfigs;
                this.clusterTargets = fetchDataResponse.clusterTargets;
                this.ssmTargets = fetchDataResponse.ssmTargets;
                this.envs = fetchDataResponse.envs;
            })
            .command(
                'login',
                'Login through your identity provider',
                (yargs) => {
                    return loginCmdBuilder(yargs);
                },
                async (argv) => {
                    await loginHandler(this.configService, this.logger, argv, this.keySplittingService);
                }
            )
            .command(
                'connect <targetString>',
                'Connect to a target',
                (yargs) => {
                    return connectCmdBuilder(yargs, this.targetTypeChoices);
                },
                async (argv) => {
                    const parsedTarget = await disambiguateTarget(argv.targetType, argv.targetString, this.logger, this.dynamicConfigs, this.ssmTargets, this.envs);

                    await connectHandler(this.configService, this.logger, this.mixpanelService, parsedTarget);
                }
            )
            .command(
                'tunnel [tunnelString]',
                'Tunnel to a cluster',
                (yargs) => {
                    return tunnelCmdBuilder(yargs);
                },
                async (argv) => {
                    if (argv.tunnelString) {
                        const [connectUser, connectCluster] = argv.tunnelString.split('@');

                        await startKubeDaemonHandler(argv, connectUser, connectCluster, this.clusterTargets, this.configService, this.logger, this.loggerConfigService);
                    } else {
                        await kubeStatusHandler(this.configService, this.logger);
                    }
                }
            )
            .command(
                ['policy [type]'],
                false, // This removes the command from the help text
                (yargs) => {
                    return policyCmdBuilder(yargs, this.policyTypeChoices);
                },
                async (argv) => {
                    await listPoliciesHandler(argv, this.configService, this.logger, this.ssmTargets, this.dynamicConfigs, this.clusterTargets, this.envs);
                }
            )
            .command(
                'describe-cluster <clusterName>',
                'Get detailed information about a certain cluster',
                (yargs) => {
                    return describeClusterCmdBuilder(yargs);
                },
                async (argv) => {
                    await describeClusterHandler(argv.clusterName, this.configService, this.logger, this.clusterTargets, this.envs);
                }
            )
            .command(
                'disconnect',
                'Disconnect a Zli Daemon',
                (yargs) => {
                    return disconnectCmdBuilder(yargs);
                },
                async (_) => {
                    await disconnectHandler(this.configService, this.logger);
                }
            )
            .command(
                'attach-to-connection <connectionId>',
                'Attach to an open zli connection',
                (yargs) => {
                    return attachCmdBuilder(yargs);
                },
                async (argv) => {
                    if (!isGuid(argv.connectionId)){
                        this.logger.error(`Passed connection id ${argv.connectionId} is not a valid Guid`);
                        await cleanExit(1, this.logger);
                    }
                    await attachHandler(this.configService, this.logger, argv.connectionId);
                }
            )
            .command(
                'close [connectionId]',
                'Close an open zli connection',
                (yargs) => {
                    return closeConnectionCmdBuilder(yargs);
                },
                async (argv) => {
                    if (! argv.all && ! isGuid(argv.connectionId)){
                        this.logger.error(`Passed connection id ${argv.connectionId} is not a valid Guid`);
                        await cleanExit(1, this.logger);
                    }
                    await closeConnectionHandler(this.configService, this.logger, argv.connectionId, argv.all);
                }
            )
            .command(
                ['list-targets', 'lt'],
                'List all targets (filters available)',
                (yargs) => {
                    return listTargetsCmdBuilder(yargs, this.targetTypeChoices, this.targetStatusChoices);
                },
                async (argv) => {
                    await listTargetsHandler(this.configService,this.logger, argv, this.dynamicConfigs, this.ssmTargets, this.clusterTargets, this.envs);
                }
            )
            .command(
                ['list-connections', 'lc'],
                'List all open zli connections',
                (yargs) => {
                    return listConnectionsCmdBuilder(yargs);
                },
                async (argv) => {
                    await listConnectionsHandler(argv, this.configService, this.logger, this.ssmTargets);
                }
            )
            .command(
                ['user [policyName] [idpEmail]'],
                false, // This removes the command from the help text
                (yargs) => {
                    return userCmdBuilder(yargs);
                },
                async (argv) => {
                    if (!! argv.add) {
                        await addUserToPolicyHandler(argv.idpEmail, argv.policyName, this.configService, this.logger);
                    } else if (!! argv.delete) {
                        await deleteUserFromPolicyHandler(argv.idpEmail, argv.policyName, this.configService, this.logger);
                    } else if (!(!!argv.add && !!argv.delete)) {
                        await listUsersHandler(argv, this.configService, this.logger);
                    } else {
                        this.logger.error(`Invalid flags combination. Please see help.`);
                        await cleanExit(1, this.logger);
                    }
                }
            )
            .command(
                ['group [policyName] [groupName]'],
                false, // This removes the command from the help text
                (yargs) => {
                    return groupCmdBuilder(yargs);
                },
                async (argv) => {
                    if (!! argv.add) {
                        await addGroupToPolicyHandler(argv.groupName, argv.policyName, this.configService, this.logger);
                    } else if (!! argv.delete) {
                        await deleteGroupFromPolicyHandler(argv.groupName, argv.policyName, this.configService, this.logger);
                    } else if (!(!!argv.add && !!argv.delete)) {
                        await fetchGroupsHandler(argv, this.configService, this.logger);
                    } else {
                        this.logger.error(`Invalid flags combination. Please see help.`);
                        await cleanExit(1, this.logger);
                    }
                }
            )
            .command(
                ['targetUser <policyName> [user]'],
                false, // This removes the command from the help text
                (yargs) => {
                    return targetUserCmdBuilder(yargs);
                },
                async (argv) => {
                    if (!! argv.add) {
                        await addTargetUserHandler(argv.user, argv.policyName, this.configService, this.logger);
                    } else if (!! argv.delete) {
                        await deleteTargetUserHandler(argv.user, argv.policyName, this.configService, this.logger);
                    } else if (!(!!argv.add && !!argv.delete)) {
                        await listTargetUsersHandler(this.configService, this.logger, argv, argv.policyName);
                    } else {
                        this.logger.error(`Invalid flags combination. Please see help.`);
                        await cleanExit(1, this.logger);
                    }
                }
            )
            .command(
                'ssh-proxy-config',
                'Generate ssh configuration to be used with the ssh-proxy command',
                (_) => {},
                async (_) => {
                    // ref: https://nodejs.org/api/process.html#process_process_argv0
                    let processName = process.argv0;

                    // handle npm install edge case
                    // note: node will also show up when running 'npm run start -- ssh-proxy-config'
                    // so for devs, they should not rely on generating configs from here and should
                    // map their dev executables in the ProxyCommand output
                    if(processName.includes('node')) processName = 'zli';

                    sshProxyConfigHandler(this.configService, this.logger, processName);
                }
            )
            .command(
                'ssh-proxy <host> <user> <port> <identityFile>',
                'ssh proxy command (run ssh-proxy-config command to generate configuration)',
                (yargs) => {
                    return sshProxyCmdBuilder(yargs);
                },
                async (argv) => {
                    let prefix = 'bzero-';
                    const configName = this.configService.getConfigName();
                    if(configName != 'prod') {
                        prefix = `${configName}-${prefix}`;
                    }

                    if(! argv.host.startsWith(prefix)) {
                        this.logger.error(`Invalid host provided must have form ${prefix}<target>. Target must be either target id or name`);
                        await cleanExit(1, this.logger);
                    }

                    // modify argv to have the targetString and targetType params
                    const targetString = argv.user + '@' + argv.host.substr(prefix.length);
                    const parsedTarget = await disambiguateTarget('ssm', targetString, this.logger, this.dynamicConfigs, this.ssmTargets, this.envs);

                    if(argv.port < 1 || argv.port > 65535)
                    {
                        this.logger.warn(`Port ${argv.port} outside of port range [1-65535]`);
                        await cleanExit(1, this.logger);
                    }

                    const sshTunnelParameters: SshTunnelParameters = {
                        parsedTarget: parsedTarget,
                        port: argv.port,
                        identityFile: argv.identityFile
                    };

                    await sshProxyHandler(this.configService, this.logger, sshTunnelParameters, this.keySplittingService, this.envMap);
                }
            )
            .command(
                'configure',
                'Returns config file path',
                () => {},
                async () => {
                    await configHandler(this.logger, this.configService, this.loggerConfigService);
                }
            )
            .command(
                'generate-bash',
                'Returns a bash script to autodiscover a target.',
                (yargs) => {
                    return generateBashCmdBuilder(process.argv, yargs) ;
                },
                async (argv) => {
                    await generateBashHandler(argv, this.logger, this.configService, this.envs);
                },
            )
            .command(
                'autodiscovery-script <operatingSystem> <targetName> <environmentName> [agentVersion]',
                'Returns autodiscovery script. [deprecated. Use generate-bash instead]',
                (yargs) => {
                    return autoDiscoveryScriptCommandBuilder(yargs);
                },
                async (argv) => {
                    await autoDiscoveryScriptHandler(argv, this.logger, this.configService, this.envs);
                }
            )
            .command(
                'generate <typeOfConfig> [clusterName]',
                'Generate a different types of configuration files',
                (yargs) => {
                    return generateKubeCmdBuilder(yargs);
                },
                async (argv) => {
                    if (argv.typeOfConfig == 'kubeConfig') {
                        await generateKubeconfigHandler(argv, this.configService, this.logger);
                    } else if (argv.typeOfConfig == 'kubeYaml') {
                        await generateKubeYamlHandler(argv, this.envs, this.configService, this.logger);
                    }
                }
            )
            .command(
                'logout',
                'Deauthenticate the client',
                () => {},
                async () => {
                    await logoutHandler(this.configService, this.logger);
                }
            )
            .command('kube', 'Kubectl wrapper catch all', (yargs) => {
                return yargs.example('$0 kube -- get pods', '');
            }, async (argv: any) => {
                // This expects that the kube command will go after the --
                const listOfCommands = argv._.slice(1); // this removes the 'kube' part of 'zli kube -- ...'
                await bctlHandler(this.configService, this.logger, listOfCommands);
            })
            .option('configName', {type: 'string', choices: ['prod', 'stage', 'dev'], default: this.envMap['configName'], hidden: true})
            .option('debug', {type: 'boolean', default: false, describe: 'Flag to show debug logs'})
            .option('silent', {alias: 's', type: 'boolean', default: false, describe: 'Silence all zli messages, only returns command output'})
            .strictCommands() // if unknown command, show help
            .demandCommand() // if no command, show help
            .help() // auto gen help message
            .showHelpOnFail(false)
            .epilog(`Note:
 - <targetString> format: ${targetStringExample}

For command specific help: zli <cmd> help

Command arguments key:
 - <arg> is required
 - [arg] is optional or sometimes required

Need help? https://cloud.bastionzero.com/support`)
            .argv; // returns argv of yargs
    }
}