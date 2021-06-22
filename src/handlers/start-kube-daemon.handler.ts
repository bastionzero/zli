import {
    HubConnection,
    HubConnectionBuilder,
    HubConnectionState,
  } from "@microsoft/signalr";
const http = require('http');
const https = require('https')

import { ConfigService } from '../config.service/config.service';

// Define our ports and host
const HOSTNAME = 'localhost';
const PORT = 1234;


// Define our server URL 
const BCTL_SERVER = 'bctl-server.bastionzero.com'


export async function startKubeDaemonHandler(configService: ConfigService) {
    const server = http.createServer(async (req: any, res: any) => {
        console.log(`Handling request for ${req.url}`);

        // Build our websocket
        const connectionBuilder = new HubConnectionBuilder();
        connectionBuilder.withUrl(
        `${configService.serviceUrl()}api/v1/hub/kube-connect?session_id=${configService.sessionId()}`,
        {
            accessTokenFactory: () => configService.getAuth(),
        }
        );
        let connection = connectionBuilder.build();

        // Add our handlers
        connection.on('ReceiveDataClient', async (req: any) => {
            console.log('here');
        });

        // Start the websocket
        await connection.start()

        // Modify our headers
        let headers = req.headers;
        headers['Authorization'] = 'Bearer 1234';

        await connection.invoke('SendData', {
            'headers': JSON.stringify(headers),
            'method': req.method,
            'body': JSON.stringify(req.body),
            'path': req.url
        });

    
        // Close the connection
        connection.stop()

        
        // // Modify our headers
        
        // delete headers['host']

        // let options = {
        //     hostname: BCTL_SERVER,
        //     port: 443,
        //     path: req.url,
        //     method: req.method,
        //     headers: headers,
        //     protocol: 'https:'
        // }

        // const httpsRequest = https.request(options, (httpsResponse: any) => {
        //     var body = '';

        //     httpsResponse.on('data', function(chunk: any){
        //         body += chunk;
        //     });
        
        //     httpsResponse.on('end', function () {
        //         // if (httpsResponse.statusCode === 200) {
        //         //     try {
        //         //         // if (req.getHeader('Accept-Encoding') != 'gzip') {
        //         //         //     var data = JSON.parse(body);
        //         //         // }
        //         //         // data is available here:
        //         //     } catch (e) {
        //         //         console.log('Error parsing JSON!');
        //         //     }
        //         // } else {
        //         //     console.log('Status:', httpsResponse.statusCode);
        //         // }
        //         if (httpsResponse.statusCode != 200) {
        //             console.log('Status:', httpsResponse.statusCode);
        //         }

        //         // Return the result of the 
        //         res.statusCode = httpsResponse.statusCode;
        //         res.headers = httpsResponse.headers
        //         res.end(body);
        //     });
        // })
        // httpsRequest.on('error', (error: any) => {
        //     console.log('An error', error);
        //     res.statusCode = 500;
        // });
        // httpsRequest.end()
    });

    server.listen(PORT, HOSTNAME, () => {
        console.log(`Kube Daemon running at https://${HOSTNAME}:${PORT}/`);
    });                                 
}