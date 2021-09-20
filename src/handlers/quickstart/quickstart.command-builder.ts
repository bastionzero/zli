import yargs from 'yargs';
import os from 'os';
import fs from 'fs';

export type quickstartArgs = { sshConfigFile: string }

const defaultSshConfigFilePath = `${os.homedir()}/.ssh/config`;

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
        )
        .check(function (argv) {
            if (argv.sshConfigFile === undefined) {
                // User did not pass in sshConfigFile parameter. Use default parameter
                argv.sshConfigFile = defaultSshConfigFilePath;
                if (! fs.existsSync(argv.sshConfigFile)) {
                    throw new Error(`Cannot read/access file at default path: ${argv.sshConfigFile}\nUse \`zli quickstart --sshConfigFile <filePath>\` to read a different file`);
                }
            } else {
                // User passed in sshConfigFile
                if (! fs.existsSync(argv.sshConfigFile)) {
                    throw new Error(`Cannot read/access file at path: ${argv.sshConfigFile}`);
                }
            }

            return true;
        });
}