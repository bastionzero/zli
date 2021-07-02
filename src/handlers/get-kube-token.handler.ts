import { config } from 'yargs';
import { ConfigService } from '../config.service/config.service';


const ONE_HOUR = 60 * 60 * 1000; /* ms */

export async function getKubeTokenHandler(
    configService: ConfigService
) {
    var kubeConfig = configService.getKubeConfig();
    if (kubeConfig == undefined) {
        throw new Error('Uninitialized zli Kube config')
    }
    console.log(kubeConfig['token'])
}