import { SessionState, TargetType } from "./types";
import yargs from "yargs";
import { ConfigService } from "./config.service/config.service";
import { ConnectionService, EnvironmentsService, SessionService, SshTargetService, SsmTargetService } from "./http.service/http.service";
import { OAuthService } from "./oauth.service";
import { ShellTerminal } from "./terminal/terminal";
import chalk from "chalk";
import Table from 'cli-table3';
import termsize from 'term-size';
import { read } from "fs";


export class CliDriver
{
    private configService: ConfigService;

    private thoumMessage(message: string): void
    {
        console.log(chalk.magenta(`thoum >>> ${message}`));
    }

    public start()
    {
        yargs(process.argv.slice(2)) // returns array of argv
        .scriptName("thoum")
        .usage('$0 <cmd> [args]')
        .middleware(async (argv) => {
            this.configService = new ConfigService(<string> argv.configName);
            var ouath = new OAuthService(this.configService.authUrl(), this.configService.callbackListenerPort());

            // All times related to oauth are in epoch second
            const now: number = Date.now() / 1000;
            
            if(this.configService.tokenSet() && this.configService.tokenSet().expires_at < now && this.configService.tokenSetExpireTime() > now)
            {
                this.thoumMessage('Refreshing oauth');
                // refresh using existing creds
                var newTokenSet = await ouath.refresh(this.configService.tokenSet());
                this.configService.setTokenSet(newTokenSet);
            } else if(this.configService.tokenSetExpireTime() < now) {
                this.thoumMessage('Log in required, opening browser');
                // renew with log in flow
                ouath.login((tokenSet, expireTime) => this.configService.setTokenSet(tokenSet, expireTime));
                await ouath.oauthFinished;
            }
        })
        .command(
            'connect [targetType] [targetId] [targetUser]', 
            'Connect to a target', 
            (yargs) => {
                yargs.positional('targetType', {
                    type: 'string',
                    describe: 'ssm or ssh',
                    choices: ['ssm', 'ssh'],
                    demandOption: 'Target Type must be selected { ssm | ssh }'
                }).positional('targetId', {
                    type: 'string',
                    describe: 'GUID of target',
                    demandOption: 'Target Id must be provided (GUID)'
                }).positional('targetUser', {
                    type: 'string',
                    describe: 'The username on the target to connect as, required for SSM',
                    demandOption: false 
                })
            },
            async (argv) => {
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

                const targetType = <TargetType> argv.targetType;
                const targetId = <string> argv.targetId;
                // We do the following for ssh since we are required to pass
                // in a user although it does not get read at any point
                // TODO: fix how enums are parsed and compared
                const targetUser = argv.targetType === "ssh" ? "totally-a-user" : <string> argv.targetUser;

                // make a new connection
                const connectionService = new ConnectionService(this.configService);
                const connectionId = await connectionService.CreateConnection(targetType, targetId, cliSessionId, targetUser);

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
                    if (key.ctrl && key.name === 'q') {
                        // close the session
                        await connectionService.CloseConnection(connectionId).catch();
                        terminal.dispose();
                        process.exit(0);
                    } else {
                        terminal.writeString(key.sequence);
                    }
                });
                this.thoumMessage('CTRL+Q to exit thoum');
            }
        )
        .command(
            'list-targets', 
            'List all SSM and SSH targets', 
            () => {},
            async () => {
                const ssmTargetService = new SsmTargetService(this.configService);
                const ssmList = await ssmTargetService.ListSsmTargets();

                const sshTargetService = new SshTargetService(this.configService);
                const sshList = await sshTargetService.ListSsmTargets();

                const envService = new EnvironmentsService(this.configService);
                const envs = await envService.ListEnvironments();
                
                // ref: https://github.com/cli-table/cli-table3
                var table = new Table({
                    head: ['Type', 'Name', 'Environment', 'Id']
                , colWidths: [6, 16, 16, 38]
                });

                ssmList.forEach(ssm => table.push(['ssm', ssm.name, envs.filter(e => e.id == ssm.environmentId).pop().name, ssm.id]));
                sshList.forEach(ssh => table.push(['ssh', ssh.alias, envs.filter(e => e.id == ssh.environmentId).pop().name, ssh.id]));
                
                const tableString = table.toString(); // hangs if you try to print directly to console
                console.log(tableString);
            }
        )
        .command(
            'config', 
            'Returns config file path', 
            () => {}, 
            () => {
                this.thoumMessage(`You can edit your config here: ${this.configService.configPath()}`);
            }
        )
        .option('configName', {type: 'string', choices: ['prod', 'stage', 'dev'], default: 'prod', hidden: true})
        .help()
        .argv;
    }
}