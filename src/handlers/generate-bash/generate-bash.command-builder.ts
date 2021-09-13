import yargs from 'yargs';

const targetNameSchemes = ['do', 'aws', 'time', 'hostname'] as const;
export type TargetNameScheme = typeof targetNameSchemes[number];

export type generateBashArgs = { environment: string } &
{ targetNameScheme: TargetNameScheme } &
{ agentVersion: string } &
{ os: string } &
{ targetName: string } &
{ outputFile: string }

export function generateBashCmdBuilder(processArgs : string[], yargs: yargs.Argv<{}>): yargs.Argv<generateBashArgs> {
    return yargs
        .option(
            'environment',
            {
                type: 'string',
                demandOption: false,
                alias: 'e',
                default: 'Default',
                describe: 'Specifies the target\'s environment',
            }
        )
        .option(
            'targetNameScheme',
            {
                demandOption: false,
                choices: targetNameSchemes,
                default: 'hostname' as TargetNameScheme,
                conflicts: 'targetName',
                describe: 'Configures the target name from a specific source. Flag cannot be used with --targetName',
            }
        )
        .option(
            'agentVersion',
            {
                type: 'string',
                demandOption: false,
                default: 'latest',
                describe: 'Use a specific version of the agent',
            }
        )
        .option(
            'os',
            {
                type: 'string',
                demandOption: false,
                choices: ['centos', 'ubuntu', 'universal'],
                default: 'universal',
                describe: 'Assume a specific operating system',
            }
        )
        .option(
            'targetName',
            {
                type: 'string',
                demandOption: false,
                conflicts: 'targetNameScheme',
                alias: 'n',
                default: undefined,
                describe: 'Set the target name explicitly. Flag cannot be used with --targetNameScheme'
            }
        )
        .option(
            'outputFile',
            {
                type: 'string',
                demandOption: false,
                alias: 'o',
                describe: 'Write the script to a file'
            }
        )
        .check(function (argv) {
            if (processArgs.find(arg => new RegExp('targetNameScheme').test(arg)) === undefined && argv.targetName !== undefined) {
                // If user did not pass --targetNameScheme but
                // did pass something for the targetName flag

                // We must look at process.argv, and not
                // yargs.argv, because yargs.argv does not have
                // a way to check if user actually passed in
                // something for a flag that has a default !=
                // undefined.

                // We must override the default of "hostname"
                // and set it to undefined so that the
                // targetName flag can be passed in by itself
                // and not run into "mutually exclusive"
                // problem with defaults. See:
                // https://github.com/yargs/yargs/issues/929
                argv.targetNameScheme = undefined;
            }
            return true;
        })
        .example('generate-bash --targetName my-target -e my-custom-env', '')
        .example('generate-bash --targetNameScheme time', '')
        .example('generate-bash -o script.sh', 'Writes the script to a file "script.sh" in the current directory');
}