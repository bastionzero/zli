import chalk from "chalk";
import got from "got/dist/source";
import { SemVer } from "semver";
import { thoumError, thoumWarn } from "../cli-driver";

import { name as appName, version } from '../../package.json';

export async function checkVersionMiddleware() {
    await new CheckVersionMiddleware().checkNewVersion();
}

interface ManifestFile {
    version: string;
}

const downloadLinks = `
MacOS:      https://download-cli.clunk80.com/release/latest/bin/thoum-macos
Linux:      https://download-cli.clunk80.com/release/latest/bin/thoum-linux
Windows:    https://download-cli.clunk80.com/release/latest/bin/thoum-win.exe`;

class CheckVersionMiddleware {
    constructor() {}

    public async checkNewVersion() {
        let manifestFile = await this.getManifestFile();

        let latestVersion = new SemVer(manifestFile.version);
        let currentVersion = new SemVer(version);

        if (latestVersion > currentVersion) {
            thoumWarn(`New version of ${appName} available: ${latestVersion} (current version ${currentVersion})`);
        }

        if(latestVersion.major > currentVersion.major) {
            thoumError(`Version ${currentVersion} is no longer supported. Please download latest version of ${appName}`);
            console.log(chalk.bold(downloadLinks));

            process.exit(1);
        }

    }

    private async getManifestFile() : Promise<ManifestFile> {
        var resp: ManifestFile = await got.get("https://download-cli.clunk80.com/release/latest/MANIFEST").json();
        return resp;
    }
}