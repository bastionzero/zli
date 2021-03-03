import Conf from 'conf/dist/source';
const ellipticcurve = require("starkbank-ecdsa");

import { Logger } from '../../src/logger.service/logger';

type KeySplittingConfigSchema = {
    idToken: string, 
    latestIdToken: string,
    cerRand: string, 
    cerRandSig: string,
    privateKey: string,
    publicKey: string
}

export class KeySplittingService {
    private logger: Logger
    private config: Conf<KeySplittingConfigSchema>;
    private publicKey: typeof ellipticcurve.PublicKey;
    private privateKey: typeof ellipticcurve.PrivateKey;


    constructor(configName: string, logger: Logger) {
        this.logger = logger;

        this.config = new Conf<KeySplittingConfigSchema>({
            projectName: 'bastionzero-keysplitting-zli',
            configName: configName, // prod, stage, dev
            defaults: {
                idToken: undefined,
                latestIdToken: undefined,
                cerRand: undefined, 
                cerRandSig: undefined,
                privateKey: undefined,
                publicKey: undefined
            },
            accessPropertiesByDotNotation: true,
            clearInvalidConfig: true    // if config is invalid, delete
        });

        this.generateKeys();
    }

    public updateId(idToken: string) {
        this.config.set('idToken', idToken);
        this.config.set('latestIdToken', idToken);
        this.logger.debug('Updated idToken and latestIdToken');
    }

    public updateLatestId(latestIdToken: string) {
        this.config.set('latestIdToken', latestIdToken);
        this.logger.debug('Updated latestIdToken');
    }

    private generateKeys() {
        // Helper function to check if keys are undefined and, generate new ones
        var PrivateKey = ellipticcurve.PrivateKey;
        if (this.config.get('privateKey') == undefined) {
            // We need to create and store new keys for the user
            // Create our keys
            this.privateKey = new PrivateKey();
            this.publicKey = this.privateKey.publicKey();

            // Store our keys
            this.config.set('privateKey', this.privateKey.toPem());
            this.config.set('publicKey', this.publicKey.toPem());

            this.logger.debug('Generated new private and public keys');
        } else {
            // We need to load in our keys
            this.privateKey = PrivateKey.fromPem(this.config.get('privateKey'));
            this.publicKey = this.privateKey.publicKey();

            // Validate the public key
            if (this.publicKey.toPem() != this.config.get('publicKey')) {
                let errorString = `Error loading keys, please check your KeySplitting configuration: ${this.config.path}`;
                this.logger.error(errorString);
                throw new Error(errorString);
            }

            this.logger.debug('Loaded in previous keys')
        }
    }
}