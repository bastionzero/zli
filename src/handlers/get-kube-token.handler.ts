
import { ConfigService } from '../config.service/config.service';


export async function getKubeTokenHandler(
    configService: ConfigService
) {
    var kubeConfig = configService.getKubeConfig();
    if (kubeConfig == undefined) {
        throw new Error('Uninitialized zli Kube config')
    }
    console.log(kubeConfig['token'])
}