import got, { Got, HTTPError } from 'got/dist/source';
import { Dictionary } from 'lodash';
import { ConfigService } from '../config/config.service';
import FormData from 'form-data';
import { Logger } from '../logger/logger.service';
import { cleanExit } from '../../handlers/clean-exit.handler';

const fs = require('fs');

export class HttpService
{
    // ref for got: https://github.com/sindresorhus/got
    protected httpClient: Got;
    private baseUrl: string;
    protected configService: ConfigService;
    private authorized: boolean;
    private logger: Logger;

    constructor(configService: ConfigService, serviceRoute: string, logger: Logger, authorized: boolean = true)
    {
        this.configService = configService;
        this.authorized = authorized;
        this.logger = logger;
        this.baseUrl = `${this.configService.serviceUrl()}${serviceRoute}`;

        this.httpClient = got.extend({
            prefixUrl: this.baseUrl,
            // Remember to set headers before calling API
            hooks: {
                beforeRequest: [
                    (options) => this.logger.trace(`Making request to: ${options.url}`)
                ],
                afterResponse: [
                    (response, _) => {
                        this.logger.trace(`Request completed to: ${response.url}`);
                        return response;
                    }
                ]
            }
            // throwHttpErrors: false // potentially do this if we want to check http without exceptions
        });
    }

    private setHeaders()
    {
        const headers: Dictionary<string> = {};

        if(this.authorized) headers['Authorization'] = this.configService.getAuthHeader();
        if(this.authorized && this.configService.sessionId()) headers['X-Session-Id'] = this.configService.sessionId();

        // append headers
        this.httpClient = this.httpClient.extend({ headers: headers });
    }

    private async handleHttpException(route: string, error: HTTPError) : Promise<void>
    {
        this.logger.debug(`Error in ${this.baseUrl}${route}`);
        let errorMessage = error.message;

        if(!error.response) {
            this.logger.error(`HttpService Error:\n${errorMessage}`);
            await cleanExit(1, this.logger);
        }

        // Pull out the specific error message from the back end
        if(error.response.body)
            errorMessage = JSON.stringify(JSON.parse(error.response.body as string), null, 2);

        if(error.response.statusCode === 401) {
            this.logger.error(`Authentication Error:\n${errorMessage}`);
        } else if(error.response.statusCode === 502) {
            this.logger.error('Service is offline');
        } else if(error.response.statusCode === 500) {
            // Handle 500 errors by printing out our custom exception message
            this.logger.error(`Server Error:\n${errorMessage}`);
        } else if(error.response.statusCode === 404) {
            this.logger.error(`Resource not found:\n Status code: 404 at ${error.request.requestUrl}`);
        } else {
            this.logger.error(`Unknown Error:\nStatusCode: ${error.response.statusCode}\n${errorMessage}`);
        }

        await cleanExit(1, this.logger);
    }

    protected getFormDataFromRequest(request: any): FormData {
        return Object.keys(request).reduce((formData, key) => {
            formData.append(key, request[key]);
            return formData;
        }, new FormData());
    }

    protected async Get<TResp>(route: string, queryParams: Dictionary<string>) : Promise<TResp>
    {
        this.setHeaders();

        try {
            const resp : TResp = await this.httpClient.get(
                route,
                {
                    searchParams: queryParams,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
            return resp;
        } catch(error) {
            this.handleHttpException(route, error);
        }
    }

    protected async Post<TReq, TResp>(route: string, body: TReq) : Promise<TResp>
    {
        this.setHeaders();

        try {
            const resp : TResp = await this.httpClient.post(
                route,
                {
                    json: body,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
            return resp;
        } catch(error) {
            this.handleHttpException(route, error);
        }
    }

    protected async FormPostWithException<TReq, TResp>(route: string, body: TReq): Promise<TResp> {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);

        const resp : TResp = await this.httpClient.post(
            route,
            {
                body: formBody
            }
        ).json();
        return resp;
    }

    protected async FormPost<TReq, TResp>(route: string, body: TReq) : Promise<TResp>
    {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);

        try {
            const resp : TResp = await this.httpClient.post(
                route,
                {
                    body: formBody
                }
            ).json();
            return resp;
        } catch (error) {
            this.handleHttpException(route, error);
        }
    }

    // Returns a request object that you can add response handlers to at a higher layer
    protected FormStream<TReq>(route: string, body: TReq, localPath: string) : Promise<void>
    {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);
        const whereToSave = localPath.endsWith('/') ? localPath + `bzero-download-${Math.floor(Date.now() / 1000)}` : localPath;

        return new Promise((resolve, reject) => {
            try {
                const requestStream = this.httpClient.stream.post(
                    route,
                    {
                        isStream: true,
                        body: formBody
                    }
                );

                // Buffer is returned by 'data' event
                requestStream.on('data', (response: Buffer) => {
                    fs.writeFileSync(whereToSave, response);
                });

                requestStream.on('end', () => {
                    this.logger.info('File download complete');
                    this.logger.info(whereToSave);
                    resolve();
                });
            } catch (error) {
                this.handleHttpException(route, error);
                reject(error);
            }
        });
    }
}