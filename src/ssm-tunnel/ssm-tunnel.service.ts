import util from 'util';
import crypto from 'crypto';
import fs from 'fs';

import SshPK from 'sshpk';
import async from 'async';
import { Observable, Subject } from 'rxjs';
import { HubConnection, HubConnectionBuilder, HubConnectionState } from '@microsoft/signalr';

import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { KeySplittingService } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';

import { SsmTargetService } from '../http.service/http.service';
import { SsmTargetSummary } from '../http.service/http.service.types';
import { SsmTunnelWebsocketService } from '../../webshell-common-ts/ssm-tunnel-websocket.service/ssm-tunnel-websocket.service';
import { ZliAuthConfigService } from '../config.service/zli-auth-config.service';
import { SsmTunnelTargetInfo } from '../../webshell-common-ts/ssm-tunnel-websocket.service/ssm-tunnel-websocket.types';

export class SsmTunnelService
{
    private ssmTunnelWebsocketService: SsmTunnelWebsocketService;
    private sendQueue: async.QueueObject<Buffer>;
    private errorSubject: Subject<string> = new Subject<string>();
    public errors: Observable<string> = this.errorSubject.asObservable();

    constructor(
        private logger: Logger,
        private configService: ConfigService,
        private keySplittingService: KeySplittingService
    )
    {
        // https://caolan.github.io/async/v3/docs.html#queue
        this.sendQueue = async.queue(async (data: Buffer, cb) => {
            await this.ssmTunnelWebsocketService.sendData(data);
            cb();
        });
    }

    public async setupWebsocketTunnel(
        hostName: string,
        userName: string,
        port: number,
        identityFile: string
    ) : Promise<boolean> {
        try {
            let target = await this.getSsmTargetFromHostString(hostName);

            this.ssmTunnelWebsocketService = new SsmTunnelWebsocketService(
                this.logger,
                this.keySplittingService,
                new ZliAuthConfigService(this.configService),
                target as SsmTunnelTargetInfo
            );

            // Forward errors from the SsmTunnelWebsocketService
            this.ssmTunnelWebsocketService.errors.subscribe(err => this.errorSubject.next(err));

            await this.setupEphemeralSshKey(identityFile);
            let pubKey = await this.extractPubKeyFromIdentityFile(identityFile);

            await this.ssmTunnelWebsocketService.setupWebsocketTunnel(userName, port, pubKey);

            return true;
        } catch(err) {
            this.logger.error(err);
            this.errorSubject.next(err);
            return false;
        }
    }

    public sendData(data: Buffer) {
        this.sendQueue.push(data);
    }

    private async setupEphemeralSshKey(identityFile: string): Promise<void> {
        let bzeroSshKeyPath = this.configService.sshKeyPath();

        // Generate a new ssh key for each new tunnel as long as the identity
        // file provided is managed by bzero
        // TODO #39: Change the lifetime of this key?
        if(identityFile === bzeroSshKeyPath) {
            let privateKey = await this.generateEphemeralSshKey();
            await util.promisify(fs.writeFile)(bzeroSshKeyPath, privateKey, {
                mode: '0600'
            });
        }
    }

    private async generateEphemeralSshKey() : Promise<string> {
        // Generate a new ephemeral key to use
        this.logger.info('Generating an ephemeral ssh key');

        let { publicKey, privateKey } = await util.promisify(crypto.generateKeyPair)('rsa', {
            modulusLength: 4096,
            publicKeyEncoding: {
                type: 'spki',
                format: 'pem'
            },
            privateKeyEncoding: {
                type: 'pkcs1',
                format: 'pem'
            }
        });

        return privateKey;
    }

    private async extractPubKeyFromIdentityFile(identityFileName: string): Promise<SshPK.Key> {
        let identityFile = await this.readIdentityFile(identityFileName);

        // Use ssh-pk library to convert the public key to ssh RFC 4716 format
        // https://stackoverflow.com/a/54406021/9186330
        // https://github.com/joyent/node-sshpk/blob/4342c21c2e0d3860f5268fd6fd8af6bdeddcc6fc/lib/key.js#L234
        return SshPK.parseKey(identityFile, 'auto');
    }

    private async readIdentityFile(identityFileName: string): Promise<string> {
        return util.promisify(fs.readFile)(identityFileName, 'utf8');
    }

    private async getSsmTargetFromHostString(host: string): Promise<SsmTargetSummary> {
        let prefix = 'bzero-';

        if(! host.startsWith(prefix)) {
            throw new Error(`Invalid host provided must have form ${prefix}<target>. Target must be either target id or name`);
        }

        let targetString = host.substr(prefix.length);

        const ssmTargetService = new SsmTargetService(this.configService, this.logger);
        let ssmTargets = await ssmTargetService.ListSsmTargets(true);

        const guidPattern = /^[0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}$/;
        if(guidPattern.test(targetString)) {
            // target id
            let targetId = targetString;
            if(! ssmTargets.some(t => t.id == targetId)) {
                throw new Error(`No ssm target exists with id ${targetId}`);
            }

            return ssmTargets.filter(t => t.id == targetId)[0];
        } else {
            // target name
            let targetName = targetString;
            let matchedTarget = ssmTargets.filter(t => t.name == targetName);

            if(matchedTarget.length == 0) {
                throw new Error(`No ssm target exists with name ${targetName}`);
            }

            if(matchedTarget.length > 1) {
                throw new Error(`Multiple targets found with name ${targetName}`);
            }

            return matchedTarget[0];
        }
    }
}