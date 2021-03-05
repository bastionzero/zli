import { Logger } from '../logger.service/logger';
import { ConfigInterface } from '../../webshell-common-ts/keysplitting.service/keysplitting.service.types';
import { KeySplittingBase } from '../../webshell-common-ts/keysplitting.service/keysplitting.service';

export class KeySplittingService extends KeySplittingBase {
    private logger: Logger

    constructor(config: ConfigInterface, logger: Logger) {
        super(config);
        this.logger = logger;
    }

    public reset() {
        super.reset();
        this.logger.debug('Reset keysplitting service');
    }

    public updateId(idToken: string) {
        super.updateId(idToken);
        this.logger.debug('Updated idToken and latestIdToken');
    }

    public updateLatestId(latestIdToken: string) {
        super.updateLatestId(latestIdToken);
        this.logger.debug('Updated latestIdToken');
    }

    public createNonce() {
        let nonce = super.createNonce();
        this.logger.debug(`Creating new nonce: ${nonce}`);
        return nonce;
    }

    public async init() {
        // Generate our keys and load them in
        this.generateKeys();

        // Generate our cerRan and cerRanSig 
        await this.generateCerRand();
    }

    public async generateCerRand() {
        super.generateCerRand();
        this.logger.debug('Generated or loaded cerRand and cerRandSig');
    }

    public generateKeys() {
        super.generateKeys();
        this.logger.debug('Loaded keysplitting keys')
    }
}