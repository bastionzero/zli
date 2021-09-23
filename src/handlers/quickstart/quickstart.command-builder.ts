import yargs from 'yargs';
import os from 'os';

export type quickstartArgs = { sshConfigFile: string }

export const defaultSshConfigFilePath = `${os.homedir()}/.ssh/config`;

export function quickstartCmdBuilder(yargs: yargs.Argv<{}>): yargs.Argv<quickstartArgs> {
    return yargs
        .option(
            'sshConfigFile',
            {
                type: 'string',
                demandOption: false,
                alias: 'c',
                describe: `Path to ssh config file [default: ${defaultSshConfigFilePath}]`,
                default: undefined
            }
        );
}