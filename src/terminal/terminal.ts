import { BehaviorSubject, Observable, Subject } from 'rxjs';
import { SshShellWebsocketService } from '../../webshell-common-ts/shell-websocket.service/ssh-shell-websocket.service';
import { isAgentKeysplittingReady, SsmShellWebsocketService } from '../../webshell-common-ts/shell-websocket.service/ssm-shell-websocket.service';
import { IDisposable } from '../../webshell-common-ts/utility/disposable';
import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';

import { ConfigService } from '../config.service/config.service';
import { IShellWebsocketService, ShellEvent, ShellEventType, TerminalSize } from '../../webshell-common-ts/shell-websocket.service/shell-websocket.service.types';
import { ZliAuthConfigService } from '../config.service/zli-auth-config.service';
import { Logger } from '../logger.service/logger';
import { SsmTargetService } from '../http.service/http.service';
import { TargetType } from '../types';
import { ParsedTargetString } from '../types';
import { SsmTargetSummary } from '../http.service/http.service.types';

export class ShellTerminal implements IDisposable
{
    private shellWebsocketService : IShellWebsocketService;
    private attached: boolean = false;

    // stdin
    private inputSubject: Subject<string> = new Subject<string>();
    private resizeSubject: Subject<TerminalSize> = new Subject<TerminalSize>();
    private blockInput: boolean = true;
    private terminalRunningStream: BehaviorSubject<boolean> = new BehaviorSubject<boolean>(true);
    public terminalRunning: Observable<boolean> = this.terminalRunningStream.asObservable();

    constructor(private logger: Logger, private configService: ConfigService, private connectionId: string, private parsedTarget: ParsedTargetString)
    {
    }

    private async createShellWebsocketService() : Promise<IShellWebsocketService> {
        const targetType = this.parsedTarget.type;

        if(targetType === TargetType.SSH) {
            return this.createSshShellWebsocketService();
        } else if(targetType === TargetType.SSM || targetType === TargetType.DYNAMIC) {
            const ssmTargetService = new SsmTargetService(this.configService, this.logger);
            const ssmTargetInfo = await ssmTargetService.GetSsmTarget(this.parsedTarget.id);
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

    public async start(termSize: TerminalSize)
    {
        this.shellWebsocketService = await this.createShellWebsocketService();

        // Handle writing to stdout
        // TODO: bring this up a level
        this.shellWebsocketService.outputData.subscribe(data => {
            process.stdout.write(Buffer.from(data, 'base64'));
        });

        // initial terminal size
        await this.shellWebsocketService.start();

        this.shellWebsocketService.shellEventData.subscribe(
            async (shellEvent: ShellEvent) => {
                this.logger.trace(`Got new shell event update: ${JSON.stringify(shellEvent)}`);

                switch(shellEvent.type) {
                case ShellEventType.Ready:
                    this.shellWebsocketService.sendShellConnect(termSize.rows, termSize.columns);
                    break;
                case ShellEventType.Start:
                    this.blockInput = false;
                    this.attached = true;
                    this.terminalRunningStream.next(true);
                    break;
                case ShellEventType.Unattached:
                    this.logger.warn('Another active session has been detected and input has been suspended. Press ALT+R at any time to resume this terminal');
                    this.blockInput = true;
                    this.attached = false;
                    break;
                case ShellEventType.Disconnect:
                case ShellEventType.Delete:
                    this.dispose();
                    break;
                default:
                    this.logger.warn(`unhandled shell event type ${shellEvent.type}`);
                }
            },
            (error: any) => {
                this.terminalRunningStream.error(error);
            },
            () => {
                this.terminalRunningStream.error(undefined);
            }
        );
    }

    public resize(resizeEvent: TerminalSize)
    {
        this.logger.trace(`New terminal resize event (rows: ${resizeEvent.rows} cols: ${resizeEvent.columns})`);

        if(! this.blockInput)
            this.resizeSubject.next({rows: resizeEvent.rows, columns: resizeEvent.columns});
    }

    public async writeString(input: string) : Promise<void> {
        if(! this.attached && input === '\u001br') {
            await this.shellWebsocketService.shellReattach();
            this.attached = true;
            this.blockInput = false;
            return;
        }

        if(! this.blockInput) {
            this.inputSubject.next(input);
        }
    }

    public writeBytes(input: Uint8Array) : void {
        this.writeString(new TextDecoder('utf-8').decode(input));
    }

    public dispose() : void
    {
        if(this.shellWebsocketService)
            this.shellWebsocketService.dispose();

        this.terminalRunningStream.complete();
    }
}