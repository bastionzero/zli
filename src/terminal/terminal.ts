import { BehaviorSubject } from 'rxjs';
import { IDisposable, WebsocketStream } from '../websocket.service/websocket.service';
import termsize from 'term-size';
import { ConfigService } from '../config.service/config.service';

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

    constructor(configService: ConfigService, connectionUrl: string)
    {
        this.websocketStream = new WebsocketStream(configService, connectionUrl, this.inputSubject, this.resizeSubject);
    }

    public start()
    {        
        // https://nodejs.org/api/process.html#process_signal_events -> SIGWINCH
        // https://github.com/nodejs/node/issues/16194
        // https://nodejs.org/api/process.html#process_a_note_on_process_i_o
        // TODO bring this up a level?
        process.stdout.on('resize', () => 
        {
            const resizeEvent = termsize();
            this.resizeSubject.next({rows: resizeEvent.rows, columns: resizeEvent.columns});
        });

        this.websocketStream.outputData.subscribe(data => 
        {
            process.stdout.write(data);
        });
        
        const terminalSize = termsize();
        this.websocketStream.start(terminalSize.rows, terminalSize.columns);
        // TOTO: abstract the process.std to a NodeJS.WriteStream 
        // member and pass in from constructor
        
        // TODO, pause input on disconnected states
        // this.websocketStream.shellStateData.subscribe() 

    }

    public writeString(input: string) : void {
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