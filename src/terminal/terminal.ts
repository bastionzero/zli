import { BehaviorSubject, Observable, Subject, Subscription } from 'rxjs';
import { SshShellWebsocketService } from '../../webshell-common-ts/shell-websocket.service/ssh-shell-websocket.service';
import { isAgentKeysplittingReady, SsmShellWebsocketService } from '../../webshell-common-ts/shell-websocket.service/ssm-shell-websocket.service';
import { IDisposable } from '../../webshell-common-ts/utility/disposable';
import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';

import { ConfigService } from '../config.service/config.service';
import { IShellWebsocketService, ShellEvent, ShellEventType, TerminalSize } from '../../webshell-common-ts/shell-websocket.service/shell-websocket.service.types';
import { ZliAuthConfigService } from '../config.service/zli-auth-config.service';
import { Logger } from '../logger.service/logger';
import { ConnectionService, SsmTargetService } from '../http.service/http.service';
import { TargetType } from '../types';
import { SsmTargetSummary } from '../http.service/http.service.types';

export class ShellTerminal implements IDisposable
{
    private shellWebsocketService : IShellWebsocketService;
    private shellEventDataSubscription: Subscription;
    private currentTerminalSize: TerminalSize;

    // stdin
    private inputSubject: Subject<string> = new Subject<string>();
    private resizeSubject: Subject<TerminalSize> = new Subject<TerminalSize>();
    private blockInput: boolean = true;
    private terminalRunningStream: BehaviorSubject<boolean> = new BehaviorSubject<boolean>(true);
    public terminalRunning: Observable<boolean> = this.terminalRunningStream.asObservable();

    constructor(private logger: Logger, private configService: ConfigService, private connectionService: ConnectionService, private connectionId: string)
    {
    }

    private async createShellWebsocketService() : Promise<IShellWebsocketService> {
        const connectionInfo = await this.connectionService.GetConnection(this.connectionId);
        const targetType = connectionInfo.serverType;
        const targetId = connectionInfo.serverId;

        if(targetType === TargetType.SSH) {
            return this.createSshShellWebsocketService();
        } else if(targetType === TargetType.SSM || targetType === TargetType.DYNAMIC) {
            const ssmTargetService = new SsmTargetService(this.configService, this.logger);
            const ssmTargetInfo = await ssmTargetService.GetSsmTarget(targetId);
            if( isAgentKeysplittingReady(ssmTargetInfo.agentVersion)) {
                return this.createSsmShellWebsocketService(ssmTargetInfo);
            } else {
                this.logger.warn(`Agent version ${ssmTargetInfo.agentVersion} not compatible with keysplitting...falling back to non-keysplitting shell`);
                return this.createSshShellWebsocketService();
            }
        } else {
            throw new Error(`Unhandled target type ${targetType}`);
        }
    }

    private createSshShellWebsocketService(): IShellWebsocketService {
        return new SshShellWebsocketService(
            this.logger,
            new ZliAuthConfigService(this.configService, this.logger),
            this.connectionId,
            this.inputSubject,
            this.resizeSubject
        );
    }

    private createSsmShellWebsocketService(ssmTargetInfo: SsmTargetSummary): IShellWebsocketService {
        return new SsmShellWebsocketService(
            new KeySplittingService(this.configService, this.logger),
            ssmTargetInfo,
            this.logger,
            new ZliAuthConfigService(this.configService, this.logger),
            this.connectionId,
            this.inputSubject,
            this.resizeSubject
        );
    }

    public async start(termSize: TerminalSize): Promise<void>
    {
        this.currentTerminalSize = termSize;
        this.shellWebsocketService = await this.createShellWebsocketService();

        // Handle writing to stdout
        // TODO: bring this up a level
        this.shellWebsocketService.outputData.subscribe(data => {
            process.stdout.write(Buffer.from(data, 'base64'));
        });

        // initial terminal size
        await this.shellWebsocketService.start();

        this.shellEventDataSubscription = this.shellWebsocketService.shellEventData.subscribe(
            async (shellEvent: ShellEvent) => {
                this.logger.debug(`Got new shell event: ${shellEvent.type}`);

                switch(shellEvent.type) {
                case ShellEventType.Ready:
                    this.shellWebsocketService.sendShellConnect(this.currentTerminalSize.rows, this.currentTerminalSize.columns);
                    break;
                case ShellEventType.Start:
                    this.blockInput = false;
                    this.terminalRunningStream.next(true);
                    // Send initial terminal dimensions
                    this.resize(this.currentTerminalSize);
                    break;
                case ShellEventType.Unattached:
                    // When another client connects (web app) handle this by
                    // exiting this ZLI process without closing the
                    // connection and effectively transferring ownership of
                    // the connection to the web app. We do not support
                    // re-attaching within the same ZLI command.
                    this.logger.error('Web App session has been detected.');
                    this.terminalRunningStream.complete();
                    break;
                case ShellEventType.Disconnect:
                    this.terminalRunningStream.error('Target Disconnected.');
                    break;
                case ShellEventType.Delete:
                    this.terminalRunningStream.error('Connection was closed.');
                    break;
                default:
                    this.logger.warn(`Unhandled shell event type ${shellEvent.type}`);
                }
            },
            (error: any) => {
                this.terminalRunningStream.error(error);
            },
            () => {
                this.terminalRunningStream.error('ShellEventData subscription completed prematurely');
            }
        );
    }

    public resize(terminalSize: TerminalSize): void
    {
        this.logger.trace(`New terminal resize event (rows: ${terminalSize.rows} cols: ${terminalSize.columns})`);

        // Save the new terminal dimensions even if the shell input is blocked
        // so that when we start the shell we initialize the terminal dimensions
        // correctly
        this.currentTerminalSize = terminalSize;

        if(! this.blockInput)
            this.resizeSubject.next({rows: terminalSize.rows, columns: terminalSize.columns});
    }

    public writeString(input: string) : void {
        if(! this.blockInput) {
            this.inputSubject.next(input);
        }
    }

    public writeBytes(input: Uint8Array) : void {
        this.writeString(new TextDecoder('utf-8').decode(input));
    }

    public dispose() : void
    {
        // First unsubscribe to shell event subscription because this wil be
        // completed when disposing the shellWebsocketService
        if(this.shellEventDataSubscription)
            this.shellEventDataSubscription.unsubscribe();

        if(this.shellWebsocketService)
            this.shellWebsocketService.dispose();

        this.terminalRunningStream.complete();
    }
}