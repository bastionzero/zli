import path from 'path';
import Conf from 'conf/dist/source';
import { error } from 'console';


// All logs are written to a single file, ~~~forever~~~
export type LoggerConfigSchema = {
    logPath: string
}

export class LoggerConfigService {
    private config: Conf<LoggerConfigSchema>;

    constructor(configName: string) {
        this.config = new Conf<LoggerConfigSchema>({
            projectName: 'bastionzero-logger',
            configName: configName,
            defaults: {
                logPath: undefined
            }
        });

        if(! this.config.get('logPath'))
            this.config.set('logPath', this.generateLogPath(configName));
    }

    private generateLogPath(configName: string): string {

        switch (configName) {
        case 'prod':
            return path.join(path.dirname(this.config.path), 'bastionzero-zli.log');

        case 'stage':
            return path.join(path.dirname(this.config.path), 'bastionzero-zli-stage.log');

        case 'dev':
            return path.join(path.dirname(this.config.path), 'bastionzero-zli-dev.log');
        default:
            throw error('Unrecognized configName');
        }
    }

    public logPath(): string {
        return this.config.get('logPath');
    }

    public configPath(): string {
        return this.config.path;
    }
}
