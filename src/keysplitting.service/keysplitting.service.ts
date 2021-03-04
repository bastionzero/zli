import Conf from 'conf/dist/source';
import { SHA3 } from 'sha3';
import * as crypto from 'crypto';
const ellipticcurve = require("starkbank-ecdsa");

import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';
import { BZECert } from '../keysplitting-types'

export interface KeySplittingConfigSchema {
    initialIdToken: string,
    cerRand: Buffer, 
    cerRandSig: string,
    privateKey: string,
    publicKey: string
}

export class KeySplittingService {
    private logger: Logger
    private config: ConfigService;
    private publicKey: typeof ellipticcurve.PublicKey;
    private privateKey: typeof ellipticcurve.PrivateKey;
    private data: KeySplittingConfigSchema;


    constructor(config: ConfigService, logger: Logger) {
        this.logger = logger;
        this.config = config;
        this.data = this.config.loadKeySplitting();
        this.init();
    }

    public updateId(idToken: string) {
        this.data.initialIdToken = idToken;
        this.config.updateKeySplitting(this.data);
        this.logger.debug('Updated idToken and latestIdToken');
    }

    public updateLatestId(latestIdToken: string) {
        this.data.initialIdToken = latestIdToken;
        this.config.updateKeySplitting(this.data);
        this.logger.debug('Updated latestIdToken');
    }

    public getConfig() {
        return this.data;
    }

    public createNonce() {
        // Helper function to create a Nonce 
        const hashClient = new SHA3(512);
        const hashString = "".concat(this.data.publicKey, this.data.cerRandSig, this.data.cerRand.toString());
        this.logger.debug(`Creating new nonce: ${hashString}`);

        // Update and return
        hashClient.update(hashString);
        return hashClient.digest('hex');
    }

    public async getBZECert(currentIdToken: string): Promise<BZECert> {
        return {
            InitialIdToken: this.data.initialIdToken,
            CurrentIdToken: currentIdToken,
            ClientPublicKey: this.data.publicKey,
            Rand: this.data.cerRand.toString(),
            SignatureOnRand: this.data.cerRandSig
        }
    }

    public async getBZECertHash(currentIdToken: string): Promise<string> {
        let BZECert = this.getBZECert(currentIdToken);
        const hashClient = new SHA3(512);
        hashClient.update(BZECert.toString());
        return hashClient.digest('hex');
    }

    private init() {
        // Generate our keys and load them in
        this.generateKeys();

        // Generate our cerRan and cerRanSig 
        this.generateCerRand();
    }

    private generateCerRand() {
        // Helper function to generate and store our cerRand and cerRandSig
        var cerRand = crypto.randomBytes(32)
        this.data.cerRand = cerRand;

        var Ecdsa = ellipticcurve.Ecdsa;
        var cerRandSig = Ecdsa.sign(cerRand, this.privateKey);
        this.data.cerRandSig = cerRandSig;

        // Update our config
        this.config.updateKeySplitting(this.data);

        this.logger.debug('Generated new cerRand and cerRandSig');
    }

    private generateKeys() {
        // Helper function to check if keys are undefined and, generate new ones
        var PrivateKey = ellipticcurve.PrivateKey;
        if (this.data.privateKey == undefined) {
            // We need to create and store new keys for the user
            // Create our keys
            this.privateKey = new PrivateKey();
            this.publicKey = this.privateKey.publicKey();

            // Store our keys
            this.data.privateKey = this.privateKey.toPem();
            this.data.publicKey = this.publicKey.toPem();
            this.config.updateKeySplitting(this.data);

            this.logger.debug('Generated new private and public keys');
        } else {
            // We need to load in our keys
            this.privateKey = PrivateKey.fromPem(this.data.privateKey);
            this.publicKey = this.privateKey.publicKey();

            // Validate the public key
            if (this.publicKey.toPem() != this.data.publicKey) {
                let errorString = `Error loading keys, please check your KeySplitting configuration: ${this.config.configPath()}`;
                this.logger.error(errorString);
                throw new Error(errorString);
            }

            this.logger.debug('Loaded in previous keys')
        }
    }
}