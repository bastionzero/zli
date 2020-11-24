import { BehaviorSubject } from 'rxjs';
import { IDisposable, WebsocketStream } from '../websocket.service/websocket.service';
import { ConfigService } from '../config.service/config.service';
import chalk from 'chalk';

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
        this.websocketStream.shellStateData.subscribe(newState => {
            if(! newState.disconnected && ! newState.loading)
                this.blockInput = false;
            else
            {
                this.blockInput = true;

                if(! newState.loading) 
                    console.log(chalk.red('\nthoum >>> Disconnection detected'));
            }
        }); 
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
        this.websocketStream.dispose();
    }
    
}