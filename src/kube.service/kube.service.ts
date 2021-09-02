import { spawn } from 'child_process';
import { ConfigService } from '../../src/config.service/config.service';

export async function killDaemon(configService: ConfigService) {
    const kubeConfig = configService.getKubeConfig();

    // then kill the daemon
    if (kubeConfig['localPid'] != null) {
        // First try to kill the process
        if (process.platform === "win32") {
            spawn('taskkill', ['/F', '/T', '/PID', kubeConfig['localPid'].toString()]);
        } else {
            spawn('pkill', ['-P', kubeConfig['localPid'].toString()]);
        }

        // Update the config
        kubeConfig['localPid'] = null;
        configService.setKubeConfig(kubeConfig);

        return true
    } else {
        return false
    }
}