import util from 'util';
import crypto from 'crypto';
import fs from 'fs';

import SshPK from 'sshpk';
import async from 'async';
import { Observable, Subject } from 'rxjs';

import { KeySplittingService } from '../../../webshell-common-ts/keysplitting.service/keysplitting.service';
import { SsmTargetInfo } from '../../../webshell-common-ts/keysplitting.service/keysplitting-types';

import { Logger } from '../logger/logger.service';
import { ConfigService } from '../config/config.service';

import { SsmTunnelWebsocketService } from '../../../webshell-common-ts/ssm-tunnel-websocket.service/ssm-tunnel-websocket.service';
import { ZliAuthConfigService } from '../config/zli-auth-config.service';
import { ParsedTargetString } from '../common.types';
import { SsmTargetService } from '../ssm-target/ssm-target.service';

export class SsmTunnelService
{
    private ssmTunnelWebsocketService: SsmTunnelWebsocketService;
    private sendQueue: async.QueueObject<Buffer>;
    private errorSubject: Subject<string> = new Subject<string>();
    public errors: Observable<string> = this.errorSubject.asObservable();

    constructor(
        private logger: Logger,
        private configService: ConfigService,
        private keySplittingService: KeySplittingService,
        private keysplittingEnabled: boolean
    )
    {
        // https://caolan.github.io/async/v3/docs.html#queue
        this.sendQueue = async.queue(async (data: Buffer, cb) => {
            await this.ssmTunnelWebsocketService.sendData(data);
            cb();
        });

        if(keysplittingEnabled) {
            this.logger.info('Keysplitting Enabled! Will attempt keysplitting on all agents that return agent version!');
        } else {
            this.logger.info('Keysplitting Disabled!');
        }
    }

    public async setupWebsocketTunnel(
        parsedTarget: ParsedTargetString,
        port: number,
        identityFile: string
    ) : Promise<boolean> {
        try {
            // target is ssmtargetsummary
            const ssmTargetService = new SsmTargetService(this.configService, this.logger);
            const target = await ssmTargetService.GetSsmTarget(parsedTarget.id);

            this.ssmTunnelWebsocketService = new SsmTunnelWebsocketService(
                this.logger,
                this.keySplittingService,
                new ZliAuthConfigService(this.configService, this.logger),
                target as SsmTargetInfo
            );

            // Forward errors from the SsmTunnelWebsocketService
            this.ssmTunnelWebsocketService.errors.subscribe(err => this.errorSubject.next(err));

            await this.setupEphemeralSshKey(identityFile);
            const pubKey = await this.extractPubKeyFromIdentityFile(identityFile);

            await this.ssmTunnelWebsocketService.setupWebsocketTunnel(parsedTarget.user, port, pubKey, this.keysplittingEnabled);

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

    public async closeTunnel(): Promise<void> {
        await this.ssmTunnelWebsocketService.closeConnection();
    }

    private async setupEphemeralSshKey(identityFile: string): Promise<void> {
        const bzeroSshKeyPath = this.configService.sshKeyPath();

        // Generate a new ssh key for each new tunnel as long as the identity
        // file provided is managed by bzero
        // TODO #39: Change the lifetime of this key?
        if(identityFile === bzeroSshKeyPath) {
            const privateKey = await this.generateEphemeralSshKey();
            await util.promisify(fs.writeFile)(bzeroSshKeyPath, privateKey, {
                mode: '0600'
            });
        }
    }

    private async generateEphemeralSshKey() : Promise<string> {
        // Generate a new ephemeral key to use
        this.logger.info('Generating an ephemeral ssh key');

        const { privateKey } = await util.promisify(crypto.generateKeyPair)('rsa', {
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
        const identityFile = await this.readIdentityFile(identityFileName);

        // Use ssh-pk library to convert the public key to ssh RFC 4716 format
        // https://stackoverflow.com/a/54406021/9186330
        // https://github.com/joyent/node-sshpk/blob/4342c21c2e0d3860f5268fd6fd8af6bdeddcc6fc/lib/key.js#L234
        return SshPK.parseKey(identityFile, 'auto');
    }

    private async readIdentityFile(identityFileName: string): Promise<string> {
        return util.promisify(fs.readFile)(identityFileName, 'utf8');
    }
}