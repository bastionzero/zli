import { BehaviorSubject, Observable } from 'rxjs';
import { IDisposable, WebsocketStream } from '../websocket.service/websocket.service';
import { ConfigService } from '../config.service/config.service';
import { ShellState } from '../websocket.service/websocket.service.types';

export interface TerminalSize
{
    rows: number;
    columns: number;
}

export class ShellTerminal implements IDisposable
{
    private websocketStream : WebsocketStream;
    // stdin
    private inputSubject: BehaviorSubject<string> = new BehaviorSubject<string>(null);
    private resizeSubject: BehaviorSubject<TerminalSize> = new BehaviorSubject<TerminalSize>({rows: 0, columns: 0});
    private blockInput: boolean = true;
    private terminalRunningStream: BehaviorSubject<boolean> = new BehaviorSubject<boolean>(true);
    public terminalRunning: Observable<boolean> = this.terminalRunningStream.asObservable();

    constructor(configService: ConfigService, connectionUrl: string)
    {
        this.websocketStream = new WebsocketStream(configService, connectionUrl, this.inputSubject, this.resizeSubject);
    }

    public start(termSize: TerminalSize)
    {
        // Handle writing to stdout
        // TODO: bring this up a level
        this.websocketStream.outputData.subscribe(data => process.stdout.write(data));

        // initial terminal size
        this.websocketStream.start(termSize.rows, termSize.columns);
        
        // Pauses input on disconnected states
        this.websocketStream.shellStateData.subscribe(
            (newState: ShellState) => {
                if(! newState.disconnected && ! newState.loading)
                {
                    this.blockInput = false;
                    this.terminalRunningStream.next(true);
                }
                else
                {
                    this.blockInput = true;
                    
                    // TODO: offer reconnect flow here
                    if(! newState.loading)
                    {
                        this.terminalRunningStream.error('Disconnection detected');
                    }
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
        if(! this.blockInput)
            this.resizeSubject.next({rows: resizeEvent.rows, columns: resizeEvent.columns});
    }

    public writeString(input: string) : void {
        if(! this.blockInput)
            this.inputSubject.next(input);
    }

    public writeBytes(input: Uint8Array) : void {
        this.writeString(new TextDecoder("utf-8").decode(input));
    }

    public dispose() : void
    {
        if(this.websocketStream)
            this.websocketStream.dispose();
        
        this.terminalRunningStream.complete();
    }
}