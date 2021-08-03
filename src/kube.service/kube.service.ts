import { spawn } from 'child_process';
import { ConfigService } from '../../src/config.service/config.service';

export async function killDaemon(configService: ConfigService) {
    const kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if (kubeConfig['localPid'] != null) {
        // First try to kill the process
        spawn('pkill', ['-P', kubeConfig['localPid'].toString()]);

        // Update the config
        kubeConfig['localPid'] = null;
        configService.setKubeConfig(kubeConfig);
    }
}