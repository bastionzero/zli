import { BehaviorSubject, Observable, Subject, Subscription } from 'rxjs';
import { isAgentKeysplittingReady, ShellWebsocketService } from '../../webshell-common-ts/shell-websocket.service/shell-websocket.service';
import { IDisposable } from '../../webshell-common-ts/utility/disposable';
import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';

import { ConfigService } from '../config.service/config.service';
import { IShellWebsocketService, ShellEvent, ShellEventType, TerminalSize } from '../../webshell-common-ts/shell-websocket.service/shell-websocket.service.types';
import { ZliAuthConfigService } from '../config.service/zli-auth-config.service';
import { Logger } from '../logger.service/logger';
import { ConnectionService, SsmTargetService } from '../http.service/http.service';
import { TargetType } from '../types';
import { ConnectionSummary, SsmTargetSummary } from '../http.service/http.service.types';

export class ShellTerminal implements IDisposable
{
    private shellWebsocketService : IShellWebsocketService;
    private shellEventDataSubscription: Subscription;
    private currentTerminalSize: TerminalSize;

    private refreshTargetInfoOnReady: boolean = false;

    // stdin
    private inputSubject: Subject<string> = new Subject<string>();
    private resizeSubject: Subject<TerminalSize> = new Subject<TerminalSize>();
    private blockInput: boolean = true;
    private terminalRunningStream: BehaviorSubject<boolean> = new BehaviorSubject<boolean>(true);
    public terminalRunning: Observable<boolean> = this.terminalRunningStream.asObservable();

    constructor(private logger: Logger, private configService: ConfigService, private connectionSummary: ConnectionSummary)
    {
    }

    private async createShellWebsocketService() : Promise<IShellWebsocketService> {
        const targetType = this.connectionSummary.serverType;
        const targetId = this.connectionSummary.serverId;

        if (targetType === TargetType.SSM || targetType === TargetType.DYNAMIC) {
            const ssmTargetService = new SsmTargetService(this.configService, this.logger);
            const ssmTargetInfo = await ssmTargetService.GetSsmTarget(targetId);

            // Check the agent version is keysplitting compatible
            this.checkAgentVersion(ssmTargetInfo);

            const connectionService = new ConnectionService(this.configService, this.logger);
            const shellConnectionAuthDetails = await connectionService.GetShellConnectionAuthDetails(this.connectionSummary.id);

            return new ShellWebsocketService(
                new KeySplittingService(this.configService, this.logger),
                ssmTargetInfo,
                this.logger,
                new ZliAuthConfigService(this.configService, this.logger),
                this.connectionSummary.id,
                shellConnectionAuthDetails.connectionNodeId,
                shellConnectionAuthDetails.authToken,
                this.inputSubject,
                this.resizeSubject
            );
        } else {
            throw new Error(`Unhandled connection type ${targetType}`);
        }
    }

    private checkAgentVersion(ssmTargetInfo: SsmTargetSummary) {
        // Bastion may not know agent version right away so skip this check if
        // we havent determined the agent version yet
        if (ssmTargetInfo.agentVersion === '') {
            this.refreshTargetInfoOnReady = true;
            return;
        }

        if (!isAgentKeysplittingReady(ssmTargetInfo.agentVersion)) {
            throw new Error(`Agent version ${ssmTargetInfo.agentVersion} not compatible with keysplitting...`);
        }
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

        // Replay the existing output if any and only then continue with the shell start flow
        this.shellWebsocketService.replayData.subscribe(data => {// Maybe a "wait for only one input" should be used here instead of subscribe?
            process.stdout.write(Buffer.from(data, 'base64'));
            this.shellWebsocketService.sendReplayDone(termSize.rows, termSize.columns);
        });

        // initial terminal size
        await this.shellWebsocketService.start();

        this.shellEventDataSubscription = this.shellWebsocketService.shellEventData.subscribe(
            async (shellEvent: ShellEvent) => {
                this.logger.debug(`Got new shell event: ${shellEvent.type}`);

                switch(shellEvent.type) {
                case ShellEventType.Ready:
                    if (this.refreshTargetInfoOnReady) {
                        const ssmTargetService = new SsmTargetService(this.configService, this.logger);
                        const ssmTargetInfo = await ssmTargetService.GetSsmTarget(this.connectionSummary.serverId);
                        this.shellWebsocketService.updateTargetInfo(ssmTargetInfo);
                    }
                    const replayOutput: boolean = true; // Maybe there is a better place for this endpoint versioning?
                    this.shellWebsocketService.sendShellConnect(this.currentTerminalSize.rows, this.currentTerminalSize.columns, replayOutput);
                    break;
                case ShellEventType.Start:
                    this.blockInput = false;
                    this.terminalRunningStream.next(true);
                    // Trigger resize to force the terminal to refresh the output
                    const tempTerminalSize : TerminalSize = {rows: this.currentTerminalSize.rows + 1, columns: this.currentTerminalSize.columns + 1};
                    this.resizeSubject.next({rows: tempTerminalSize.rows, columns: tempTerminalSize.columns});
                    // Send initial terminal dimensions
                    this.resize(this.currentTerminalSize);
                    break;
                case ShellEventType.Unattached:
                    // When another client connects handle this by
                    // exiting this ZLI process without closing the
                    // connection and effectively transferring ownership of
                    // the connection to the other client
                    this.logger.error('Another client has attached to this connection.');
                    this.terminalRunningStream.complete();
                    break;
                case ShellEventType.Disconnect:
                    this.terminalRunningStream.error('Target Disconnected.');
                    break;
                case ShellEventType.Delete:
                    this.terminalRunningStream.error('Connection was closed.');
                    break;
                case ShellEventType.BrokenWebsocket:
                    this.blockInput = true;
                    this.logger.warn('BastionZero: 503 service unavailable. Reconnecting...');
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
        } else {
            // char code 3 is SIGINT
            if( input.charCodeAt(0) === 3 )
                this.terminalRunningStream.error('Terminal killed');
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