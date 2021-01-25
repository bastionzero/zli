import { IdP, SessionState, TargetType } from "./types";
import { 
    checkTargetTypeAndStringPair, 
    findSubstring, 
    parsedTargetString, 
    parseTargetString, 
    getTableOfTargets, 
    targetStringExample, 
    targetStringExampleNoPath, 
    TargetSummary
} from './utils';
import yargs from "yargs";
import { ConfigService } from "./config.service/config.service";
import { 
    ConnectionService, 
    EnvironmentService, 
    FileService, 
    MfaService, 
    SessionService, 
    SshTargetService, 
    SsmTargetService, 
    UserService
} from "./http.service/http.service";
import { OAuthService } from "./oauth.service/oauth.service";
import { ShellTerminal } from "./terminal/terminal";
import termsize from 'term-size';
import { UserinfoResponse } from "openid-client";
import { MixpanelService } from "./mixpanel.service/mixpanel.service";
import { checkVersionMiddleware } from "./middlewares/check-version-middleware";
import { oauthMiddleware } from "./middlewares/oauth-middleware";
import fs from 'fs'
import { EnvironmentDetails, ConnectionState, MfaActionRequired } from "./http.service/http.service.types";
import { includes } from "lodash";
import { version } from '../package.json';
import qrcode from 'qrcode';
import { Logger } from './logger.service/logger'
import { LoggerConfigService } from "./logger-config.service/logger-config.service";

export class CliDriver
{
    private configService: ConfigService;
    private loggerConfigService: LoggerConfigService;
    private userInfo: UserinfoResponse; // sub and email
    private logger: Logger;

    private mixpanelService: MixpanelService;

    private sshTargets: Promise<TargetSummary[]>;
    private ssmTargets: Promise<TargetSummary[]>;
    private envs: Promise<EnvironmentDetails[]>;

    // use the following to shortcut middleware according to command
    private noOauthCommands: string[] = ['config', 'login', 'logout'];
    private noMixpanelCommands: string[] = ['config', 'login', 'logout'];
    private noFetchCommands: string[] = ['config', 'login', 'logout'];

    public start()
    {
        yargs(process.argv.slice(2))
        .scriptName("thoum")
        .usage('$0 <cmd> [args]')
        .wrap(null)
        .middleware(checkVersionMiddleware)
        .middleware((argv) => {
            // Configure our logger
            this.loggerConfigService = new LoggerConfigService(<string> argv.configName);
            this.logger = new Logger(this.loggerConfigService, !!argv.debug);

            // Config init
            this.configService = new ConfigService(<string> argv.configName, this.logger);
        })
        .middleware(async (argv) => {
            if(includes(this.noOauthCommands, argv._[0]))
                return;

            // OAuth
            this.userInfo = await oauthMiddleware(this.configService, this.logger);
            const me = this.configService.me(); // if you have logged in, this should be set
            const sessionId = this.configService.sessionId();
            this.logger.info(`Logged in as: ${this.userInfo.email}, clunk80-id:${me.id}, session-id:${sessionId}`);
        })
        .middleware(async (argv) => {
            if(includes(this.noMixpanelCommands, argv._[0]))
                return;

            // Mixpanel tracking
            this.mixpanelService = new MixpanelService(
                this.configService.mixpanelToken(),
                this.userInfo.sub
            );

            // Only captures args, not options at the moment. Capturing configName flag
            // does not matter as that is handled by which mixpanel token is used
            // TODO: capture options and flags
            this.mixpanelService.TrackCliCall(
                'CliCommand', 
                {
                    "cli-version": version,
                    "command": argv._[0],
                    args: argv._.slice(1)
                }
            );
        })
        .middleware((argv) => {
            if(includes(this.noFetchCommands, argv._[0]))
                return;

            // Greedy fetch of some data that we use frequently 
            const ssmTargetService = new SsmTargetService(this.configService, this.logger);
            const sshTargetService = new SshTargetService(this.configService, this.logger);
            const envService = new EnvironmentService(this.configService, this.logger);

            this.ssmTargets = ssmTargetService.ListSsmTargets()
                .then(result => 
                    result.map<TargetSummary>((ssm, _index, _array) => {
                        return {type: TargetType.SSM, id: ssm.id, name: ssm.name, environmentId: ssm.environmentId};
                    })
                );


            this.sshTargets = sshTargetService.ListSshTargets()
                .then(result => 
                    result.map<TargetSummary>((ssh, _index, _array) => {
                        return {type: TargetType.SSH, id: ssh.id, name: ssh.alias, environmentId: ssh.environmentId};
                })
            );

            this.envs = envService.ListEnvironments();
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
                .example('login Google', 'Login with Google');
            },
            async (argv) => {
                // Clear previous log in info
                this.configService.logout();

                const provider = <IdP> argv.provider;
                await this.configService.loginSetup(provider);
                
                // Can only create oauth service after loginSetup completes
                const oAuthService = new OAuthService(this.configService, this.logger);
                if(! oAuthService.isAuthenticated())
                {
                    this.logger.info('Login required, opening browser');
                    await oAuthService.login((t) => this.configService.setTokenSet(t));
                    this.userInfo = await oAuthService.userInfo();
                }
                
                // Register user log in and get User Session Id
                const userService = new UserService(this.configService, this.logger);
                const registerResponse = await userService.Register();
                this.configService.setSessionId(registerResponse.userSessionId);
                
                // Check if we must MFA and act upon it
                const mfaService = new MfaService(this.configService, this.logger);
                switch(registerResponse.mfaActionRequired)
                {
                    case MfaActionRequired.NONE:
                        break;
                    case MfaActionRequired.TOTP:
                        if(! argv.mfa)
                        {
                            this.logger.warn('MFA token required for this account');
                            this.logger.info('Please try logging in again with \'--mfa <token\' flag');
                            this.configService.logout();
                            process.exit(1);
                        }

                        await mfaService.SendTotp(argv.mfa);
                        
                        break;
                    case MfaActionRequired.RESET:
                        this.logger.info('MFA reset detected, requesting new MFA token');
                        this.logger.info('Please scan the following QR code with your device (Google Authenticator recommended)');

                        const resp = await mfaService.ResetSecret();
                        var data = await qrcode.toString(resp.mfaSecretUrl, {type: 'terminal', scale: 2});
                        console.log(data);

                        this.logger.info('Please log in again with \'--mfa token\'');
                        
                        break;
                    default:
                        this.logger.warn(`Unexpected MFA response ${registerResponse.mfaActionRequired}`);
                        break;
                }

                const me = await userService.Me();
                this.configService.setMe(me);
                this.logger.info(`Logged in as: ${this.userInfo.email}, clunk80-id:${me.id}, session-id:${registerResponse.userSessionId}`)
                
                process.exit(0);
            }
        )
        // TODO: https://github.com/yargs/yargs/blob/master/docs/advanced.md#commanddirdirectory-opts
        // <requiredPositional>, [optionalPositional]
        .command(
            'connect <targetType> <targetString>',
            'Connect to a target',
            (yargs) => {
                // you must return the yarg for the handler to have types
                return yargs
                .positional('targetType', {
                    type: 'string',
                    choices: ['ssm', 'ssh']
                })
                .positional('targetString', {
                    type: 'string',
                })
                .example('connect ssm ssm-user@neat-name', 'SSM connect example')
                .example('connect ssh dbda775d-e37c-402b-aa76-bbb0799fd775', 'SSH connect example');
            },
            async (argv) => {
                
                const parsedTarget = await this.disambiguateTargetName(argv.targetType, argv.targetString);

                // call list session
                const sessionService = new SessionService(this.configService, this.logger);
                const listSessions = await sessionService.ListSessions();

                // space names are not unique, make sure to find the latest active one
                const cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space' && s.state == SessionState.Active); // TODO: cli-space name can be changed in config

                // maybe make a session
                let cliSessionId: string;
                if(cliSpace.length === 0)
                {
                    cliSessionId =  await sessionService.CreateSession('cli-space');
                } else {
                    // there should only be 1 active 'cli-space' session
                    cliSessionId = cliSpace.pop().id;
                }

                // We do the following for ssh since we are required to pass
                // in a user although it does not get read at any point
                const targetUser = parsedTarget.type === TargetType.SSH ? 'ssh' : parsedTarget.user;

                // make a new connection
                const connectionService = new ConnectionService(this.configService, this.logger);
                // if SSM user does not exist then resp.connectionId will throw a 
                // 'TypeError: Cannot read property 'connectionId' of undefined'
                // so we need to catch and return undefined
                const connectionId = await connectionService.CreateConnection(parsedTarget.type, parsedTarget.id, cliSessionId, targetUser).catch(() => undefined);

                if(! connectionId)
                {
                    this.logger.error('Connection creation failed');
                    if(parsedTarget.type === TargetType.SSM)
                    {
                        const targetEnvId = (await this.ssmTargets).filter(ssm => ssm.id == parsedTarget.id)[0].environmentId;
                        const targetEnvName = (await this.envs).filter(e => e.id == targetEnvId)[0].name;
                        this.logger.error(`You may not have a policy for targetUser ${parsedTarget.user} in environment ${targetEnvName}`);
                        this.logger.info('You can find SSM user policies in the web app');
                    } else {
                        this.logger.info('Please check your polices in the web app for this target and/or environment');
                    }

                    process.exit(1);
                }

                // connect to target and run terminal
                var terminal = new ShellTerminal(this.configService, connectionId);
                terminal.start(termsize());

                this.mixpanelService.TrackNewConnection(parsedTarget.type);

                // Terminal resize event logic
                // https://nodejs.org/api/process.html#process_signal_events -> SIGWINCH
                // https://github.com/nodejs/node/issues/16194
                // https://nodejs.org/api/process.html#process_a_note_on_process_i_o
                process.stdout.on(
                    'resize', 
                    () => {
                        const resizeEvent = termsize();
                        terminal.resize(resizeEvent);
                    }
                );
                
                // If we detect a disconnection, close the connection immediately
                terminal.terminalRunning.subscribe(
                    () => {},
                    async (error) => {
                        if(error)
                        {
                            this.logger.error(error);
                            this.logger.warn('Target may have gone offline or space/connection closed from another client');
                        }

                        terminal.dispose();
                        
                        this.logger.debug('Cleaning up connection...');
                        const conn = await connectionService.GetConnection(connectionId);
                        // if connection not already closed
                        if(conn.state == ConnectionState.Open)
                            await connectionService.CloseConnection(connectionId);

                        this.logger.debug('Connection closed');
                        
                        if(error)
                            process.exit(1);

                        process.exit(0);
                    },
                    () => {}
                );

                // To get 'keypress' events you need the following lines
                // ref: https://nodejs.org/api/readline.html#readline_readline_emitkeypressevents_stream_interface
                const readline = require('readline');
                readline.emitKeypressEvents(process.stdin);
                process.stdin.setRawMode(true);
                process.stdin.on('keypress', (_, key) => terminal.writeString(key.sequence));
            }
        )
        .command(
            ['list-targets', 'lt'],
            'List all SSM and SSH targets (filters available)',
            (yargs) => {
                return yargs
                .option(
                    'targetType',
                    {
                        type: 'string',
                        choices: ['ssm', 'ssh'],
                        demandOption: false,
                        alias: 't'
                    },
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
            },
            async (argv) => {
                let ssmList = await this.ssmTargets;
                let sshList = await this.sshTargets;
                let envs = await this.envs;

                // find all envIds with substring search
                // filter targets down by endIds
                if(argv.env)
                {
                    const envIdFilter = envs.filter(e => findSubstring(argv.env, e.name)).map(e => e.id);

                    ssmList = ssmList.filter(ssm => envIdFilter.includes(ssm.environmentId));
                    sshList = sshList.filter(ssh => envIdFilter.includes(ssh.environmentId));
                }

                // filter targets by name/alias
                if(argv.name)
                {
                    ssmList = ssmList.filter(ssm => findSubstring(argv.name, ssm.name));
                    sshList = sshList.filter(ssh => findSubstring(argv.name, ssh.name));
                }

                let tableString: string;

                // push targets to printed table, maybe filter by targetType
                if(argv.targetType === 'ssm')
                {
                    tableString = getTableOfTargets(ssmList, envs);
                } else if(argv.targetType === 'ssh') {
                    tableString = getTableOfTargets(sshList, envs);
                } else {
                    tableString = getTableOfTargets(ssmList.concat(sshList), envs);
                }

                console.log(tableString);
                process.exit(0);
            }
        )
        .command(
            'copy <targetType> <source> <destination>',
            'Upload/download a file to target',
            (yargs) => {
                return yargs
                .positional('targetType', {
                    type: 'string',
                    choices: ['ssm', 'ssh']
                })
                .positional('source', {
                    type: 'string'
                })
                .positional('destination', {
                    type: 'string'
                })
                .example('copy ssm ssm-user@95b72b50-d09c-49fa-8825-332abfeb013e:/home/ssm-user/file.txt /Users/coolUser/newFileName.txt', 'SSM Download example')
                .example('copy ssm /Users/coolUser/secretFile.txt ssm-user@neat-target:/home/ssm-user/newFileName', 'SSM Upload example')
                .example('copy ssh /Users/coolUser/neatFile.txt cool-alias:/home/ssm-user/newFileName.txt', 'SSH Upload example');
            },
            async (argv) => {
                const fileService = new FileService(this.configService, this.logger);

                const sourceParsedString = parseTargetString(argv.targetType, argv.source);
                const destParsedString = parseTargetString(argv.targetType, argv.destination);
                const parsedTarget = sourceParsedString || destParsedString; // one of these will be undefined so javascript will use the other

                // figure out upload or download
                // would be undefined if not parsed properly
                if(destParsedString)
                {
                    // upload case
                    const fh = fs.createReadStream(argv.source);
                    if(fh.readableLength === 0)
                    {
                        this.logger.warn(`File ${argv.source} does not exist or cannot be read`);
                        process.exit(1);
                    }

                    await fileService.uploadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, fh, parsedTarget.user);
                    this.logger.info('File upload complete');

                } else if(sourceParsedString) {
                    // download case
                    await fileService.downloadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, argv.destination, parsedTarget.user);
                } else {
                    this.logger.error('Invalid target string, must follow syntax:');
                    this.logger.error(targetStringExample);
                }
                process.exit(0);
            }
        )
        .command(
            'config',
            'Returns config file path',
            () => {},
            () => {
                this.logger.info(`You can edit your config here: ${this.configService.configPath()}`);
                this.logger.info(`You can find your log files here: ${this.loggerConfigService.logPath()}`);
                process.exit(0);
            }
        ).command(
            'logout',
            'Deauthenticate the client',
            () => {},
            async () => {
                // Deletes the auth tokens from the config which will force the
                // user to login again before running another command
                this.configService.logout();
                process.exit(0);
            }
        )
        .option('configName', {type: 'string', choices: ['prod', 'stage', 'dev'], default: 'prod', hidden: true})
        .option('debug', {type: 'boolean', default: false, describe: 'Flag to show debug logs'})
        .strict() // if unknown command, show help
        .demandCommand() // if no command, show help
        .help() // auto gen help message
        .epilog(`Note:
 - <targetString> format: ${targetStringExample}
 - TargetStrings only require targetUser for SSM
 - TargetPath can be omitted

For command specific help: thoum <cmd> help

Command arguments key:
 - <arg> is required
 - [arg] is optional or sometimes required

Need help? https://app.clunk80.com/support`)
        .argv; // returns argv of yargs
    }

    // Figure out target id based on target name and target type.
    // Also preforms error checking on target type and target string passed in
    private async disambiguateTargetName(argvTargetType: string, argvTargetString: string) : Promise<parsedTargetString>
    {
        let parsedTarget = parseTargetString(argvTargetType, argvTargetString);

        if(! parsedTarget)
        {
            this.logger.error('Invalid target string, must follow syntax:');
            this.logger.error(targetStringExampleNoPath);
            process.exit(1);
        }

        if(! checkTargetTypeAndStringPair(parsedTarget))
        {   
            // print warning
            if(parsedTarget.type === TargetType.SSH)
            {
                this.logger.warn('Cannot specify targetUser for SSH connections');
                this.logger.warn('Please try your previous command without the targetUser');
                this.logger.warn('Target string for SSH: targetId[:path]');
            } else {
                this.logger.warn('Must specify targetUser for SSM connections');
                this.logger.warn('Target string for SSM: targetUser@targetId[:path]');
            }

            process.exit(1);
        }
            

        if(parsedTarget.name)
        {
            let matchedNamedTargets: TargetSummary[] = [];

            switch(parsedTarget.type)
            {
                case TargetType.SSM:
                    matchedNamedTargets = (await this.ssmTargets).filter(ssm => ssm.name === parsedTarget.name);
                    break;
                case TargetType.SSH:
                    matchedNamedTargets = (await this.sshTargets).filter(ssh => ssh.name === parsedTarget.name);
                    break;
                default:
                    this.logger.error(`Invalid TargetType passed ${parsedTarget.type}`);
                    process.exit(1);
            }

            if(matchedNamedTargets.length < 1)
            {
                this.logger.error(`No ${parsedTarget.type} targets found with name ${parsedTarget.name}`);
                this.logger.warn('Target names are case sensitive');
                this.logger.warn('To see list of all targets run: \'thoum lt\'');
                process.exit(1);
            } else if(matchedNamedTargets.length == 1)
            {
                // the rest of the flow will work as before since the targetId has now been disambiguated
                parsedTarget.id = matchedNamedTargets.pop().id;
            } else {
                // ambiguous target id, warn user, exit process
                this.logger.warn(`Multiple ${parsedTarget.type} targets found with name ${parsedTarget.name}`);
                const tableString = getTableOfTargets(matchedNamedTargets, await this.envs);
                console.log(tableString);
                this.logger.info('Please connect using \'target id\' instead of target name');
                process.exit(1);
            }
        }

        return parsedTarget;
    }
}