import { spawn } from 'child_process';
import { ConfigService } from '../../src/config.service/config.service';

export async function killDaemon(configService: ConfigService) {
    const kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if (kubeConfig['localPid'] != null) {
        // First try to kill the process
        if (process.platform === 'win32') {
            spawn('taskkill', ['/F', '/T', '/PID', kubeConfig['localPid'].toString()]);
        } else if (process.platform === 'linux') {
            spawn('pkill', ['-s', kubeConfig['localPid'].toString()]);
        } else {
            spawn('kill', ['-9', kubeConfig['localPid'].toString()]);
        }

        // Update the config
        kubeConfig['localPid'] = null;
        configService.setKubeConfig(kubeConfig);

        return true;
    } else {
        return false;
    }
}