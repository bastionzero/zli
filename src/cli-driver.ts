import { ClusterSummary, IdP, TargetStatus, TargetSummary, TargetType } from './types';
import {
    disambiguateTarget,
    isGuid,
    targetStringExample
} from './utils';
import { ConfigService } from './config.service/config.service';
import { MixpanelService } from './mixpanel.service/mixpanel.service';
import { checkVersionMiddleware } from './middlewares/check-version-middleware';
import { EnvironmentDetails, PolicyType } from './http.service/http.service.types';
import { Logger } from './logger.service/logger';
import { LoggerConfigService } from './logger-config.service/logger-config.service';
import { KeySplittingService } from '../webshell-common-ts/keysplitting.service/keysplitting.service';
import { cleanExit } from './handlers/clean-exit.handler';

// Handlers
import { initMiddleware, oAuthMiddleware, fetchDataMiddleware, mixpanelTrackingMiddleware } from './handlers/middleware.handler';
import { sshProxyConfigHandler } from './handlers/ssh-proxy-config.handler';
import { sshProxyHandler, SshTunnelParameters } from './handlers/ssh-proxy.handler';
import { loginHandler } from './handlers/login.handler';
import { connectHandler } from './handlers/connect.handler';
import { listTargetsHandler } from './handlers/list-targets.handler';
import { configHandler } from './handlers/config.handler';
import { logoutHandler } from './handlers/logout.handler';
import { startKubeDaemonHandler } from './handlers/start-kube-daemon.handler';
import { autoDiscoveryScriptHandler } from './handlers/autodiscovery-script-handler';
import { listConnectionsHandler } from './handlers/list-connections.handler';
import { listUsersHandler } from './handlers/list-users.handler';
import { attachHandler } from './handlers/attach.handler';
import { closeConnectionHandler } from './handlers/close-connection.handler';
import { generateKubeconfigHandler } from './handlers/generate-kubeconfig.handler';
import { addTargetUserHandler } from './handlers/add-target-user.handler';
import { deleteTargetUserHandler } from './handlers/delete-target-user.handler';
import { addUserToPolicyHandler } from './handlers/add-user-policy.handler';
import { deleteUserFromPolicyHandler } from './handlers/delete-user-policy.handler';
import { generateKubeYamlHandler } from './handlers/generate-kube-yaml.handler';
import { disconnectHandler } from './handlers/disconnect.handler';
import { kubeStatusHandler } from './handlers/status.handler';
import { bctlHandler } from './handlers/bctl.handler';
import { listPoliciesHandler } from './handlers/list-policies.handler';
import { listTargetUsersHandler } from './handlers/list-target-users.handler';
import { fetchGroupsHandler } from './handlers/fetch-groups.handler';

// 3rd Party Modules
import { Dictionary, includes } from 'lodash';
import yargs from 'yargs';
import { describeClusterHandler } from './handlers/describe-cluster.handler';
import { deleteGroupFromPolicyHandler } from './handlers/delete-group-policy.handler';
import { addGroupToPolicyHandler } from './handlers/add-group-policy.handler';

export class CliDriver
{
    private configService: ConfigService;
    private keySplittingService: KeySplittingService
    private loggerConfigService: LoggerConfigService;
    private logger: Logger;

    private mixpanelService: MixpanelService;

    private ssmTargets: Promise<TargetSummary[]>;
    private dynamicConfigs: Promise<TargetSummary[]>;
    private clusterTargets: Promise<ClusterSummary[]>;
    private envs: Promise<EnvironmentDetails[]>;

    // use the following to shortcut middleware according to command
    private oauthCommands: string[] = [
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
        'group'
    ];

    private mixpanelCommands: string[] = [
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
        'group'
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
        'group'
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
                await checkVersionMiddleware(this.logger);
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
                'login <provider>',
                'Login through a specific provider',
                (yargs) => {
                    return yargs
                        .positional('provider', {
                            type: 'string',
                            choices: [IdP.Google, IdP.Microsoft]
                        })
                        .option(
                            'mfa',
                            {
                                type: 'string',
                                demandOption: false,
                                alias: 'm'
                            }
                        )
                        .example('login Google', 'Login with Google')
                        .example('login Microsoft --mfa 123456', 'Login with Microsoft and enter MFA');
                },
                async (argv) => {
                    await loginHandler(this.configService, this.logger, argv, this.keySplittingService);
                }
            )
            .command(
                'connect <targetString>',
                'Connect to a target',
                (yargs) => {
                    return yargs
                        .positional('targetString', {
                            type: 'string',
                        })
                        .option(
                            'targetType',
                            {
                                type: 'string',
                                choices: this.targetTypeChoices,
                                demandOption: false,
                                alias: 't'
                            },
                        )
                        .example('connect ssm-user@neat-target', 'SSM connect example, uniquely named ssm target')
                        .example('connect --targetType dynamic ssm-user@my-dat-config', 'DAT connect example with a DAT configuration whose name is my-dat-config');
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
                    return yargs
                        .positional('tunnelString', {
                            type: 'string',
                            default: null,
                            demandOption: false,
                        }).option('customPort', {
                            type: 'number',
                            default: -1,
                            demandOption: false
                        })
                        .example('proxy admin@neat-cluster', 'Connect to neat-cluster as the admin Kube RBAC role');
                },
                async (argv) => {
                    if (argv.tunnelString) {
                        // TODO make this smart parsing
                        const connectUser = argv.tunnelString.split('@')[0];
                        const connectCluster = argv.tunnelString.split('@')[1];

                        await startKubeDaemonHandler(argv, connectUser, connectCluster, this.clusterTargets, this.configService, this.logger);
                    } else {
                        await kubeStatusHandler(this.configService, this.logger);
                    }
                }
            )
            .command(
                ['policy [type]'],
                false, // This removes the command from the help text
                (yargs) => {
                    return yargs
                        .option(
                            'type',
                            {
                                type: 'string',
                                choices: this.policyTypeChoices,
                                alias: 't',
                                demandOption: false
                            }
                        )
                        .option(
                            'json',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'j',
                            }
                        )
                        .example('policy --json', 'List all policies, output as json, pipeable');
                },
                async (argv) => {
                    await listPoliciesHandler(argv, this.configService, this.logger, this.ssmTargets, this.dynamicConfigs, this.clusterTargets, this.envs);
                }
            )
            .command(
                'describe-cluster <clusterName>',
                'Get detailed information about a certain cluster',
                (yargs) => {
                    return yargs
                        .positional('clusterName', {
                            type: 'string',
                        })
                        .example('status test-cluster', '');
                },
                async (argv) => {
                    await describeClusterHandler(argv.clusterName, this.configService, this.logger, this.clusterTargets, this.envs);
                }
            )
            .command(
                'disconnect',
                'Disconnect a Zli Daemon',
                (yargs) => {
                    return yargs
                        .example('disconnect', 'Disconnect a local Zli Daemon');
                },
                async (_) => {
                    await disconnectHandler(this.configService, this.logger);
                }
            )
            .command(
                'attach-to-connection <connectionId>',
                'Attach to an open zli connection',
                (yargs) => {
                    return yargs
                        .positional('connectionId', {
                            type: 'string',
                        })
                        .example('attach d5b264c7-534c-4184-a4e4-3703489cb917', 'attach example, unique connection id');
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
                    return yargs
                        .positional('connectionId', {
                            type: 'string',
                        })
                        .option(
                            'all',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'a'
                            }
                        )
                        .example('close d5b264c7-534c-4184-a4e4-3703489cb917', 'close example, unique connection id')
                        .example('close all', 'close all connections in cli-space');
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
                    return yargs
                        .option(
                            'targetType',
                            {
                                type: 'string',
                                choices: this.targetTypeChoices,
                                demandOption: false,
                                alias: 't'
                            }
                        )
                        .option(
                            'env',
                            {
                                type: 'string',
                                demandOption: false,
                                alias: 'e'
                            }
                        )
                        .option(
                            'name',
                            {
                                type: 'string',
                                demandOption: false,
                                alias: 'n'
                            }
                        )
                        .option(
                            'status',
                            {
                                type: 'string',
                                array: true,
                                choices: this.targetStatusChoices,
                                alias: 'u'
                            }
                        )
                        .option(
                            'detail',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'd'
                            }
                        )
                        .option(
                            'showId',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'i'
                            }
                        )
                        .option(
                            'json',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'j',
                            }
                        )
                        .example('lt -t ssm', 'List all SSM targets only')
                        .example('lt -i', 'List all targets and show unique ids')
                        .example('lt -e prod --json --silent', 'List all targets targets in prod, output as json, pipeable');
                },
                async (argv) => {
                    await listTargetsHandler(this.configService,this.logger, argv, this.dynamicConfigs, this.ssmTargets, this.clusterTargets, this.envs);
                }
            )
            .command(
                ['list-connections', 'lc'],
                'List all open zli connections',
                (yargs) => {
                    return yargs
                        .option(
                            'json',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'j',
                            }
                        )
                        .example('lc --json', 'List all open zli connections, output as json, pipeable');
                },
                async (argv) => {
                    await listConnectionsHandler(argv, this.configService, this.logger, this.ssmTargets);
                }
            )
            .command(
                ['user [policyName] [idpEmail]'],
                false, // This removes the command from the help text
                (yargs) => {
                    return yargs
                        .option(
                            'add',
                            {
                                type: 'boolean',
                                demandOption: false,
                                alias: 'a',
                                implies: ['idpEmail', 'policyName']
                            }
                        )
                        .option(
                            'delete',
                            {
                                type: 'boolean',
                                demandOption: false,
                                alias: 'd',
                                implies: ['idpEmail', 'policyName']
                            }
                        )
                        .conflicts('add', 'delete')
                        .positional('idpEmail',
                            {
                                type: 'string',
                                default: null,
                                demandOption: false,
                            }
                        )
                        .positional('policyName',
                            {
                                type: 'string',
                                default: null,
                                demandOption: false,
                            }
                        )
                        .option(
                            'json',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'j',
                            }
                        )
                        .example('user --json', 'List all users, output as json, pipeable')
                        .example('user --add test@test.com test-cluster', 'Adds the test@test.com IDP user to test-cluster policy')
                        .example('user -d test@test.com test-cluster', 'Removes the test@test.com IDP user from test-cluster policy');
                },
                async (argv) => {
                    if (!! argv.add) {
                        await addUserToPolicyHandler(argv.idpEmail, argv.policyName, this.clusterTargets, this.configService, this.logger);
                    } else if (!! argv.delete) {
                        await deleteUserFromPolicyHandler(argv.idpEmail, argv.policyName, this.clusterTargets, this.configService, this.logger);
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
                    return yargs
                        .option(
                            'add',
                            {
                                type: 'boolean',
                                demandOption: false,
                                alias: 'a',
                                implies: ['groupName', 'policyName']
                            }
                        )
                        .option(
                            'delete',
                            {
                                type: 'boolean',
                                demandOption: false,
                                alias: 'd',
                                implies: ['groupName', 'policyName']
                            }
                        )
                        .conflicts('add', 'delete')
                        .positional('groupName',
                            {
                                type: 'string',
                                default: null,
                                demandOption: false,
                            }
                        )
                        .positional('policyName',
                            {
                                type: 'string',
                                default: null,
                                demandOption: false,
                            }
                        )
                        .option(
                            'json',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'j',
                            }
                        )
                        .example('group --json', 'List all groups, output as json, pipeable')
                        .example('group --add cool-policy engineering-group', 'Adds the engineering-group IDP group to cool-policy policy')
                        .example('group -d cool-policy engineering-group', 'Deletes the engineering-group IDP group from the cool-policy policy');
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
                    return yargs
                        .option(
                            'add',
                            {
                                type: 'boolean',
                                demandOption: false,
                                alias: 'a',
                                implies: ['user', 'policyName']
                            }
                        )
                        .option(
                            'delete',
                            {
                                type: 'boolean',
                                demandOption: false,
                                alias: 'd',
                                implies: ['user', 'policyName']
                            }
                        )
                        .conflicts('add', 'delete')
                        .positional('user',
                            {
                                type: 'string',
                                default: null,
                                demandOption: false,
                            }
                        )
                        .positional('policyName',
                            {
                                type: 'string',
                                default: null,
                                demandOption: true,
                            }
                        )
                        .option(
                            'json',
                            {
                                type: 'boolean',
                                default: false,
                                demandOption: false,
                                alias: 'j',
                            }
                        )
                        .example('targetUser --json', 'List all target users, output as json, pipeable')
                        .example('targetUser --add cool-policy centos', 'Adds the centos user to cool-policy')
                        .example('targetUser -d test-cluster admin', 'Removes the admin RBAC Role to the test-cluster policy');
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
                    return yargs
                        .positional('host', {
                            type: 'string',
                        })
                        .positional('user', {
                            type: 'string',
                        })
                        .positional('port', {
                            type: 'number',
                        })
                        .positional('identityFile', {
                            type: 'string'
                        });
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
                'autodiscovery-script <operatingSystem> <targetName> <environmentName> [agentVersion]',
                'Returns autodiscovery script',
                (yargs) => {
                    return yargs
                        .positional('operatingSystem', {
                            type: 'string',
                            choices: ['centos', 'ubuntu']
                        })
                        .positional('targetName', {
                            type: 'string'
                        })
                        .positional('environmentName', {
                            type: 'string',
                        })
                        .positional('agentVersion', {
                            type: 'string',
                            default: 'latest'
                        })
                        .option(
                            'outputFile',
                            {
                                type: 'string',
                                demandOption: false,
                                alias: 'o'
                            }
                        )
                        .example('autodiscovery-script centos sample-target-name Default', '');
                },
                async (argv) => {
                    await autoDiscoveryScriptHandler(argv, this.logger, this.configService, this.envs);
                }
            )
            .command(
                'generate <typeOfConfig> [clusterName]',
                'Generate a different types of configuration files',
                (yargs) => {
                    return yargs
                        .positional('typeOfConfig', {
                            type: 'string',
                            choices: ['kubeConfig', 'kubeYaml']

                        }).positional('clusterName', {
                            type: 'string',
                            default: null
                        }).option('namespace', {
                            type: 'string',
                            default: '',
                            demandOption: false
                        }).option('labels', {
                            type: 'array',
                            default: [],
                            demandOption: false
                        }).option('customPort', {
                            type: 'number',
                            default: -1,
                            demandOption: false
                        }).option('outputFile', {
                            type: 'string',
                            demandOption: false,
                            alias: 'o',
                            default: null
                        })
                        .option('environmentId', {
                            type: 'string',
                            default: null
                        })
                        .example('generate kubeYaml testcluster', '')
                        .example('generate kubeConfig', '')
                        .example('generate kubeYaml --labels testkey:testvalue', '');
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
            .command('$0', 'Kubectl wrapper catch all', () => {}, async (_) => {
                await bctlHandler(this.configService, this.logger);
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