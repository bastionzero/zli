import got, { Got, HTTPError } from 'got/dist/source';
import { Dictionary } from 'lodash';
import { ConfigService } from '../config/config.service';
import FormData from 'form-data';
import { Logger } from '../logger/logger.service';

export class HttpService {
    // ref for got: https://github.com/sindresorhus/got
    protected httpClient: Got;
    private baseUrl: string;
    protected configService: ConfigService;
    private authorized: boolean;
    private logger: Logger;

    constructor(configService: ConfigService, serviceRoute: string, logger: Logger, authorized: boolean = true) {
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

    private setHeaders() {
        const headers: Dictionary<string> = {};

        if (this.authorized) headers['Authorization'] = this.configService.getAuthHeader();
        if (this.authorized && this.configService.sessionId()) headers['X-Session-Id'] = this.configService.sessionId();

        // append headers
        this.httpClient = this.httpClient.extend({ headers: headers });
    }

    private getHttpErrorMessage(route: string, error: HTTPError): string {
        this.logger.debug(`Error in ${this.baseUrl}${route}`);
        let errorMessage = error.message;

        if (!error.response) {
            return `HttpService Error:\n${errorMessage}`;
        }

        if (error.response.statusCode === 401) {
            return `Authentication Error:\n${errorMessage}`;
        } else if (error.response.statusCode === 502) {
            return 'Service is offline';
        } else if (error.response.statusCode === 500) {
            // Handle 500 errors by returning our custom exception message
            // Pull out the specific error message from the back end
            if (error.response.body) {
                try {
                    const parsedJSON = JSON.parse(error.response.body as string);
                    errorMessage = JSON.stringify(parsedJSON, null, 2);
                } catch (e) {
                    errorMessage = '';
                }
            }
            return `Server Error:\n${errorMessage}`;
        } else if (error.response.statusCode === 404) {
            return `Resource not found:\n Status code: 404 at ${error.request.requestUrl}`;
        } else {
            return `Unknown Error:\nStatusCode: ${error.response.statusCode}\n${errorMessage}`;
        }
    }

    protected getFormDataFromRequest(request: any): FormData {
        return Object.keys(request).reduce((formData, key) => {
            formData.append(key, request[key]);
            return formData;
        }, new FormData());
    }

    protected async Get<TResp>(route: string, queryParams: Dictionary<string>): Promise<TResp> {
        this.setHeaders();

        try {
            const resp: TResp = await this.httpClient.get(
                route,
                {
                    searchParams: queryParams,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
            return resp;
        } catch (error) {
            throw new Error(this.getHttpErrorMessage(route, error));
        }
    }

    protected async Post<TReq, TResp>(route: string, body: TReq): Promise<TResp> {
        this.setHeaders();

        try {
            const resp: TResp = await this.httpClient.post(
                route,
                {
                    json: body,
                    parseJson: text => JSON.parse(text),
                }
            ).json();
            return resp;
        } catch (error) {
            throw new Error(this.getHttpErrorMessage(route, error));
        }
    }

    protected async FormPostWithException<TReq, TResp>(route: string, body: TReq): Promise<TResp> {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);

        const resp: TResp = await this.httpClient.post(
            route,
            {
                body: formBody
            }
        ).json();
        return resp;
    }

    protected async FormPost<TReq, TResp>(route: string, body: TReq): Promise<TResp> {
        this.setHeaders();

        const formBody = this.getFormDataFromRequest(body);

        try {
            const resp: TResp = await this.httpClient.post(
                route,
                {
                    body: formBody
                }
            ).json();
            return resp;
        } catch (error) {
            throw new Error(this.getHttpErrorMessage(route, error));
        }
    }
}