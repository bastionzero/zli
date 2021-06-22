import { ConfigService } from '../config.service/config.service';


const ONE_HOUR = 60 * 60 * 1000; /* ms */

export async function getKubeTokenHandler(
    configService: ConfigService
) {
    let experationTime = (new Date(Date.now() + ONE_HOUR)).toISOString();

    let token = {
        'kind': 'ExecCredential', 
        'apiVersion': 'client.authentication.k8s.io/v1alpha1',
        'spec': {},
        'status': {
            'expirationTimestamp': experationTime,
            'token': '1234',
            'user': 'test-user'
        }
    }
    console.log(token);
}