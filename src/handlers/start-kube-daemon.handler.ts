import {
    HubConnection,
    HubConnectionBuilder,
    HubConnectionState,
    LogLevel,
} from "@microsoft/signalr";
import { ConfigService } from '../config.service/config.service';
import { SignalRLogger } from "../../webshell-common-ts/logging/signalr-logger";
import { Logger } from "../logger.service/logger";
import { v4 as uuidv4 } from 'uuid';


const https = require('https');
const fs = require('fs');
const WebSocket = require('ws');
const spdy = require('spdy')
const spdyp2p = require('libp2p-spdy')

const pull = require('pull-stream')
const toPull = require('stream-to-pull-stream')
const tcp = require('net')


export async function startKubeDaemonHandler(connectUser: string, connectCluster: string, configService: ConfigService, logger: Logger) {
    // First build a connection and open a websocket
    var connection = await buildWebsocket(configService, connectUser, logger);

    var kubeConfig = configService.getKubeConfig()
    if (kubeConfig == undefined) {
        throw new Error('Uninitialized zli Kube config')
    }

    // const options = {
    //     key: fs.readFileSync(kubeConfig['keyPath']),
    //     cert: fs.readFileSync(kubeConfig['certPath'])
    // };
    var options = {
        // Private key
        key: fs.readFileSync(kubeConfig['keyPath']),
       
        // Fullchain file or cert file (prefer the former)
        cert: fs.readFileSync(kubeConfig['certPath']),
       
        // **optional** SPDY-specific options
        spdy: {
          protocols: ['spdy/3.1'],
          plain: false,
       
          // **optional**
          // Parse first incoming X_FORWARDED_FOR frame and put it to the
          // headers of every request.
          // NOTE: Use with care! This should not be used without some proxy that
          // will *always* send X_FORWARDED_FOR
          'x-forwarded-for': true,
       
          connection: {
            windowSize: 1024 * 1024, // Server's window size
       
            // **optional** if true - server will send 3.1 frames on 3.0 *plain* spdy
            autoSpdy31: false
          }
        }
      };
       
    
    const token = kubeConfig['token']

    // Set up our server and websocket handlers
    const server = spdy.createServer(options, async (req: any, res: any) => {  // spdy.createServer
        // Verify our token, first extract the token from the headers
        var headers = req.headers;
        let tokenAndCommand = headers['authorization'].split('++++');

        // Remove the bearer and add the ++++ so we can perform our check
        let tokenToBeVerified = `${tokenAndCommand[0].replace('Bearer ', '')}++++`;

        if (tokenToBeVerified != token) {
            // Reject the request if the token do not match to prevent port rebinding attacks
            logger.error(`Unverified token passed: ${token}`)
            res.statusCode = 401
            res.end();
            return;
        }

        console.log(`Handling ${req.method} for ${req.url}`);

        // if (req.url.includes('exec')) {
        //     console.log('Handling exec');
            // console.log(req.isSpdy);

            // var connSocket = req.socket;
            // const socket = tcp.connect(9999, '10.0.1.12')
            // console.log(connSocket);

            // const listener = spdyp2p.listener(toPull(connSocket))

            // // console.log(listener);
            // // const conn = listener.newStream((err: any, connSocket: any) => { })
            // // console.log(conn);
            // // conn.on('error', (err: any) => {console.log('here? errior?')})

            // listener.on('Stdin', (stream: any) => {
            //     console.log('INCOMMING MUXED STREAM - stdin')
            //     console.log(stream)
            // })

            // listener.on('stream', (stream: any) => {
            //     console.log('INCOMMING MUXED STREAM - stream')
            //     pull(
            //         stream,
            //         pull.drain((data: any) => {
            //           console.log('DO I GET DATA?', data)
            //         })
            //       )
            // })
            
            // listener.emit('stdin', 'test-stdin');
            // listener.emit('stdout', 'test-stdout');

            // await new Promise(resolve => setTimeout(resolve, 3));

            // const conn = listener.newStream((err: any, conn: any) => {})

            // conn.on('error', (err: any) => {})
            // var stream = res.push(req.url, {
            //     request: {
            //       accept: '*/*'
            //     },
            //     response: {
            //       'content-type': 'application/json'
            //     }
            // });
            // stream.on('error', function() {
            // });
            // stream.end('alert("hello from push stream!");');
        
            // res.end('<script src="/main.js"></script>');
            // res.end();

        //     res.statusCode = 101
        //     res.setHeader("Upgrade", "SPDY/3.1")
        //     res.setHeader("X-Stream-Protocol-Version", "v4.channel.k8s.io")
        //     res.end()
        //     return; 
        // }
        
        // Generate a unique requestIdentifier
        const requestIdentifier = await generateUniqueId();

        // Extract the command bring run from the token if present 
        let commandBeingRun = 'N/A';
        if (tokenAndCommand[1] != '') {
            commandBeingRun = tokenAndCommand[1];
        }

        // extract the log uniqudId from the bearer token if present,
        var logId = undefined;
        if (tokenAndCommand.length == 3) {
            logId = tokenAndCommand[2];
        } else {
            // Else Generate a unique logId
            logId = await generateUUID()
        }
        
        // Add our handlers
        connection.on('DataToClient', async (req: any) => {
            // Wait for our requestIdentifier message to come back
            if (req.requestIdentifier == requestIdentifier) {
                try {
                    console.log(`GOT RESPONSE: ${req.headers}`)
                } catch {
                }
                res.statusCode = req.statusCode;

                var headersParsed = JSON.parse(req.headers);

                
                for (var key in headersParsed) {
                    res.setHeader(key, headersParsed[key])
                }
                res.end(req.content);
            }
        });

        // Remove the token from the Auth header, and add the command if it exists
        delete headers['authorization']


        await connection.invoke('DataFromClient', {
            'headers': headers,
            'method': req.method,
            'body': JSON.stringify(req.body),
            'endpoint': req.url,
            'requestIdentifier': requestIdentifier,
            'kubeCommand': commandBeingRun,
            'logId': logId
        });
    });

    // Start the websocket
    await connection.start()

    // Start the server
    const host =kubeConfig['localHost']
    const port = kubeConfig['localPort'].toString()
    server.listen(port, host, () => {
        logger.info(`Kube Daemon running at https://${host}:${port}/ for ${connectCluster} for role ${connectUser}`);
    });                       
    
    // const server = https.createServer({
    //     key: fs.readFileSync(kubeConfig['keyPath']),
    //     cert: fs.readFileSync(kubeConfig['certPath'])
    // });



    // const wss = new WebSocket.Server({ server });

    // wss.on('connection', function connection(ws: any) {
    //     console.log('connected');

    //     ws.on('message', function incoming(message: any) {
    //         console.log('received: %s', message);
    //     });


    // });

    // server.listen(1234);
    // var server = spdy.createServer(options, function (req: any, res: any) {
    //     console.log(req.url)
    //     console.log(req.method)
    //     res.writeHead(200);
    //     res.end('hello world!');
    // });
       
    // server.listen(1234);

}

async function generateUUID(): Promise<string> {
    return uuidv4();
}

async function generateUniqueId(): Promise<number> {
    // Helper function to generate a uniqueId
    // ret number: unique number
    return Math.round(Math.random() * 1000);
}

async function buildWebsocket(configService: ConfigService, connectUser: string, logger: Logger): Promise<HubConnection> {
    const connectionBuilder = new HubConnectionBuilder();
    connectionBuilder.withUrl(
    `${configService.serviceUrl()}api/v1/hub/kube?session_id=${configService.sessionId()}&assume_role=${connectUser}`,
    {
        accessTokenFactory: () => configService.getAuth(),
    }
    )
    .configureLogging(new SignalRLogger(logger))
    .withAutomaticReconnect()
    .configureLogging(LogLevel.None);
    return connectionBuilder.build();
}

// function convert(obj: any) {
//     return Object.keys(obj).map(key => ({
//         name: key,
//         value: obj[key],
//         type: "foo"
//     }));
// }