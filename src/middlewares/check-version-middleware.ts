import chalk from 'chalk';
import got from 'got/dist/source';
import { SemVer } from 'semver';

import { name as appName, version } from '../../package.json';
import { Logger } from '../../src/logger.service/logger';

export async function checkVersionMiddleware(logger: Logger) {
    await new CheckVersionMiddleware().checkNewVersion(logger);
}

interface ManifestFile {
    version: string;
}

const downloadLinks = `
MacOS:      https://download-cli.clunk80.com/release/latest/bin/zli-macos
Linux:      https://download-cli.clunk80.com/release/latest/bin/zli-linux
Windows:    https://download-cli.clunk80.com/release/latest/bin/zli-win.exe`;

class CheckVersionMiddleware {
    constructor() {}

    public async checkNewVersion(logger: Logger) {
        let manifestFile = await this.getManifestFile();

        let latestVersion = new SemVer(manifestFile.version);
        let currentVersion = new SemVer(version);

        if (latestVersion > currentVersion) {
            logger.warn(`New version of ${appName} available: ${latestVersion} (current version ${currentVersion})`);
        }

        if(latestVersion.major > currentVersion.major) {
            logger.error(`Version ${currentVersion} is no longer supported. Please download latest version of ${appName}`);
            console.log(chalk.bold(downloadLinks));

            process.exit(1);
        }

    }

    private async getManifestFile() : Promise<ManifestFile> {
        var resp: ManifestFile = await got.get('https://download-cli.clunk80.com/release/latest/MANIFEST').json();
        return resp;
    }
}