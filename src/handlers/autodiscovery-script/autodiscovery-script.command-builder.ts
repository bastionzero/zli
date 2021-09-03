import yargs from "yargs";

export function autoDiscoveryScriptCommandBuilder(yargs : yargs.Argv<{}>) :
yargs.Argv<
{operatingSystem: string} &
{targetName: string} &
{environmentName: string} &
{agentVersion: string} &
{outputFile: string}
> {
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
        .example('autodiscovery-script centos sample-target-name Default', '');
}