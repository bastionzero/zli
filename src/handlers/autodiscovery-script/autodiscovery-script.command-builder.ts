import yargs from 'yargs';

export type autoDiscoveryScriptArgs = {operatingSystem: string} &
{targetName: string} &
{environmentName: string} &
{agentVersion: string} &
{outputFile: string}

export function autoDiscoveryScriptCommandBuilder(yargs : yargs.Argv<{}>) : yargs.Argv<autoDiscoveryScriptArgs> {
    return yargs
        .positional('operatingSystem', {
            type: 'string',
            choices: ['centos', 'ubuntu']
        })
        .positional('targetName', {
            type: 'string'
        })
        .positional('environmentName', {
            type: 'string',
        })
        .positional('agentVersion', {
            type: 'string',
            default: 'latest'
        })
        .option(
            'outputFile',
            {
                type: 'string',
                demandOption: false,
                alias: 'o'
            }
        )
        .example('$0 autodiscovery-script centos sample-target-name Default', '');
}