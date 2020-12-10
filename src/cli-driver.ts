import { SessionState, TargetType } from "./types";
import { checkTargetTypeAndStringPair, findSubstring, parseTargetString, parseTargetType, targetStringExample, targetStringExampleNoPath, thoumError, thoumMessage, thoumWarn } from './utils';
import yargs from "yargs";
import { ConfigService } from "./config.service/config.service";
import { ConnectionService, EnvironmentService, FileService, SessionService, SshTargetService, SsmTargetService } from "./http.service/http.service";
import { OAuthService } from "./oauth.service/oauth.service";
import { ShellTerminal } from "./terminal/terminal";
import Table from 'cli-table3';
import termsize from 'term-size';
import { UserinfoResponse } from "openid-client";
import { MixpanelService } from "./mixpanel.service/mixpanel.service";
import { checkVersionMiddleware } from "./middlewares/check-version-middleware";
import { oauthMiddleware } from "./middlewares/oauth-middleware";
import fs from 'fs'

export class CliDriver
{
    private configService: ConfigService;
    private userInfo: UserinfoResponse; // sub and email

    private mixpanelService: MixpanelService;

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
                .example('connect ssm ssm-user@95b72b50-d09c-49fa-8825-332abfeb013e', 'SSM connect example')
                .example('connect ssh dbda775d-e37c-402b-aa76-bbb0799fd775', 'SSH connect example');
            },
            async (argv) => {

                // ssm and ssh range checking preformed by yargs via choices
                const targetType = parseTargetType(argv.targetType);

                // Extra colon added to account for no path being passed into this command
                const parsedTarget = parseTargetString(argv.targetString);

                if(! parsedTarget)
                {
                    thoumError('Invalid target string, must follow syntax:');
                    thoumError(targetStringExampleNoPath);
                    process.exit(1);
                }

                if(! checkTargetTypeAndStringPair(targetType, parsedTarget))
                    process.exit(1);

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
                const targetUser = targetType == TargetType.SSH ? 'totally-a-user' : parsedTarget.targetUser;

                // make a new connection
                const connectionService = new ConnectionService(this.configService);
                const connectionId = await connectionService.CreateConnection(targetType, parsedTarget.targetId, cliSessionId, targetUser);

                this.mixpanelService.TrackNewConnection(targetType);

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
                const ssmTargetService = new SsmTargetService(this.configService);
                let ssmList = await ssmTargetService.ListSsmTargets();

                const sshTargetService = new SshTargetService(this.configService);
                let sshList = await sshTargetService.ListSsmTargets();

                const envService = new EnvironmentService(this.configService);
                const envs = await envService.ListEnvironments();

                // ref: https://github.com/cli-table/cli-table3
                var table = new Table({
                    head: ['Type', 'Name', 'Environment', 'Id']
                , colWidths: [6, 16, 16, 38]
                });


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
                    sshList = sshList.filter(ssh => findSubstring(argv.name, ssh.alias));
                }

                // push targets to printed table, maybe filter by targetType
                if(argv.targetType === 'ssm')
                {
                    ssmList.forEach(ssm => table.push(['ssm', ssm.name, envs.filter(e => e.id == ssm.environmentId).pop().name, ssm.id]));
                } else if(argv.targetType === 'ssh') {
                    sshList.forEach(ssh => table.push(['ssh', ssh.alias, envs.filter(e => e.id == ssh.environmentId).pop().name, ssh.id]));
                } else {
                    ssmList.forEach(ssm => table.push(['ssm', ssm.name, envs.filter(e => e.id == ssm.environmentId).pop().name, ssm.id]));
                    sshList.forEach(ssh => table.push(['ssh', ssh.alias, envs.filter(e => e.id == ssh.environmentId).pop().name, ssh.id]));
                }

                const tableString = table.toString(); // hangs if you try to print directly to console
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
                .example('copy ssm /Users/coolUser/secretFile.txt ssm-user@95b72b50-d09c-49fa-8825-332abfeb013e:/home/ssm-user/newFileName', 'SSM Upload example')
                .example('copy ssh /Users/coolUser/neatFile.txt 05c6bdea-8623-4b49-83ad-f7f301fea7e1:/home/ssm-user/newFileName.txt', 'SSH Upload example');
            },
            async (argv) => {
                const fileService = new FileService(this.configService);

                const targetType = parseTargetType(argv.targetType);
                const sourceParsedString = parseTargetString(argv.source);
                const destParsedString = parseTargetString(argv.destination);
                const parsedTarget = sourceParsedString || destParsedString; // one of these will be undefined so javascript will use the other

                if(! checkTargetTypeAndStringPair(targetType, parsedTarget))
                    process.exit(1);

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

                    await fileService.uploadFile(parsedTarget.targetId, targetType, parsedTarget.targetPath, fh, parsedTarget.targetUser);
                    thoumMessage('File upload complete');

                } else if(sourceParsedString) {
                    // download case
                    await fileService.downloadFile(parsedTarget.targetId, targetType, parsedTarget.targetPath, argv.destination, parsedTarget.targetUser);
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
}