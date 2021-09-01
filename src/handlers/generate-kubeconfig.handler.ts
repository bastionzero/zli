import util from 'util';

import { Logger } from '../logger.service/logger';
import { ConfigService } from '../config.service/config.service';

const path = require('path');
const fs = require('fs');
const pem = require('pem');

export async function generateKubeconfigHandler(
    argv: any,
    configService: ConfigService,
    logger: Logger
) {
    // Check if we already have generated a cert/key
    let kubeConfig = configService.getKubeConfig();

    if (kubeConfig == undefined) {
        logger.info('No KubeConfig has been generated before, generating key and cert for local daemon...');

        // Create and save key/cert
        const createCertPromise = new Promise<void>(async (resolve, reject) => {
            // Define pem here as if the config has already been created, this codeblock will never be executed
            pem.createCertificate({ days: 999, selfSigned: true }, async function (err: any, keys: any) {
                if (err) {
                    throw err;
                }

                // Get the path of where we want to save
                const pathToConfig = path.dirname(configService.configPath());
                const pathToKey = `${pathToConfig}/kubeKey.pem`;
                const pathToCert = `${pathToConfig}/kubeCert.pem`;

                // Now save the key and cert
                await fs.writeFile(pathToKey, keys.serviceKey, function (err: any) {
                    if (err) {
                        logger.error('Error writing key to file!');
                        reject();
                        return;
                    }
                    logger.debug('Generated and saved key file');
                });

                await fs.writeFile(pathToCert, keys.certificate, function (err: any) {
                    if (err) {
                        logger.error('Error writing cert to file!');
                        reject();
                        return;
                    }
                    logger.debug('Generated and saved cert file');
                });

                // Generate a token that can be used for auth
                const randtoken = require('rand-token');
                const token = randtoken.generate(128) + '++++';

                // Find an open port, define it here as if the config has already been created, this codeblock will never be executed
                const findPort = require('find-open-port');
                const localPort = new Promise<number>(async (resolve, reject) => {
                    findPort().then((port: any) => {
                        resolve(port);
                    });
                });

                // Now save the path in the configService
                kubeConfig = {
                    keyPath: pathToKey,
                    certPath: pathToCert,
                    token: token,
                    localHost: 'localhost',
                    localPort: await localPort,
                    localPid: null,
                    assumeRole: null,
                    assumeCluster: null
                };
                configService.setKubeConfig(kubeConfig);
                resolve();
            });
        });

        // TODO: try/catch block
        await createCertPromise;
    }

    // See if the user passed in a custom port
    var daemonPort = kubeConfig['localPort'].toString();
    if (argv.customPort != -1) {
        daemonPort = argv.customPort.toString();
    }

    // Now generate a kubeConfig
    const clientKubeConfig = `
apiVersion: v1
clusters:
- cluster:
    server: https://${kubeConfig['localHost']}:${daemonPort}
    insecure-skip-tls-verify: true
  name: bctl-server
contexts:
- context:
    cluster: bctl-server
    user: ${configService.me()['email']}
  name: bctl-server
current-context: bctl-server
preferences: {}
users:
  - name: ${configService.me()['email']}
    user:
      token: "${kubeConfig['token']}"
    `;

    // Show it to the user or write to file
    if (argv.outputFile) {
        await util.promisify(fs.writeFile)(argv.outputFile,clientKubeConfig);
    } else {
        logger.info(clientKubeConfig);
    }
}