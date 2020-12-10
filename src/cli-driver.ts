import { SessionState, TargetType } from "./types";
import { checkTargetTypeAndStringPair, findSubstring, parsedTargetString, parseTargetString, getTableOfTargets, targetStringExample, targetStringExampleNoPath, TargetSummary, thoumError, thoumMessage, thoumWarn } from './utils';
import yargs from "yargs";
import { ConfigService } from "./config.service/config.service";
import { ConnectionService, EnvironmentService, FileService, SessionService, SshTargetService, SsmTargetService } from "./http.service/http.service";
import { OAuthService } from "./oauth.service/oauth.service";
import { ShellTerminal } from "./terminal/terminal";
import termsize from 'term-size';
import { UserinfoResponse } from "openid-client";
import { MixpanelService } from "./mixpanel.service/mixpanel.service";
import { checkVersionMiddleware } from "./middlewares/check-version-middleware";
import { oauthMiddleware } from "./middlewares/oauth-middleware";
import fs from 'fs'
import { EnvironmentDetails, } from "./http.service/http.service.types";

export class CliDriver
{
    private configService: ConfigService;
    private userInfo: UserinfoResponse; // sub and email

    private mixpanelService: MixpanelService;

    private sshTargets: Promise<TargetSummary[]>;
    private ssmTargets: Promise<TargetSummary[]>;
    private envs: Promise<EnvironmentDetails[]>;

    public start()
    {
        yargs(process.argv.slice(2))
        .scriptName("thoum")
        .usage('$0 <cmd> [args]')
        .wrap(null)
        .middleware(checkVersionMiddleware)
        .middleware((argv) =>
        {
            // Config init
            this.configService = new ConfigService(<string> argv.configName);
        })
        .middleware(async () => {
            // OAuth
            this.userInfo = await oauthMiddleware(this.configService);
            thoumMessage(`Logged in as: ${this.userInfo.email}, clunk80-id:${this.userInfo.sub}`);
        })
        .middleware(async (argv) => {
            // Mixpanel tracking
            this.mixpanelService = new MixpanelService(
                this.configService.mixpanelToken(),
                this.userInfo.sub
            );

            // Only captures args, not options at the moment. Capturing configName flag
            // does not matter as that is handled by which mixpanel token is used
            // TODO: capture options and flags
            this.mixpanelService.TrackCliCall('CliCommand', { args: argv._ } );
        })
        .middleware(() => {
            // Greedy fetch of some data that we use frequently 
            const ssmTargetService = new SsmTargetService(this.configService);
            const sshTargetService = new SshTargetService(this.configService);
            const envService = new EnvironmentService(this.configService);

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
                const sessionService = new SessionService(this.configService);
                const listSessions = await sessionService.ListSessions();

                // space names are not unique, make sure to find the latest active one
                var cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space' && s.state == SessionState.Active); // TODO: cli-space name can be changed in config

                // maybe make a session
                var cliSessionId: string;
                if(cliSpace.length === 0)
                {
                    cliSessionId =  await sessionService.CreateSession('cli-space');
                } else {
                    // there should only be 1 active 'cli-space' session
                    cliSessionId = cliSpace.pop().id;
                }

                // We do the following for ssh since we are required to pass
                // in a user although it does not get read at any point
                // TODO: fix how enums are parsed and compared
                const targetUser = parsedTarget.type == TargetType.SSH ? 'totally-a-user' : parsedTarget.user;

                // make a new connection
                const connectionService = new ConnectionService(this.configService);
                const connectionId = await connectionService.CreateConnection(parsedTarget.type, parsedTarget.id, cliSessionId, targetUser);

                this.mixpanelService.TrackNewConnection(parsedTarget.type);

                // run terminal
                const queryString = `?connectionId=${connectionId}`;
                const connectionUrl = `${this.configService.serviceUrl()}api/v1/hub/ssh/${queryString}`;

                var terminal = new ShellTerminal(this.configService, connectionUrl);
                terminal.start(termsize());

                // Terminal resize event logic
                // https://nodejs.org/api/process.html#process_signal_events -> SIGWINCH
                // https://github.com/nodejs/node/issues/16194
                // https://nodejs.org/api/process.html#process_a_note_on_process_i_o
                process.stdout.on('resize', () =>
                {
                    const resizeEvent = termsize();
                    terminal.resize(resizeEvent);
                });

                // To get 'keypress' events you need the following lines
                // ref: https://nodejs.org/api/readline.html#readline_readline_emitkeypressevents_stream_interface
                const readline = require('readline');
                readline.emitKeypressEvents(process.stdin);
                process.stdin.setRawMode(true);
                process.stdin.on('keypress', async (str, key) => {
                    if (key.ctrl && key.name === 'p') {
                        // close the session
                        await connectionService.CloseConnection(connectionId).catch();
                        terminal.dispose();
                        process.exit(0);
                    } else {
                        terminal.writeString(key.sequence);
                    }
                });
                thoumMessage('CTRL+P to escape terminal');
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
                const fileService = new FileService(this.configService);

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
                        thoumWarn(`File ${argv.source} does not exist or cannot be read`);
                        process.exit(1);
                    }

                    await fileService.uploadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, fh, parsedTarget.user);
                    thoumMessage('File upload complete');

                } else if(sourceParsedString) {
                    // download case
                    await fileService.downloadFile(parsedTarget.id, parsedTarget.type, parsedTarget.path, argv.destination, parsedTarget.user);
                } else {
                    thoumError('Invalid target string, must follow syntax:');
                    thoumError(targetStringExample);
                }
                process.exit(0);
            }
        )
        .command(
            'config',
            'Returns config file path',
            () => {},
            () => {
                thoumMessage(`You can edit your config here: ${this.configService.configPath()}`);
                process.exit(0);
            }
        ).command(
            'logout',
            'Deauthenticate the client',
            () => {},
            async () => {
                var ouath = new OAuthService(this.configService.authUrl(), this.configService.callbackListenerPort());
                await ouath.logout(this.configService.tokenSet());
                this.configService.logout();
                process.exit(0);
            }
        )
        .option('configName', {type: 'string', choices: ['prod', 'stage', 'dev'], default: 'prod', hidden: true})
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
            thoumError('Invalid target string, must follow syntax:');
            thoumError(targetStringExampleNoPath);
            process.exit(1);
        }

        if(! checkTargetTypeAndStringPair(parsedTarget))
            process.exit(1);

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
                    thoumError(`Invalid TargetType passed ${parsedTarget.type}`);
                    process.exit(1);
            }

            if(matchedNamedTargets.length < 1)
            {
                thoumError(`No ${parsedTarget.type} targets found with name ${parsedTarget.name}`);
                thoumWarn('Target names are case sensitive');
                thoumWarn('To see list of all targets run: \'thoum lt\'');
                process.exit(1);
            } else if(matchedNamedTargets.length == 1)
            {
                // the rest of the flow will work as before since the targetId has now been disambiguated
                parsedTarget.id = matchedNamedTargets.pop().id;
            } else {
                // ambiguous target id, warn user, exit process
                thoumWarn(`Multiple ${parsedTarget.type} targets found with name ${parsedTarget.name}`);
                const tableString = getTableOfTargets(matchedNamedTargets, await this.envs);
                console.log(tableString);
                thoumMessage('Please connect using \'target id\' instead of target name');
                process.exit(1);
            }
        }

        return parsedTarget;
    }
}