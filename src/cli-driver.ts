import { TargetType } from "./types";
import { argv } from "process";
import yargs from "yargs";
import { ConfigService } from "./config.service/config.service";
import { ConnectionService, SessionService } from "./http.service/http.service";
import { OAuthService } from "./oauth.service";
import { ShellTerminal } from "./terminal/terminal";

export class CliDriver
{
    private configService: ConfigService;

    public start()
    {
        yargs(process.argv.slice(2)) // returns array of argv
        .scriptName("thoum")
        .usage('$0 <cmd> [args]')
        .middleware(async (argv) => {
            this.configService = new ConfigService(<string> argv.configName);
            var ouath = new OAuthService(this.configService.authUrl());

            // All times related to oauth are in epoch second
            const now: number = new Date().getSeconds();
            
            if(this.configService.tokenSet() && this.configService.tokenSet().expires_at < now && this.configService.tokenSetExpireTime() > now)
            {
                // refresh using existing creds
                var newTokenSet = await ouath.refresh(this.configService.tokenSet());
                this.configService.setTokenSet(newTokenSet);
            } else {
                // renew with log in flow
                await ouath.login((tokenSet, expireTime) => this.configService.setTokenSet(tokenSet, expireTime));
            }

            await ouath.oauthFinished;
        })
        .command('connect [targetType] [targetId]', 'Connect to a target', (yargs) => {
            yargs.positional('targetType', {
                type: 'string',
                describe: 'ssm or ssh',
                choices: ['ssm', 'ssh'],
                demandOption: 'Target Type must be selected { ssm | ssh }'
            }).positional('targetId', {
                type: 'string',
                describe: 'GUID of target',
                demandOption: 'Target Id must be provided (GUID)'
            })
        }, async (argv) => {
            // call list session
            const sessionService = new SessionService(this.configService);
            const listSessions = await sessionService.ListSessions();

            var cliSpace = listSessions.sessions.filter(s => s.displayName === 'cli-space'); // TODO: cli-space name can be changed in config

            // maybe make a session
            var cliSessionId: string;
            if(cliSpace.length === 0)
            {
                const resp =  await sessionService.CreateSession('cli-space');
                cliSessionId = resp;
            } else {
                // there should only be 1
                cliSessionId = cliSpace.pop().id;
            }

            // make a new connection
            const connectionService = new ConnectionService(this.configService);
            const connectionId = await connectionService.CreateConnection(<TargetType> argv.targetType, <string> argv.targetId, cliSessionId);

            // run terminal
            const queryString = `?connectionId=${connectionId}`;
            const connectionUrl = `${this.configService.serviceUrl()}api/v1/hub/ssh/${queryString}`;

            var terminal = new ShellTerminal(this.configService, connectionUrl);
            terminal.start();

            // To get 'keypress' events you need the following lines
            // ref: https://nodejs.org/api/readline.html#readline_readline_emitkeypressevents_stream_interface
            const readline = require('readline');
            readline.emitKeypressEvents(process.stdin);
            process.stdin.setRawMode(true);
            process.stdin.on('keypress', (str, key) => {
                if (key.ctrl && key.name === 'q') {
                    // close the session
                    connectionService.CloseConnection(connectionId).catch();
                    terminal.dispose();
                    process.exit();
                } else {
                    terminal.writeString(str);
                }
            });
            console.log('CTRL+Q to exit thoum');
        })

        // .command('config [configName]', 'Set up your config file', (yargs) => {
        //     yargs.positional('configName', {
        //         type: 'string',
        //         describe: 'The name of the config you are editing (default: prod)',
        //         demandOption: 'true',
        //         default: 'prod',
        //         choices: ['prod', 'staging', 'dev'],
        //     }) 
        // }, (argv) => {
        //     console.log('edit config flow', argv);
        // })
        .option('configName', {type: 'string', choices: ['prod', 'stage', 'dev'], default: 'prod'})
        .help()
        .argv;
    }
}