import { HubConnection, HubConnectionBuilder } from '@microsoft/signalr';
import { BehaviorSubject, Observable, Subscription } from 'rxjs';
import { ConfigService } from '../config.service/config.service';
import { TerminalSize } from '../terminal/terminal';
import { ShellHubIncomingMessages, ShellHubOutgoingMessages, ShellState } from './websocket.service.types';


// ref: https://gist.github.com/dsherret/cf5d6bec3d0f791cef00
export interface IDisposable
{
    dispose() : void;
}

// Reflects the IShell interface
export class WebsocketStream implements IDisposable
{
    private connectionUrl : string;
    private websocket : HubConnection;

    // stdout
    private outputSubject: BehaviorSubject<Uint8Array> = new BehaviorSubject<Uint8Array>(new Uint8Array());
    public outputData: Observable<Uint8Array> = this.outputSubject.asObservable();
    
    // stdin
    private inputSubscription: Subscription;
    private resizeSubscription: Subscription;

    // shell state
    private shellStateSubject: BehaviorSubject<ShellState> = new BehaviorSubject<ShellState>({loading: true, disconnected: false});
    public shellStateData: Observable<ShellState> = this.shellStateSubject.asObservable();

    constructor(private configService: ConfigService, connectionUrl: string, inputStream: BehaviorSubject<string>, resizeStream: BehaviorSubject<TerminalSize>)
    {
        this.connectionUrl = connectionUrl;

        this.inputSubscription = inputStream.asObservable().subscribe(
            async (data) => 
            {   
                if(this.websocket && this.websocket.connectionId)
                    this.websocket.invoke(
                        ShellHubOutgoingMessages.shellInput,
                        {Data: data}
                    );
            }
        );

        this.resizeSubscription = resizeStream.asObservable().subscribe(
            async (terminalSize) => {
                if(this.websocket && this.websocket.connectionId)
                    this.websocket.invoke(
                        ShellHubOutgoingMessages.shellGeometry,
                        terminalSize
                    );
            }
        );
    }

    public start(rows: number, cols: number) // take in terminal size?
    {
        this.websocket = this.createConnection();

        this.websocket.start().then(_ => {
            // start the shell terminal on the backend, use current terminal dimensions
            // to setup the terminal size
            this.sendShellConnect(rows, cols);
        });

        this.websocket.on(
            ShellHubIncomingMessages.shellOutput,
            req => 
            {   
                // ref: https://git.coolaj86.com/coolaj86/atob.js/src/branch/master/node-atob.js
                var decodedOutput = Buffer.from(req.data, 'base64');
                this.outputSubject.next(decodedOutput);
            }
        );
        this.websocket.on(
            ShellHubIncomingMessages.shellStart, 
            () => this.shellStateSubject.next({loading: false, disconnected: false})
        );
        this.websocket.on(
            ShellHubIncomingMessages.shellDisconnect,
            () => this.shellStateSubject.next({loading: false, disconnected: true})
        );
    }

    public sendShellConnect(rows: number, cols: number)
    {
        if(this.websocket && this.websocket.connectionId)
            this.websocket.invoke(
                ShellHubOutgoingMessages.shellConnect, 
                { TerminalRows: rows, TerminalColumns: cols }
            );
    }


    private createConnection(): HubConnection {
        const connectionBuilder = new HubConnectionBuilder();
        connectionBuilder.withUrl(
            this.connectionUrl, 
            { headers: {authorization: this.configService.getAuthHeader() } }
        );
    
        return connectionBuilder.build();
    }
    
    private destroyConnection() {
        if(this.websocket) {
            this.websocket.stop(); // maybe await on this for server not to complain
            this.websocket = undefined;
        }
    }

    public dispose() : void
    {
        this.shellStateSubject.next({loading: false, disconnected: true});
        this.resizeSubscription.unsubscribe();
        this.inputSubscription.unsubscribe();
        this.destroyConnection();
    }
}